package rest

import "github.com/gin-gonic/gin"

// API The RESTFul API
type API struct {
	Name     string
	Source   string
	REST     REST
	handlers map[string]map[string]interface{}
}

// REST The RESTFul API
type REST struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Group       string `json:"-"`
	Description string `json:"description,omitempty"`
	Guard       string `json:"guard,omitempty"`
	Paths       []Path `json:"paths,omitempty"`
}

// Path The RESTFul API Route path
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

// Out The RESTFul API output
type Out struct {
	Status  int               `json:"status"`
	Type    string            `json:"type,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Option the server option
type Option struct {
	Mode string `json:"mode,omitempty"` // the server mode production / development
	Root string `json:"root,omitempty"` // the root route path of the RESTFul API server
}

// In the in struct
type In struct {
	handler func(c *gin.Context, name string) interface{}
	varname string
}
