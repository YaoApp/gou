package mcp

import (
	"fmt"
	"sync"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/mcp/client"
	"github.com/yaoapp/gou/mcp/process"
	"github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/kun/exception"
)

// Clients the mcp clients
var clients = map[string]Client{}

// Servers the mcp services
var servers = map[string]Server{}

// clientsLock protects the clients map from concurrent access
var clientsLock sync.RWMutex

// LoadServer load the mcp server
func LoadServer(path, id string) (Server, error) {
	return nil, nil
}

// LoadServerSource load the mcp server source
func LoadServerSource(server, id string) (Server, error) {
	return nil, nil
}

// LoadClientSource load the mcp client source with optional mapping data
func LoadClientSource(dsl, id string, mappingData ...*types.MappingData) (Client, error) {
	return LoadClientSourceWithType(dsl, id, "", mappingData...)
}

// LoadClientSourceWithType load the mcp client source with type and optional mapping data
func LoadClientSourceWithType(dsl, id, clientType string, mappingData ...*types.MappingData) (Client, error) {
	if id == "" {
		return nil, fmt.Errorf("client id is required")
	}

	// Parse DSL
	var clientDSL types.ClientDSL
	err := application.Parse(fmt.Sprintf("mcp.%s.yao", id), []byte(dsl), &clientDSL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MCP client DSL: %w", err)
	}

	// Set ID if not provided in DSL
	if clientDSL.ID == "" {
		clientDSL.ID = id
	}

	// Set Type if provided and not already set
	if clientType != "" && clientDSL.Type == "" {
		clientDSL.Type = clientType
	}

	// Set Name if not provided in DSL
	if clientDSL.Name == "" {
		clientDSL.Name = id
	}

	// Process environment variables
	clientDSL.URL = helper.EnvString(clientDSL.URL)
	clientDSL.Endpoint = helper.EnvString(clientDSL.Endpoint)
	clientDSL.AuthorizationToken = helper.EnvString(clientDSL.AuthorizationToken)
	clientDSL.Command = helper.EnvString(clientDSL.Command)

	// Process environment variables in env map
	if clientDSL.Env != nil {
		for key, value := range clientDSL.Env {
			clientDSL.Env[key] = helper.EnvString(value)
		}
	}

	// Process environment variables in arguments
	if clientDSL.Arguments != nil {
		for i, arg := range clientDSL.Arguments {
			clientDSL.Arguments[i] = helper.EnvString(arg)
		}
	}

	// Yao Process
	var mcpClient Client
	switch clientDSL.Transport {

	// Yao Process based client
	case types.TransportProcess:
		// Load mapping data for process-based client
		var mapping *types.MappingData
		if len(mappingData) > 0 && mappingData[0] != nil {
			mapping = mappingData[0]
		} else {
			// Try to load from filesystem
			mapping, err = process.LoadMappingFromSource(id, &clientDSL, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to load mapping data: %w", err)
			}
		}

		// Store mapping in global registry
		process.SetMapping(id, mapping)

		// Create client
		mcpClient, err = process.New(&clientDSL)
		if err != nil {
			// Clean up mapping on error
			process.RemoveMapping(id)
			return nil, fmt.Errorf("failed to create MCP client: %w", err)
		}

	// HTTP, SSE, STDIO based client
	default:
		mcpClient, err = client.New(&clientDSL)
		if err != nil {
			return nil, fmt.Errorf("failed to create MCP client: %w", err)
		}
	}

	// Store client in the global map with write lock
	clientsLock.Lock()
	clients[id] = mcpClient
	clientsLock.Unlock()

	return mcpClient, nil
}

// LoadClient load the mcp client
func LoadClient(path, id string) (Client, error) {
	return LoadClientWithType(path, id, "")
}

// LoadClientWithType load the mcp client from file with a specific type
func LoadClientWithType(path, id, clientType string) (Client, error) {
	// Read file content
	data, err := application.App.Read(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP client file %s: %w", path, err)
	}

	// Parse DSL to check if it's a process-based client
	var clientDSL types.ClientDSL
	err = application.Parse(path, data, &clientDSL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MCP client DSL: %w", err)
	}

	// If it's a process-based client, try to load mapping from filesystem
	if clientDSL.Transport == types.TransportProcess {
		// Try to load mapping data based on file path
		mapping, err := process.LoadMappingFromFile(path, id, &clientDSL)
		if err != nil {
			// If mapping load fails but tools/resources/prompts are defined, return error
			if clientDSL.Tools != nil || clientDSL.Resources != nil || clientDSL.Prompts != nil {
				return nil, fmt.Errorf("failed to load mapping data: %w", err)
			}
		}

		// If id is not provided, extract from file path
		if id == "" {
			id = extractClientIDFromPath(path)
		}

		// Call LoadClientSourceWithType with mapping data
		return LoadClientSourceWithType(string(data), id, clientType, mapping)
	}

	// For non-process clients, if id is not provided, extract from file path
	if id == "" {
		id = extractClientIDFromPath(path)
	}

	// For non-process clients
	return LoadClientSourceWithType(string(data), id, clientType)
}

