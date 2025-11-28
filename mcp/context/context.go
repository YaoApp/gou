package context

import (
	"context"
)

// Context wraps context.Context and provides MCP-specific context information
type Context struct {
	context.Context
	TraceID string // Trace ID for request tracking
}

// New creates a new MCP Context from a standard context.Context
func New(ctx context.Context) *Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Context{
		Context: ctx,
	}
}

// WithTraceID creates a new MCP Context with a trace ID
func WithTraceID(ctx context.Context, traceID string) *Context {
	return &Context{
		Context: ctx,
		TraceID: traceID,
	}
}
