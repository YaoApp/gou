package llm

// AuthMode defines how an LLM connector authenticates with its API.
type AuthMode string

const (
	AuthBearer  AuthMode = "bearer"    // Authorization: Bearer {key}
	AuthAPIKey  AuthMode = "api-key"   // api-key: {key} (Azure OpenAI)
	AuthXAPIKey AuthMode = "x-api-key" // x-api-key: {key} (Anthropic)
)

// ParamSpec constrains values for a single API request-body parameter.
// An empty ParamSpec ({}) means "allowed, no constraints".
type ParamSpec struct {
	Min           *float64      `json:"min,omitempty"`
	Max           *float64      `json:"max,omitempty"`
	AllowedValues []interface{} `json:"allowed_values,omitempty"`
}

// LLMConnector provides typed access to LLM-specific metadata.
// OpenAI and Anthropic connectors implement this interface.
// Consumers should prefer these methods over digging into Setting() maps.
//
// This interface intentionally does NOT embed connector.Connector to avoid
// import cycles (connector root package imports connector subpackages).
// Use type assertion: if llm, ok := conn.(llm.LLMConnector); ok { ... }
type LLMConnector interface {
	GetAuthMode() AuthMode
	GetURL() string // API base URL (host > proxy > provider default). Does NOT include endpoint paths.
	GetKey() string
	GetModel() string
	GetSupportedParams() map[string]*ParamSpec // nil → use provider-type defaults
	GetCapabilities() *Capabilities
}
