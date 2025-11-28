package jsapi

import (
	"context"

	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// MCP represents a JavaScript MCP client wrapper
type MCP struct {
	ClientID string
	Client   mcp.Client
	Context  context.Context
}

// NewMCP creates a new MCP JavaScript object
// Usage in JS: const client = new MCP("client_id")
func NewMCP(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		v8ctx := info.Context()

		// Require client ID as first argument
		if len(args) < 1 || !args[0].IsString() {
			return bridge.JsException(v8ctx, "MCP constructor requires client ID as string")
		}

		clientID := args[0].String()

		// Get MCP client
		client, err := mcp.Select(clientID)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to get MCP client: "+err.Error())
		}

		// Create MCP wrapper
		mcpWrapper := &MCP{
			ClientID: clientID,
			Client:   client,
			Context:  context.Background(),
		}

		// Create JavaScript object
		jsObject, err := mcpWrapper.NewObject(v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to create MCP object: "+err.Error())
		}

		return jsObject
	})
}

// NewObject creates a JavaScript object for the MCP client
func (m *MCP) NewObject(v8ctx *v8go.Context) (*v8go.Value, error) {
	jsObject := v8go.NewObjectTemplate(v8ctx.Isolate())

	// Set internal field count to 1 to store the goValueID
	jsObject.SetInternalFieldCount(1)

	// Register MCP wrapper in global bridge registry
	goValueID := bridge.RegisterGoObject(m)

	// Set primitive fields
	jsObject.Set("id", m.ClientID)

	// Set release methods
	// Release() - Public method for manual resource cleanup (recommended)
	// __release() - Automatic cleanup when GC collects the object (fallback)
	releaseFunc := m.releaseMethod(v8ctx.Isolate(), goValueID)
	jsObject.Set("Release", releaseFunc)
	jsObject.Set("__release", releaseFunc)

	// Set methods - Tool operations
	jsObject.Set("ListTools", m.listToolsMethod(v8ctx.Isolate()))
	jsObject.Set("CallTool", m.callToolMethod(v8ctx.Isolate()))
	jsObject.Set("CallTools", m.callToolsMethod(v8ctx.Isolate()))
	jsObject.Set("CallToolsParallel", m.callToolsParallelMethod(v8ctx.Isolate()))

	// Set methods - Resource operations
	jsObject.Set("ListResources", m.listResourcesMethod(v8ctx.Isolate()))
	jsObject.Set("ReadResource", m.readResourceMethod(v8ctx.Isolate()))

	// Set methods - Prompt operations
	jsObject.Set("ListPrompts", m.listPromptsMethod(v8ctx.Isolate()))
	jsObject.Set("GetPrompt", m.getPromptMethod(v8ctx.Isolate()))

	// Set methods - Sample operations
	jsObject.Set("ListSamples", m.listSamplesMethod(v8ctx.Isolate()))
	jsObject.Set("GetSample", m.getSampleMethod(v8ctx.Isolate()))

	// Create instance
	instance, err := jsObject.NewInstance(v8ctx)
	if err != nil {
		bridge.ReleaseGoObject(goValueID)
		return nil, err
	}

	// Store the goValueID in internal field (index 0)
	obj, err := instance.Value.AsObject()
	if err != nil {
		bridge.ReleaseGoObject(goValueID)
		return nil, err
	}

	err = obj.SetInternalField(0, goValueID)
	if err != nil {
		bridge.ReleaseGoObject(goValueID)
		return nil, err
	}

	return instance.Value, nil
}

// releaseMethod releases the Go object from the global bridge registry
// Can be called manually via Release() or automatically via __release during GC
func (m *MCP) releaseMethod(iso *v8go.Isolate, goValueID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		thisObj, err := info.This().AsObject()
		if err == nil && thisObj.InternalFieldCount() > 0 {
			goValueID := thisObj.GetInternalField(0)
			if goValueID != nil && goValueID.IsString() {
				bridge.ReleaseGoObject(goValueID.String())
			}
		}
		return v8go.Undefined(info.Context().Isolate())
	})
}

// listToolsMethod implements client.ListTools(cursor)
// Usage: const tools = await client.ListTools()
func (m *MCP) listToolsMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		cursor := ""
		if len(args) > 0 && args[0].IsString() {
			cursor = args[0].String()
		}

		resp, err := m.Client.ListTools(m.Context, cursor)
		if err != nil {
			return bridge.JsException(v8ctx, "ListTools failed: "+err.Error())
		}

		jsResp, err := bridge.JsValue(v8ctx, resp)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert response: "+err.Error())
		}

		return jsResp
	})
}

