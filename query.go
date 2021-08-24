package gou

// Query 数据查询器模型
type Query struct{}

// LoadQuery 载入查询器
func LoadQuery() {}

// Run 执行查询，并返回结果(支持上下文)
func (query *Query) Run() {}
