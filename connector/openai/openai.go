package openai

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
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
	Azure string `json:"azure,omitempty"` // "true" or "false"
}

// Register the connections from dsl
func (o *Connector) Register(file string, id string, dsl []byte) error {
	o.id = id
	o.file = file
	err := application.Parse(file, dsl, o)
	if err != nil {
		return err
	}

	o.Options.Proxy = helper.EnvString(o.Options.Proxy)
	o.Options.Model = helper.EnvString(o.Options.Model)
	o.Options.Key = helper.EnvString(o.Options.Key)
	o.Options.Azure = helper.EnvString(o.Options.Azure)
	return nil
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

// Schema get connector schema interface
func (o *Connector) Schema() (schema.Schema, error) {
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
		"azure": o.Options.Azure,
	}
}
