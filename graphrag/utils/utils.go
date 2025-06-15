package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kaptinlin/jsonrepair"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/http"
)

// FileReader wraps os.File to implement io.ReadSeeker with Close method
type FileReader struct {
	*os.File
}

// Close closes the underlying file
func (f *FileReader) Close() error {
	return f.File.Close()
}

// OpenFileAsReader opens a file and returns a ReadSeeker with Close method
func OpenFileAsReader(filename string) (*FileReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return &FileReader{File: file}, nil
}

// PostLLM sends a POST request to LLM API using connector
func PostLLM(ctx context.Context, conn connector.Connector, endpoint string, payload map[string]interface{}) (interface{}, error) {
	setting := conn.Setting()

	// Get host from connector settings
	host, ok := setting["host"].(string)
	if !ok || host == "" {
		return nil, fmt.Errorf("no host found in connector settings")
	}

	// Build full URL
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(host, "/"), endpoint)

	// Get API key
	apiKey, ok := setting["key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("API key is not set")
	}

	// Make HTTP request (exactly like openai.go)
	r := http.New(url)
	r.SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	r.SetHeader("Content-Type", "application/json")
	r.WithContext(ctx)

	resp := r.Post(payload)
	if resp.Status != 200 {
		return nil, fmt.Errorf("request failed with status: %d, data: %v", resp.Status, resp.Data)
	}

	return resp.Data, nil
}

// StreamLLM sends a streaming request to LLM service with real-time callback
func StreamLLM(ctx context.Context, conn connector.Connector, endpoint string, payload map[string]interface{}, callback func(data []byte) error) error {
	// Get connector settings
	setting := conn.Setting()

	// Get host from connector settings
	host, ok := setting["host"].(string)
	if !ok || host == "" {
		return fmt.Errorf("no host found in connector settings")
	}

	// Build full URL
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(host, "/"), endpoint)

	// Get API key
	key, ok := setting["key"].(string)
	if !ok || key == "" {
		return fmt.Errorf("API key is not set")
	}

	// Add stream parameter to payload
	streamPayload := make(map[string]interface{})
	for k, v := range payload {
		streamPayload[k] = v
	}
	streamPayload["stream"] = true

	// Create HTTP request
	req := http.New(url).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", key))

	// Stream handler function
	streamHandler := func(data []byte) int {
		// Call user callback
		if callback != nil {
			if err := callback(data); err != nil {
				return http.HandlerReturnError
			}
		}
		return http.HandlerReturnOk
	}

	// Make streaming request with provided context
	err := req.Stream(ctx, "POST", streamPayload, streamHandler)
	if err != nil {
		return fmt.Errorf("streaming request failed: %w", err)
	}

	return nil
}

// ParseJSONOptions parses JSON options string and returns a map
func ParseJSONOptions(optionsStr string) (map[string]interface{}, error) {
	if optionsStr == "" {
		return make(map[string]interface{}), nil
	}

	var options map[string]interface{}
	if err := json.Unmarshal([]byte(optionsStr), &options); err != nil {
		return nil, fmt.Errorf("failed to parse JSON options: %w", err)
	}

	return options, nil
}

// StreamChunkData represents parsed data from streaming chunk
type StreamChunkData struct {
	Content    string                 `json:"content"`
	Arguments  string                 `json:"arguments"`
	Positions  []SemanticPosition     `json:"positions,omitempty"`
	IsToolcall bool                   `json:"is_toolcall"`
	Finished   bool                   `json:"finished"`
	Error      string                 `json:"error,omitempty"`
	Raw        map[string]interface{} `json:"raw,omitempty"`
}

// SemanticPosition represents a semantic segment position
type SemanticPosition struct {
	StartPos int `json:"start_pos"`
	EndPos   int `json:"end_pos"`
}

// StreamParser handles parsing of streaming LLM responses
type StreamParser struct {
	accumulatedContent   string
	accumulatedArguments string
	isToolcall           bool
	finished             bool
}

// NewStreamParser creates a new stream parser
func NewStreamParser(isToolcall bool) *StreamParser {
	return &StreamParser{
		isToolcall: isToolcall,
	}
}

