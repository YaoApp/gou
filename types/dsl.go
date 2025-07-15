package types

import "time"

// MetaInfo The meta info of the DSL (Each DSL has its own meta info)
type MetaInfo struct {
	Label       string    `json:"label,omitempty"`       // The label of the DSL
	Description string    `json:"description,omitempty"` // The description of the DSL ( markdown or plain text )
	Tags        []string  `json:"tags,omitempty"`        // The tags of the DSL
	Readonly    bool      `json:"readonly,omitempty"`    // The DSL is readonly
	Builtin     bool      `json:"builtin,omitempty"`     // The DSL is builtin
	Sort        int       `json:"sort,omitempty"`        // The sort of the DSL
	Mtime       time.Time `json:"mtime,omitempty"`       // The mtime of the DSL
	Ctime       time.Time `json:"ctime,omitempty"`       // The ctime of the DSL
}
