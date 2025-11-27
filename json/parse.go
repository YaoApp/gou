package json

import (
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/kaptinlin/jsonrepair"
	"gopkg.in/yaml.v3"
)

// DetectFormat attempts to detect the format of the data
// Returns: "yaml", "json", "jsonc", or "" if uncertain
func DetectFormat(data string) string {
	trimmed := strings.TrimSpace(data)
	if len(trimmed) == 0 {
		return ""
	}

	// Check for YAML indicators
	// YAML typically has key: value format without braces at start
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		lines := strings.Split(trimmed, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) == 0 || strings.HasPrefix(line, "#") {
				continue
			}
			// YAML-style key: value (not inside quotes)
			if strings.Contains(line, ":") && !strings.Contains(line, "\":") && !strings.Contains(line, ":{") {
				return "yaml"
			}
			break
		}
	}

	// Check for JSONC (has comments)
	if strings.Contains(trimmed, "//") || strings.Contains(trimmed, "/*") {
		return "jsonc"
	}

	// Default to JSON
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "json"
	}

	return ""
}

// Parse parses data in multiple formats (JSON, JSONC, Yao, YAML) with auto-repair
// Parameters:
//   - data: the data string to parse
//   - hint: optional format hint (e.g., "file.yaml", ".json", "yaml")
//     If not provided, will attempt to auto-detect the format
func Parse(data string, hint ...string) (interface{}, error) {
	format := ""
	if len(hint) > 0 {
		format = strings.ToLower(hint[0])
	} else {
		// Auto-detect format if no hint provided
		detected := DetectFormat(data)
		if detected != "" {
			format = detected
		}
	}

	var res interface{}
	var err error

	// Determine format from hint or auto-detection
	switch {
	case strings.Contains(format, "yaml") || strings.Contains(format, "yml") || strings.HasSuffix(format, ".yaml") || strings.HasSuffix(format, ".yml"):
		// Parse as YAML
		err = yaml.Unmarshal([]byte(data), &res)
		if err != nil {
			return nil, err
		}
		return res, nil

	case strings.Contains(format, "yao") || strings.Contains(format, "jsonc") || strings.HasSuffix(format, ".yao") || strings.HasSuffix(format, ".jsonc"):
		// Parse as JSONC/Yao (remove comments first)
		cleaned := TrimComments([]byte(data))
		err = jsoniter.Unmarshal(cleaned, &res)
		if err != nil {
			return nil, err
		}
		return res, nil

	default:
		// Parse as JSON with progressive fallback
		// 1. Try standard JSON
		err = jsoniter.UnmarshalFromString(data, &res)
		if err == nil {
			return res, nil
		}

		// 2. Try removing comments (might be JSONC)
		cleaned := TrimComments([]byte(data))
		err = jsoniter.Unmarshal(cleaned, &res)
		if err == nil {
			return res, nil
		}

		// 3. Try auto-repair (for LLM-generated broken JSON)
		repaired, errRepair := jsonrepair.JSONRepair(data)
		if errRepair != nil {
			return nil, err // Return original error
		}

		err = jsoniter.UnmarshalFromString(repaired, &res)
		if err != nil {
			return nil, err
		}

		return res, nil
	}
}

// ParseTyped parses data into a typed pointer with format detection
func ParseTyped(data string, v interface{}, hint ...string) error {
	format := ""
	if len(hint) > 0 {
		format = strings.ToLower(hint[0])
	} else {
		// Auto-detect format if no hint provided
		detected := DetectFormat(data)
		if detected != "" {
			format = detected
		}
	}

	var err error

	switch {
	case strings.Contains(format, "yaml") || strings.Contains(format, "yml") || strings.HasSuffix(format, ".yaml") || strings.HasSuffix(format, ".yml"):
		return yaml.Unmarshal([]byte(data), v)

	case strings.Contains(format, "yao") || strings.Contains(format, "jsonc") || strings.HasSuffix(format, ".yao") || strings.HasSuffix(format, ".jsonc"):
		cleaned := TrimComments([]byte(data))
		return jsoniter.Unmarshal(cleaned, v)

	default:
		// Try standard JSON
		err = jsoniter.UnmarshalFromString(data, v)
		if err == nil {
			return nil
		}

		// Try removing comments
		cleaned := TrimComments([]byte(data))
		err = jsoniter.Unmarshal(cleaned, v)
		if err == nil {
			return nil
		}

		// Try auto-repair
		repaired, errRepair := jsonrepair.JSONRepair(data)
		if errRepair != nil {
			return err
		}

		return jsoniter.UnmarshalFromString(repaired, v)
	}
}

// ParseFile parses data based on file extension
// Supports: .json, .jsonc, .yao, .yaml, .yml
func ParseFile(filename string, data []byte, v interface{}) error {
	ext := filepath.Ext(filename)
	hint := ext
	if ext == "" {
		hint = filename
	}
	return ParseTyped(string(data), v, hint)
}

// Repair attempts to repair broken JSON
func Repair(data string) (string, error) {
	return jsonrepair.JSONRepair(data)
}
