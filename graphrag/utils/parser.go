package utils

import (
	"fmt"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/kaptinlin/jsonrepair"
	"github.com/yaoapp/gou/graphrag/types"
)

// Parser handles streaming LLM responses and accumulates data
type Parser struct {
	Content   string
	Arguments string
	Toolcall  bool
	finished  bool
	mutex     sync.Mutex
}

// NewSemanticParser creates a new semantic parser
func NewSemanticParser(isToolcall bool) *Parser {
	return &Parser{Toolcall: isToolcall}
}

// ParseSemanticToolcall parses the final arguments of the toolcall
func (parser *Parser) ParseSemanticToolcall(finalArguments string) ([]types.Position, error) {
	parser.mutex.Lock()
	defer parser.mutex.Unlock()

	parser.Arguments = finalArguments

	return parser.tryParseToolcallPositions()
}

// ParseSemanticRegular parses the final content of the regular
func (parser *Parser) ParseSemanticRegular(finalContent string) ([]types.Position, error) {
	parser.mutex.Lock()
	defer parser.mutex.Unlock()

	parser.Content = finalContent

	return parser.tryParseRegularPositions()
}

// ParseSemanticPositions parses a single streaming chunk returns a semantic chunk
func (parser *Parser) ParseSemanticPositions(chunkData []byte) ([]types.Position, error) {
	if parser.Toolcall {
		return parser.parseSemanticToolcall(chunkData)
	}
	return parser.parseSemanticRegular(chunkData)
}

// parseSemanticToolcall parses a single streaming chunk returns a semantic toolcall
func (parser *Parser) parseSemanticToolcall(chunkData []byte) ([]types.Position, error) {
	parser.mutex.Lock()
	defer parser.mutex.Unlock()

	// Skip empty chunks
	if len(chunkData) == 0 {
		return nil, nil
	}

	// Handle SSE format (data: prefix)
	dataStr := string(chunkData)
	if strings.HasPrefix(dataStr, "data: ") {
		dataStr = strings.TrimPrefix(dataStr, "data: ")
		if strings.TrimSpace(dataStr) == "[DONE]" {
			parser.finished = true
			return parser.tryParseToolcallPositions()
		}
		chunkData = []byte(dataStr)
	}

	// Parse the streaming chunk JSON
	var chunkObj map[string]interface{}
	if err := jsoniter.Unmarshal(chunkData, &chunkObj); err != nil {
		// If JSON parsing fails, return empty for now
		return nil, nil
	}

	// Extract tool call arguments from streaming response
	choices, ok := chunkObj["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, nil
	}

	choice := choices[0].(map[string]interface{})
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	// Check for tool calls in delta
	if toolCalls, ok := delta["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		toolCall := toolCalls[0].(map[string]interface{})
		if function, ok := toolCall["function"].(map[string]interface{}); ok {
			if args, ok := function["arguments"].(string); ok {
				parser.Arguments += args
			}
		}
	}

	// Check finish reason
	if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" {
		parser.finished = true
		return parser.tryParseToolcallPositions()
	}

	// If not finished, try to parse what we have so far
	return parser.tryParseToolcallPositions()
}

// parseSemanticRegular parses a single streaming chunk returns a semantic regular
func (parser *Parser) parseSemanticRegular(chunkData []byte) ([]types.Position, error) {
	parser.mutex.Lock()
	defer parser.mutex.Unlock()

	// Skip empty chunks
	if len(chunkData) == 0 {
		return nil, nil
	}

	// Handle SSE format (data: prefix)
	dataStr := string(chunkData)
	if strings.HasPrefix(dataStr, "data: ") {
		dataStr = strings.TrimPrefix(dataStr, "data: ")
		if strings.TrimSpace(dataStr) == "[DONE]" {
			parser.finished = true
			return parser.tryParseRegularPositions()
		}
		chunkData = []byte(dataStr)
	}

	// Parse the streaming chunk JSON
	var chunkObj map[string]interface{}
	if err := jsoniter.Unmarshal(chunkData, &chunkObj); err != nil {
		// If JSON parsing fails, return empty for now
		return nil, nil
	}

	// Extract content from streaming response
	choices, ok := chunkObj["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, nil
	}

	choice := choices[0].(map[string]interface{})
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	// Extract content from delta
	if content, ok := delta["content"].(string); ok {
		parser.Content += content
	}
	// Check finish reason
	if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" {
		parser.finished = true
		return parser.tryParseRegularPositions()
	}

	// If not finished, try to parse what we have so far
	return parser.tryParseRegularPositions()
}

