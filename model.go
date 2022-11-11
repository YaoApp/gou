package gou

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/capsule"
)

// Models 已载入模型
var Models = map[string]*Model{}

// SetModelLogger 设定模型 Logger
func SetModelLogger(output io.Writer, level log.Level) {
	log.SetLevel(level)
	log.SetOutput(output)
}

// LoadModelReturn 加载数据模型
func LoadModelReturn(source string, name string) (model *Model, err error) {
	defer func() { err = exception.Catch(recover()) }()
	model = LoadModel(source, name)
	return model, nil
}

// LoadModel 载入数据模型
func LoadModel(source string, name string) *Model {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	metadata := MetaData{}
	err := helper.UnmarshalFile(input, &metadata)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	mod := &Model{
		ID:       name,
		Name:     name,
		Source:   source,
		MetaData: metadata,
	}

	// 解析常用数值
	columns := map[string]*Column{} // 字段映射表
	columnNames := []interface{}{}  // 字段名称清单
	PrimaryKey := "id"              // 字段主键
	uniqueColumns := []*Column{}    // 唯一字段清单

	// 补充字段(软删除)
	if mod.MetaData.Option.SoftDeletes {
		mod.MetaData.Columns = append(mod.MetaData.Columns, Column{
			Label:    "删除标记",
			Name:     "deleted_at",
			Type:     "timestamp",
			Comment:  "删除标记",
			Nullable: true,
		})
	}

	// 补充时间戳(软删除)
	if mod.MetaData.Option.Timestamps {
		mod.MetaData.Columns = append(mod.MetaData.Columns,
			Column{
				Label:    "创建时间",
				Name:     "created_at",
				Type:     "timestamp",
				Comment:  "创建时间",
				Nullable: true,
			},
			Column{
				Label:    "更新时间",
				Name:     "updated_at",
				Type:     "timestamp",
				Comment:  "更新时间",
				Nullable: true,
			},
		)
	}

	for i, column := range mod.MetaData.Columns {
		mod.MetaData.Columns[i].model = mod // 链接所属模型
		columns[column.Name] = &mod.MetaData.Columns[i]
		columnNames = append(columnNames, column.Name)
		if strings.ToLower(column.Type) == "id" || column.Primary == true {
			PrimaryKey = column.Name
		}

		// 唯一字段
		if column.Unique || column.Primary {
			uniqueColumns = append(uniqueColumns, columns[column.Name])
		}
	}

	// 唯一索引
	for _, index := range mod.MetaData.Indexes {
		if strings.ToLower(index.Type) == "unique" {
			for _, name := range index.Columns {
				col, has := columns[name]
				if has {
					uniqueColumns = append(uniqueColumns, col)
				}
			}
		} else if strings.ToLower(index.Type) == "primary" {
			for _, name := range index.Columns {
				col, has := columns[name]
				if has {
					PrimaryKey = col.Name
					uniqueColumns = append(uniqueColumns, col)
				}
			}
		}
	}

	mod.Columns = columns
	mod.ColumnNames = columnNames
	mod.PrimaryKey = PrimaryKey
	mod.UniqueColumns = uniqueColumns
	mod.Driver = capsule.Schema().MustGetConnection().Config.Driver

	Models[name] = mod
	return mod
}

// Reload 更新模型
func (mod *Model) Reload() *Model {
	*mod = *LoadModel(mod.Source, mod.Name)
	return mod
}

// Migrate 数据迁移
func (mod *Model) Migrate(force bool) error {
	if force {
		err := mod.DropTable()
		if err != nil {
			return err
		}
	}

	has, err := mod.HasTable()
	if err != nil {
		return err
	}

	if !has {
		err := mod.CreateTable()
		if err != nil {
			return err
		}

		_, errs := mod.InsertValues()
		if errs != nil && len(errs) > 0 {
			for _, err := range errs {
				log.Error("[Migrate] %s", err.Error())
			}
			return fmt.Errorf("%d values error, please check the logs", len(errs))
		}
		return nil
	}

	return mod.SaveTable()
}

// Select 读取已加载模型
func Select(name string) *Model {
	mod, has := Models[name]
	if !has {
		exception.New(
			fmt.Sprintf("Model:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return mod
}

// Validate 数值校验
func (mod *Model) Validate(row maps.MapStrAny) []ValidateResponse {
	res := []ValidateResponse{}
	for name, value := range row {
		column, has := mod.Columns[name]
		if !has {
			continue
		}

		// 如果允许为 null
		if value == nil && column.Nullable {
			continue
		}

		success, messages := column.Validate(value, row)
		if !success {
			res = append(res, ValidateResponse{
				Column:   column.Name,
				Messages: messages,
			})
		}
	}
	return res
}
