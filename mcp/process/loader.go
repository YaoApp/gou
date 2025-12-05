package process

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/application"
	gouJSON "github.com/yaoapp/gou/json"
	"github.com/yaoapp/gou/mcp/types"
)

// LoadMapping loads mapping data (tools, resources, prompts) for a process-based MCP client
func LoadMapping(clientID string, dsl *types.ClientDSL, mappingBasePath string) (*types.MappingData, error) {
	mapping := &types.MappingData{
		Tools:     make(map[string]*types.ToolSchema),
		Resources: make(map[string]*types.ResourceSchema),
		Prompts:   make(map[string]*types.PromptSchema),
	}

	// Load tools
	if dsl.Tools != nil {
		for toolName, processName := range dsl.Tools {
			tool, err := loadToolSchema(clientID, toolName, processName, mappingBasePath)
			if err != nil {
				return nil, fmt.Errorf("failed to load tool %s: %w", toolName, err)
			}
			mapping.Tools[toolName] = tool
		}
	}

	// Load resources
	if dsl.Resources != nil {
		for resourceName, processName := range dsl.Resources {
			resource, err := loadResourceSchema(clientID, resourceName, processName, mappingBasePath)
			if err != nil {
				return nil, fmt.Errorf("failed to load resource %s: %w", resourceName, err)
			}
			mapping.Resources[resourceName] = resource
		}
	}

	// Load prompts
	if dsl.Prompts != nil {
		for promptName, processName := range dsl.Prompts {
			prompt, err := loadPromptSchema(clientID, promptName, processName, mappingBasePath)
			if err != nil {
				return nil, fmt.Errorf("failed to load prompt %s: %w", promptName, err)
			}
			mapping.Prompts[promptName] = prompt
		}
	}

	return mapping, nil
}

// loadToolSchema loads a tool schema from mapping directory
func loadToolSchema(clientID, toolName, processName, basePath string) (*types.ToolSchema, error) {
	tool := &types.ToolSchema{
		Name:    toolName,
		Process: processName,
	}

	// Convert clientID dots to slashes (e.g., "foo.bar" → "foo/bar")
	clientPath := strings.ReplaceAll(clientID, ".", "/")

	// Load input schema (required)
	inputPath := filepath.Join(basePath, clientPath, "schemes", toolName+".in.yao")
	inputData, err := application.App.Read(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read input schema %s: %w", inputPath, err)
	}

	// Parse input schema as JSON
	var inputSchema map[string]interface{}
	err = application.Parse(inputPath, inputData, &inputSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input schema %s: %w", inputPath, err)
	}

	// Validate input schema is a valid JSON Schema
	if err := validateJSONSchema(inputSchema, inputPath); err != nil {
		return nil, err
	}

	// Extract description if present
	if desc, ok := inputSchema["description"].(string); ok {
		tool.Description = desc
	}

	// Marshal back to RawMessage
	tool.InputSchema, err = json.Marshal(inputSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input schema: %w", err)
	}

	// Load output schema (optional)
	outputPath := filepath.Join(basePath, clientPath, "schemes", toolName+".out.yao")
	outputData, err := application.App.Read(outputPath)
	if err == nil {
		var outputSchema map[string]interface{}
		err = application.Parse(outputPath, outputData, &outputSchema)
		if err == nil {
			// Validate output schema is a valid JSON Schema
			if err := validateJSONSchema(outputSchema, outputPath); err != nil {
				return nil, err
			}
			tool.OutputSchema, _ = json.Marshal(outputSchema)
		}
	}

	return tool, nil
}

// validateJSONSchema validates that a schema is a valid JSON Schema
func validateJSONSchema(schema map[string]interface{}, path string) error {
	// Use gou/json package to validate the JSON Schema structure
	err := gouJSON.ValidateSchema(schema)
	if err != nil {
		return fmt.Errorf("invalid JSON Schema %s: %w", path, err)
	}
	return nil
}

