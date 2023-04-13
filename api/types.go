package api

// API 数据接口
type API struct {
	ID   string `jsong:"id"`
	Name string
	File string
	Type string
	HTTP HTTP
}

// HTTP http 协议服务
type HTTP struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Group       string `json:"group,omitempty"`
	Guard       string `json:"guard,omitempty"`
	Paths       []Path `json:"paths,omitempty"`
}

// Path HTTP Path
type Path struct {
	Label       string        `json:"label,omitempty"`
	Description string        `json:"description,omitempty"`
	Path        string        `json:"path"`
	Method      string        `json:"method"`
	Process     string        `json:"process"`
	Guard       string        `json:"guard,omitempty"`
	In          []interface{} `json:"in,omitempty"`
	Out         Out           `json:"out,omitempty"`
}

// Out http 输出
type Out struct {
	Status   int               `json:"status"`
	Type     string            `json:"type,omitempty"`
	Body     interface{}       `json:"body,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Stream   bool              `json:"stream,omitempty"`
	Redirect *Redirect         `json:"redirect,omitempty"`
}

// Redirect out redirect
type Redirect struct {
	Code     int    `json:"code,omitempty"`
	Location string `json:"location,omitempty"`
}

type ssEventData struct {
	Name    string
	Message interface{}
}
