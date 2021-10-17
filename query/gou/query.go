package gou

import (
	"bytes"
	"io"
	"os"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/query/share"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/dbal/query"
)

// cache 查询器解析缓存
var cache = map[interface{}]*Query{}

// Query Query share.DSL
type Query struct {
	QueryDSL
	Query        query.Query
	GetTableName GetTableName
	Bindings     map[string]interface{}
	AESKey       string
}

// GetTableName 读取表格名称
type GetTableName = func(string) string

// Make 创建 Gou Query share.DSL
func Make(input []byte) *Query {

	var dsl QueryDSL
	err := jsoniter.Unmarshal(input, &dsl)
	if err != nil {
		exception.New("DSL 解析失败 %s", 500, err.Error()).Throw()
	}

	errs := dsl.Validate()
	if len(errs) > 0 {
		exception.New("%s", 400, errs[0]).Ctx(errs).Throw()
	}

	return &Query{QueryDSL: dsl}
}

// Read 创建 Gou Query share.DSL (输入接口)
func Read(reader io.Reader) *Query {
	buf := bytes.NewBuffer(nil)
	_, err := io.Copy(buf, reader)
	if err != nil {
		exception.New("读取数据失败 %s", 500, err.Error()).Throw()
	}
	return Make(buf.Bytes())
}

// Open 创建 Gou Query share.DSL (文件)
func Open(filename string) *Query {
	file, err := os.Open(filename)
	if err != nil {
		exception.New("读取文件失败 %s", 500, err.Error()).Throw()
	}
	defer file.Close()
	var reader io.Reader = file
	return Read(reader)
}

// With 关联查询器
func (gou *Query) With(qb query.Query, getTableName ...GetTableName) *Query {
	gou.Query = qb.New()
	if len(getTableName) > 0 {
		return gou.TableName(getTableName[0])
	}
	return gou
}

// Bind 绑定动态数据
func (gou *Query) Bind(data map[string]interface{}) *Query {
	gou.Bindings = data
	return gou
}

// SetAESKey 设定 AES KEY
func (gou *Query) SetAESKey(key string) *Query {
	gou.AESKey = key
	return gou
}

// New 克隆对象
func New() *Query {
	var new Query = Query{}
	return &new
}

// TableName 绑定数据模型数据表读取方式
func (gou *Query) TableName(getTableName GetTableName) *Query {
	gou.GetTableName = getTableName
	return gou
}

// ToSQL 返回查询语句
func (gou Query) ToSQL() string {
	if gou.Query == nil {
		exception.New("未绑定数据连接", 500).Throw()
	}
	return gou.Query.ToSQL()
}

// GetBindings 返回SQL绑定数据
func (gou Query) GetBindings() []interface{} {
	if gou.Query == nil {
		exception.New("未绑定数据连接", 500).Throw()
	}
	return gou.Query.GetBindings()
}

// ==================================================
// share.DSL Interface
// ==================================================

// Load 加载查询条件
func (gou *Query) Load(data interface{}) share.DSL {
	// if query, has := cache[data]; has {
	// 	query.Query = gou.Query.New()
	// 	query.Bindings = map[string]interface{}{}
	// 	return query
	// }

	input, err := jsoniter.Marshal(data)
	if err != nil {
		exception.New("加载失败%s", 500, err.Error()).Throw()
	}

	query := Make(input)
	query.Query = gou.Query.New()
	query.AESKey = gou.AESKey
	query.GetTableName = gou.GetTableName
	errs := query.Validate()
	if len(errs) > 0 {
		exception.New("查询条件错误", 500).Ctx(errs).Throw()
	}
	query.Build()

	// cache[data] = query
	return query
}

// Run 执行查询根据查询条件返回结果
func (gou Query) Run(data maps.Map) interface{} {

	bindings := gou.GetBindings()
	sql := gou.ToSQL()

	for i := range bindings {
		bindings[i] = share.Bind(bindings[i], data)
	}

	qb := gou.Query.New()
	qb.SQL(sql, bindings...).MustGet()

	// utils.Dump(rows)
	// fmt.Println("----", "\n", gou.ToSQL())
	// utils.Dump(bindings)
	return []share.Record{}
}

// Get 执行查询并返回数据记录集合
func (gou Query) Get() []share.Record {
	return []share.Record{}
}

// Paginate 执行查询并返回带分页信息的数据记录数组
func (gou Query) Paginate() share.Paginate {
	return share.Paginate{}
}

// First 执行查询并返回一条数据记录
func (gou Query) First() share.Record {
	return share.Record{}
}
