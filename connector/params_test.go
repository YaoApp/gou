package connector

import (
	"testing"

	"github.com/yaoapp/gou/llm"
	gouTypes "github.com/yaoapp/gou/types"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
)

// mockLLMConnector implements both Connector and llm.LLMConnector for testing.
type mockLLMConnector struct {
	typ      int
	settings map[string]interface{}
	authMode llm.AuthMode
	url      string
	key      string
	model    string
	params   map[string]*llm.ParamSpec
	caps     *llm.Capabilities
}

func (m *mockLLMConnector) Register(string, string, []byte) error         { return nil }
func (m *mockLLMConnector) Query() (query.Query, error)                   { return nil, nil }
func (m *mockLLMConnector) Schema() (schema.Schema, error)                { return nil, nil }
func (m *mockLLMConnector) Close() error                                  { return nil }
func (m *mockLLMConnector) ID() string                                    { return "mock" }
func (m *mockLLMConnector) Is(t int) bool                                 { return m.typ == t }
func (m *mockLLMConnector) Setting() map[string]interface{}               { return m.settings }
func (m *mockLLMConnector) GetMetaInfo() gouTypes.MetaInfo                { return gouTypes.MetaInfo{} }
func (m *mockLLMConnector) GetAuthMode() llm.AuthMode                     { return m.authMode }
func (m *mockLLMConnector) GetURL() string                                { return m.url }
func (m *mockLLMConnector) GetKey() string                                { return m.key }
func (m *mockLLMConnector) GetModel() string                              { return m.model }
func (m *mockLLMConnector) GetSupportedParams() map[string]*llm.ParamSpec { return m.params }
func (m *mockLLMConnector) GetCapabilities() *llm.Capabilities            { return m.caps }

// plainConnector implements only Connector (not LLMConnector).
type plainConnector struct {
	typ      int
	settings map[string]interface{}
}

func (p *plainConnector) Register(string, string, []byte) error { return nil }
func (p *plainConnector) Query() (query.Query, error)           { return nil, nil }
func (p *plainConnector) Schema() (schema.Schema, error)        { return nil, nil }
func (p *plainConnector) Close() error                          { return nil }
func (p *plainConnector) ID() string                            { return "plain" }
func (p *plainConnector) Is(t int) bool                         { return p.typ == t }
func (p *plainConnector) Setting() map[string]interface{}       { return p.settings }
func (p *plainConnector) GetMetaInfo() gouTypes.MetaInfo        { return gouTypes.MetaInfo{} }

func TestFilterRequestBodyParams_WithSupportedParams(t *testing.T) {
	max06 := 0.6
	conn := &mockLLMConnector{
		typ: OPENAI,
		params: map[string]*llm.ParamSpec{
			"temperature": {Max: &max06},
			"max_tokens":  nil,
			"thinking":    nil,
		},
		settings: map[string]interface{}{
			"host":        "https://api.example.com",
			"key":         "sk-test",
			"model":       "gpt-4",
			"temperature": 0.9,
			"max_tokens":  4096,
			"thinking":    map[string]interface{}{"type": "enabled"},
			"auth_mode":   "bearer",
		},
	}

	result := FilterRequestBodyParams(conn.settings, conn)

	if result["temperature"].(float64) != 0.6 {
		t.Errorf("expected temperature clamped to 0.6, got %v", result["temperature"])
	}
	if result["max_tokens"].(int) != 4096 {
		t.Errorf("expected max_tokens 4096, got %v", result["max_tokens"])
	}
	if result["thinking"] == nil {
		t.Error("expected thinking to be present")
	}
	if _, ok := result["host"]; ok {
		t.Error("host should be filtered out")
	}
	if _, ok := result["auth_mode"]; ok {
		t.Error("auth_mode should be filtered out")
	}
	if _, ok := result["key"]; ok {
		t.Error("key should be filtered out")
	}
}

