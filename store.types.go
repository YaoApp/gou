package gou

// Store the kv-store setting
type Store struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Connector   string                 `json:"connector,omitempty"`
	Type        string                 `json:"type,omitempty"` // warning: type is deprecated in the future new version
	Option      map[string]interface{} `json:"option,omitempty"`
}
