package query

import "github.com/yaoapp/xun/dbal/query"

// Gou Query DSL
type Gou struct {
	GouQueryDSL
	qb query.Query
}

// Run 执行查询根据查询条件返回结果
func (gou Gou) Run() interface{} {
	return []Record{}
}

// Get 执行查询并返回数据记录集合
func (gou Gou) Get() []Record {
	return []Record{}
}

// Paginate 执行查询并返回带分页信息的数据记录数组
func (gou Gou) Paginate() Paginate {
	return Paginate{}
}

// First 执行查询并返回一条数据记录
func (gou Gou) First() Record {
	return Record{}
}