// ParseStreamChunk parses a single streaming chunk and returns accumulated data with positions
func (sp *StreamParser) ParseStreamChunk(chunkData []byte) (*StreamChunkData, error) {
	// Skip empty chunks
	if len(chunkData) == 0 {
		return &StreamChunkData{
			Content:    sp.accumulatedContent,
			Arguments:  sp.accumulatedArguments,
			IsToolcall: sp.isToolcall,
			Finished:   sp.finished,
		}, nil
	}

	// Parse the chunk as JSON with error tolerance
	chunkStr := strings.TrimSpace(string(chunkData))

	// Handle SSE format (data: prefix)
	chunkStr = strings.TrimPrefix(chunkStr, "data: ")

	// Skip [DONE] marker
	if chunkStr == "[DONE]" {
		sp.finished = true
		// Try to parse final accumulated content for positions
		positions := sp.parseAccumulatedContent()
		return &StreamChunkData{
			Content:    sp.accumulatedContent,
			Arguments:  sp.accumulatedArguments,
			Positions:  positions,
			IsToolcall: sp.isToolcall,
			Finished:   true,
		}, nil
	}

	// Parse JSON with tolerance
	var chunkObj map[string]interface{}
	if err := TolerantJSONUnmarshal([]byte(chunkStr), &chunkObj); err != nil {
		// If parsing fails, return current state without error
		return &StreamChunkData{
			Content:    sp.accumulatedContent,
			Arguments:  sp.accumulatedArguments,
			IsToolcall: sp.isToolcall,
			Finished:   sp.finished,
			Error:      fmt.Sprintf("JSON parse error: %v", err),
			Raw:        map[string]interface{}{"raw_chunk": chunkStr},
		}, nil
	}

	// Extract content based on toolcall or regular response
	if sp.isToolcall {
		sp.parseToolcallChunk(chunkObj)
	} else {
		sp.parseRegularChunk(chunkObj)
	}

	// Try to parse positions from accumulated content
	positions := sp.parseAccumulatedContent()

	return &StreamChunkData{
		Content:    sp.accumulatedContent,
		Arguments:  sp.accumulatedArguments,
		Positions:  positions,
		IsToolcall: sp.isToolcall,
		Finished:   sp.finished,
		Raw:        chunkObj,
	}, nil
}

// parseAccumulatedContent tries to parse semantic positions from accumulated content
func (sp *StreamParser) parseAccumulatedContent() []SemanticPosition {
	var contentToParse string

	if sp.isToolcall {
		// For toolcall, parse from accumulated arguments
		contentToParse = sp.accumulatedArguments
	} else {
		// For regular response, parse from accumulated content
		contentToParse = sp.accumulatedContent
	}

	if strings.TrimSpace(contentToParse) == "" {
		return nil
	}

	// Try to extract and parse JSON positions
	return sp.extractPositionsFromText(contentToParse)
}

// extractPositionsFromText extracts semantic positions from text content
func (sp *StreamParser) extractPositionsFromText(text string) []SemanticPosition {
	// For toolcall, the text should be JSON arguments
	if sp.isToolcall {
		return sp.parseToolcallPositions(text)
	}

	// For regular response, extract JSON from the content
	return sp.parseRegularPositions(text)
}

// parseToolcallPositions parses positions from toolcall arguments
func (sp *StreamParser) parseToolcallPositions(arguments string) []SemanticPosition {
	arguments = strings.TrimSpace(arguments)
	if arguments == "" {
		return nil
	}

	// Try to parse the arguments as JSON directly first
	var args map[string]interface{}
	if err := TolerantJSONUnmarshal([]byte(arguments), &args); err != nil {
		// If parsing fails, try to complete the JSON
		completedJSON := sp.completeJSON(arguments)
		if err := TolerantJSONUnmarshal([]byte(completedJSON), &args); err != nil {
			// If still fails, try jsonrepair as last resort
			repairedJSON, repairErr := jsonrepair.JSONRepair(arguments)
			if repairErr != nil {
				return nil
			}
			if err := TolerantJSONUnmarshal([]byte(repairedJSON), &args); err != nil {
				return nil
			}
		}
	}

	// Extract segments from arguments
	segments, ok := args["segments"].([]interface{})
	if !ok {
		return nil
	}

	var positions []SemanticPosition
	for _, seg := range segments {
		segMap, ok := seg.(map[string]interface{})
		if !ok {
			continue
		}

		startPos, startOk := segMap["start_pos"]
		endPos, endOk := segMap["end_pos"]
		if !startOk || !endOk {
			continue
		}

		// Convert to int (handle both float64 and int)
		var start, end int
		switch v := startPos.(type) {
		case float64:
			start = int(v)
		case int:
			start = v
		default:
			continue
		}

		switch v := endPos.(type) {
		case float64:
			end = int(v)
		case int:
			end = v
		default:
			continue
		}

		// Validate position values
		if start < 0 || end < 0 || start >= end {
			continue
		}

		positions = append(positions, SemanticPosition{
			StartPos: start,
			EndPos:   end,
		})
	}

	return positions
}