// loadResourceSchema loads a resource schema from mapping directory
func loadResourceSchema(clientID, resourceName, processName, basePath string) (*types.ResourceSchema, error) {
	resource := &types.ResourceSchema{
		Name:    resourceName,
		Process: processName,
	}

	// Convert clientID dots to slashes (e.g., "foo.bar" → "foo/bar")
	clientPath := strings.ReplaceAll(clientID, ".", "/")

	// Load resource definition
	resourcePath := filepath.Join(basePath, clientPath, "resources", resourceName+".res.yao")
	resourceData, err := application.App.Read(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource schema %s: %w", resourcePath, err)
	}

	// Parse resource schema
	var resourceDef struct {
		Description string                    `json:"description"`
		URI         string                    `json:"uri"`
		MimeType    string                    `json:"mimeType"`
		Parameters  []types.ResourceParameter `json:"parameters"`
		Meta        map[string]interface{}    `json:"meta"`
	}
	err = application.Parse(resourcePath, resourceData, &resourceDef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource schema %s: %w", resourcePath, err)
	}

	resource.Description = resourceDef.Description
	resource.URI = resourceDef.URI
	resource.MimeType = resourceDef.MimeType
	resource.Parameters = resourceDef.Parameters
	resource.Meta = resourceDef.Meta

	return resource, nil
}

// loadPromptSchema loads a prompt schema from mapping directory
func loadPromptSchema(clientID, promptName, processName, basePath string) (*types.PromptSchema, error) {
	prompt := &types.PromptSchema{
		Name: promptName,
	}

	// Convert clientID dots to slashes (e.g., "foo.bar" → "foo/bar")
	clientPath := strings.ReplaceAll(clientID, ".", "/")

	// Load prompt template
	promptPath := filepath.Join(basePath, clientPath, "prompts", promptName+".pmt.yao")
	promptData, err := application.App.Read(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt template %s: %w", promptPath, err)
	}

	// Parse prompt schema
	var promptDef struct {
		Description string                 `json:"description"`
		Template    string                 `json:"template"`
		Arguments   []types.PromptArgument `json:"arguments"`
		Meta        map[string]interface{} `json:"meta"`
	}
	err = application.Parse(promptPath, promptData, &promptDef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt template %s: %w", promptPath, err)
	}

	prompt.Description = promptDef.Description
	prompt.Template = promptDef.Template
	prompt.Arguments = promptDef.Arguments
	prompt.Meta = promptDef.Meta

	return prompt, nil
}

// LoadMappingFromFile loads mapping data based on MCP client file path
// Examples:
//
//	mcps/dsl.mcp.yao -> clientID: "dsl", mapping: mcps/mapping/dsl/
//	mcps/foo/bar.mcp.yao -> clientID: "foo.bar", mapping: mcps/mapping/foo/bar/
//	assistants/expense/mcps/tools.mcp.yao -> clientID: "tools", mapping: assistants/expense/mcps/mapping/tools/
func LoadMappingFromFile(filePath string, clientID string, dsl *types.ClientDSL) (*types.MappingData, error) {
	// If no tools/resources/prompts defined, return empty mapping
	if dsl.Tools == nil && dsl.Resources == nil && dsl.Prompts == nil {
		return &types.MappingData{
			Tools:     make(map[string]*types.ToolSchema),
			Resources: make(map[string]*types.ResourceSchema),
			Prompts:   make(map[string]*types.PromptSchema),
		}, nil
	}

	// Determine mapping base path based on file location
	mappingBasePath := "mcps/mapping"

	// Normalize path separators
	normalizedPath := filepath.ToSlash(filePath)

	// Check if this is an assistant MCP (contains /mcps/ in path)
	// Pattern: <prefix>/mcps/<filename>.mcp.yao
	if strings.Contains(normalizedPath, "/mcps/") {
		// Extract the directory containing mcps/
		parts := strings.Split(normalizedPath, "/mcps/")
		if len(parts) == 2 {
			// Use the parent directory + mcps/mapping as base
			// e.g., "assistants/expense/mcps/tools.mcp.yao" -> "assistants/expense/mcps/mapping"
			mappingBasePath = parts[0] + "/mcps/mapping"
		}
	}

	// Extract mapping lookup ID from file path
	// For assistants, we want just the filename without path
	// For standard mcps, we want the full path under mcps/
	mappingLookupID := clientID
	if mappingLookupID == "" {
		// Parse file path to extract client ID
		// Examples:
		//   mcps/dsl.mcp.yao -> dsl
		//   mcps/foo/bar.mcp.yao -> foo.bar
		//   assistants/expense/mcps/tools.mcp.yao -> tools

		// Remove extensions
		pathWithoutExt := normalizedPath
		pathWithoutExt = strings.TrimSuffix(pathWithoutExt, ".mcp.yao")
		pathWithoutExt = strings.TrimSuffix(pathWithoutExt, ".mcp.json")
		pathWithoutExt = strings.TrimSuffix(pathWithoutExt, ".mcp.jsonc")

		// Extract just the filename part after last /mcps/
		if strings.Contains(pathWithoutExt, "/mcps/") {
			parts := strings.Split(pathWithoutExt, "/mcps/")
			if len(parts) == 2 {
				mappingLookupID = parts[1]
			}
		} else if strings.HasPrefix(pathWithoutExt, "mcps/") {
			// Remove "mcps/" prefix for standard mcps
			mappingLookupID = pathWithoutExt[5:]
		} else {
			mappingLookupID = filepath.Base(pathWithoutExt)
		}

		// Replace slashes with dots for nested paths
		mappingLookupID = strings.ReplaceAll(mappingLookupID, "/", ".")
	} else if strings.Contains(normalizedPath, "/mcps/") {
		// If clientID was provided but this is an assistant MCP,
		// extract the path after /mcps/ for mapping lookup
		// e.g., "assistants/tests/mcpload/mcps/nested/tool.mcp.yao" -> "nested.tool"
		parts := strings.Split(normalizedPath, "/mcps/")
		if len(parts) == 2 {
			filename := parts[1]
			filename = strings.TrimSuffix(filename, ".mcp.yao")
			filename = strings.TrimSuffix(filename, ".mcp.json")
			filename = strings.TrimSuffix(filename, ".mcp.jsonc")
			// Support nested paths: "nested/tool" -> "nested.tool"
			mappingLookupID = strings.ReplaceAll(filename, "/", ".")
		}
	}

	// Load mapping from the determined path
	return LoadMapping(mappingLookupID, dsl, mappingBasePath)
}

