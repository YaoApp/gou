package mcp

// Clients the mcp clients
var clients = map[string]Client{}

// Servers the mcp services
var servers = map[string]Server{}

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
	return nil, nil
}

// LoadClient load the mcp client
func LoadClient(file, id string) (Client, error) {
	return nil, nil
}

// Select select the mcp client or server
func Select(id string) (Client, error) {
	return nil, nil
}
