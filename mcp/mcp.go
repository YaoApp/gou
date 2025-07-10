package mcp

import (
	"fmt"
	"sync"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/mcp/client"
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

// LoadClientSource load the mcp client source
func LoadClientSource(dsl, id string) (Client, error) {
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

	// Set Name if not provided in DSL
	if clientDSL.Name == "" {
		clientDSL.Name = id
	}

	// Process environment variables
	clientDSL.URL = helper.EnvString(clientDSL.URL)
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

	// Create client instance
	mcpClient, err := client.New(&clientDSL)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Store client in the global map with write lock
	clientsLock.Lock()
	clients[id] = mcpClient
	clientsLock.Unlock()

	return mcpClient, nil
}

// LoadClient load the mcp client
func LoadClient(file, id string) (Client, error) {
	// Read file content
	data, err := application.App.Read(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP client file %s: %w", file, err)
	}

	// Call LoadClientSource
	return LoadClientSource(string(data), id)
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
