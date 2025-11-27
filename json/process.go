package json

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("json", map[string]process.Handler{
		// Standard encoding/decoding (fast)
		"encode": ProcessEncode,
		"decode": ProcessDecode,

		// Universal parsing (supports all formats + auto-repair)
		"parse": ProcessParse,

		// Repair broken JSON
		"repair": ProcessRepair,

		// JSON Schema validation
		"validate":       ProcessValidate,
		"validateschema": ProcessValidateSchema,
	})
}

// ProcessEncode json.Encode
// Standard JSON encoding
// Args: value interface{} - The value to encode
// Returns: JSON string
func ProcessEncode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	res, err := Encode(process.Args[0])
	if err != nil {
		exception.New("json.encode error: %s", 500, err).Throw()
	}
	return res
}

// ProcessDecode json.Decode
// Standard JSON decoding
// Args: data string - JSON string to decode
// Returns: decoded value
func ProcessDecode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	data := process.ArgsString(0)
	res, err := Decode(data)
	if err != nil {
		exception.New("json.decode error: %s", 500, err).Throw()
	}
	return res
}

// ProcessParse json.Parse
// Universal parser supporting JSON, JSONC, Yao, YAML with auto-repair
// Args:
//   - data string - The data string to parse
//   - hint string (optional) - Format hint (e.g., "file.yaml", ".json", "yaml")
//
// Returns: parsed value
//
// Usage:
//
//	// Parse JSON (with auto-repair if broken)
//	var data = Process("json.Parse", jsonString)
//
//	// Parse YAML
//	var data = Process("json.Parse", yamlString, "file.yaml")
//
//	// Parse Yao/JSONC (with comments)
//	var data = Process("json.Parse", yaoString, ".yao")
func ProcessParse(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	data := process.ArgsString(0)

	var hint []string
	if len(process.Args) > 1 {
		hint = []string{process.ArgsString(1)}
	}

	res, err := Parse(data, hint...)
	if err != nil {
		exception.New("json.parse error: %s", 500, err).Throw()
	}
	return res
}

// ProcessRepair json.Repair
// Repairs broken JSON
// Args: data string - Broken JSON string
// Returns: repaired JSON string
func ProcessRepair(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	data := process.ArgsString(0)
	repaired, err := Repair(data)
	if err != nil {
		exception.New("json.repair error: %s", 500, err).Throw()
	}
	return repaired
}

// ProcessValidate json.Validate
// Validates data against a JSON Schema
// Args:
//   - data interface{} - The data to validate
//   - schema interface{} - The JSON Schema (map, string, or []byte)
//
// Returns: nil if valid, error message string if invalid
//
// Usage:
//
//	// Validate with schema map
//	var result = Process("json.Validate", data, schema)
//	if (result != null) {
//	    log.Error("Validation failed: " + result)
//	}
//
//	// Validate with JSON string
//	var schemaStr = '{"type": "object", "properties": {"name": {"type": "string"}}}'
//	var result = Process("json.Validate", {"name": "John"}, schemaStr)
func ProcessValidate(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	data := process.Args[0]
	schema := process.Args[1]

	err := Validate(data, schema)
	if err != nil {
		return err.Error()
	}
	return nil
}

// ProcessValidateSchema json.ValidateSchema
// Validates a JSON Schema structure
// Args: schema interface{} - The JSON Schema to validate
// Returns: nil if valid, error message string if invalid
func ProcessValidateSchema(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	schema := process.Args[0]

	err := ValidateSchema(schema)
	if err != nil {
		return err.Error()
	}
	return nil
}
