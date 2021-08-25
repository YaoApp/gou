package gou

// API 数据接口
type API struct {
	Name   string
	Source string
	Type   string
	HTTP   HTTP
}

// HTTP http 协议服务
type HTTP struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description,omitempty"`
	Group       string   `json:"group,omitempty"`
	Guard       string   `json:"guard,omitempty"`
	Table       bool     `json:"table,omitempty"`
	Disabled    []string `json:"disabled,omitempty"`
	Enabled     []string `json:"enabled,omitempty"`
	Paths       []Path   `json:"paths,omitempty"`
}

// Path HTTP Path
type Path struct {
	Path    string   `json:"path"`
	Method  string   `json:"method"`
	Guard   string   `json:"guard,omitempty"`
	Type    string   `json:"type,omitempty"`
	Process string   `json:"process"`
	In      []string `json:"in,omitempty"`
	Out     Out      `json:"out,omitempty"`
}

// Out http 输出
type Out struct {
	Status  int         `json:"status"`
	Type    string      `json:"type,omitempty"`
	Content interface{} `json:"content,omitempty"`
}