// extractClientIDFromPath extracts client ID from file path
// Examples:
//
//	mcps/dsl.mcp.yao -> dsl
//	mcps/foo/bar.mcp.yao -> foo.bar
func extractClientIDFromPath(filePath string) string {
	// Remove .mcp.yao extension
	if len(filePath) < 8 || filePath[len(filePath)-8:] != ".mcp.yao" {
		return ""
	}

	pathWithoutExt := filePath[:len(filePath)-8]

	// Remove "mcps/" prefix if present
	if len(pathWithoutExt) > 5 && pathWithoutExt[:5] == "mcps/" {
		pathWithoutExt = pathWithoutExt[5:]
	}

	// Normalize path separators and replace with dots
	// Example: "foo/bar" -> "foo.bar"
	pathWithoutExt = fmt.Sprintf("%s", pathWithoutExt)
	clientID := ""
	for i := 0; i < len(pathWithoutExt); i++ {
		if pathWithoutExt[i] == '/' || pathWithoutExt[i] == '\\' {
			clientID += "."
		} else {
			clientID += string(pathWithoutExt[i])
		}
	}
	return clientID
}

// Select select the mcp client or server
func Select(id string) (Client, error) {
	clientsLock.RLock()
	defer clientsLock.RUnlock()

	client, exists := clients[id]
	if !exists {
		return nil, fmt.Errorf("MCP client %s not found", id)
	}
	return client, nil
}

// Exists Check if the client is loaded
func Exists(id string) bool {
	clientsLock.RLock()
	defer clientsLock.RUnlock()

	_, exists := clients[id]
	return exists
}

// GetClient get the client by id (similar to model.Select but without throwing exception)
func GetClient(id string) Client {
	clientsLock.RLock()
	defer clientsLock.RUnlock()

	client, exists := clients[id]
	if !exists {
		exception.New(
			fmt.Sprintf("MCP Client:%s; not found", id),
			400,
		).Throw()
	}
	return client
}

// UnloadClient unload the client by id
func UnloadClient(id string) {
	clientsLock.Lock()
	defer clientsLock.Unlock()

	delete(clients, id)

	// Also remove mapping if it's a process-based client
	process.RemoveMapping(id)
}

// ListClients list all loaded clients
func ListClients() []string {
	clientsLock.RLock()
	defer clientsLock.RUnlock()

	keys := make([]string, 0, len(clients))
	for id := range clients {
		keys = append(keys, id)
	}
	return keys
}

// UpdateClientMapping updates the mapping data for a process-based MCP client
func UpdateClientMapping(id string, tools map[string]*types.ToolSchema, resources map[string]*types.ResourceSchema, prompts map[string]*types.PromptSchema) error {
	clientsLock.RLock()
	_, exists := clients[id]
	clientsLock.RUnlock()

	if !exists {
		return fmt.Errorf("MCP client %s not found", id)
	}

	return process.UpdateMapping(id, tools, resources, prompts)
}

// RemoveClientMappingItems removes specific tools/resources/prompts from a process-based MCP client
func RemoveClientMappingItems(id string, toolNames []string, resourceNames []string, promptNames []string) error {
	clientsLock.RLock()
	_, exists := clients[id]
	clientsLock.RUnlock()

	if !exists {
		return fmt.Errorf("MCP client %s not found", id)
	}

	return process.RemoveMappingItems(id, toolNames, resourceNames, promptNames)
}

// GetClientMapping returns the mapping data for a process-based MCP client
func GetClientMapping(id string) (*types.MappingData, error) {
	clientsLock.RLock()
	_, exists := clients[id]
	clientsLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("MCP client %s not found", id)
	}

	mapping, exists := process.GetMapping(id)
	if !exists {
		return nil, fmt.Errorf("no mapping data found for client %s", id)
	}

	return mapping, nil
}
