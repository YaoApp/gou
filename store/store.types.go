package store

// Instance the kv-store setting
type Instance struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Connector   string                 `json:"connector,omitempty"`
	Type        string                 `json:"type,omitempty"` // warning: type is deprecated in the future new version
	Option      map[string]interface{} `json:"option,omitempty"`
}
