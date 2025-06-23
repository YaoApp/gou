package utils

import (
	"fmt"
	"strings"
	"sync"
	"time"

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

// NewExtractionParser creates a new extraction parser for tool call parsing
func NewExtractionParser() *Parser {
	return &Parser{Toolcall: true}
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

// ParseExtractionToolcall parses the final arguments of the extraction toolcall
func (parser *Parser) ParseExtractionToolcall(finalArguments string) ([]types.Node, []types.Relationship, error) {
	parser.mutex.Lock()
	defer parser.mutex.Unlock()

	parser.Arguments = finalArguments

	return parser.tryParseExtractionToolcall()
}

// ParseSemanticPositions parses a single streaming chunk returns a semantic chunk
func (parser *Parser) ParseSemanticPositions(chunkData []byte) ([]types.Position, error) {
	if parser.Toolcall {
		return parser.parseSemanticToolcall(chunkData)
	}
	return parser.parseSemanticRegular(chunkData)
}

// ParseExtractionEntities parses a single streaming chunk for extraction and returns extraction progress
func (parser *Parser) ParseExtractionEntities(chunkData []byte) ([]types.Node, []types.Relationship, error) {
	parser.mutex.Lock()
	defer parser.mutex.Unlock()

	// Skip empty chunks
	if len(chunkData) == 0 {
		return nil, nil, nil
	}

	// Handle SSE format (data: prefix)
	dataStr := string(chunkData)
	if strings.HasPrefix(dataStr, "data: ") {
		dataStr = strings.TrimPrefix(dataStr, "data: ")
		if strings.TrimSpace(dataStr) == "[DONE]" {
			parser.finished = true
			return parser.tryParseExtractionToolcall()
		}
		chunkData = []byte(dataStr)
	}

	// Parse the streaming chunk JSON
	var chunkObj map[string]interface{}
	if err := jsoniter.Unmarshal(chunkData, &chunkObj); err != nil {
		// If JSON parsing fails, return empty for now
		return nil, nil, nil
	}

	// Extract tool call arguments from streaming response
	choices, ok := chunkObj["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, nil, nil
	}

	choice := choices[0].(map[string]interface{})
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return nil, nil, nil
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
		return parser.tryParseExtractionToolcall()
	}

	// If not finished, try to parse what we have so far
	return parser.tryParseExtractionToolcall()
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

// tryParseExtractionToolcall attempts to parse entities and relationships from accumulated tool call arguments
func (parser *Parser) tryParseExtractionToolcall() ([]types.Node, []types.Relationship, error) {
	arguments := strings.TrimSpace(parser.Arguments)
	if len(arguments) < 10 {
		return nil, nil, nil
	}

	// Try to complete incomplete JSON by truncating to last complete object
	arguments = parser.completeExtractionJSON(arguments)

	// First try to parse with jsoniter
	var args map[string]interface{}
	err := jsoniter.UnmarshalFromString(arguments, &args)
	if err != nil {
		// Try to repair the JSON using jsonrepair
		repaired, errRepair := jsonrepair.JSONRepair(arguments)
		if errRepair != nil {
			// If JSON repair fails, return error for debugging
			return nil, nil, fmt.Errorf("failed to repair extraction JSON: %w (original: %s)", errRepair, arguments)
		}

		// Retry with repaired JSON
		err = jsoniter.UnmarshalFromString(repaired, &args)
		if err != nil {
			// If even repaired JSON fails, return error for debugging
			return nil, nil, fmt.Errorf("failed to parse repaired extraction JSON: %w (repaired: %s)", err, repaired)
		}
	}

	// Parse entities and relationships
	var nodes []types.Node
	var relationships []types.Relationship

	// Parse entities
	if entitiesRaw, ok := args["entities"].([]interface{}); ok {
		for _, entityRaw := range entitiesRaw {
			if entityMap, ok := entityRaw.(map[string]interface{}); ok {
				node := types.Node{
					ExtractionMethod: types.ExtractionMethodLLM,
					CreatedAt:        time.Now().Unix(),
					Version:          1,
					Status:           types.EntityStatusActive,
				}

				if id, ok := entityMap["id"].(string); ok {
					node.ID = id
				}
				if name, ok := entityMap["name"].(string); ok {
					node.Name = name
				}
				if entityType, ok := entityMap["type"].(string); ok {
					node.Type = entityType
				}
				if description, ok := entityMap["description"].(string); ok {
					node.Description = description
				}
				if confidence, ok := entityMap["confidence"].(float64); ok {
					node.Confidence = confidence
				}

				nodes = append(nodes, node)
			}
		}
	}

	// Parse relationships
	if relationshipsRaw, ok := args["relationships"].([]interface{}); ok {
		for _, relationshipRaw := range relationshipsRaw {
			if relationshipMap, ok := relationshipRaw.(map[string]interface{}); ok {
				relationship := types.Relationship{
					ExtractionMethod: types.ExtractionMethodLLM,
					CreatedAt:        time.Now().Unix(),
					Version:          1,
					Status:           types.EntityStatusActive,
				}

				if startNode, ok := relationshipMap["start_node"].(string); ok {
					relationship.StartNode = startNode
				}
				if endNode, ok := relationshipMap["end_node"].(string); ok {
					relationship.EndNode = endNode
				}
				if relType, ok := relationshipMap["type"].(string); ok {
					relationship.Type = relType
				}
				if description, ok := relationshipMap["description"].(string); ok {
					relationship.Description = description
				}
				if confidence, ok := relationshipMap["confidence"].(float64); ok {
					relationship.Confidence = confidence
				}

				relationships = append(relationships, relationship)
			}
		}
	}

	return nodes, relationships, nil
}

// completeExtractionJSON tries to complete incomplete extraction JSON
func (parser *Parser) completeExtractionJSON(jsonStr string) string {
	original := strings.TrimSpace(jsonStr)

	// If empty or too short, return minimal valid JSON
	if len(original) < 10 {
		return `{"entities":[],"relationships":[]}`
	}

	// Check for duplicate JSON objects (streaming issue)
	// Look for pattern like }{"entities" which indicates concatenated objects
	duplicatePattern := `}{"entities"`
	if strings.Contains(original, duplicatePattern) {
		// Find the first complete JSON object
		firstEnd := strings.Index(original, duplicatePattern)
		if firstEnd > 0 {
			original = original[:firstEnd+1] // Keep only the first complete object
		}
	}

	// Check if JSON is already complete
	if strings.HasSuffix(original, "}") && strings.Count(original, "{") == strings.Count(original, "}") {
		return original
	}

	// Try to find a valid truncation point by looking for complete JSON objects
	// Start from the end and work backwards to find the last complete object
	var truncated string
	braceCount := 0
	bracketCount := 0
	inString := false
	escapeNext := false

	for i := len(original) - 1; i >= 0; i-- {
		char := original[i]

		// Handle string escaping
		if escapeNext {
			escapeNext = false
			continue
		}
		if char == '\\' {
			escapeNext = true
			continue
		}

		// Track string boundaries
		if char == '"' {
			inString = !inString
			continue
		}

		// Skip characters inside strings
		if inString {
			continue
		}

		// Count braces and brackets
		switch char {
		case '}':
			braceCount++
		case '{':
			braceCount--
		case ']':
			bracketCount++
		case '[':
			bracketCount--
		}

		// Check if we have a balanced structure at this point
		if braceCount == 0 && bracketCount == 0 {
			// Check if this looks like a complete extraction JSON
			candidate := original[:i+1]
			if strings.Contains(candidate, `"entities"`) || strings.Contains(candidate, `"relationships"`) {
				// This looks like a good truncation point
				truncated = candidate
				break
			}
		}
	}

	// If we found a good truncation point, use it
	if truncated != "" {
		// Ensure it has both entities and relationships fields
		if !strings.Contains(truncated, `"entities"`) {
			// Insert entities field at the beginning
			if strings.HasPrefix(truncated, "{") {
				truncated = `{"entities":[],` + truncated[1:]
			}
		}
		if !strings.Contains(truncated, `"relationships"`) {
			// Add relationships field before closing
			if strings.HasSuffix(truncated, "}") {
				truncated = truncated[:len(truncated)-1] + `,"relationships":[]}`
			}
		}
		return truncated
	}

	// If no good truncation found, try to build a minimal valid JSON
	// Look for any entities or relationships data we can salvage
	entitiesStart := strings.Index(original, `"entities":[`)
	relationshipsStart := strings.Index(original, `"relationships":[`)

	result := `{"entities":[],"relationships":[]}`

	// Try to extract entities if found
	if entitiesStart >= 0 {
		entitiesEnd := findArrayEnd(original, entitiesStart+len(`"entities":`))
		if entitiesEnd > entitiesStart {
			entitiesData := original[entitiesStart+len(`"entities":`) : entitiesEnd+1]
			result = `{"entities":` + entitiesData + `,"relationships":[]}`
		}
	}

	// Try to extract relationships if found
	if relationshipsStart >= 0 {
		relationshipsEnd := findArrayEnd(original, relationshipsStart+len(`"relationships":`))
		if relationshipsEnd > relationshipsStart {
			relationshipsData := original[relationshipsStart+len(`"relationships":`) : relationshipsEnd+1]
			if entitiesStart >= 0 {
				// Replace the relationships part
				result = strings.Replace(result, `"relationships":[]`, `"relationships":`+relationshipsData, 1)
			} else {
				result = `{"entities":[],"relationships":` + relationshipsData + `}`
			}
		}
	}

	return result
}

// findArrayEnd finds the end position of a JSON array starting at the given position
func findArrayEnd(jsonStr string, startPos int) int {
	if startPos >= len(jsonStr) {
		return -1
	}

	// Skip whitespace to find the opening bracket
	i := startPos
	for i < len(jsonStr) && (jsonStr[i] == ' ' || jsonStr[i] == '\t' || jsonStr[i] == '\n') {
		i++
	}

	if i >= len(jsonStr) || jsonStr[i] != '[' {
		return -1
	}

	// Count brackets to find the matching closing bracket
	bracketCount := 0
	inString := false
	escapeNext := false

	for i < len(jsonStr) {
		char := jsonStr[i]

		if escapeNext {
			escapeNext = false
			i++
			continue
		}

		if char == '\\' {
			escapeNext = true
			i++
			continue
		}

		if char == '"' {
			inString = !inString
			i++
			continue
		}

		if !inString {
			if char == '[' {
				bracketCount++
			} else if char == ']' {
				bracketCount--
				if bracketCount == 0 {
					return i
				}
			}
		}

		i++
	}

	return -1
}
