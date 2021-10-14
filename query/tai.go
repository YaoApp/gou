package query

import (
	"github.com/yaoapp/gou/query/share"
	"github.com/yaoapp/gou/query/tai"
)

// Tai Query DSL
type Tai tai.QueryDSL

// Run 执行查询根据查询条件返回结果
func (tai Tai) Run() interface{} {
	return []share.Record{}
}

// Get 执行查询并返回数据记录集合
func (tai Tai) Get() []share.Record {
	return []share.Record{}
}

// Paginate 执行查询并返回带分页信息的数据记录数组
func (tai Tai) Paginate() share.Paginate {
	return share.Paginate{}
}

// First 执行查询并返回一条数据记录
func (tai Tai) First() share.Record {
	return share.Record{}
}
