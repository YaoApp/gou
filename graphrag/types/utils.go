package types

import "strings"

// SafeExtractInt safely converts interface{} to int, handling various numeric types
func SafeExtractInt(value interface{}, defaultValue int) int {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return defaultValue
	}
}

// SafeExtractFloat64 safely converts interface{} to float64, handling various numeric types
func SafeExtractFloat64(value interface{}, defaultValue float64) float64 {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	default:
		return defaultValue
	}
}

// SafeExtractBool safely extracts bool values from interface{}
func SafeExtractBool(value interface{}, defaultValue bool) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	return defaultValue
}

// SafeExtractString safely extracts string values from interface{}
func SafeExtractString(value interface{}, defaultValue string) string {
	if s, ok := value.(string); ok {
		return s
	}
	return defaultValue
}

// SafeExtractMap safely extracts map[string]interface{} from interface{}
func SafeExtractMap(value interface{}) (map[string]interface{}, bool) {
	if m, ok := value.(map[string]interface{}); ok {
		return m, true
	}
	return nil, false
}

// DeepMergeMetadata deeply merges metadata maps, with source values taking precedence
func DeepMergeMetadata(target, source map[string]interface{}) map[string]interface{} {
	if target == nil {
		target = make(map[string]interface{})
	}
	if source == nil {
		return target
	}

	result := make(map[string]interface{})

	// Copy all keys from target first
	for k, v := range target {
		result[k] = v
	}

	// Merge source into result
	for key, sourceValue := range source {
		if targetValue, exists := result[key]; exists {
			// If both are maps, merge recursively
			if targetMap, targetIsMap := targetValue.(map[string]interface{}); targetIsMap {
				if sourceMap, sourceIsMap := sourceValue.(map[string]interface{}); sourceIsMap {
					result[key] = DeepMergeMetadata(targetMap, sourceMap)
					continue
				}
			}
		}
		// For non-map values or when target doesn't exist, source takes precedence
		result[key] = sourceValue
	}

	return result
}

// ExtractNestedValue safely extracts nested values from metadata using dot notation
// Example: ExtractNestedValue(metadata, "chunk_details.depth", 0)
func ExtractNestedValue(metadata map[string]interface{}, path string, defaultValue interface{}) interface{} {
	if metadata == nil {
		return defaultValue
	}

	keys := strings.Split(path, ".")
	current := metadata

	for i, key := range keys {
		if i == len(keys)-1 {
			// Last key, return the value
			if value, exists := current[key]; exists {
				return value
			}
			return defaultValue
		}

		// Intermediate key, must be a map
		if next, exists := current[key]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return defaultValue
			}
		} else {
			return defaultValue
		}
	}

	return defaultValue
}
