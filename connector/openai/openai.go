package openai

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
)

// Connector connector
type Connector struct {
	id      string
	file    string
	Name    string  `json:"name"`
	Options Options `json:"options"`
	types.MetaInfo
}

// Options the openai connector option
type Options struct {
	Host  string `json:"host,omitempty"`  // API endpoint, e.g. "https://api.openai.com" or custom endpoint
	Proxy string `json:"proxy,omitempty"` // (Deprecated) API endpoint, use Host instead. For backward compatibility only.
	Model string `json:"model,omitempty"` // Model name, e.g. "gpt-4o"
	Key   string `json:"key"`             // API key
	Azure string `json:"azure,omitempty"` // "true" or "false" for Azure OpenAI
}

// Note: HTTP proxy (HTTPS_PROXY, HTTP_PROXY environment variables) is handled by http.GetTransport automatically

// Register the connections from dsl
func (o *Connector) Register(file string, id string, dsl []byte) error {
	o.id = id
	o.file = file
	err := application.Parse(file, dsl, o)
	if err != nil {
		return err
	}

	o.Options.Host = helper.EnvString(o.Options.Host)
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

	// Determine API endpoint
	// Priority: Host > Proxy (backward compatibility) > default
	host := "https://api.openai.com"
	if o.Options.Host != "" {
		host = o.Options.Host
	} else if o.Options.Proxy != "" {
		// Backward compatibility: use Proxy as API endpoint
		host = o.Options.Proxy
	}

	// Note: HTTP proxy is handled via HTTPS_PROXY/HTTP_PROXY environment variables
	// by http.GetTransport, not configured here
	return map[string]interface{}{
		"host":  host,
		"key":   o.Options.Key,
		"model": o.Options.Model,
		"azure": o.Options.Azure,
	}
}

// GetMetaInfo returns the meta information
func (o *Connector) GetMetaInfo() types.MetaInfo {
	return o.MetaInfo
}
