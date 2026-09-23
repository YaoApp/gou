package typesafe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/llm"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
)

const defaultEndpoint = "/v1/systemone"

// Connector implements the TypeSafe AI (Jev) decision connector.
// One implementation serves both Tao proxy and BYOK direct connections;
// only Host, Endpoint, and Key differ between the two.
type Connector struct {
	id      string
	file    string
	Name    string  `json:"name"`
	Options Options `json:"options"`
	types.MetaInfo
	types.ConnectorMetadata
}

// Options configures the TypeSafe connector.
type Options struct {
	Host     string `json:"host,omitempty"`     // Tao: base URL, BYOK: "https://api.typesafe.ai"
	Model    string `json:"model,omitempty"`    // default "jev-latest"
	Key      string `json:"key"`                // API key (Tao key or TypeSafe key)
	Endpoint string `json:"endpoint,omitempty"` // Tao: from /v1/services path, BYOK: defaults to /v1/systemone

	Capabilities *llm.Capabilities `json:"capabilities,omitempty"`

	// Extra parameters forwarded verbatim into the API request body.
	ExtraBody map[string]interface{} `json:"extra_body,omitempty"`
}

// Register parses the connector DSL and stores configuration.
func (c *Connector) Register(file string, id string, dsl []byte) error {
	c.id = id
	c.file = file
	err := application.Parse(file, dsl, c)
	if err != nil {
		return err
	}

	c.Options.Host = helper.EnvString(c.Options.Host)
	c.Options.Model = helper.EnvString(c.Options.Model)
	c.Options.Key = helper.EnvString(c.Options.Key)
	c.Options.Endpoint = helper.EnvString(c.Options.Endpoint)
	return nil
}

// Is reports whether the connector matches the given type constant.
func (c *Connector) Is(typ int) bool {
	return 12 == typ // TYPESAFE constant value
}

// ID returns the connector identifier.
func (c *Connector) ID() string {
	return c.id
}

// Query is not supported by decision connectors.
func (c *Connector) Query() (query.Query, error) {
	return nil, nil
}

// Schema is not supported by decision connectors.
func (c *Connector) Schema() (schema.Schema, error) {
	return nil, nil
}

// Close releases resources (none for HTTP-based connector).
func (c *Connector) Close() error {
	return nil
}

// Setting returns the connector configuration as a map.
func (c *Connector) Setting() map[string]interface{} {
	return map[string]interface{}{
		"host":     c.GetURL(),
		"key":      c.Options.Key,
		"model":    c.Options.Model,
		"endpoint": c.endpoint(),
	}
}

// GetMetaInfo returns the meta information.
func (c *Connector) GetMetaInfo() types.MetaInfo {
	return c.MetaInfo
}

// --- LLMConnector interface ---

// GetAuthMode returns bearer authentication.
func (c *Connector) GetAuthMode() llm.AuthMode {
	return llm.AuthBearer
}

// GetURL returns the API host. Does NOT include endpoint paths.
func (c *Connector) GetURL() string {
	if c.Options.Host != "" {
		return c.Options.Host
	}
	return "https://api.typesafe.ai"
}

// GetKey returns the API key.
func (c *Connector) GetKey() string {
	return c.Options.Key
}

// GetModel returns the configured model name.
func (c *Connector) GetModel() string {
	if c.Options.Model != "" {
		return c.Options.Model
	}
	return "jev-latest"
}

// GetSupportedParams returns nil (TypeSafe has a fixed request format).
func (c *Connector) GetSupportedParams() map[string]*llm.ParamSpec {
	return nil
}

// GetCapabilities returns the decision capability.
func (c *Connector) GetCapabilities() *llm.Capabilities {
	if c.Options.Capabilities != nil {
		return c.Options.Capabilities
	}
	return &llm.Capabilities{Decision: true}
}

// --- DecisionConnector interface ---

// Decide sends a typed decision request and returns structured answers.
// URL is constructed as Host + Endpoint (no BuildAPIURL, no /v1 prefix insertion).
func (c *Connector) Decide(req *llm.DecisionRequest) (*llm.DecisionResponse, error) {
	if req.Model == "" {
		req.Model = c.GetModel()
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("typesafe: marshal request: %w", err)
	}

	url := strings.TrimRight(c.GetURL(), "/") + c.endpoint()
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("typesafe: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.GetKey())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("typesafe: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("typesafe: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("typesafe: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result llm.DecisionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("typesafe: unmarshal response: %w", err)
	}
	return &result, nil
}

// endpoint returns the API endpoint path, defaulting to /v1/systemone for BYOK.
func (c *Connector) endpoint() string {
	if c.Options.Endpoint != "" {
		return c.Options.Endpoint
	}
	return defaultEndpoint
}
