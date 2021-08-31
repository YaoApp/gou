package gou

import (
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xun/dbal/query"
)

// QueryStack 查询栈
type QueryStack struct {
	QueryBuilders []query.Query
	Wheres        []map[string]interface{}
	Current       int
}

// NewQueryStack 创建查询栈
func NewQueryStack() *QueryStack {
	return &QueryStack{
		QueryBuilders: []query.Query{},
		Wheres:        []map[string]interface{}{},
		Current:       -1,
	}
}

// Push 添加查询器
func (stack *QueryStack) Push(qb query.Query, wheres map[string]interface{}) {
	stack.QueryBuilders = append(stack.QueryBuilders, qb)
	stack.Wheres = append(stack.Wheres, wheres)
	stack.Current = stack.Current + 1
}

// Merge 合并 Stack
func (stack *QueryStack) Merge(new *QueryStack) {
	curr := stack.Current
	for i, qb := range new.QueryBuilders {
		stack.QueryBuilders = append(stack.QueryBuilders, qb)
		stack.Wheres = append(stack.Wheres, new.Wheres[i])
	}
	stack.Current = curr
}

// Len 查询器数量
func (stack *QueryStack) Len() int {
	return len(stack.QueryBuilders)
}

// Query 返回查询器
func (stack *QueryStack) Query() query.Query {
	if stack.Current < 0 {
		return nil
	}
	return stack.QueryBuilders[stack.Current]
}

// Where 返回查询器查询条件
func (stack *QueryStack) Where() map[string]interface{} {
	if stack.Current < 0 {
		return nil
	}
	return stack.Wheres[stack.Current]
}

// Next 返回下一个查询器
func (stack *QueryStack) Next() int {
	next := stack.Current + 1
	if next < stack.Len() {
		stack.Current = next
		return next
	}
	return -1
}

// Prev 返回上一个查询器
func (stack *QueryStack) Prev() int {
	prev := stack.Current - 1
	if prev > 0 {
		stack.Current = prev
		return prev
	}
	return -1
}

// Run 执行查询栈
func (stack *QueryStack) Run() {
	for i, qb := range stack.QueryBuilders {
		utils.Dump(qb.ToSQL(), qb.MustGet(), stack.Wheres[i], "---")
	}
}
