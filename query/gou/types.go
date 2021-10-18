package gou

import (
	"regexp"
)

// Star 数组索引 *
const Star = -1

// RegSpaces 连续空格
var RegSpaces = regexp.MustCompile("[ ]+")

// RegCommaSpaces 连续空格 + 逗号+连续空格 "  ,  " | "," | ", " | " ,"
var RegCommaSpaces = regexp.MustCompile("[ ]*,[ ]*")

// RegIsNumber 字段表达式字段是否为数字
var RegIsNumber = regexp.MustCompile("^[0-9]{1}[\\.]*[0-9]*$")

// RegArrayIndex 字段表达式数组下标
var RegArrayIndex = regexp.MustCompile("\\[([0-9\\*]+)\\]")

// RegAlias 别名正则表达式
var RegAlias = regexp.MustCompile("[ ]+[Aa][Ss][ ]+")

// RegTable Table 正则表达式
var RegTable = regexp.MustCompile("^[A-Za-z0-9_\u4e00-\u9fa5]+$")

// RegFieldTable 字段表达式字段中的指定的数据表
var RegFieldTable = regexp.MustCompile("^[$]*([A-Za-z0-9_\u4e00-\u9fa5]+)\\.")

// RegField 字段表达式字段为数据表字段
var RegField = regexp.MustCompile("^[A-Za-z0-9_\u4e00-\u9fa5]+$")

// RegFieldFun 字段表达式为函数
var RegFieldFun = regexp.MustCompile("^\\:([A-Za-z0-9_]+)\\((.*)\\)$")

// RegFieldType 字段表达式的类型声明
var RegFieldType = regexp.MustCompile("\\([ ]*([a-zA-Z0-9, ]+)[ ]*\\)[ ]*$")

// RegFieldIsArrayObject 字段表达式字段是否为数组
var RegFieldIsArrayObject = regexp.MustCompile("\\.[A-Za-z0-9_\u4e00-\u9fa5]+")

// RegFieldIsArray 字段表达式字段是否为数组
var RegFieldIsArray = regexp.MustCompile("^([A-Za-z0-9_\u4e00-\u9fa5]+)([@\\[]{1})")

// M map[string]interface{} 别名
type M = map[string]interface{}

// F float64 别名
type F = float64

// Any interface{} 别名
type Any = interface{}

// QueryDSL Gou Query Domain Specific Language
type QueryDSL struct {
	Select   []Expression `json:"select"`              // 查询字段列表
	From     *Table       `json:"from,omitempty"`      // 查询数据表名称或数据模型
	Wheres   []Where      `json:"wheres,omitempty"`    // 数据查询条件
	Orders   Orders       `json:"orders,omitempty"`    // 排序条件
	Groups   *Groups      `json:"groups,omitempty"`    // 聚合条件
	Havings  []Having     `json:"havings,omitempty"`   // 聚合查询结果筛选条件
	First    interface{}  `json:"first,omitempty"`     // 限定读取单条数据
	Limit    interface{}  `json:"limit,omitempty"`     // 限定读取记录的数量
	Offset   interface{}  `json:"offset,omitempty"`    // 记录开始位置
	Page     interface{}  `json:"page,omitempty"`      // 分页查询当前页面页码
	PageSize interface{}  `json:"pagesize,omitempty"`  // 每页读取记录的数量
	DataOnly interface{}  `json:"data-only,omitempty"` // 设定为 true, 查询结果为 []Record; 设定为 false, 查询结果为 Paginate
	Unions   []QueryDSL   `json:"unions,omitempty"`    // 联合查询
	SubQuery *QueryDSL    `json:"query,omitempty"`     // 子查询
	Alias    string       `json:"name,omitempty"`      // 子查询别名
	Joins    []Join       `json:"joins,omitempty"`     // 表连接
	SQL      *SQL         `json:"sql,omitempty"`       // SQL语句
	Comment  string       `json:"comment,omitempty"`   // 查询条件注释
}

// Expression 字段表达式
type Expression struct {
	Origin        string       // 原始数据
	Table         string       // 数据表名称
	Field         string       // 字段名称
	Value         interface{}  // 常量数值
	FunName       string       // 函数名称
	FunArgs       []Expression // 函数参数表
	Alias         string       // 字段别名
	Type          *FieldType   // 字段类型(用于自动转换和JSON Table)
	Index         int          // 数组字段索引 const Star -1 全部, 0 ~ n 数组
	Key           string       // 对象字段键名
	IsModel       bool         // 数据表是否为模型 $model.name
	IsFun         bool         // 是否为函数  :max(name)
	IsConst       bool         // 是否为常量  1,0.618, 'foo'
	IsNumber      bool         // 是否为数字常量
	IsString      bool         // 是否为字符串常量
	IsArray       bool         // 是否为数组  array@, array[0], array[*]
	IsObject      bool         // 是否为对象  object$.foo
	IsAES         bool         // 是否为加密字段  name*
	IsArrayObject bool         // 是否为对象数组  array@.foo.bar
	IsBinding     bool         // 是否为绑定参数  ?:name
}

// FieldType 字段类型(用于自动转换和JSON Table)
type FieldType struct {
	Name      string `json:"name,omitempty"`      // JSON数组字段类型(用于生成 JSON Table)
	Length    int    `json:"length,omitempty"`    // 字段长度，对 string 等类型字段有效
	Precision int    `json:"precision,omitempty"` // 字段位数(含小数位)，对 float、decimal 等类型字段有效
	Scale     int    `json:"scale,omitempty"`     // 字段小数位位数，对 float、decimal 等类型字段有效
}

// FieldNode 字段表达式节点
type FieldNode struct {
	Index int
	Field *Expression
}

// Table 数据表名称或数据模型
type Table struct {
	Alias   string // 别名
	Name    string // 名称
	IsModel bool   // 是否为数据模型
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
	From    *Table      `json:"from"`              // 查询数据表名称或数据模型
	Key     *Expression `json:"key"`               // 关联连接表字段名称
	Foreign *Expression `json:"foreign"`           // 关联目标表字段名称(需指定表名或别名)
	Left    bool        `json:"left,omitempty"`    // true 连接方式为 LEFT JOIN, 默认为 false 连接方式为 JOIN
	Right   bool        `json:"right,omitempty"`   // true 连接方式为 RIGHT JOIN, 默认为 false 连接方式为 JOIN
	Comment string      `json:"comment,omitempty"` // 关联条件注释
}

// SQL 语句
type SQL struct {
	STMT    string        `json:"stmt,omitempty"`    // SQL 语句
	Args    []interface{} `json:"args,omitempty"`    // 绑定参数表
	Comment string        `json:"comment,omitempty"` // SQL语句注释
}
