package types

import (
	"context"
	"encoding/json"
	"time"

	"github.com/yaoapp/gou/types"
)

// TransportType the transport type
type TransportType string

const (
	// TransportHTTP the HTTP transport
	TransportHTTP TransportType = "http"
	// TransportSSE the SSE transport
	TransportSSE TransportType = "sse"
	// TransportStdio the Stdio transport
	TransportStdio TransportType = "stdio"
	// TransportProcess the Yao transport ( For Yao internal processes)
	TransportProcess TransportType = "process"
)

// Protocol version
const (
	ProtocolVersion = "2025-06-18"
)

// SampleItemType represents the type of item for samples
type SampleItemType string

// Sample item types
const (
	SampleTool     SampleItemType = "tool"
	SampleResource SampleItemType = "resource"
)

// Message types
const (
	TypeRequest      = "request"
	TypeResponse     = "response"
	TypeNotification = "notification"
)

// ================================
// DSLs Of the MCP Server and Client
// ================================

// ClientDSL the MCP Client DSL
type ClientDSL struct {
	ID        string        `json:"id,omitempty"`      // The ID of the MCP Client (required)
	Name      string        `json:"name"`              // The name of the MCP Client (required)
	Version   string        `json:"version,omitempty"` // The version of the MCP Client
	Type      string        `json:"type,omitempty"`    // The type of the MCP Client (e.g., "standard", "agent", "system")
	Transport TransportType `json:"transport"`         // One of the TransportType (required)

	types.MetaInfo

	// Client capabilities configuration
	EnableSampling    bool `json:"enable_sampling,omitempty"`    // Enable sampling capability
	EnableRoots       bool `json:"enable_roots,omitempty"`       // Enable roots capability
	RootsListChanged  bool `json:"roots_list_changed,omitempty"` // Whether to be notified of root changes
	EnableElicitation bool `json:"enable_elicitation,omitempty"` // Enable elicitation capability

	// For HTTP, SSE transport
	URL                string `json:"url,omitempty"`                 // For HTTP、SSE transport (optional)
	Endpoint           string `json:"endpoint,omitempty"`            // API endpoint path (e.g., "/api/mcp")
	AuthorizationToken string `json:"authorization_token,omitempty"` // For HTTP、SSE transport

	// For stdio transport
	Command   string            `json:"command,omitempty"`   // for stdio transport
	Arguments []string          `json:"arguments,omitempty"` // for stdio transport
	Env       map[string]string `json:"env,omitempty"`       // for stdio transport

	Timeout string `json:"timeout,omitempty"` // for HTTP、SSE transport (1s, 1m, 1h, 1d)

	// For process transport - mapping data (tools, prompts, resources)
	Tools     map[string]string `json:"tools,omitempty"`     // tool_name -> process_name mapping
	Resources map[string]string `json:"resources,omitempty"` // resource_name -> process_name mapping
	Prompts   map[string]string `json:"prompts,omitempty"`   // prompt_name -> process_name mapping
}

// ================================

// Message JSON-RPC 2.0 message structure
type Message struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard error codes
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
)

// Initialize request and response
type InitializeRequest struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    ClientCapabilities     `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
	Meta            map[string]interface{} `json:"meta,omitempty"`
}

type InitializeResponse struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    ServerCapabilities     `json:"capabilities"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
	Meta            map[string]interface{} `json:"meta,omitempty"`
}

// Client and Server info
type ClientInfo struct {
	ID          string        `json:"id,omitempty"`          // Client ID
	Name        string        `json:"name"`                  // Client name
	Version     string        `json:"version,omitempty"`     // Client version
	Type        string        `json:"type,omitempty"`        // Client type (standard, agent, system)
	Transport   TransportType `json:"transport,omitempty"`   // Transport type
	Label       string        `json:"label,omitempty"`       // Display label
	Description string        `json:"description,omitempty"` // Description
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Implementation describes the name and version of an MCP implementation
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Capabilities
type ClientCapabilities struct {
	Sampling     *SamplingCapability    `json:"sampling,omitempty"`
	Roots        *RootsCapability       `json:"roots,omitempty"`
	Elicitation  *ElicitationCapability `json:"elicitation,omitempty"`
	Experimental map[string]interface{} `json:"experimental,omitempty"`
}

type ServerCapabilities struct {
	Resources    *ResourcesCapability   `json:"resources,omitempty"`
	Tools        *ToolsCapability       `json:"tools,omitempty"`
	Prompts      *PromptsCapability     `json:"prompts,omitempty"`
	Logging      *LoggingCapability     `json:"logging,omitempty"`
	Experimental map[string]interface{} `json:"experimental,omitempty"`
}

type SamplingCapability struct{}

type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ElicitationCapability struct{}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type LoggingCapability struct{}

// Resource types
type Resource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
}

