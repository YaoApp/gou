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
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Group       string `json:"group,omitempty"`
	Guard       string `json:"guard,omitempty"`
	Paths       []Path `json:"paths,omitempty"`
}

// Path HTTP Path
type Path struct {
	Label       string   `json:"label,omitempty"`
	Description string   `json:"description,omitempty"`
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Process     string   `json:"process"`
	Guard       string   `json:"guard,omitempty"`
	In          []string `json:"in,omitempty"`
	Out         Out      `json:"out,omitempty"`
}

// Out http 输出
type Out struct {
	Status  int               `json:"status"`
	Type    string            `json:"type,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Server API 服务配置
type Server struct {
	Debug  bool     `json:"debug,omitempty"`
	Port   int      `json:"port,omitempty"`
	Host   string   `json:"host,omitempty"`
	Root   string   `json:"root,omitempty"`   // API 根目录
	Allows []string `json:"allows,omitempty"` // 许可跨域访问域名
}

// Socket struct
type Socket struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Mode        string `json:"mode,omitempty"`      // Server | client
	Reconnect   int    `json:"reconnect,omitempty"` // max times try to reconnect server when connection break (client mode only)
	Description string `json:"description,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Host        string `json:"host,omitempty"`
	Port        string `json:"port,omitempty"`
	Process     string `json:"process,omitempty"`
	Timeout     int    `json:"timeout,omitempty"` // timeout (seconds)
	BufferSize  int    `json:"buffer,omitempty"`  // bufferSize
	KeepAlive   int    `json:"keep,omitempty"`    // -1 not keep alive, 0 keep alive always, keep alive n seconds.
}
