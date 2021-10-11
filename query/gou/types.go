package gou

// QueryDSL Gou Query Domain Specific Language
type QueryDSL struct {
	Select   []Expression `json:"select"`              // 查询字段列表
	From     *Table       `json:"from,omitempty"`      // 查询数据表名称或数据模型
	Wheres   []Where      `json:"wheres,omitempty"`    // 数据查询条件
	Orders   Orders       `json:"orders,omitempty"`    // 排序条件
	Groups   *Groups      `json:"groups,omitempty"`    // 聚合条件
	Havings  []Having     `json:"havings,omitempty"`   // 聚合查询结果筛选条件
	Limit    int          `json:"limit,omitempty"`     // 限定读取记录的数量
	Offset   int          `json:"offset,omitempty"`    // 记录开始位置
	Page     int          `json:"page,omitempty"`      // 分页查询当前页面页码
	PageSize int          `json:"pagesize,omitempty"`  // 每页读取记录的数量
	DataOnly bool         `json:"data-only,omitempty"` // 设定为 true, 查询结果为 []Record; 设定为 false, 查询结果为 Paginate
	Unions   []QueryDSL   `json:"unions,omitempty"`    // 联合查询
	Query    *QueryDSL    `json:"query,omitempty"`     // 子查询
	Joins    []Join       `json:"joins,omitempty"`     // 表连接
	SQL      *SQL         `json:"sql,omitempty"`       // SQL语句
	Comment  string       `json:"comment,omitempty"`   // 查询条件注释
}

// QueryDSLSugar ?支持语法糖
type QueryDSLSugar struct {
	QueryDSL
	Select interface{} `json:"select"`           // 字段列表
	Orders interface{} `json:"orders,omitempty"` // 排序条件
	Groups interface{} `json:"groups,omitempty"` // 聚合条件
}

// Expression 字段表达式
type Expression struct{ string }

// Table 数据表名称或数据模型
type Table struct {
	Alias string // 别名
	Name  string // 名称
}

// Condition 查询条件
type Condition struct {
	Field           *Expression `json:"field"`             // 查询字段
	Value           interface{} `json:"value,omitempty"`   // 匹配数值
	ValueExpression *Expression `json:"-"`                 // 数值表达式
	OP              string      `json:"op"`                // 匹配关系运算符
	OR              bool        `json:"or,omitempty"`      // true 查询条件逻辑关系为 or, 默认为 false 查询条件逻辑关系为 and
	Query           *QueryDSL   `json:"query,omitempty"`   // 子查询, 如设定 query 则忽略 value 数值。
	Comment         string      `json:"comment,omitempty"` // 查询条件注释
}

// Where 查询条件
type Where struct {
	Condition
	Wheres []Where `json:"wheres,omitempty"` // 分组查询。用于 condition 1 and ( condition 2 OR condition 3) 的场景
}

// Orders 排序条件集合
type Orders []Order

// Order 排序条件
type Order struct {
	Field   *Expression `json:"field"`             // 排序字段
	Sort    string      `json:"sort,omitempty"`    // 排序方式
	Comment string      `json:"comment,omitempty"` // 查询条件注释
}

// Groups 聚合条件集合
type Groups []Group

// Group 聚合条件
type Group struct {
	Field   *Expression `json:"field"`             // 排序字段
	Rollup  string      `json:"rollup,omitempty"`  // 同时返回多层级统计结果，对应聚合字段数值的名称。
	Comment string      `json:"comment,omitempty"` // 查询条件注释
}

// Having 聚合结果筛选条件
type Having struct {
	Condition
	Havings []Having `json:"havings,omitempty"` // 分组查询。用于 condition 1 and ( condition 2 OR condition 3) 的场景
}

// Join 数据表连接
type Join struct {
	From    Table       `json:"from"`            // 查询数据表名称或数据模型
	Key     *Expression `json:"key"`             // 关联连接表字段名称
	Foreign *Expression `json:"foreign"`         // 关联目标表字段名称(需指定表名或别名)
	Left    bool        `json:"left,omitempty"`  // true 连接方式为 LEFT JOIN, 默认为 false 连接方式为 JOIN
	Right   bool        `json:"right,omitempty"` // true 连接方式为 RIGHT JOIN, 默认为 false 连接方式为 JOIN
}

// SQL 语句
type SQL struct {
	STMT string        `json:"stmt,omitempty"` // SQL 语句
	Args []interface{} `json:"args,omitempty"` // 绑定参数表
}
