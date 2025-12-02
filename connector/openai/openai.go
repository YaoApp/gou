package openai

import (
	"strings"

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

	// Model Capabilities
	Capabilities *Capabilities `json:"capabilities,omitempty"`
}

// Capabilities defines the capabilities of a language model
// This configuration is loaded from agent/models.yml
type Capabilities struct {
	Vision                interface{} `json:"vision,omitempty" yaml:"vision,omitempty"`                                 // Supports vision/image input: bool or VisionFormat string ("openai", "claude"/"base64", "default")
	Audio                 bool        `json:"audio,omitempty" yaml:"audio,omitempty"`                                   // Supports audio input/output
	ToolCalls             bool        `json:"tool_calls,omitempty" yaml:"tool_calls,omitempty"`                         // Supports tool/function calling
	Reasoning             bool        `json:"reasoning,omitempty" yaml:"reasoning,omitempty"`                           // Supports reasoning/thinking mode (o1, DeepSeek R1)
	Streaming             bool        `json:"streaming,omitempty" yaml:"streaming,omitempty"`                           // Supports streaming responses
	JSON                  bool        `json:"json,omitempty" yaml:"json,omitempty"`                                     // Supports JSON mode
	Multimodal            bool        `json:"multimodal,omitempty" yaml:"multimodal,omitempty"`                         // Supports multimodal input
	TemperatureAdjustable bool        `json:"temperature_adjustable,omitempty" yaml:"temperature_adjustable,omitempty"` // Supports temperature adjustment (reasoning models typically don't)
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

	setting := map[string]interface{}{
		"host":  host,
		"key":   o.Options.Key,
		"model": o.Options.Model,
		"azure": o.Options.Azure,
	}

	// Add capabilities with defaults if not provided
	capabilities := o.Options.Capabilities
	if capabilities == nil {
		// Try to get default capabilities based on model name (convert to lowercase first)
		modelLower := strings.ToLower(o.Options.Model)
		capabilities = GetDefaultCapabilities(modelLower)

		// If no matching pattern found, use minimal standard (all disabled)
		if capabilities == nil {
			capabilities = &Capabilities{
				Vision:                false,
				ToolCalls:             false,
				Audio:                 false,
				Reasoning:             false,
				Streaming:             false,
				JSON:                  false,
				Multimodal:            false,
				TemperatureAdjustable: true, // Default to true for non-reasoning models
			}
		}
	}

	// Auto-detect TemperatureAdjustable if not explicitly set in config
	// Reasoning models typically don't support temperature adjustment
	// If Reasoning is true but TemperatureAdjustable wasn't explicitly set, default to false
	if capabilities.Reasoning && !capabilities.TemperatureAdjustable {
		// Reasoning model with TemperatureAdjustable=false is expected
	} else if !capabilities.Reasoning && !capabilities.TemperatureAdjustable {
		// Non-reasoning model should support temperature by default
		capabilities.TemperatureAdjustable = true
	}

	setting["capabilities"] = capabilities

	// Note: HTTP proxy is handled via HTTPS_PROXY/HTTP_PROXY environment variables
	// by http.GetTransport, not configured here
	return setting
}

// GetMetaInfo returns the meta information
func (o *Connector) GetMetaInfo() types.MetaInfo {
	return o.MetaInfo
}
