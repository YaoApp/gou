package query

import (
	"github.com/yaoapp/kun/maps"
)

// DSL QueryDSL Interface
type DSL interface {
	Run() interface{}   // 执行查询根据查询条件返回结果
	Get() []Record      // 执行查询并返回数据记录集合
	Paginate() Paginate // 执行查询并返回带分页信息的数据记录数组
	First() Record      // 执行查询并返回一条数据记录
}

// Record 数据记录
type Record maps.MapStrAny

// Paginate 带分页信息的数据记录数组
type Paginate struct {
	Items     []Record `json:"items"`    // 数据记录集合
	Total     int      `json:"total"`    // 总记录数
	Next      int      `json:"next"`     // 下一页，如没有下一页返回 -1
	Prev      int      `json:"prev"`     // 上一页，如没有上一页返回 -1
	Page      int      `json:"page"`     // 当前页码
	PageSize  int      `json:"pagesize"` // 每页记录数量
	PageCount int      `json:"pagecnt"`  // 总页数
}
