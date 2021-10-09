package query

import (
	"bufio"
	"io"
	"os"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/dbal/query"
)

// Query Query DSL
type Query struct {
	gou.QueryDSL
	qb query.Query
}

// Gou 创建 Gou Query DSL
func Gou(input []byte) *Query {

	var dsl gou.QueryDSL
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

// GouReader 创建 Gou Query DSL (输入接口)
func GouReader(reader io.Reader) *Query {
	var bytes []byte
	_, err := reader.Read(bytes)
	if err != nil {
		exception.New("读取数据失败 %s", 500, err.Error()).Throw()
	}
	return Gou(bytes)
}

// GouFile 创建 Gou Query DSL (文件)
func GouFile(filename string) *Query {
	file, err := os.Open(filename)
	if err != nil {
		exception.New("读取文件失败 %s", 500, err.Error()).Throw()
	}
	defer file.Close()
	return GouReader(bufio.NewReader(file))
}

// With 关联查询器
func (gou *Query) With(qb query.Query) DSL {
	gou.qb = qb
	return gou
}

// ==================================================
// DSL Interface
// ==================================================

// Run 执行查询根据查询条件返回结果
func (gou Query) Run() interface{} {
	return []Record{}
}

// Get 执行查询并返回数据记录集合
func (gou Query) Get() []Record {
	return []Record{}
}

// Paginate 执行查询并返回带分页信息的数据记录数组
func (gou Query) Paginate() Paginate {
	return Paginate{}
}

// First 执行查询并返回一条数据记录
func (gou Query) First() Record {
	return Record{}
}
