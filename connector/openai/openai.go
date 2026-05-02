package openai

import (
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/llm"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
)

// Connector connector
type Connector struct {
	id              string
	file            string
	Name            string                    `json:"name"`
	AuthModeVal     llm.AuthMode              `json:"auth_mode,omitempty"`
	SupportedParams map[string]*llm.ParamSpec `json:"supported_params,omitempty"`
	Options         Options                   `json:"options"`
	types.MetaInfo
}

// Options the openai connector option
type Options struct {
	Host  string `json:"host,omitempty"`  // API endpoint, e.g. "https://api.openai.com" or custom endpoint
	Proxy string `json:"proxy,omitempty"` // (Deprecated) API endpoint, use Host instead. For backward compatibility only.
	Model string `json:"model,omitempty"` // Model name, e.g. "gpt-4o"
	Key   string `json:"key"`             // API key

	// Model Capabilities
	Capabilities *Capabilities `json:"capabilities,omitempty"`

	// Thinking mode configuration (for models that support reasoning/thinking)
	// Example: {"type": "enabled"} or {"type": "disabled"}
	Thinking interface{} `json:"thinking,omitempty"`

	// Request parameters that can be passed to sandbox proxy
	MaxTokens   int      `json:"max_tokens,omitempty"`  // Maximum tokens for response
	Temperature *float64 `json:"temperature,omitempty"` // Temperature for response (use pointer to distinguish 0 from unset)

	// Supported API protocols. Defaults to ["openai"] when empty.
	// Dual-protocol gateways (e.g. LiteLLM) declare ["openai", "anthropic"].
	Protocols []string `json:"protocols,omitempty"`

	// Extra parameters forwarded verbatim into the API request body.
	// Aligned with LiteLLM's extra_body convention.
	// Examples: reasoning, enable_thinking, thinkingConfig.
	ExtraBody map[string]interface{} `json:"extra_body,omitempty"`
}

// Capabilities is an alias for llm.Capabilities for backward compatibility.
// New code should use llm.Capabilities directly.
type Capabilities = llm.Capabilities

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

	if o.AuthModeVal == "" {
		o.AuthModeVal = llm.AuthBearer
	}
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

// Setting get the connection setting.
// Returns ALL data (metadata + API params + capabilities) for backward compatibility.
// New code should prefer the typed LLMConnector methods instead.
func (o *Connector) Setting() map[string]interface{} {
	host := o.GetURL()

	setting := map[string]interface{}{
		"host":      host,
		"key":       o.Options.Key,
		"model":     o.Options.Model,
		"auth_mode": string(o.AuthModeVal),
	}

	setting["capabilities"] = o.resolveCapabilities()

	if o.Options.Thinking != nil {
		setting["thinking"] = o.Options.Thinking
	}
	if o.Options.MaxTokens > 0 {
		setting["max_tokens"] = o.Options.MaxTokens
	}
	if o.Options.Temperature != nil {
		setting["temperature"] = *o.Options.Temperature
	}
	if len(o.Options.Protocols) > 0 {
		setting["protocols"] = o.Options.Protocols
	}

	for k, v := range o.Options.ExtraBody {
		if _, exists := setting[k]; !exists {
			setting[k] = v
		}
	}

	return setting
}

// --- LLMConnector interface implementation ---

// GetAuthMode returns the authentication mode (bearer/api-key/x-api-key).
func (o *Connector) GetAuthMode() llm.AuthMode {
	return o.AuthModeVal
}

// GetURL returns the API base URL. Does NOT include endpoint paths like /chat/completions.
func (o *Connector) GetURL() string {
	if o.Options.Host != "" {
		return o.Options.Host
	}
	if o.Options.Proxy != "" {
		return o.Options.Proxy
	}
	return "https://api.openai.com"
}

// GetKey returns the API key.
func (o *Connector) GetKey() string {
	return o.Options.Key
}

// GetModel returns the configured model name.
func (o *Connector) GetModel() string {
	return o.Options.Model
}

// GetSupportedParams returns the explicit supported_params whitelist.
// nil means "use provider-type defaults" in FilterRequestBodyParams.
func (o *Connector) GetSupportedParams() map[string]*llm.ParamSpec {
	return o.SupportedParams
}

// GetCapabilities returns the model capabilities with defaults applied.
func (o *Connector) GetCapabilities() *llm.Capabilities {
	return o.resolveCapabilities()
}

// GetMetaInfo returns the meta information
func (o *Connector) GetMetaInfo() types.MetaInfo {
	return o.MetaInfo
}

// resolveCapabilities returns capabilities with model-based defaults applied.
func (o *Connector) resolveCapabilities() *llm.Capabilities {
	capabilities := o.Options.Capabilities
	if capabilities == nil {
		modelLower := strings.ToLower(o.Options.Model)
		capabilities = GetDefaultCapabilities(modelLower)
		if capabilities == nil {
			capabilities = &Capabilities{
				TemperatureAdjustable: true,
			}
		}
	}

	if !capabilities.Reasoning && !capabilities.TemperatureAdjustable {
		capabilities.TemperatureAdjustable = true
	}
	return capabilities
}