type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
}

type ListResourcesRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

type ListResourcesResponse struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

type ReadResourceRequest struct {
	URI string `json:"uri"`
}

type ReadResourceResponse struct {
	Contents []ResourceContent `json:"contents"`
}

type ResourceUpdatedNotification struct {
	URI string `json:"uri"`
}

// Tool types
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema json.RawMessage        `json:"inputSchema"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
}

type ListToolsRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

type ListToolsResponse struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

type CallToolRequest struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments,omitempty"`
}

type CallToolResponse struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContentType represents the type of tool content
type ToolContentType string

const (
	ToolContentTypeText     ToolContentType = "text"
	ToolContentTypeImage    ToolContentType = "image"
	ToolContentTypeResource ToolContentType = "resource"
)

type ToolContent struct {
	Type ToolContentType `json:"type"`

	// Text content
	Text string `json:"text,omitempty"`

	// Image content (base64 encoded)
	Data     string `json:"data,omitempty"`     // base64-encoded image data
	MimeType string `json:"mimeType,omitempty"` // MIME type for image

	// Resource content
	Resource *EmbeddedResource `json:"resource,omitempty"`
}

// Batch tool call types
type ToolCall struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments,omitempty"`
}

type CallToolsBatchRequest struct {
	Tools []ToolCall `json:"tools"`
}

type CallToolsResponse struct {
	Results []CallToolResponse `json:"results"`
}

// Deprecated: Use CallToolsResponse instead
type CallToolsBatchResponse = CallToolsResponse

// Prompt types
type Prompt struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
}

type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type ListPromptsRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

type ListPromptsResponse struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

type GetPromptRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type GetPromptResponse struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptRole represents the role of a prompt message
type PromptRole string

const (
	PromptRoleUser      PromptRole = "user"
	PromptRoleAssistant PromptRole = "assistant"
)

type PromptMessage struct {
	Role    PromptRole    `json:"role"`
	Content PromptContent `json:"content"`
}

// PromptContentType represents the type of prompt content
type PromptContentType string

const (
	PromptContentTypeText     PromptContentType = "text"
	PromptContentTypeImage    PromptContentType = "image"
	PromptContentTypeAudio    PromptContentType = "audio"
	PromptContentTypeResource PromptContentType = "resource"
)

type PromptContent struct {
	Type PromptContentType `json:"type"`

	// Text content
	Text string `json:"text,omitempty"`

	// Image content
	Data     string `json:"data,omitempty"`     // base64-encoded data for image/audio
	MimeType string `json:"mimeType,omitempty"` // MIME type for image/audio

	// Audio content (same fields as image)
	// Data and MimeType are reused

	// Resource content
	Resource *EmbeddedResource `json:"resource,omitempty"`
}

// EmbeddedResource represents an embedded resource in prompt content
type EmbeddedResource struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64-encoded binary data
}

// Sample types - for training/example data from .jsonl files
type SampleData struct {
	Index    int    `json:"index"`               // Auto-generated index (set by loader)
	Name     string `json:"name,omitempty"`      // Optional sample name
	ItemName string `json:"item_name,omitempty"` // Tool/Resource name (set by loader)

	// Multi-purpose fields (semantics depend on item type)
	Input  map[string]interface{} `json:"input,omitempty"`  // For tools: input args; For resources: parsed from URI
	Output interface{}            `json:"output,omitempty"` // For tools: expected output; Not used for resources
	URI    string                 `json:"uri,omitempty"`    // For resources: full URI with params
	Data   interface{}            `json:"data,omitempty"`   // For resources: response data

	Metadata  map[string]interface{} `json:"metadata,omitempty"`  // Optional metadata (e.g., description)
	Timestamp string                 `json:"timestamp,omitempty"` // Optional timestamp
}

type ListSamplesResponse struct {
	Samples []SampleData `json:"samples"`
	Total   int          `json:"total"`
}

// Sampling types (LLM text generation - MCP protocol standard)
type SamplingRequest struct {
	Model          string                 `json:"model"`
	Messages       []SamplingMessage      `json:"messages"`
	SystemPrompt   string                 `json:"systemPrompt,omitempty"`
	IncludeContext string                 `json:"includeContext,omitempty"`
	Temperature    float64                `json:"temperature,omitempty"`
	MaxTokens      int                    `json:"maxTokens,omitempty"`
	StopSequences  []string               `json:"stopSequences,omitempty"`
	Meta           map[string]interface{} `json:"meta,omitempty"`
}

type SamplingMessage struct {
	Role    string          `json:"role"`
	Content SamplingContent `json:"content"`
}

type SamplingContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

