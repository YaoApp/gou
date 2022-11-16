package gou

import (
	"net/textproto"
)

// API 数据接口
type API struct {
	ID     string `jsong:"id"`
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
	Status   int               `json:"status"`
	Type     string            `json:"type,omitempty"`
	Body     interface{}       `json:"body,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Redirect *Redirect         `json:"redirect,omitempty"`
}

// Redirect out redirect
type Redirect struct {
	Code     int    `json:"code,omitempty"`
	Location string `json:"location,omitempty"`
}

// Server API 服务配置
type Server struct {
	Debug  bool     `json:"debug,omitempty"`
	Port   int      `json:"port,omitempty"`
	Host   string   `json:"host,omitempty"`
	Root   string   `json:"root,omitempty"`   // API 根目录
	Allows []string `json:"allows,omitempty"` // 许可跨域访问域名
}

// UploadFile upload file
type UploadFile struct {
	Name     string               `json:"name"`
	TempFile string               `json:"tempFile"`
	Size     int64                `json:"size"`
	Header   textproto.MIMEHeader `json:"mimeType"`
}