// parseRegularPositions parses positions from regular response content
func (sp *StreamParser) parseRegularPositions(content string) []SemanticPosition {
	// Extract JSON array from content
	jsonStr := sp.extractJSONFromText(content)
	if jsonStr == "" {
		return nil
	}

	// Try to complete incomplete JSON
	completedJSON := sp.completeJSON(jsonStr)

	// Parse positions
	var positions []SemanticPosition
	if err := TolerantJSONUnmarshal([]byte(completedJSON), &positions); err != nil {
		return nil
	}

	return positions
}

// extractJSONFromText extracts JSON array from text content
func (sp *StreamParser) extractJSONFromText(text string) string {
	// Remove markdown code blocks
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```", "")

	// Find JSON array boundaries
	start := strings.Index(text, "[")
	if start == -1 {
		return ""
	}

	// Find the last ] that could close the array
	end := strings.LastIndex(text, "]")
	if end == -1 || end <= start {
		// Array not closed yet, return what we have so far
		return text[start:]
	}

	return text[start : end+1]
}

// completeJSON tries to complete incomplete JSON for parsing
func (sp *StreamParser) completeJSON(jsonStr string) string {
	jsonStr = strings.TrimSpace(jsonStr)

	// If it's toolcall arguments, try to complete the object
	if sp.isToolcall {
		return sp.completeToolcallJSON(jsonStr)
	}

	// For regular response, try to complete the array
	return sp.completeArrayJSON(jsonStr)
}

// completeToolcallJSON completes incomplete toolcall JSON
func (sp *StreamParser) completeToolcallJSON(jsonStr string) string {
	jsonStr = strings.TrimSpace(jsonStr)

	// If it doesn't start with {, add it
	if !strings.HasPrefix(jsonStr, "{") {
		jsonStr = "{" + jsonStr
	}

	// Check if we have segments array structure
	if !strings.Contains(jsonStr, "segments") && !strings.Contains(jsonStr, "\"segments\"") {
		// If no segments found, try to wrap content in segments structure
		// This handles cases where only the array content is provided
		if strings.HasPrefix(jsonStr, "{") && !strings.Contains(jsonStr, "segments") {
			// Extract any array content and wrap it
			if strings.Contains(jsonStr, "[") {
				arrayStart := strings.Index(jsonStr, "[")
				arrayContent := jsonStr[arrayStart:]
				jsonStr = `{"segments":` + arrayContent + `}`
			} else if strings.Contains(jsonStr, `"start_pos"`) {
				// Looks like segment objects without array wrapper
				jsonStr = `{"segments":[` + jsonStr[1:] // Remove opening { and wrap in segments array
			}
		}
	}

	// Count braces and brackets to see if we need to close them
	openBraces := strings.Count(jsonStr, "{") - strings.Count(jsonStr, "}")
	openBrackets := strings.Count(jsonStr, "[") - strings.Count(jsonStr, "]")

	// Close any open brackets first (arrays)
	for i := 0; i < openBrackets; i++ {
		jsonStr += "]"
	}

	// Close any open braces (objects)
	for i := 0; i < openBraces; i++ {
		jsonStr += "}"
	}

	// If we have segments but it's not properly structured, try to fix it
	if strings.Contains(jsonStr, "segments") && !strings.Contains(jsonStr, `"segments":[`) {
		// Try to find and fix segments structure
		segmentPos := strings.Index(jsonStr, "segments")
		if segmentPos > 0 {
			prefix := jsonStr[:segmentPos]
			suffix := jsonStr[segmentPos:]

			// Ensure proper JSON structure for segments
			if !strings.Contains(prefix, `"segments"`) {
				suffix = `"` + suffix
			}
			if !strings.Contains(suffix, ":[") && strings.Contains(suffix, "[") {
				suffix = strings.Replace(suffix, "segments", "segments", 1)
				if !strings.Contains(suffix, ":") {
					arrayStart := strings.Index(suffix, "[")
					if arrayStart > 0 {
						suffix = suffix[:arrayStart] + ":" + suffix[arrayStart:]
					}
				}
			}
			jsonStr = prefix + suffix
		}
	}

	return jsonStr
}

