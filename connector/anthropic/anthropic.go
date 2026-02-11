package anthropic

import (
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
)

// Connector anthropic connector
type Connector struct {
	id      string
	file    string
	Name    string  `json:"name"`
	Options Options `json:"options"`
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
}

// Capabilities defines the capabilities of an Anthropic model
type Capabilities struct {
	Vision                interface{} `json:"vision,omitempty" yaml:"vision,omitempty"`
	Audio                 bool        `json:"audio,omitempty" yaml:"audio,omitempty"`
	ToolCalls             bool        `json:"tool_calls,omitempty" yaml:"tool_calls,omitempty"`
	Reasoning             bool        `json:"reasoning,omitempty" yaml:"reasoning,omitempty"`
	Streaming             bool        `json:"streaming,omitempty" yaml:"streaming,omitempty"`
	JSON                  bool        `json:"json,omitempty" yaml:"json,omitempty"`
	Multimodal            bool        `json:"multimodal,omitempty" yaml:"multimodal,omitempty"`
	TemperatureAdjustable bool        `json:"temperature_adjustable,omitempty" yaml:"temperature_adjustable,omitempty"`
}

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

// Setting get the connection setting
func (c *Connector) Setting() map[string]interface{} {

	// Determine API endpoint
	host := "https://api.anthropic.com"
	if c.Options.Host != "" {
		host = c.Options.Host
	} else if c.Options.Proxy != "" {
		host = c.Options.Proxy
	}

	// API version
	version := "2023-06-01"
	if c.Options.Version != "" {
		version = c.Options.Version
	}

	setting := map[string]interface{}{
		"host":    host,
		"key":     c.Options.Key,
		"model":   c.Options.Model,
		"version": version,
	}

	// Add capabilities with defaults if not provided
	capabilities := c.Options.Capabilities
	if capabilities == nil {
		modelLower := strings.ToLower(c.Options.Model)
		capabilities = GetDefaultCapabilities(modelLower)

		if capabilities == nil {
			capabilities = &Capabilities{
				Vision:                "claude",
				ToolCalls:             true,
				Audio:                 false,
				Reasoning:             false,
				Streaming:             true,
				JSON:                  true,
				Multimodal:            true,
				TemperatureAdjustable: true,
			}
		}
	}

	setting["capabilities"] = capabilities

	// Add thinking configuration if present
	if c.Options.Thinking != nil {
		setting["thinking"] = c.Options.Thinking
	}

	// Add max_tokens if specified
	if c.Options.MaxTokens > 0 {
		setting["max_tokens"] = c.Options.MaxTokens
	}

	// Add temperature if specified
	if c.Options.Temperature != nil {
		setting["temperature"] = *c.Options.Temperature
	}

	return setting
}

// GetMetaInfo returns the meta information
func (c *Connector) GetMetaInfo() types.MetaInfo {
	return c.MetaInfo
}
