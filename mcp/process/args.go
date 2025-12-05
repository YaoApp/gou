package process

import (
	"fmt"
	"strings"
)

// extractProcessArgs extracts process arguments based on x-process-args mapping
// If processArgs is nil/empty, returns the entire arguments object as a single parameter
// Otherwise, extracts values according to the mapping rules:
//   - "$args.field" or "$field" - extract field from arguments
//   - "$args.nested.field" - extract nested field using dot notation
//   - ":arguments" - pass entire arguments object
//   - other strings - pass as constant values
//
// extraArgs will be appended to the result after the mapped arguments
func extractProcessArgs(processArgs []string, arguments interface{}, extraArgs ...interface{}) ([]interface{}, error) {
	// Default behavior: no mapping, pass entire arguments object
	if len(processArgs) == 0 {
		result := []interface{}{arguments}
		result = append(result, extraArgs...)
		return result, nil
	}

	result := make([]interface{}, 0, len(processArgs)+len(extraArgs))

	for _, argSpec := range processArgs {
		value, err := extractArgValue(argSpec, arguments)
		if err != nil {
			return nil, fmt.Errorf("failed to extract argument '%s': %w", argSpec, err)
		}
		result = append(result, value)
	}

	// Append extra arguments
	result = append(result, extraArgs...)

	return result, nil
}

// extractArgValue extracts a single argument value based on the spec
func extractArgValue(argSpec string, arguments interface{}) (interface{}, error) {
	// Special case: pass entire arguments object
	if argSpec == ":arguments" {
		return arguments, nil
	}

	// Extract from arguments object
	if strings.HasPrefix(argSpec, "$") {
		path := strings.TrimPrefix(argSpec, "$")

		// Remove "args." prefix if present (allow both $args.field and $field)
		path = strings.TrimPrefix(path, "args.")

		// Extract value from arguments map
		argsMap, ok := arguments.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("arguments must be an object for field extraction, got %T", arguments)
		}

		value, err := extractNestedValue(argsMap, path)
		if err != nil {
			return nil, err
		}

		return value, nil
	}

	// Constant value
	return argSpec, nil
}

// extractNestedValue extracts a value from a nested map using dot notation
// Examples:
//   - "name" -> map["name"]
//   - "contact.phone" -> map["contact"]["phone"]
//   - "address.city.name" -> map["address"]["city"]["name"]
func extractNestedValue(data map[string]interface{}, path string) (interface{}, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	parts := strings.Split(path, ".")
	var current interface{} = data

	for i, part := range parts {
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("path segment '%s' at position %d is not an object", strings.Join(parts[:i], "."), i)
		}

		value, exists := currentMap[part]
		if !exists {
			// Return nil for missing optional fields (not an error)
			return nil, nil
		}

		current = value
	}

	return current, nil
}

// extractURIParams extracts parameters from a URI using a template
// Example:
//   - template: "customers://{id}", uri: "customers://123" -> {"id": "123"}
//   - template: "file://{path}", uri: "file:///home/user/file.txt" -> {"path": "/home/user/file.txt"}
func extractURIParams(template, uri string) (map[string]string, error) {
	// Simple implementation for URI template parameters
	// This handles basic {param} patterns in the URI template

	params := make(map[string]string)

	// Find all {param} patterns in template
	templateParts := strings.Split(template, "{")
	if len(templateParts) == 1 {
		// No parameters in template
		return params, nil
	}

	// Build a pattern to match
	currentURI := uri
	currentTemplate := template

	for strings.Contains(currentTemplate, "{") {
		startIdx := strings.Index(currentTemplate, "{")
		endIdx := strings.Index(currentTemplate, "}")
		if endIdx == -1 {
			return nil, fmt.Errorf("malformed URI template: unclosed '{' in %s", template)
		}

		paramName := currentTemplate[startIdx+1 : endIdx]
		prefix := currentTemplate[:startIdx]
		suffix := ""
		if endIdx+1 < len(currentTemplate) {
			suffix = currentTemplate[endIdx+1:]
		}

		// Remove prefix from URI
		if !strings.HasPrefix(currentURI, prefix) {
			return nil, fmt.Errorf("URI %s does not match template %s", uri, template)
		}
		currentURI = currentURI[len(prefix):]

		// Extract parameter value (everything up to the next template part)
		var paramValue string
		if suffix == "" {
			// Last parameter, take the rest
			paramValue = currentURI
			currentURI = ""
		} else {
			// Find the suffix in URI
			suffixIdx := strings.Index(currentURI, suffix)
			if suffixIdx == -1 {
				return nil, fmt.Errorf("URI %s does not match template %s", uri, template)
			}
			paramValue = currentURI[:suffixIdx]
			currentURI = currentURI[suffixIdx:]
		}

		params[paramName] = paramValue

		// Move to next template segment
		currentTemplate = suffix
	}

	return params, nil
}

// extractResourceArgs extracts process arguments for resource reading
// Similar to extractProcessArgs but also handles URI template parameters
// Supports:
//   - "$args.field" - extract from query/parameters
//   - "$uri.field" - extract from URI template parameters
//   - ":arguments" - pass entire arguments object
//   - constant values
func extractResourceArgs(processArgs []string, uri string, uriTemplate string, parameters map[string]interface{}) ([]interface{}, error) {
	// Default behavior: pass URI as the only argument
	if len(processArgs) == 0 {
		return []interface{}{uri}, nil
	}

	// Extract URI parameters if template is provided
	uriParams := make(map[string]string)
	if uriTemplate != "" {
		var err error
		uriParams, err = extractURIParams(uriTemplate, uri)
		if err != nil {
			return nil, fmt.Errorf("failed to extract URI parameters: %w", err)
		}
	}

	result := make([]interface{}, 0, len(processArgs))

	for _, argSpec := range processArgs {
		value, err := extractResourceArgValue(argSpec, uri, uriParams, parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to extract argument '%s': %w", argSpec, err)
		}
		result = append(result, value)
	}

	return result, nil
}

// extractResourceArgValue extracts a single resource argument value
func extractResourceArgValue(argSpec string, uri string, uriParams map[string]string, parameters map[string]interface{}) (interface{}, error) {
	// Special cases
	if argSpec == ":uri" {
		return uri, nil
	}
	if argSpec == ":arguments" || argSpec == ":parameters" {
		return parameters, nil
	}

	// Extract from URI parameters
	if strings.HasPrefix(argSpec, "$uri.") {
		paramName := strings.TrimPrefix(argSpec, "$uri.")
		value, exists := uriParams[paramName]
		if !exists {
			return nil, nil // Missing URI parameter
		}
		return value, nil
	}

	// Extract from query/parameters
	if strings.HasPrefix(argSpec, "$") {
		path := strings.TrimPrefix(argSpec, "$")

		// Remove "args." or "params." prefix if present
		if strings.HasPrefix(path, "args.") {
			path = strings.TrimPrefix(path, "args.")
		} else if strings.HasPrefix(path, "params.") {
			path = strings.TrimPrefix(path, "params.")
		}

		value, err := extractNestedValue(parameters, path)
		if err != nil {
			return nil, err
		}
		return value, nil
	}

	// Constant value
	return argSpec, nil
}
