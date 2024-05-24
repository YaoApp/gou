package types

// Schema the schema interface
type Schema interface {
	SetOption(option interface{}) error

	Close() error
	Create(name string) error
	Drop(name string) error

	Tables(prefix ...string) ([]string, error)
	TableExists(name string) (bool, error)
	TableGet(name string) (Blueprint, error)
	TableCreate(name string, blueprint Blueprint) error
	TableSave(name string, blueprint Blueprint) error
	TableDrop(name string) error
	TableRename(name string, new string) error
	TableDiff(name Blueprint, another Blueprint) (Diff, error)

	ColumnAdd(name string, column Column) error
	ColumnAlt(name string, column Column) error
	ColumnDel(name string, columns ...string) error

	IndexAdd(name string, index Index) error
	IndexAlt(name string, index Index) error
	IndexDel(name string, indexes ...string) error
}

// Diff the different of schema
type Diff struct {
	Columns struct {
		Add []Column
		Del []Column
		Alt []Column
	}
	Indexes struct {
		Add []Index
		Del []Index
		Alt []Index
	}
	Option map[string]bool
}

// Blueprint the blueprint of schema
type Blueprint struct {
	Columns []Column        `json:"columns,omitempty"`
	Indexes []Index         `json:"indexes,omitempty"`
	Option  BlueprintOption `json:"option,omitempty"`
}

// BlueprintOption the blueprint option
type BlueprintOption struct {
	Timestamps  bool `json:"timestamps,omitempty"`   // + created_at, updated_at fields
	SoftDeletes bool `json:"soft_deletes,omitempty"` // + deleted_at field
	Trackings   bool `json:"trackings,omitempty"`    // + created_by, updated_by, deleted_by fields
	Constraints bool `json:"constraints,omitempty"`  // + 约束定义
	Permission  bool `json:"permission,omitempty"`   // + __permission fields
	Logging     bool `json:"logging,omitempty"`      // + __logging_id fields
	Readonly    bool `json:"read_only,omitempty"`    // + Ignore the migrate operation
}

// Column the field description struct
type Column struct {
	Name          string      `json:"name"`
	Label         string      `json:"label,omitempty"`
	Type          string      `json:"type,omitempty"`
	Title         string      `json:"title,omitempty"`
	Description   string      `json:"description,omitempty"`
	Comment       string      `json:"comment,omitempty"`
	Length        int         `json:"length,omitempty"`
	Precision     int         `json:"precision,omitempty"`
	Scale         int         `json:"scale,omitempty"`
	Nullable      bool        `json:"nullable,omitempty"`
	Option        []string    `json:"option,omitempty"`
	Default       interface{} `json:"default,omitempty"`
	DefaultRaw    string      `json:"default_raw,omitempty"`
	Generate      string      `json:"generate,omitempty"` // Increment, UUID,...
	Crypt         string      `json:"crypt,omitempty"`    // AES, PASSWORD, AES-256, AES-128, PASSWORD-HASH, ...
	Index         bool        `json:"index,omitempty"`
	Unique        bool        `json:"unique,omitempty"`
	Primary       bool        `json:"primary,omitempty"`
	Origin        string      `json:"origin,omitempty"`
	RemoveIndex   bool        `json:"-"`
	RemoveUnique  bool        `json:"-"`
	RemovePrimary bool        `json:"-"`
}

// Index the search index struct
type Index struct {
	Comment string   `json:"comment,omitempty"`
	Name    string   `json:"name,omitempty"`
	Columns []string `json:"columns,omitempty"`
	Type    string   `json:"type,omitempty"` // primary,unique,index,match
	Origin  string   `json:"origin,omitempty"`
}
