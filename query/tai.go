package query

// Tai Query DSL
type Tai TaiQueryDSL

// Run 执行查询根据查询条件返回结果
func (tai Tai) Run() interface{} {
	return []Record{}
}

// Get 执行查询并返回数据记录集合
func (tai Tai) Get() []Record {
	return []Record{}
}

// Paginate 执行查询并返回带分页信息的数据记录数组
func (tai Tai) Paginate() Paginate {
	return Paginate{}
}

// First 执行查询并返回一条数据记录
func (tai Tai) First() Record {
	return Record{}
}