func TestFilterRequestBodyParams_TypeDefaults_OpenAI(t *testing.T) {
	conn := &plainConnector{
		typ: OPENAI,
		settings: map[string]interface{}{
			"host":         "https://api.openai.com",
			"key":          "sk-test",
			"model":        "gpt-4",
			"temperature":  0.7,
			"auth_mode":    "bearer",
			"capabilities": &llm.Capabilities{Streaming: true},
		},
	}

	result := FilterRequestBodyParams(conn.settings, conn)

	if result["temperature"].(float64) != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", result["temperature"])
	}
	if _, ok := result["host"]; ok {
		t.Error("host should not be in result")
	}
	if _, ok := result["auth_mode"]; ok {
		t.Error("auth_mode should not be in result")
	}
	if _, ok := result["capabilities"]; ok {
		t.Error("capabilities should not be in result")
	}
}

func TestFilterRequestBodyParams_TypeDefaults_Anthropic(t *testing.T) {
	conn := &plainConnector{
		typ: ANTHROPIC,
		settings: map[string]interface{}{
			"host":      "https://api.anthropic.com",
			"key":       "sk-ant-test",
			"model":     "claude-sonnet-4",
			"version":   "2023-06-01",
			"thinking":  map[string]interface{}{"type": "enabled"},
			"protocols": []string{"anthropic"},
		},
	}

	result := FilterRequestBodyParams(conn.settings, conn)

	if result["thinking"] == nil {
		t.Error("thinking should be in result (valid Anthropic param)")
	}
	if _, ok := result["host"]; ok {
		t.Error("host should not be in result")
	}
	if _, ok := result["version"]; ok {
		t.Error("version should not be in result")
	}
	if _, ok := result["protocols"]; ok {
		t.Error("protocols should not be in result")
	}
}

func TestFilterRequestBodyParams_UnknownType(t *testing.T) {
	conn := &plainConnector{
		typ: 999, // unknown
		settings: map[string]interface{}{
			"host":         "https://custom.api.com",
			"key":          "custom-key",
			"model":        "custom-model",
			"temperature":  0.5,
			"custom_param": "value",
		},
	}

	result := FilterRequestBodyParams(conn.settings, conn)

	// Unknown type: strip metadata, keep everything else
	if _, ok := result["host"]; ok {
		t.Error("host should be stripped")
	}
	if _, ok := result["key"]; ok {
		t.Error("key should be stripped")
	}
	if result["temperature"].(float64) != 0.5 {
		t.Error("temperature should pass through")
	}
	if result["custom_param"].(string) != "value" {
		t.Error("custom_param should pass through")
	}
}

func TestFilterRequestBodyParams_NilInputs(t *testing.T) {
	result := FilterRequestBodyParams(nil, nil)
	if result == nil || len(result) != 0 {
		t.Error("expected empty map for nil settings")
	}

	result = FilterRequestBodyParams(map[string]interface{}{"host": "x", "temp": 0.5}, nil)
	if _, ok := result["host"]; ok {
		t.Error("host should be stripped even with nil conn")
	}
}

func TestClampValue_Min(t *testing.T) {
	min := 0.1
	spec := &llm.ParamSpec{Min: &min}
	result := clampValue(0.05, spec)
	if result.(float64) != 0.1 {
		t.Errorf("expected 0.1, got %v", result)
	}
}

func TestClampValue_Max(t *testing.T) {
	max := 0.6
	spec := &llm.ParamSpec{Max: &max}
	result := clampValue(0.9, spec)
	if result.(float64) != 0.6 {
		t.Errorf("expected 0.6, got %v", result)
	}
}

func TestClampValue_AllowedValues(t *testing.T) {
	spec := &llm.ParamSpec{AllowedValues: []interface{}{"auto", "none"}}
	result := clampAllowed("invalid", spec.AllowedValues)
	if result.(string) != "auto" {
		t.Errorf("expected 'auto', got %v", result)
	}

	result = clampAllowed("none", spec.AllowedValues)
	if result.(string) != "none" {
		t.Errorf("expected 'none', got %v", result)
	}
}

func TestConnectorTypeName(t *testing.T) {
	openai := &plainConnector{typ: OPENAI}
	if connectorTypeName(openai) != "openai" {
		t.Error("expected openai")
	}

	anthropic := &plainConnector{typ: ANTHROPIC}
	if connectorTypeName(anthropic) != "anthropic" {
		t.Error("expected anthropic")
	}

	unknown := &plainConnector{typ: 999}
	if connectorTypeName(unknown) != "" {
		t.Error("expected empty string")
	}

	if connectorTypeName(nil) != "" {
		t.Error("expected empty string for nil")
	}
}