// callToolMethod implements client.CallTool(name, arguments)
// Usage: const result = await client.CallTool("tool_name", { arg1: "value1" })
func (m *MCP) callToolMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 || !args[0].IsString() {
			return bridge.JsException(v8ctx, "CallTool requires tool name as first argument")
		}

		toolName := args[0].String()

		// Parse arguments (optional)
		var arguments interface{}
		if len(args) > 1 {
			goArgs, err := bridge.GoValue(args[1], v8ctx)
			if err != nil {
				return bridge.JsException(v8ctx, "failed to parse arguments: "+err.Error())
			}
			arguments = goArgs
		}

		resp, err := m.Client.CallTool(m.Context, toolName, arguments)
		if err != nil {
			return bridge.JsException(v8ctx, "CallTool failed: "+err.Error())
		}

		jsResp, err := bridge.JsValue(v8ctx, resp)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert response: "+err.Error())
		}

		return jsResp
	})
}

// callToolsMethod implements client.CallTools(toolCalls)
// Usage: const results = await client.CallTools([{name: "tool1", arguments: {...}}, ...])
func (m *MCP) callToolsMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "CallTools requires tool calls array")
		}

		// Parse tool calls array
		goValue, err := bridge.GoValue(args[0], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to parse tool calls: "+err.Error())
		}

		toolCallsArray, ok := goValue.([]interface{})
		if !ok {
			return bridge.JsException(v8ctx, "tool calls must be an array")
		}

		// Convert to []types.ToolCall
		toolCalls := make([]types.ToolCall, 0, len(toolCallsArray))
		for i, tc := range toolCallsArray {
			tcMap, ok := tc.(map[string]interface{})
			if !ok {
				return bridge.JsException(v8ctx, "tool call must be an object")
			}

			toolCall := types.ToolCall{}
			if name, ok := tcMap["name"].(string); ok {
				toolCall.Name = name
			} else {
				return bridge.JsException(v8ctx, "tool call["+string(rune(i))+"].name is required")
			}

			if args, ok := tcMap["arguments"]; ok {
				toolCall.Arguments = args
			}

			toolCalls = append(toolCalls, toolCall)
		}

		resp, err := m.Client.CallTools(m.Context, toolCalls)
		if err != nil {
			return bridge.JsException(v8ctx, "CallTools failed: "+err.Error())
		}

		jsResp, err := bridge.JsValue(v8ctx, resp)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert response: "+err.Error())
		}

		return jsResp
	})
}

// callToolsParallelMethod implements client.CallToolsParallel(toolCalls)
// Usage: const results = await client.CallToolsParallel([{name: "tool1", arguments: {...}}, ...])
func (m *MCP) callToolsParallelMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "CallToolsParallel requires tool calls array")
		}

		// Parse tool calls array (same as CallTools)
		goValue, err := bridge.GoValue(args[0], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to parse tool calls: "+err.Error())
		}

		toolCallsArray, ok := goValue.([]interface{})
		if !ok {
			return bridge.JsException(v8ctx, "tool calls must be an array")
		}

		// Convert to []types.ToolCall
		toolCalls := make([]types.ToolCall, 0, len(toolCallsArray))
		for i, tc := range toolCallsArray {
			tcMap, ok := tc.(map[string]interface{})
			if !ok {
				return bridge.JsException(v8ctx, "tool call must be an object")
			}

			toolCall := types.ToolCall{}
			if name, ok := tcMap["name"].(string); ok {
				toolCall.Name = name
			} else {
				return bridge.JsException(v8ctx, "tool call["+string(rune(i))+"].name is required")
			}

			if args, ok := tcMap["arguments"]; ok {
				toolCall.Arguments = args
			}

			toolCalls = append(toolCalls, toolCall)
		}

		resp, err := m.Client.CallToolsParallel(m.Context, toolCalls)
		if err != nil {
			return bridge.JsException(v8ctx, "CallToolsParallel failed: "+err.Error())
		}

		jsResp, err := bridge.JsValue(v8ctx, resp)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert response: "+err.Error())
		}

		return jsResp
	})
}

