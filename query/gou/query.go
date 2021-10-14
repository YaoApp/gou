package gou

import (
	"bytes"
	"io"
	"os"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/query/share"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/dbal/query"
)

// Query Query share.DSL
type Query struct {
	QueryDSL
	Query        query.Query
	GetTableName func(string) string
}

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
func (gou *Query) With(qb query.Query) share.DSL {
	gou.Query = qb
	return gou
}

// Bind 绑定数据模型
func (gou *Query) Bind(getTableName func(string) string) share.DSL {
	gou.GetTableName = getTableName
	return gou
}

// ==================================================
// share.DSL Interface
// ==================================================

// Run 执行查询根据查询条件返回结果
func (gou Query) Run() interface{} {
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