// tryParseToolcallPositions attempts to parse positions from accumulated tool call arguments
func (parser *Parser) tryParseToolcallPositions() ([]types.Position, error) {
	arguments := strings.TrimSpace(parser.Arguments)
	if len(arguments) < 10 {
		return nil, nil
	}

	// Try to complete incomplete JSON by truncating to last complete object
	arguments = parser.completeJSON(arguments)

	// First try to parse with jsoniter
	var args map[string]interface{}
	err := jsoniter.UnmarshalFromString(arguments, &args)
	if err != nil {
		// Try to repair the JSON using jsonrepair
		repaired, errRepair := jsonrepair.JSONRepair(arguments)
		if errRepair != nil {
			return nil, fmt.Errorf("failed to repair JSON: %w", errRepair)
		}

		// Retry with repaired JSON
		err = jsoniter.UnmarshalFromString(repaired, &args)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repaired JSON: %w", err)
		}
	}

	// Extract segments from arguments
	segments, ok := args["segments"].([]interface{})
	if !ok {
		return nil, nil
	}

	var positions []types.Position
	for _, seg := range segments {
		segMap, ok := seg.(map[string]interface{})
		if !ok {
			continue
		}

		startPos := parser.toInt(segMap["s"])
		endPos := parser.toInt(segMap["e"])

		if startPos >= 0 && endPos > startPos {
			positions = append(positions, types.Position{
				StartPos: startPos,
				EndPos:   endPos,
			})
		}
	}

	return positions, nil
}

// completeJSON tries to complete incomplete JSON by truncating to last complete object
func (parser *Parser) completeJSON(jsonStr string) string {
	original := strings.TrimSpace(jsonStr)

	// Check if JSON ends with }] or }]} (with possible whitespace/newlines)
	if strings.HasSuffix(original, "}]") || strings.HasSuffix(original, "}]}") {
		return original // Already complete
	}

	// Look for incomplete objects by finding the last complete one
	// Pattern: Find the last },{"s" which indicates an incomplete object follows
	lastValidEndPos := -1

	// Find all occurrences of },{"s" or }, {"s"
	for i := 0; i < len(original)-4; i++ {
		if original[i] == '}' {
			// Check what follows after the }
			j := i + 1
			// Skip whitespace after }
			for j < len(original) && (original[j] == ' ' || original[j] == '\n' || original[j] == '\t') {
				j++
			}

			// Check if we have ,{"s pattern (beginning of incomplete object)
			if j < len(original) && original[j] == ',' {
				remaining := original[j+1:]
				remaining = strings.TrimSpace(remaining)
				if strings.HasPrefix(remaining, `{"s`) {
					// This } ends a complete object, and what follows is incomplete
					lastValidEndPos = i
				}
			}
		}
	}

	// If we found a truncation point, use it
	if lastValidEndPos > 0 {
		truncated := original[:lastValidEndPos+1] // Include the closing }
		// Add proper closing for segments array and object
		if strings.Contains(truncated, `"segments":[`) && !strings.HasSuffix(truncated, "}]") {
			truncated += "]}"
		}
		return truncated
	}

	return original // Return as-is if we can't find a good truncation point
}

// tryParseRegularPositions attempts to parse positions from accumulated content
func (parser *Parser) tryParseRegularPositions() ([]types.Position, error) {
	content := strings.TrimSpace(parser.Content)
	if len(content) < 10 {
		return nil, nil
	}

	// Extract JSON array from content
	jsonStr := parser.extractJSONArray(content)
	if jsonStr == "" {
		return nil, nil
	}

	// First try to parse as generic maps to handle different field names
	var rawPositions []map[string]interface{}
	err := jsoniter.UnmarshalFromString(jsonStr, &rawPositions)
	if err != nil {
		// Try to repair the JSON using jsonrepair
		repaired, errRepair := jsonrepair.JSONRepair(jsonStr)
		if errRepair != nil {
			return nil, fmt.Errorf("failed to repair JSON: %w", errRepair)
		}

		// Retry with repaired JSON
		err = jsoniter.UnmarshalFromString(repaired, &rawPositions)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repaired JSON: %w", err)
		}
	}

	// Convert to types.Position, handling both start_pos/end_pos and s/e formats
	var positions []types.Position
	for _, rawPos := range rawPositions {
		var startPos, endPos int

		// Try start_pos/end_pos format first
		if startPosVal, exists := rawPos["start_pos"]; exists {
			startPos = parser.toInt(startPosVal)
		} else if sVal, exists := rawPos["s"]; exists {
			startPos = parser.toInt(sVal)
		}

		if endPosVal, exists := rawPos["end_pos"]; exists {
			endPos = parser.toInt(endPosVal)
		} else if eVal, exists := rawPos["e"]; exists {
			endPos = parser.toInt(eVal)
		}

		if startPos >= 0 && endPos > startPos {
			positions = append(positions, types.Position{
				StartPos: startPos,
				EndPos:   endPos,
			})
		}
	}

	return positions, nil
}

// extractJSONArray extracts JSON array from text content
func (parser *Parser) extractJSONArray(text string) string {
	// Remove markdown code blocks

	if len(text) < 10 {
		return ""
	}

	text = strings.TrimSpace(text)
	text = strings.Trim(text, "\r")
	text = strings.Trim(text, "\n")
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```", "")
	text = strings.ReplaceAll(text, "\n", "")

	// Try to complete incomplete JSON by truncating to last complete object
	text = parser.completeJSON(text)
	return text
}

// toInt safely converts interface{} to int
func (parser *Parser) toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return -1
	}
}