// completeArrayJSON completes incomplete array JSON
func (sp *StreamParser) completeArrayJSON(jsonStr string) string {
	if !strings.HasPrefix(jsonStr, "[") {
		jsonStr = "[" + jsonStr
	}

	// If the array is not closed, close it
	if !strings.HasSuffix(jsonStr, "]") {
		// Remove trailing comma if present
		jsonStr = strings.TrimSuffix(strings.TrimSpace(jsonStr), ",")

		// Count open braces to close them properly
		openBraces := strings.Count(jsonStr, "{") - strings.Count(jsonStr, "}")
		for i := 0; i < openBraces; i++ {
			jsonStr += "}"
		}

		jsonStr += "]"
	}

	return jsonStr
}

// parseToolcallChunk parses toolcall streaming response
func (sp *StreamParser) parseToolcallChunk(chunkObj map[string]interface{}) {
	choices, ok := chunkObj["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return
	}

	choice := choices[0].(map[string]interface{})
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return
	}

	// Check for tool calls in delta
	if toolCalls, ok := delta["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		toolCall := toolCalls[0].(map[string]interface{})
		if function, ok := toolCall["function"].(map[string]interface{}); ok {
			if args, ok := function["arguments"].(string); ok {
				sp.accumulatedArguments += args
			}
		}
	}

	// Check finish reason
	if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" {
		sp.finished = true
	}
}

// parseRegularChunk parses regular streaming response
func (sp *StreamParser) parseRegularChunk(chunkObj map[string]interface{}) {
	choices, ok := chunkObj["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return
	}

	choice := choices[0].(map[string]interface{})
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return
	}

	// Extract content from delta
	if content, ok := delta["content"].(string); ok {
		sp.accumulatedContent += content
	}

	// Check finish reason
	if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" {
		sp.finished = true
	}
}

// TolerantJSONUnmarshal unmarshals JSON with error tolerance using JSONRepair
func TolerantJSONUnmarshal(data []byte, v interface{}) error {
	// First try normal unmarshal
	if err := json.Unmarshal(data, v); err == nil {
		return nil
	}

	// If failed, try to repair JSON
	repairedJSON, err := jsonrepair.JSONRepair(string(data))
	if err != nil {
		return fmt.Errorf("failed to repair JSON: %w", err)
	}

	// Try unmarshal again with repaired JSON
	if err := json.Unmarshal([]byte(repairedJSON), v); err != nil {
		return fmt.Errorf("failed to unmarshal repaired JSON: %w", err)
	}

	return nil
}

// GetSemanticPrompt returns semantic analysis prompt, user-defined or default
func GetSemanticPrompt(userPrompt string) string {
	if strings.TrimSpace(userPrompt) != "" {
		return userPrompt
	}

	return GetDefaultSemanticPrompt()
}

// GetDefaultSemanticPrompt returns the default semantic segmentation prompt
func GetDefaultSemanticPrompt() string {
	return `You are an expert text analyst. Please analyze the following text and segment it into semantically coherent chunks. Each chunk should represent a complete thought, topic, or concept.

Instructions:
1. Identify natural semantic boundaries in the text
2. Each segment should be meaningful and self-contained
3. Avoid splitting sentences or related concepts
4. Aim for segments that are roughly balanced in size but prioritize semantic coherence
5. Return the segments as a JSON array with start_pos and end_pos positions

Output format:
[
  {"start_pos": 0, "end_pos": 150},
  {"start_pos": 150, "end_pos": 300},
  ...
]

Requirements:
- start_pos and end_pos are character positions (integers)
- Segments should not overlap
- All positions should be within the text boundaries
- Do not include the actual text content in your response, only positions`
}
