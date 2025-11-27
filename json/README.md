# JSON Package

A comprehensive JSON utility package for the Gou framework, designed for the AI era with support for JSON, JSONC (JSON with Comments), YAML, and automatic repair of malformed JSON.

## Features

- **Standard JSON Operations**: Fast encoding/decoding using `jsoniter`
- **Universal Parser**: Single `Parse()` function supporting JSON, JSONC, YAML with auto-detection
- **JSON Repair**: Automatically fix malformed JSON from LLM outputs
- **Schema Validation**: JSON Schema validation using draft 2020-12
- **Comment Support**: Handle JSONC/Yao format (JSON with comments)

## Go Usage

### Basic Encoding/Decoding

```go
import "github.com/yaoapp/gou/json"

// Encode to JSON string
data := map[string]interface{}{"name": "Alice", "age": 30}
jsonStr, err := json.Encode(data)

// Decode from JSON string
var result map[string]interface{}
err = json.Decode([]byte(jsonStr), &result)

// Decode with type inference
result, err := json.DecodeTyped([]byte(jsonStr))
```

### Universal Parsing

```go
// Auto-detect format (JSON, JSONC, YAML)
data, err := json.Parse(input)

// Parse with format hint
data, err := json.ParseWithHint(input, "yaml")

// Parse file
data, err := json.ParseFile("/path/to/file.yaml")

// Parse with type safety
var config Config
err := json.ParseTyped(input, &config)
```

### JSON Repair

```go
// Repair malformed JSON (useful for LLM outputs)
fixed, err := json.Repair(`{name: "Alice", age: 30, "incomplete": }`)
// Result: {"name": "Alice", "age": 30}
```

### Comment Handling

```go
// Remove comments from JSONC/Yao format
clean := json.TrimComments(`{
  // This is a comment
  "key": "value" /* block comment */
}`)
```

### Schema Validation

```go
// Create validator
validator, err := json.NewValidator(schemaJSON)

// Validate data
err = validator.Validate(data)

// One-shot validation
err = json.Validate(schemaJSON, data)

// Validate schema itself
err = json.ValidateSchema(schemaJSON)
```

## JavaScript Usage (Yao Process)

All functions are available as Yao processes under the `json.*` namespace.

### Basic Operations

```javascript
// Encode to JSON
const jsonStr = Process("json.Encode", { name: "Alice", age: 30 });

// Decode from JSON
const data = Process("json.Decode", jsonStr);
```

### Universal Parsing

```javascript
// Auto-detect format
const data = Process("json.Parse", jsonString);

// Parse with hint
const data = Process("json.Parse", yamlString, "yaml");

// Parse file
const data = Process("json.ParseFile", "/data/config.yaml");
```

### JSON Repair

```javascript
// Repair malformed JSON from LLM
const fixed = Process("json.Repair", '{name: "Alice", incomplete: }');
```

### Comment Handling

```javascript
// Remove comments
const clean = Process("json.TrimComments", `{
  // Comment
  "key": "value"
}`);
```

### Schema Validation

```javascript
// Validate data against schema
const schema = {
  type: "object",
  properties: {
    name: { type: "string" },
    age: { type: "integer", minimum: 0 }
  },
  required: ["name"]
};

Process("json.Validate", schema, { name: "Alice", age: 30 });

// Validate schema itself
Process("json.ValidateSchema", schema);
```

## Format Detection

The `Parse()` function automatically detects the input format:

- **JSON**: Standard JSON syntax
- **JSONC**: JSON with `//` or `/* */` comments
- **YAML**: YAML syntax (indentation-based)

You can also provide a hint (`"json"`, `"jsonc"`, `"yao"`, `"yaml"`) to skip auto-detection.

## Performance Notes

- `Encode/Decode`: Fast, optimized for performance (using `jsoniter`)
- `Parse`: Comprehensive, supports multiple formats (slightly slower)
- `Repair`: Designed for LLM outputs, may be slow on large inputs

## Use Cases

- **AI/LLM Integration**: Repair malformed JSON from language models
- **Configuration Files**: Parse JSONC/YAML configs with comments
- **Schema Validation**: Ensure data integrity for MCP tools/resources
- **Universal Parser**: One function for all JSON-like formats

## Backward Compatibility

For backward compatibility with Yao's `utils.jsonschema.*`, the following aliases are registered:

- `utils.jsonschema.Validate` → `json.Validate`
- `utils.jsonschema.ValidateSchema` → `json.ValidateSchema`

## Testing

Run tests with coverage:

```bash
cd gou/json
go test -v -cover
```

Current coverage: **92.4%**

