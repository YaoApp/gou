package gou

// Store the kv-store setting
type Store struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Option      map[string]interface{} `json:"option,omitempty"`
}
