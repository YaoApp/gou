package gou

import (
	"github.com/yaoapp/kun/maps"
)

// Relationship types
const (
	RelHasOne         = "hasOne"         // 1 v 1
	RelHasMany        = "hasMany"        // 1 v n
	RelBelongsTo      = "belongsTo"      // inverse  1 v 1 / 1 v n / n v n
	RelHasOneThrough  = "hasOneThrough"  // 1 v 1 ( t1 <-> t2 <-> t3)
	RelHasManyThrough = "hasManyThrough" // 1 v n ( t1 <-> t2 <-> t3)
	RelBelongsToMany  = "belongsToMany"  // 1 v1 / 1 v n / n v n
	RelMorphOne       = "morphOne"
	RelMorphMany      = "morphMany"
	RelMorphToMany    = "morphToMany"
	RelMorphByMany    = "morphByMany"
	RelMorphMap       = "morphMap"
)

// Model 数据模型
type Model struct {
	ID            string
	Name          string
	Source        string
	Driver        string // Driver
	MetaData      MetaData
	Columns       map[string]*Column // 字段映射表
	ColumnNames   []interface{}      // 字段名称清单
	PrimaryKey    string             // 主键(单一主键)
	PrimaryKeys   []string           // 主键(联合主键)
	UniqueColumns []*Column          // 唯一字段清单
}

// MetaData 元数据
type MetaData struct {
	Name      string              `json:"name,omitempty"`      // 元数据名称
	Connector string              `json:"connector,omitempty"` // Bind a connector, MySQL, SQLite, Postgres, Clickhouse, Tidb, Oracle support. default is SQLite
	Table     Table               `json:"table,omitempty"`     // 数据表选项
	Columns   []Column            `json:"columns,omitempty"`   // 字段定义
	Indexes   []Index             `json:"indexes,omitempty"`   // 索引定义
	Relations map[string]Relation `json:"relations,omitempty"` // 映射关系定义
	Values    []maps.MapStrAny    `json:"values,omitempty"`    // 初始数值
	Option    Option              `json:"option,omitempty"`    // 元数据配置
}

// Column the field description struct
type Column struct {
	Label       string       `json:"label,omitempty"`
	Name        string       `json:"name"`
	Type        string       `json:"type,omitempty"`
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Comment     string       `json:"comment,omitempty"`
	Length      int          `json:"length,omitempty"`
	Precision   int          `json:"precision,omitempty"`
	Scale       int          `json:"scale,omitempty"`
	Nullable    bool         `json:"nullable,omitempty"`
	Option      []string     `json:"option,omitempty"`
	Default     interface{}  `json:"default,omitempty"`
	DefaultRaw  string       `json:"default_raw,omitempty"`
	Example     interface{}  `json:"example,omitempty"`
	Generate    string       `json:"generate,omitempty"` // Increment, UUID,...
	Crypt       string       `json:"crypt,omitempty"`    // AES, PASSWORD, AES-256, AES-128, PASSWORD-HASH, ...
	Validations []Validation `json:"validations,omitempty"`
	Index       bool         `json:"index,omitempty"`
	Unique      bool         `json:"unique,omitempty"`
	Primary     bool         `json:"primary,omitempty"`
	model       *Model
}

// Validation the field validation struct
type Validation struct {
	Method  string        `json:"method"`
	Args    []interface{} `json:"args,omitempty"`
	Message string        `json:"message,omitempty"`
}

// ValidateResponse 数据校验返回结果
type ValidateResponse struct {
	Line     int      `json:"line,omitempty"`
	Column   string   `json:"column,omitempty"`
	Messages []string `json:"messages,omitempty"`
}

// Index the search index struct
type Index struct {
	Comment string   `json:"comment,omitempty"`
	Name    string   `json:"name,omitempty"`
	Columns []string `json:"columns,omitempty"`
	Type    string   `json:"type,omitempty"` // primary,unique,index,match
}

// Table the model mapping table in DB
type Table struct {
	Name        string   `json:"name,omitempty"`   // optional, if not set, the default is generate from model name. eg name.space.user, table name is name_space_user
	Prefix      string   `json:"prefix,omitempty"` // optional, the table prefix
	Comment     string   `json:"comment,omitempty"`
	Engine      string   `json:"engine,omitempty"` // InnoDB,MyISAM ( MySQL Only )
	Collation   string   `json:"collation"`
	Charset     string   `json:"charset"`
	PrimaryKeys []string `json:"primarykeys"`
}

// Relation the new xun model relation
type Relation struct {
	Name    string     `json:"-"`
	Type    string     `json:"type"`
	Key     string     `json:"key,omitempty"`
	Model   string     `json:"model,omitempty"`
	Foreign string     `json:"foreign,omitempty"`
	Links   []Relation `json:"links,omitempty"`
	Query   QueryParam `json:"query,omitempty"`
}

// Option 模型配置选项
type Option struct {
	Timestamps  bool `json:"timestamps,omitempty"`   // + created_at, updated_at 字段
	SoftDeletes bool `json:"soft_deletes,omitempty"` // + deleted_at 字段
	Trackings   bool `json:"trackings,omitempty"`    // + created_by, updated_by, deleted_by 字段
	Constraints bool `json:"constraints,omitempty"`  // + 约束定义
	Permission  bool `json:"permission,omitempty"`   // + __permission 字段
	Logging     bool `json:"logging,omitempty"`      // + __logging_id 字段
	Readonly    bool `json:"read_only,omitempty"`    // Ignore the migrate operation
}

// ColumnMap ColumnMap 字段映射
type ColumnMap struct {
	Column *Column
	Model  *Model
	Export string // 取值时的变量名
}

// ExportData the export data struct
type ExportData struct {
	Model   string          `json:"model"`
	Columns []string        `json:"columns"`
	Values  [][]interface{} `json:"values"`
}
