package model

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/capsule"
)

// Models 已载入模型
var Models = map[string]*Model{}
var lock sync.Mutex

// LoadSync load model sync
func LoadSync(file string, id string) (*Model, error) {
	lock.Lock()
	defer lock.Unlock()
	return Load(file, id)
}

// LoadSourceSync load model sync
func LoadSourceSync(source []byte, id string, file string) (*Model, error) {
	lock.Lock()
	defer lock.Unlock()
	return LoadSource(source, id, "")
}

// Load load model
func Load(file string, id string) (*Model, error) {
	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}
	return LoadSource(data, id, file)
}

// LoadSource load model from source
func LoadSource(source []byte, id string, file string) (*Model, error) {

	if file == "" {
		file = fmt.Sprintf("__source.%s.mod.yao", id)
	}

	metadata := MetaData{}
	err := application.Parse(file, source, &metadata)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	mod := &Model{
		ID:       id,
		Name:     id,
		File:     file,
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
			Label:    "::Delete At",
			Name:     "deleted_at",
			Type:     "timestamp",
			Comment:  "::Delete At",
			Nullable: true,
		})
	}

	// 补充时间戳(软删除)
	if mod.MetaData.Option.Timestamps {
		mod.MetaData.Columns = append(mod.MetaData.Columns,
			Column{
				Label:    "::Created At",
				Name:     "created_at",
				Type:     "timestamp",
				Comment:  "::Created At",
				Nullable: true,
			},
			Column{
				Label:    "::Updated At",
				Name:     "updated_at",
				Type:     "timestamp",
				Comment:  "::Updated At",
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

	if capsule.Global != nil {
		mod.Driver = capsule.Schema().MustGetConnection().Config.Driver
	}

	Models[id] = mod
	return mod, nil
}

// Reload 更新模型
func (mod *Model) Reload() (*Model, error) {
	new, err := LoadSync(mod.File, mod.ID)
	if err != nil {
		return nil, err
	}
	*mod = *new
	return mod, nil
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
func Select(id string) *Model {
	mod, has := Models[id]
	if !has {
		exception.New(
			fmt.Sprintf("Model:%s; not found", id),
			400,
		).Throw()
	}
	return mod
}

// Exists Check if the model is loaded
func Exists(id string) bool {
	_, has := Models[id]
	return has
}

// Read read the model dsl
func Read(id string) MetaData {
	return Select(id).MetaData
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
