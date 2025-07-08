package mcp

import (
	"context"
	"io"

	"github.com/yaoapp/gou/mcp/types"
)

// Client the MCP client interface
type Client interface {
	// Connection management
	Connect(ctx context.Context, options ...types.ConnectionOptions) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	State() types.ConnectionState

	// Protocol initialization
	Initialize(ctx context.Context) (*types.InitializeResponse, error)
	Initialized(ctx context.Context) error

	// Resource operations
	ListResources(ctx context.Context, cursor string) (*types.ListResourcesResponse, error)
	ReadResource(ctx context.Context, uri string) (*types.ReadResourceResponse, error)
	SubscribeResource(ctx context.Context, uri string) error
	UnsubscribeResource(ctx context.Context, uri string) error

	// Tool operations
	ListTools(ctx context.Context, cursor string) (*types.ListToolsResponse, error)
	CallTool(ctx context.Context, name string, arguments interface{}) (*types.CallToolResponse, error)
	CallToolsBatch(ctx context.Context, tools []types.ToolCall) (*types.CallToolsBatchResponse, error)

	// Prompt operations
	ListPrompts(ctx context.Context, cursor string) (*types.ListPromptsResponse, error)
	GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*types.GetPromptResponse, error)

	// Sampling operations (if supported by server)
	CreateSampling(ctx context.Context, request types.SamplingRequest) (*types.SamplingResponse, error)

	// Logging operations
	SetLogLevel(ctx context.Context, level types.LogLevel) error

	// Request cancellation
	CancelRequest(ctx context.Context, requestID interface{}) error

	// Progress tracking
	CreateProgress(ctx context.Context, total uint64) (uint64, error)
	UpdateProgress(ctx context.Context, token uint64, progress uint64) error

	// Event handling
	OnEvent(eventType string, handler func(event types.Event))
	OnNotification(method string, handler types.NotificationHandler)
	OnError(handler types.ErrorHandler)
}

// Server the MCP service interface
type Server interface {
	// Server information
	GetServerInfo() types.ServerInfo
	GetCapabilities() types.ServerCapabilities

	// Server lifecycle
	Start(ctx context.Context, transport types.Transport) error
	Stop(ctx context.Context) error
	HandleInitialize(ctx context.Context, request types.InitializeRequest) (*types.InitializeResponse, error)
	HandleInitialized(ctx context.Context) error

	// Resource handlers
	HandleListResources(ctx context.Context, request types.ListResourcesRequest) (*types.ListResourcesResponse, error)
	HandleReadResource(ctx context.Context, request types.ReadResourceRequest) (*types.ReadResourceResponse, error)
	HandleSubscribeResource(ctx context.Context, uri string) error
	HandleUnsubscribeResource(ctx context.Context, uri string) error

	// Tool handlers
	HandleListTools(ctx context.Context, request types.ListToolsRequest) (*types.ListToolsResponse, error)
	HandleCallTool(ctx context.Context, request types.CallToolRequest) (*types.CallToolResponse, error)

	// Prompt handlers
	HandleListPrompts(ctx context.Context, request types.ListPromptsRequest) (*types.ListPromptsResponse, error)
	HandleGetPrompt(ctx context.Context, request types.GetPromptRequest) (*types.GetPromptResponse, error)

	// Sampling handlers (if capability is supported)
	HandleCreateSampling(ctx context.Context, request types.SamplingRequest) (*types.SamplingResponse, error)

	// Logging handlers
	HandleSetLogLevel(ctx context.Context, request types.SetLogLevelRequest) error
	SendLogMessage(ctx context.Context, message types.LogMessage) error

	// Cancellation handlers
	HandleCancelRequest(ctx context.Context, request types.CancelRequest) error

	// Progress handlers
	HandleCreateProgress(ctx context.Context, total uint64) (uint64, error)
	SendProgressNotification(ctx context.Context, notification types.ProgressNotification) error

	// Notification sending
	SendResourceUpdated(ctx context.Context, uri string) error
	SendToolsChanged(ctx context.Context) error
	SendPromptsChanged(ctx context.Context) error
	SendRootsChanged(ctx context.Context) error

	// Request handling
	HandleRequest(ctx context.Context, request types.Message) (interface{}, error)
	HandleNotification(ctx context.Context, notification types.Message) error

	// Transport operations
	SetTransport(transport types.Transport)
	Send(ctx context.Context, message types.Message) error
	Close() error
}

// ResourceProvider provides resource functionality
type ResourceProvider interface {
	// List all available resources
	ListResources(ctx context.Context, cursor string) ([]types.Resource, string, error)
	// Read specific resource content
	ReadResource(ctx context.Context, uri string) ([]types.ResourceContent, error)
	// Check if resource exists
	HasResource(ctx context.Context, uri string) bool
	// Subscribe to resource updates
	SubscribeToResource(ctx context.Context, uri string) error
	// Unsubscribe from resource updates
	UnsubscribeFromResource(ctx context.Context, uri string) error
}

// ToolProvider provides tool functionality
type ToolProvider interface {
	// List all available tools
	ListTools(ctx context.Context, cursor string) ([]types.Tool, string, error)
	// Execute a tool with given arguments
	CallTool(ctx context.Context, name string, arguments interface{}) ([]types.ToolContent, error)
	// Check if tool exists
	HasTool(ctx context.Context, name string) bool
	// Get tool schema
	GetToolSchema(ctx context.Context, name string) (types.Tool, error)
}

