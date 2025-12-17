package text

import (
	"github.com/yaoapp/gou/process"
)

func init() {
	process.RegisterGroup("text", map[string]process.Handler{
		"extract":       ProcessExtract,
		"extractfirst":  ProcessExtractFirst,
		"extractjson":   ProcessExtractJSON,
		"extractbytype": ProcessExtractByType,
	})
}

// ProcessExtract text.Extract
// Extracts code blocks from text (typically LLM output)
// Handles markdown code blocks and direct JSON/HTML/Code detection
//
// Args:
//   - text string - The text to extract code blocks from
//
// Returns: []CodeBlock - Array of extracted code blocks
//
// Usage:
//
//	// Extract all code blocks from LLM response
//	var blocks = Process("text.Extract", llmResponse)
//	// Returns: [{"type": "json", "content": "{...}", "data": {...}}, ...]
//
//	// Handle markdown wrapped JSON
//	var blocks = Process("text.Extract", "```json\n{\"key\": \"value\"}\n```")
//	// Returns: [{"type": "json", "content": "{\"key\": \"value\"}", "data": {"key": "value"}}]
func ProcessExtract(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	text := process.ArgsString(0)
	return Extract(text)
}

// ProcessExtractFirst text.ExtractFirst
// Extracts and returns only the first code block
// Useful when you expect only one block (common in LLM responses)
//
// Args:
//   - text string - The text to extract from
//
// Returns: CodeBlock or null - The first extracted code block
//
// Usage:
//
//	var block = Process("text.ExtractFirst", llmResponse)
//	if (block != null && block.data != null) {
//	    // Use block.data directly
//	}
func ProcessExtractFirst(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	text := process.ArgsString(0)
	block := ExtractFirst(text)
	if block == nil {
		return nil
	}
	return block
}

// ProcessExtractJSON text.ExtractJSON
// Extracts the first JSON or YAML block and returns parsed data directly
// Most convenient for LLM responses that return structured data
// Supports both JSON and YAML formats
//
// Args:
//   - text string - The text containing JSON or YAML
//
// Returns: parsed data or null
//
// Usage:
//
//	// Direct JSON parsing from LLM response
//	var data = Process("text.ExtractJSON", "```json\n{\"keywords\": [\"a\", \"b\"]}\n```")
//	// Returns: {"keywords": ["a", "b"]}
//
//	// Also works without markdown wrapper
//	var data = Process("text.ExtractJSON", "{\"keywords\": [\"a\", \"b\"]}")
//	// Returns: {"keywords": ["a", "b"]}
//
//	// YAML is also supported
//	var data = Process("text.ExtractJSON", "```yaml\nkeywords:\n  - a\n  - b\n```")
//	// Returns: {"keywords": ["a", "b"]}
func ProcessExtractJSON(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	text := process.ArgsString(0)
	return ExtractJSON(text)
}

// ProcessExtractByType text.ExtractByType
// Extracts all blocks of a specific type
//
// Args:
//   - text string - The text to extract from
//   - blockType string - The type to filter by (e.g., "json", "html", "sql")
//
// Returns: []CodeBlock - Array of matching code blocks
//
// Usage:
//
//	// Extract only SQL blocks
//	var sqlBlocks = Process("text.ExtractByType", llmResponse, "sql")
func ProcessExtractByType(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	text := process.ArgsString(0)
	blockType := process.ArgsString(1)

	blocks := ExtractByType(text, blockType)
	if blocks == nil {
		return []CodeBlock{}
	}
	return blocks
}
