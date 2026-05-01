package anthropic

import (
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/llm"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
)

// Connector anthropic connector
type Connector struct {
	id              string
	file            string
	Name            string                    `json:"name"`
	AuthModeVal     llm.AuthMode              `json:"auth_mode,omitempty"`
	SupportedParams map[string]*llm.ParamSpec `json:"supported_params,omitempty"`
	Options         Options                   `json:"options"`
	types.MetaInfo
}

// Options the anthropic connector options
type Options struct {
	Host    string `json:"host,omitempty"`    // API endpoint, default: https://api.anthropic.com
	Proxy   string `json:"proxy,omitempty"`   // (Deprecated) API endpoint, use Host instead
	Model   string `json:"model,omitempty"`   // Model name, e.g. "claude-sonnet-4-20250514"
	Key     string `json:"key"`               // API key
	Version string `json:"version,omitempty"` // Anthropic API version header, default: "2023-06-01"

	// Model Capabilities
	Capabilities *Capabilities `json:"capabilities,omitempty"`

	// Thinking mode configuration
	// Example: {"type": "enabled", "budget_tokens": 10000} or {"type": "disabled"}
	Thinking interface{} `json:"thinking,omitempty"`

	// Request parameters
	MaxTokens   int      `json:"max_tokens,omitempty"`  // Maximum tokens for response (required for Anthropic)
	Temperature *float64 `json:"temperature,omitempty"` // Temperature (use pointer to distinguish 0 from unset)

	// Supported API protocols. Defaults to ["anthropic"] when empty.
	// Dual-protocol gateways declare ["openai", "anthropic"].
	Protocols []string `json:"protocols,omitempty"`
}

// Capabilities is an alias for llm.Capabilities for backward compatibility.
// New code should use llm.Capabilities directly.
type Capabilities = llm.Capabilities

// Register the connector from dsl
func (c *Connector) Register(file string, id string, dsl []byte) error {
	c.id = id
	c.file = file
	err := application.Parse(file, dsl, c)
	if err != nil {
		return err
	}

	c.Options.Host = helper.EnvString(c.Options.Host)
	c.Options.Proxy = helper.EnvString(c.Options.Proxy)
	c.Options.Model = helper.EnvString(c.Options.Model)
	c.Options.Key = helper.EnvString(c.Options.Key)
	c.Options.Version = helper.EnvString(c.Options.Version)

	if c.AuthModeVal == "" {
		c.AuthModeVal = llm.AuthXAPIKey
	}
	return nil
}

// Is checks the connector type (ANTHROPIC = 11)
func (c *Connector) Is(typ int) bool {
	return 11 == typ
}

// ID get connector id
func (c *Connector) ID() string {
	return c.id
}

// Query get connector query interface (not applicable for AI connectors)
func (c *Connector) Query() (query.Query, error) {
	return nil, nil
}

// Schema get connector schema interface (not applicable for AI connectors)
func (c *Connector) Schema() (schema.Schema, error) {
	return nil, nil
}

// Close connections
func (c *Connector) Close() error {
	return nil
}

// Setting get the connection setting.
// Returns ALL data for backward compatibility. New code should prefer LLMConnector methods.
func (c *Connector) Setting() map[string]interface{} {
	host := c.GetURL()

	version := "2023-06-01"
	if c.Options.Version != "" {
		version = c.Options.Version
	}

	setting := map[string]interface{}{
		"host":      host,
		"key":       c.Options.Key,
		"model":     c.Options.Model,
		"version":   version,
		"auth_mode": string(c.AuthModeVal),
	}

	setting["capabilities"] = c.resolveCapabilities()

	if c.Options.Thinking != nil {
		setting["thinking"] = c.Options.Thinking
	}
	if c.Options.MaxTokens > 0 {
		setting["max_tokens"] = c.Options.MaxTokens
	}
	if c.Options.Temperature != nil {
		setting["temperature"] = *c.Options.Temperature
	}
	if len(c.Options.Protocols) > 0 {
		setting["protocols"] = c.Options.Protocols
	}

	return setting
}

// --- LLMConnector interface implementation ---

// GetAuthMode returns the authentication mode (default: x-api-key for Anthropic).
func (c *Connector) GetAuthMode() llm.AuthMode {
	return c.AuthModeVal
}

// GetURL returns the API base URL. Does NOT include endpoint paths.
func (c *Connector) GetURL() string {
	if c.Options.Host != "" {
		return c.Options.Host
	}
	if c.Options.Proxy != "" {
		return c.Options.Proxy
	}
	return "https://api.anthropic.com"
}

// GetKey returns the API key.
func (c *Connector) GetKey() string {
	return c.Options.Key
}

// GetModel returns the configured model name.
func (c *Connector) GetModel() string {
	return c.Options.Model
}

// GetSupportedParams returns the explicit supported_params whitelist.
func (c *Connector) GetSupportedParams() map[string]*llm.ParamSpec {
	return c.SupportedParams
}

// GetCapabilities returns the model capabilities with defaults applied.
func (c *Connector) GetCapabilities() *llm.Capabilities {
	return c.resolveCapabilities()
}

// GetMetaInfo returns the meta information
func (c *Connector) GetMetaInfo() types.MetaInfo {
	return c.MetaInfo
}

func (c *Connector) resolveCapabilities() *llm.Capabilities {
	capabilities := c.Options.Capabilities
	if capabilities == nil {
		modelLower := strings.ToLower(c.Options.Model)
		capabilities = GetDefaultCapabilities(modelLower)
		if capabilities == nil {
			capabilities = &Capabilities{
				Vision:                "claude",
				ToolCalls:             true,
				Streaming:             true,
				JSON:                  true,
				Multimodal:            true,
				TemperatureAdjustable: true,
			}
		}
	}
	return capabilities
}
