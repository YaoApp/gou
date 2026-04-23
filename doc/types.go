package doc

// EntryType identifies the kind of documented item.
type EntryType string

const (
	TypeProcess    EntryType = "process"
	TypeJSObject   EntryType = "js_object"
	TypeJSFunction EntryType = "js_function"
	TypeJSClass    EntryType = "js_class"
	TypeOpenAPI    EntryType = "openapi"
)

// TypeValue is the universal value descriptor used for arguments, return
// values, fields and method signatures. Its Type field follows the bridge.go
// Go↔JS mapping:
//
//	Base:     null, undefined, bool, number, bigint, string, object, array,
//	          bytes, error, function, promise, external
//	Extended: void, any, union
//	Named:    Response, Paginated, CodeBlock, … (object with fields)
type TypeValue struct {
	Type     string      `json:"type"               yaml:"type"`
	Desc     string      `json:"desc,omitempty"     yaml:"desc,omitempty"`
	Example  any         `json:"example,omitempty"  yaml:"example,omitempty"`
	Fields   []TypeValue `json:"fields,omitempty"   yaml:"fields,omitempty"`
	Items    *TypeValue  `json:"items,omitempty"    yaml:"items,omitempty"`
	Variants []TypeValue `json:"variants,omitempty" yaml:"variants,omitempty"`
	Name     string      `json:"name,omitempty"     yaml:"name,omitempty"`
	Required bool        `json:"required,omitempty" yaml:"required,omitempty"`
}

// Method describes a method on a JS object or class.
type Method struct {
	Name   string      `json:"name"              yaml:"name"`
	Desc   string      `json:"desc"              yaml:"desc"`
	Args   []TypeValue `json:"args,omitempty"    yaml:"args,omitempty"`
	Return *TypeValue  `json:"return,omitempty"  yaml:"return,omitempty"`
}

// Endpoint describes an HTTP API endpoint (reserved for OpenAPI).
type Endpoint struct {
	Method string `json:"method"            yaml:"method"`
	Path   string `json:"path"              yaml:"path"`
	Guard  string `json:"guard,omitempty"   yaml:"guard,omitempty"`
}

// Entry is a single documentation item.
type Entry struct {
	Name     string      `json:"name"               yaml:"name"`
	Type     EntryType   `json:"type"               yaml:"type"`
	Group    string      `json:"group"              yaml:"group"`
	Desc     string      `json:"desc"               yaml:"desc"`
	Args     []TypeValue `json:"args,omitempty"     yaml:"args,omitempty"`
	Return   *TypeValue  `json:"return,omitempty"   yaml:"return,omitempty"`
	Methods  []Method    `json:"methods,omitempty"  yaml:"methods,omitempty"`
	Endpoint *Endpoint   `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
}

// DocFile is the top-level structure of a doc.yml file.
type DocFile struct {
	Group   string    `yaml:"group"`
	Type    EntryType `yaml:"type"`
	Entries []Entry   `yaml:"entries"`
}

// ValidationResult holds the outcome of a Validate call.
type ValidationResult struct {
	Valid      bool     `json:"valid"`
	Status     string   `json:"status"`
	Name       string   `json:"name"`
	Suggestion []string `json:"suggestion,omitempty"`
	Message    string   `json:"message"`
	Entry      *Entry   `json:"entry,omitempty"`
}

// ListOption controls filtering for List calls.
type ListOption struct {
	Group  string
	Search string
}

// StatInfo holds documentation coverage statistics for one EntryType.
type StatInfo struct {
	Total        int `json:"total"`
	Documented   int `json:"documented"`
	Undocumented int `json:"undocumented"`
}
