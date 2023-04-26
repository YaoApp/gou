package openai

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/xun/dbal/query"
)

// Connector connector
type Connector struct {
	id      string
	file    string
	Name    string  `json:"name"`
	Options Options `json:"options"`
}

// Options the redis connector option
type Options struct {
	Proxy string `json:"proxy,omitempty"`
	Model string `json:"model,omitempty"`
	Key   string `json:"key"`
}

// Register the connections from dsl
func (o *Connector) Register(file string, id string, dsl []byte) error {
	o.id = id
	o.file = file
	return application.Parse(file, dsl, o)
}

// Is the connections from dsl
func (o *Connector) Is(typ int) bool {
	return 6 == typ
}

// ID get connector id
func (o *Connector) ID() string {
	return o.id
}

// Query get connector query interface
func (o *Connector) Query() (query.Query, error) {
	return nil, nil
}

// Close connections
func (o *Connector) Close() error {
	return nil
}

// Setting get the connection setting
func (o *Connector) Setting() map[string]interface{} {

	host := "https://api.openai.com"
	if o.Options.Proxy != "" {
		host = o.Options.Proxy
	}

	return map[string]interface{}{
		"host":  host,
		"key":   o.Options.Key,
		"model": o.Options.Model,
	}
}
