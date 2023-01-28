package types

// QueryParam 数据查询器参数
type QueryParam struct {
	Model    string          `json:"model,omitempty"`
	Table    string          `json:"table,omitempty"`
	Alias    string          `json:"alias,omitempty"`
	Export   string          `json:"export,omitempty"` // 导出前缀
	Select   []interface{}   `json:"select,omitempty"` // string | dbal.Raw
	Wheres   []QueryWhere    `json:"wheres,omitempty"`
	Orders   []QueryOrder    `json:"orders,omitempty"`
	Limit    int             `json:"limit,omitempty"`
	Page     int             `json:"page,omitempty"`
	PageSize int             `json:"pagesize,omitempty"`
	Withs    map[string]With `json:"withs,omitempty"`
}

// With relations 关联查询
type With struct {
	Name  string     `json:"name"`
	Query QueryParam `json:"query,omitempty"`
}

// QueryWhere Where 查询条件
type QueryWhere struct {
	Rel    string       `json:"rel,omitempty"` // Relation Name
	Column interface{}  `json:"column,omitempty"`
	Value  interface{}  `json:"value,omitempty"`
	Method string       `json:"method,omitempty"` // where,orwhere, wherein, orwherein...
	OP     string       `json:"op,omitempty"`     // 操作 eq/gt/lt/ge/le/like...
	Wheres []QueryWhere `json:"wheres,omitempty"` // 分组查询
}

// QueryOrder Order 查询排序
type QueryOrder struct {
	Rel    string `json:"rel,omitempty"` // Relation Name
	Column string `json:"column"`
	Option string `json:"option,omitempty"` // desc, asc
}