// PromptProvider provides prompt functionality
type PromptProvider interface {
	// List all available prompts
	ListPrompts(ctx context.Context, cursor string) ([]types.Prompt, string, error)
	// Get prompt with arguments
	GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*types.GetPromptResponse, error)
	// Check if prompt exists
	HasPrompt(ctx context.Context, name string) bool
	// Get prompt schema
	GetPromptSchema(ctx context.Context, name string) (types.Prompt, error)
}

// SamplingProvider provides sampling functionality
type SamplingProvider interface {
	// Create sampling request
	CreateSampling(ctx context.Context, request types.SamplingRequest) (*types.SamplingResponse, error)
	// Check if sampling is supported
	SupportsSampling() bool
}

// LoggingProvider provides logging functionality
type LoggingProvider interface {
	// Set log level
	SetLogLevel(ctx context.Context, level types.LogLevel) error
	// Send log message
	SendLogMessage(ctx context.Context, message types.LogMessage) error
	// Get current log level
	GetLogLevel() types.LogLevel
}

// ProgressProvider provides progress tracking functionality
type ProgressProvider interface {
	// Create progress token
	CreateProgress(ctx context.Context, total uint64) (uint64, error)
	// Update progress
	UpdateProgress(ctx context.Context, token uint64, progress uint64) error
	// Complete progress
	CompleteProgress(ctx context.Context, token uint64) error
}

// NotificationProvider provides notification functionality
type NotificationProvider interface {
	// Send resource updated notification
	SendResourceUpdated(ctx context.Context, uri string) error
	// Send tools changed notification
	SendToolsChanged(ctx context.Context) error
	// Send prompts changed notification
	SendPromptsChanged(ctx context.Context) error
	// Send roots changed notification
	SendRootsChanged(ctx context.Context) error
	// Send custom notification
	SendNotification(ctx context.Context, method string, params interface{}) error
}

// MessageHandler handles incoming messages
type MessageHandler interface {
	// Handle JSON-RPC request
	HandleRequest(ctx context.Context, request types.Message) (interface{}, error)
	// Handle JSON-RPC notification
	HandleNotification(ctx context.Context, notification types.Message) error
	// Handle transport error
	HandleError(ctx context.Context, err error) error
}

// TransportFactory creates transport instances
type TransportFactory interface {
	// Create transport from config
	CreateTransport(config types.Config) (types.Transport, error)
	// Create stdio transport
	CreateStdioTransport() (types.Transport, error)
	// Create websocket transport
	CreateWebSocketTransport(url string) (types.Transport, error)
	// Create tcp transport
	CreateTCPTransport(address string) (types.Transport, error)
}

// Validator validates MCP protocol messages
type Validator interface {
	// Validate message structure
	ValidateMessage(message types.Message) error
	// Validate request parameters
	ValidateRequest(method string, params interface{}) error
	// Validate response result
	ValidateResponse(method string, result interface{}) error
	// Validate capabilities
	ValidateCapabilities(capabilities interface{}) error
}

// Serializer handles message serialization
type Serializer interface {
	// Serialize message to bytes
	Serialize(message types.Message) ([]byte, error)
	// Deserialize bytes to message
	Deserialize(data []byte) (types.Message, error)
	// Serialize any object to bytes
	Marshal(obj interface{}) ([]byte, error)
	// Deserialize bytes to object
	Unmarshal(data []byte, obj interface{}) error
}

// Logger provides logging functionality
type Logger interface {
	// Log debug message
	Debug(ctx context.Context, msg string, args ...interface{})
	// Log info message
	Info(ctx context.Context, msg string, args ...interface{})
	// Log warning message
	Warn(ctx context.Context, msg string, args ...interface{})
	// Log error message
	Error(ctx context.Context, msg string, args ...interface{})
	// Log with specific level
	Log(ctx context.Context, level types.LogLevel, msg string, args ...interface{})
}

// Registry manages MCP components
type Registry interface {
	// Register resource provider
	RegisterResourceProvider(name string, provider ResourceProvider) error
	// Register tool provider
	RegisterToolProvider(name string, provider ToolProvider) error
	// Register prompt provider
	RegisterPromptProvider(name string, provider PromptProvider) error
	// Register sampling provider
	RegisterSamplingProvider(name string, provider SamplingProvider) error

	// Get resource provider
	GetResourceProvider(name string) (ResourceProvider, bool)
	// Get tool provider
	GetToolProvider(name string) (ToolProvider, bool)
	// Get prompt provider
	GetPromptProvider(name string) (PromptProvider, bool)
	// Get sampling provider
	GetSamplingProvider(name string) (SamplingProvider, bool)

	// List all providers
	ListProviders() map[string]interface{}
}

// EventBus provides event publishing and subscription
type EventBus interface {
	// Subscribe to event
	Subscribe(eventType string, handler func(event types.Event)) error
	// Unsubscribe from event
	Unsubscribe(eventType string, handler func(event types.Event)) error
	// Publish event
	Publish(event types.Event) error
	// Close event bus
	Close() error
}

// HealthChecker checks component health
type HealthChecker interface {
	// Check if component is healthy
	IsHealthy(ctx context.Context) bool
	// Get health status
	GetHealth(ctx context.Context) map[string]interface{}
	// Register health check
	RegisterHealthCheck(name string, check func(ctx context.Context) bool) error
}

// MetricsCollector collects metrics
type MetricsCollector interface {
	// Increment counter
	IncrementCounter(name string, tags map[string]string) error
	// Set gauge value
	SetGauge(name string, value float64, tags map[string]string) error
	// Record histogram value
	RecordHistogram(name string, value float64, tags map[string]string) error
	// Get metrics
	GetMetrics() map[string]interface{}
}

// Closer provides cleanup functionality
type Closer interface {
	io.Closer
}
