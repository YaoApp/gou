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

	// For streaming, we might have partial data that doesn't contain entities/relationships yet
	// Only check for required fields if we have what looks like a complete JSON structure
	hasClosingBrace := strings.Contains(arguments, "}")
	hasEntities := strings.Contains(arguments, `"entities"`)
	hasRelationships := strings.Contains(arguments, `"relationships"`)

	// If it looks like a complete JSON (has closing brace) but missing both required fields
	if hasClosingBrace && !hasEntities && !hasRelationships {
		// Allow partial streaming chunks that are clearly incomplete and could still become valid:
		// - Contains partial field names that could become entities/relationships
		// - OR contains valid JSON field structure (quotes and colons) indicating partial entity data
		isPartialChunk := strings.Contains(arguments, `"entit`) || strings.Contains(arguments, `"relation`) ||
			(strings.Contains(arguments, `"`) && strings.Contains(arguments, `:`))

		if !isPartialChunk {
			// This looks like a complete JSON but doesn't have the required extraction fields
			return nil, nil, fmt.Errorf("invalid extraction JSON: missing required fields")
		}
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
	if len(original) < 3 {
		return `{"entities":[],"relationships":[]}`
	}

	// Handle empty object - add required arrays
	if original == "{}" {
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

	// Try to parse as-is first
	var result map[string]interface{}
	err := jsoniter.UnmarshalFromString(original, &result)
	if err == nil {
		// JSON is valid, ensure it has required arrays in correct order
		orderedResult := map[string]interface{}{
			"entities":      []interface{}{},
			"relationships": []interface{}{},
		}

		if entities, hasEntities := result["entities"]; hasEntities {
			orderedResult["entities"] = entities
		}
		if relationships, hasRelationships := result["relationships"]; hasRelationships {
			orderedResult["relationships"] = relationships
		}

		completed, marshalErr := jsoniter.MarshalToString(orderedResult)
		if marshalErr == nil {
			return completed
		}
	}

	// JSON is incomplete, try to repair it intelligently
	return parser.repairIncompleteJSON(original)
}

// repairIncompleteJSON repairs incomplete JSON by extracting valid parts
func (parser *Parser) repairIncompleteJSON(jsonStr string) string {
	// Start building the result
	result := map[string]interface{}{
		"entities":      []interface{}{},
		"relationships": []interface{}{},
	}

	// Extract entities if present
	if strings.Contains(jsonStr, `"entities"`) {
		entities := parser.extractValidEntities(jsonStr)
		if len(entities) > 0 {
			result["entities"] = entities
		}
	}

	// Extract relationships if present
	if strings.Contains(jsonStr, `"relationships"`) {
		relationships := parser.extractValidRelationships(jsonStr)
		if len(relationships) > 0 {
			result["relationships"] = relationships
		}
	}

	// Convert back to JSON
	completed, err := jsoniter.MarshalToString(result)
	if err != nil {
		return `{"entities":[],"relationships":[]}`
	}

	return completed
}

// extractValidEntities extracts valid entities from incomplete JSON
func (parser *Parser) extractValidEntities(jsonStr string) []interface{} {
	var entities []interface{}

	// Find entities array start (handle whitespace)
	entitiesStart := strings.Index(jsonStr, `"entities"`)
	if entitiesStart == -1 {
		return entities
	}

	// Find the opening bracket after "entities"
	colonPos := strings.Index(jsonStr[entitiesStart:], `:`)
	if colonPos == -1 {
		return entities
	}

	bracketPos := strings.Index(jsonStr[entitiesStart+colonPos:], `[`)
	if bracketPos == -1 {
		return entities
	}

	pos := entitiesStart + colonPos + bracketPos + 1

	// Parse each entity object
	braceCount := 0
	start := -1
	inString := false
	escapeNext := false

	for i := pos; i < len(jsonStr); i++ {
		char := jsonStr[i]

		if escapeNext {
			escapeNext = false
			continue
		}
		if char == '\\' {
			escapeNext = true
			continue
		}
		if char == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch char {
		case '{':
			if braceCount == 0 {
				start = i
			}
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 && start >= 0 {
				// Found a complete entity object
				entityStr := strings.TrimSpace(jsonStr[start : i+1])
				var entity map[string]interface{}
				if jsoniter.UnmarshalFromString(entityStr, &entity) == nil {
					// Ensure entity has at least id field
					if _, hasID := entity["id"]; hasID {
						entities = append(entities, entity)
					}
				}
				start = -1
			}
		case ']':
			// End of entities array
			return entities
		}
	}

	// Handle incomplete last entity - try to repair it
	if start >= 0 && braceCount > 0 {
		// Try to close the incomplete entity
		incompleteStr := strings.TrimSpace(jsonStr[start:])
		repairedEntity := parser.repairIncompleteEntity(incompleteStr)
		if repairedEntity != nil {
			entities = append(entities, repairedEntity)
		}
	}

	return entities
}

// extractValidRelationships extracts valid relationships from incomplete JSON
func (parser *Parser) extractValidRelationships(jsonStr string) []interface{} {
	var relationships []interface{}

	// Find relationships array start
	relStart := strings.Index(jsonStr, `"relationships":[`)
	if relStart == -1 {
		return relationships
	}

	// Start after the opening bracket
	pos := relStart + len(`"relationships":[`)

	// Parse each relationship object
	braceCount := 0
	start := -1
	inString := false
	escapeNext := false

	for i := pos; i < len(jsonStr); i++ {
		char := jsonStr[i]

		if escapeNext {
			escapeNext = false
			continue
		}
		if char == '\\' {
			escapeNext = true
			continue
		}
		if char == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch char {
		case '{':
			if braceCount == 0 {
				start = i
			}
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 && start >= 0 {
				// Found a complete relationship object
				relStr := strings.TrimSpace(jsonStr[start : i+1])
				var rel map[string]interface{}
				if jsoniter.UnmarshalFromString(relStr, &rel) == nil {
					// Ensure relationship has required fields
					if _, hasStart := rel["start_node"]; hasStart {
						if _, hasEnd := rel["end_node"]; hasEnd {
							relationships = append(relationships, rel)
						}
					}
				}
				start = -1
			}
		case ']':
			// End of relationships array
			return relationships
		}
	}

	// Don't try to repair incomplete relationships - they need both start_node and end_node
	return relationships
}

// repairIncompleteEntity tries to repair an incomplete entity object
func (parser *Parser) repairIncompleteEntity(entityStr string) map[string]interface{} {
	// Try to close incomplete nested objects first
	repaired := parser.closeIncompleteObjects(entityStr)

	// Try to parse the repaired entity
	var entity map[string]interface{}
	err := jsoniter.UnmarshalFromString(repaired, &entity)
	if err == nil {
		// Check if it has minimum required fields
		if _, hasID := entity["id"]; hasID {
			return entity
		}
	}

	return nil
}

// closeIncompleteObjects closes incomplete nested objects in a string
func (parser *Parser) closeIncompleteObjects(str string) string {
	// If we have an incomplete string (odd number of quotes), we need to handle it
	if strings.Count(str, `"`)%2 == 1 {
		// Find the last quote and see what comes after it
		lastQuote := strings.LastIndex(str, `"`)
		if lastQuote >= 0 {
			beforeQuote := str[:lastQuote]

			// If there's a comma before the incomplete string, check if we should remove it
			if strings.Contains(beforeQuote, ",") {
				lastComma := strings.LastIndex(beforeQuote, ",")
				if lastComma >= 0 {
					// The incomplete string appears to be a key without a value
					// Remove the incomplete key by cutting at the last comma
					result := beforeQuote[:lastComma]
					// Close any unclosed braces
					openBraces := strings.Count(result, "{")
					closeBraces := strings.Count(result, "}")
					for openBraces > closeBraces {
						result += "}"
						closeBraces++
					}
					return result
				}
			}
		}

		// If we can't handle it smartly, just close the string and add empty value
		result := str + `"`
		if strings.Contains(result, ",") {
			lastComma := strings.LastIndex(result, ",")
			afterComma := strings.TrimSpace(result[lastComma+1:])
			if strings.Count(afterComma, `"`) == 2 && !strings.Contains(afterComma, ":") {
				result += `: ""`
			}
		}

		// Close unclosed braces
		openBraces := strings.Count(result, "{")
		closeBraces := strings.Count(result, "}")
		for openBraces > closeBraces {
			result += "}"
			closeBraces++
		}
		return result
	}

	// No incomplete strings, just close unclosed braces
	result := str
	openBraces := strings.Count(result, "{")
	closeBraces := strings.Count(result, "}")

	for openBraces > closeBraces {
		result += "}"
		closeBraces++
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
