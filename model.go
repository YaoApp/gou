package gou

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/capsule"
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
	for i, column := range mod.MetaData.Columns {
		columns[column.Name] = &mod.MetaData.Columns[i]
		columnNames = append(columnNames, column.Name)
		if strings.ToLower(column.Type) == "id" {
			PrimaryKey = column.Name
		}
	}

	mod.Columns = columns
	mod.ColumnNames = columnNames
	mod.PrimaryKey = PrimaryKey
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
func (mod *Model) Find(id interface{}, withs ...With) (maps.MapStr, error) {

	qb := capsule.Query().Table(mod.MetaData.Table.Name)
	row, err := qb.
		Where(mod.PrimaryKey, id).
		Select(mod.SelectColumns(mod.MetaData.Table.Name)...).
		First()
	if err != nil {
		return nil, err
	}

	var res maps.MapStr = row.ToMap()
	mod.FliterOut(res)
	return res, nil
}

// SelectColumns 选择字段
func (mod *Model) SelectColumns(alias string, colums ...interface{}) []interface{} {
	if len(colums) == 0 {
		colums = mod.ColumnNames
	}
	return mod.FliterSelect(alias, colums)
}

// MustFind 查询单条记录
func (mod *Model) MustFind(id interface{}, withs ...With) maps.MapStr {
	res, err := mod.Find(id, withs...)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return res
}

// Create 创建单条数据
func (mod *Model) Create(row maps.MapStrAny) (int, error) {

	errs := mod.Validate(row) // 输入数据校验
	if len(errs) > 0 {
		exception.New("输入参数错误", 400).Ctx(errs).Throw()
	}

	mod.FliterIn(row) // 入库前输入数据预处理

	id, err := capsule.Query().
		Table(mod.MetaData.Table.Name).
		InsertGetID(row)

	if err != nil {
		return 0, err
	}

	return int(id), err
}

// MustCreate 创建单条数据, 失败抛出异常
func (mod *Model) MustCreate(row maps.MapStrAny) int {
	id, err := mod.Create(row)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return id
}

// Insert 插入多条数据
func (mod *Model) Insert(rows []maps.MapStrAny) error {
	return nil
}

// Save 保存单条数据
func (mod *Model) Save(row maps.MapStrAny) error {
	return nil
}

// Delete 删除单条记录
func (mod *Model) Delete() {}

// Search 按条件检索
func (mod *Model) Search() {}

// Import 批量导入模型
func (mod *Model) Import() {}

// Export 导出数据模型
func (mod *Model) Export() {}

// Setting 数据模型配置
func (mod *Model) Setting() {}

// List 列表界面配置
func (mod *Model) List() {}

// View 详情界面配置
func (mod *Model) View() {}

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