// listResourcesMethod implements client.ListResources(cursor)
// Usage: const resources = await client.ListResources()
func (m *MCP) listResourcesMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		cursor := ""
		if len(args) > 0 && args[0].IsString() {
			cursor = args[0].String()
		}

		resp, err := m.Client.ListResources(m.Context, cursor)
		if err != nil {
			return bridge.JsException(v8ctx, "ListResources failed: "+err.Error())
		}

		jsResp, err := bridge.JsValue(v8ctx, resp)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert response: "+err.Error())
		}

		return jsResp
	})
}

// readResourceMethod implements client.ReadResource(uri)
// Usage: const content = await client.ReadResource("resource://id")
func (m *MCP) readResourceMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 || !args[0].IsString() {
			return bridge.JsException(v8ctx, "ReadResource requires URI as first argument")
		}

		uri := args[0].String()

		resp, err := m.Client.ReadResource(m.Context, uri)
		if err != nil {
			return bridge.JsException(v8ctx, "ReadResource failed: "+err.Error())
		}

		jsResp, err := bridge.JsValue(v8ctx, resp)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert response: "+err.Error())
		}

		return jsResp
	})
}

// listPromptsMethod implements client.ListPrompts(cursor)
// Usage: const prompts = await client.ListPrompts()
func (m *MCP) listPromptsMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		cursor := ""
		if len(args) > 0 && args[0].IsString() {
			cursor = args[0].String()
		}

		resp, err := m.Client.ListPrompts(m.Context, cursor)
		if err != nil {
			return bridge.JsException(v8ctx, "ListPrompts failed: "+err.Error())
		}

		jsResp, err := bridge.JsValue(v8ctx, resp)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert response: "+err.Error())
		}

		return jsResp
	})
}

// getPromptMethod implements client.GetPrompt(name, arguments)
// Usage: const prompt = await client.GetPrompt("prompt_name", { arg1: "value1" })
func (m *MCP) getPromptMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 || !args[0].IsString() {
			return bridge.JsException(v8ctx, "GetPrompt requires prompt name as first argument")
		}

		promptName := args[0].String()

		// Parse arguments (optional)
		var arguments map[string]interface{}
		if len(args) > 1 {
			goArgs, err := bridge.GoValue(args[1], v8ctx)
			if err != nil {
				return bridge.JsException(v8ctx, "failed to parse arguments: "+err.Error())
			}
			if argsMap, ok := goArgs.(map[string]interface{}); ok {
				arguments = argsMap
			}
		}

		resp, err := m.Client.GetPrompt(m.Context, promptName, arguments)
		if err != nil {
			return bridge.JsException(v8ctx, "GetPrompt failed: "+err.Error())
		}

		jsResp, err := bridge.JsValue(v8ctx, resp)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert response: "+err.Error())
		}

		return jsResp
	})
}

// listSamplesMethod implements client.ListSamples(itemType, itemName)
// Usage: const samples = await client.ListSamples("tool", "tool_name")
func (m *MCP) listSamplesMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 2 || !args[0].IsString() || !args[1].IsString() {
			return bridge.JsException(v8ctx, "ListSamples requires itemType and itemName as arguments")
		}

		itemType := types.SampleItemType(args[0].String())
		itemName := args[1].String()

		resp, err := m.Client.ListSamples(m.Context, itemType, itemName)
		if err != nil {
			return bridge.JsException(v8ctx, "ListSamples failed: "+err.Error())
		}

		jsResp, err := bridge.JsValue(v8ctx, resp)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert response: "+err.Error())
		}

		return jsResp
	})
}

// getSampleMethod implements client.GetSample(itemType, itemName, index)
// Usage: const sample = await client.GetSample("tool", "tool_name", 0)
func (m *MCP) getSampleMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 3 || !args[0].IsString() || !args[1].IsString() {
			return bridge.JsException(v8ctx, "GetSample requires itemType, itemName, and index as arguments")
		}

		itemType := types.SampleItemType(args[0].String())
		itemName := args[1].String()

		// Parse index
		indexVal, err := bridge.GoValue(args[2], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to parse index: "+err.Error())
		}

		index := 0
		switch v := indexVal.(type) {
		case float64:
			index = int(v)
		case int:
			index = v
		case int64:
			index = int(v)
		default:
			return bridge.JsException(v8ctx, "index must be a number")
		}

		resp, err := m.Client.GetSample(m.Context, itemType, itemName, index)
		if err != nil {
			return bridge.JsException(v8ctx, "GetSample failed: "+err.Error())
		}

		jsResp, err := bridge.JsValue(v8ctx, resp)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert response: "+err.Error())
		}

		return jsResp
	})
}
