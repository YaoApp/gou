package gou

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal"
	"github.com/yaoapp/xun/logger"
)

// Models 已载入模型
var Models = map[string]*Model{}

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
		columns[column.Name] = &mod.MetaData.Columns[i]
		columnNames = append(columnNames, column.Name)
		if strings.ToLower(column.Type) == "id" {
			PrimaryKey = column.Name
		}
		// 唯一字段
		if column.Unique {
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
		}
	}

	mod.Columns = columns
	mod.ColumnNames = columnNames
	mod.PrimaryKey = PrimaryKey
	mod.UniqueColumns = uniqueColumns
	Models[name] = mod
	return mod
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

// SetModelLogger 设定模型 Logger
func SetModelLogger(output io.Writer, level logger.LogLevel) {
	logger.DefaultLogger.SetLevel(level)
	logger.DefaultLogger.SetOutput(output)
}

// Validate 数值校验
func (mod *Model) Validate(row maps.MapStrAny) []ValidateResponse {
	res := []ValidateResponse{}
	for name, value := range row {
		column, has := mod.Columns[name]
		if !has {
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

// Reload 更新模型
func (mod *Model) Reload() *Model {
	return LoadModel(mod.Source, mod.Name)
}

// Find 查询单条记录
func (mod *Model) Find(id interface{}, param QueryParam) (maps.MapStr, error) {
	param.Model = mod.Name
	param.Wheres = []QueryWhere{
		{
			Column: mod.PrimaryKey,
			Value:  id,
		},
	}

	stack := NewQueryStack(param)
	res := stack.Run()
	if len(res) <= 0 {
		return nil, fmt.Errorf("ID=%v的数据不存在", id)
	}
	return res[0], nil

}

// MustFind 查询单条记录
func (mod *Model) MustFind(id interface{}, param QueryParam) maps.MapStr {
	res, err := mod.Find(id, param)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return res
}

// Create 创建单条数据, 返回新创建数据ID
func (mod *Model) Create(row maps.MapStrAny) (int, error) {

	errs := mod.Validate(row) // 输入数据校验
	if len(errs) > 0 {
		exception.New("输入参数错误", 400).Ctx(errs).Throw()
	}

	mod.FliterIn(row) // 入库前输入数据预处理

	if mod.MetaData.Option.Timestamps {
		row.Set("created_at", dbal.Raw("NOW()"))
	}

	id, err := capsule.Query().
		Table(mod.MetaData.Table.Name).
		InsertGetID(row)

	if err != nil {
		return 0, err
	}

	return int(id), err
}

// MustCreate 创建单条数据, 返回新创建数据ID, 失败抛出异常
func (mod *Model) MustCreate(row maps.MapStrAny) int {
	id, err := mod.Create(row)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return id
}

// Save 保存单条数据, 返回数据ID
func (mod *Model) Save(row maps.MapStrAny) (int, error) {

	errs := mod.Validate(row) // 输入数据校验
	if len(errs) > 0 {
		exception.New("输入参数错误", 400).Ctx(errs).Throw()
	}

	mod.FliterIn(row) // 入库前输入数据预处理

	// 更新
	if row.Has(mod.PrimaryKey) {

		if mod.MetaData.Option.Timestamps {
			row.Set("updated_at", dbal.Raw("NOW()"))
		}

		id := row.Get(mod.PrimaryKey)
		_, err := capsule.Query().
			Table(mod.MetaData.Table.Name).
			Where(mod.PrimaryKey, id).
			Update(row)

		if err != nil {
			return 0, err
		}

		return any.Of(id).Int(), nil
	}

	// 创建
	if mod.MetaData.Option.Timestamps {
		row.Set("created_at", dbal.Raw("NOW()"))
	}

	id, err := capsule.Query().
		Table(mod.MetaData.Table.Name).
		InsertGetID(row)

	if err != nil {
		return 0, err
	}

	return int(id), err
}

// MustSave 保存单条数据, 返回数据ID, 失败抛出异常
func (mod *Model) MustSave(row maps.MapStrAny) int {
	id, err := mod.Save(row)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return id
}

// Update 按条件更新记录, 返回更新行数
func (mod *Model) Update(param QueryParam, row maps.MapStrAny) (int, error) {

	errs := mod.Validate(row) // 输入数据校验
	if len(errs) > 0 {
		exception.New("输入参数错误", 400).Ctx(errs).Throw()
	}

	mod.FliterIn(row) // 入库前输入数据预处理

	if mod.MetaData.Option.Timestamps {
		row.Set("updated_at", dbal.Raw("NOW()"))
	}

	param.Model = mod.Name
	stack := NewQueryStack(param)
	qb := stack.FirstQuery()
	effect, err := qb.Update(row)
	if err != nil {
		return 0, err
	}

	return int(effect), err
}

// MustUpdate 按条件更新记录, 返回更新行数, 失败抛出异常
func (mod *Model) MustUpdate(param QueryParam, row maps.MapStrAny) int {
	effect, err := mod.Update(param, row)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return effect
}

// Delete 删除单条记录
func (mod *Model) Delete(id interface{}) error {

	// 软删除
	if mod.MetaData.Option.SoftDeletes {
		selects := []interface{}{}
		data := maps.MapStrAny{}
		for _, col := range mod.UniqueColumns {
			selects = append(selects, col.Name)
			typ := strings.ToLower(col.Type)
			if col.Nullable {
				data[col.Name] = nil
			} else if typ == "string" {
				data[col.Name] = dbal.Raw(fmt.Sprintf("CONCAT_WS('_', '%d')", time.Now().UnixNano()))
			}
		}
		row, err := mod.Find(id, QueryParam{Select: selects})
		if err != nil {
			return err
		}

		restore, err := jsoniter.MarshalToString(row)
		if err != nil {
			return err
		}

		data["__restore_data"] = restore
		data["deleted_at"] = dbal.Raw("NOW()")

		_, err = capsule.Query().
			Table(mod.MetaData.Table.Name).
			Where(mod.PrimaryKey, id).
			Update(data)
		if err != nil {
			return err
		}

		return nil
	}

	return mod.Destroy(id)
}

// MustDelete 删除单条记录, 失败抛出异常
func (mod *Model) MustDelete(id interface{}) {
	err := mod.Delete(id)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
}

// Destroy 真删除单条记录
func (mod *Model) Destroy(id interface{}) error {
	_, err := capsule.Query().Table(mod.MetaData.Table.Name).Where("id", id).Delete()
	return err
}

// MustDestroy 真删除单条记录, 失败抛出异常
func (mod *Model) MustDestroy(id interface{}) {
	err := mod.Destroy(id)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
}

// Insert 插入多条数据
func (mod *Model) Insert(rows []maps.MapStrAny) error {
	return nil
}

// Search 按条件检索
func (mod *Model) Search(param QueryParam, page int, pagesize int) (maps.MapStr, error) {
	param.Model = mod.Name
	stack := NewQueryStack(param)
	res := stack.Paginate(page, pagesize)
	return res, nil
}

// MustSearch 按条件检索
func (mod *Model) MustSearch(param QueryParam, page int, pagesize int) maps.MapStr {
	res, err := mod.Search(param, page, pagesize)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return res
}

// Migrate 数据迁移
func (mod *Model) Migrate(force bool) {
	table := mod.MetaData.Table.Name
	schema := capsule.Schema()
	if force {
		schema.DropTableIfExists(table)
	}

	if !schema.MustHasTable(table) {
		mod.SchemaCreateTable()
		return
	}

	mod.SchemaDiffTable()
}