type SamplingResponse struct {
	Model      string          `json:"model"`
	Role       string          `json:"role"`
	Content    SamplingContent `json:"content"`
	StopReason string          `json:"stopReason,omitempty"`
}

// Progress types
type Progress struct {
	Token uint64 `json:"token"`
	Total uint64 `json:"total,omitempty"`
}

type ProgressNotification struct {
	Token    uint64 `json:"token"`
	Progress uint64 `json:"progress"`
	Total    uint64 `json:"total,omitempty"`
}

// Cancellation types
type CancelRequest struct {
	RequestID interface{} `json:"requestId"`
}

// Logging types
type LogLevel string

const (
	LogLevelDebug     LogLevel = "debug"
	LogLevelInfo      LogLevel = "info"
	LogLevelNotice    LogLevel = "notice"
	LogLevelWarning   LogLevel = "warning"
	LogLevelError     LogLevel = "error"
	LogLevelCritical  LogLevel = "critical"
	LogLevelAlert     LogLevel = "alert"
	LogLevelEmergency LogLevel = "emergency"
)

type LogMessage struct {
	Level  LogLevel               `json:"level"`
	Data   interface{}            `json:"data"`
	Logger string                 `json:"logger,omitempty"`
	Meta   map[string]interface{} `json:"meta,omitempty"`
}

type SetLogLevelRequest struct {
	Level LogLevel `json:"level"`
}

// Connection state
type ConnectionState string

const (
	StateDisconnected ConnectionState = "disconnected"
	StateConnecting   ConnectionState = "connecting"
	StateConnected    ConnectionState = "connected"
	StateInitialized  ConnectionState = "initialized"
	StateError        ConnectionState = "error"
)

// ConnectionOptions provides options for establishing connection
type ConnectionOptions struct {
	// Headers for HTTP/SSE transports (e.g., Mcp-Session-Id, Custom-Auth, etc.)
	Headers map[string]string `json:"headers,omitempty"`
	// Timeout for connection establishment
	Timeout time.Duration `json:"timeout,omitempty"`
	// Retry configuration
	MaxRetries int           `json:"max_retries,omitempty"`
	RetryDelay time.Duration `json:"retry_delay,omitempty"`
}

// Transport interface
type Transport interface {
	Start(ctx context.Context) error
	Stop() error
	Send(message Message) error
	Receive() (<-chan Message, error)
	Close() error
}

// Configuration
type Config struct {
	ServerCommand []string               `json:"serverCommand,omitempty"`
	ServerEnv     map[string]string      `json:"serverEnv,omitempty"`
	InitOptions   map[string]interface{} `json:"initOptions,omitempty"`
	Timeout       time.Duration          `json:"timeout,omitempty"`
}

// Handler function types
type RequestHandler func(ctx context.Context, request Message) (interface{}, error)
type NotificationHandler func(ctx context.Context, notification Message) error
type ErrorHandler func(ctx context.Context, err error) error

// Event types
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

const (
	EventTypeConnected    = "connected"
	EventTypeDisconnected = "disconnected"
	EventTypeError        = "error"
	EventTypeMessage      = "message"
)

// ================================
// MCP Mapping Data (for Process Transport)
// ================================

// ToolSchema represents a tool's input/output schema
type ToolSchema struct {
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Process      string          `json:"process"`                  // Yao process name
	InputSchema  json.RawMessage `json:"inputSchema"`              // JSON Schema for input
	OutputSchema json.RawMessage `json:"outputSchema,omitempty"`   // Optional JSON Schema for output
	ProcessArgs  []string        `json:"x-process-args,omitempty"` // Mapping from MCP arguments to Process positional args
}

// ResourceSchema represents a resource definition
type ResourceSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Process     string                 `json:"process"` // Yao process name
	URI         string                 `json:"uri"`     // URI template (e.g., "customer://{id}")
	MimeType    string                 `json:"mimeType,omitempty"`
	Parameters  []ResourceParameter    `json:"parameters,omitempty"` // URI parameters
	Meta        map[string]interface{} `json:"meta,omitempty"`
	ProcessArgs []string               `json:"x-process-args,omitempty"` // Mapping from URI/parameters to Process positional args
}

// ResourceParameter defines a parameter for a resource URI
type ResourceParameter struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptSchema represents a prompt template definition
type PromptSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Template    string                 `json:"template"` // Prompt template content
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
}

// MappingData contains all loaded schemas for a process-based MCP client
type MappingData struct {
	Tools     map[string]*ToolSchema     `json:"tools"`     // tool_name -> ToolSchema
	Resources map[string]*ResourceSchema `json:"resources"` // resource_name -> ResourceSchema
	Prompts   map[string]*PromptSchema   `json:"prompts"`   // prompt_name -> PromptSchema
}