// LoadMappingFromSource loads mapping data from in-memory DSL and optional mapping data
func LoadMappingFromSource(clientID string, dsl *types.ClientDSL, mappingData *types.MappingData) (*types.MappingData, error) {
	// If mapping data is provided directly, use it
	if mappingData != nil {
		return mappingData, nil
	}

	// If no tools/resources/prompts defined, return empty mapping
	if dsl.Tools == nil && dsl.Resources == nil && dsl.Prompts == nil {
		return &types.MappingData{
			Tools:     make(map[string]*types.ToolSchema),
			Resources: make(map[string]*types.ResourceSchema),
			Prompts:   make(map[string]*types.PromptSchema),
		}, nil
	}

	// Load from filesystem
	mappingBasePath := "mcps/mapping"
	return LoadMapping(clientID, dsl, mappingBasePath)
}

// UpdateMapping updates specific tools/resources/prompts in the mapping registry
func UpdateMapping(clientID string, tools map[string]*types.ToolSchema, resources map[string]*types.ResourceSchema, prompts map[string]*types.PromptSchema) error {
	mappingRegistryLock.Lock()
	defer mappingRegistryLock.Unlock()

	mapping, exists := mappingRegistry[clientID]
	if !exists {
		mapping = &types.MappingData{
			Tools:     make(map[string]*types.ToolSchema),
			Resources: make(map[string]*types.ResourceSchema),
			Prompts:   make(map[string]*types.PromptSchema),
		}
		mappingRegistry[clientID] = mapping
	}

	// Update tools
	for name, tool := range tools {
		mapping.Tools[name] = tool
	}

	// Update resources
	for name, resource := range resources {
		mapping.Resources[name] = resource
	}

	// Update prompts
	for name, prompt := range prompts {
		mapping.Prompts[name] = prompt
	}

	return nil
}

// RemoveMappingItems removes specific tools/resources/prompts from the mapping registry
func RemoveMappingItems(clientID string, toolNames []string, resourceNames []string, promptNames []string) error {
	mappingRegistryLock.Lock()
	defer mappingRegistryLock.Unlock()

	mapping, exists := mappingRegistry[clientID]
	if !exists {
		return fmt.Errorf("mapping not found for client %s", clientID)
	}

	// Remove tools
	for _, name := range toolNames {
		delete(mapping.Tools, name)
	}

	// Remove resources
	for _, name := range resourceNames {
		delete(mapping.Resources, name)
	}

	// Remove prompts
	for _, name := range promptNames {
		delete(mapping.Prompts, name)
	}

	return nil
}
