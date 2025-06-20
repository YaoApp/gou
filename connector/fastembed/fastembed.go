package fastembed

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

// Options the fastembed connector option
type Options struct {
	Host  string `json:"host,omitempty"`
	Model string `json:"model"`
	Key   string `json:"key,omitempty"`
}

// Register the connections from dsl
func (o *Connector) Register(file string, id string, dsl []byte) error {
	o.id = id
	o.file = file
	err := application.Parse(file, dsl, o)
	if err != nil {
		return err
	}

	o.Options.Host = helper.EnvString(o.Options.Host)
	o.Options.Model = helper.EnvString(o.Options.Model)
	o.Options.Key = helper.EnvString(o.Options.Key)

	// Set default host if not provided
	if o.Options.Host == "" {
		o.Options.Host = "127.0.0.1:8000"
	}

	return nil
}

// Is the connections from dsl
func (o *Connector) Is(typ int) bool {
	return 9 == typ // FASTEMBED type will be 9
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
	host := o.Options.Host
	if host == "" {
		host = "127.0.0.1:8000"
	}

	return map[string]interface{}{
		"host":  host,
		"key":   o.Options.Key,
		"model": o.Options.Model,
	}
}
