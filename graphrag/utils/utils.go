package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

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

	// endpoint not start with /, then add /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	// Host is api.openai.com & endpoint not has /v1, then add /v1
	if host == "https://api.openai.com" && !strings.HasPrefix(endpoint, "/v1") {
		endpoint = "/v1" + endpoint
	}

	// Build full URL
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(host, "/"), strings.TrimPrefix(endpoint, "/"))

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

	// endpoint not start with /, then add /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	// Host is api.openai.com & endpoint not has /v1, then add /v1
	if host == "https://api.openai.com" && !strings.HasPrefix(endpoint, "/v1") {
		endpoint = "/v1" + endpoint
	}

	// Build full URL
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(host, "/"), strings.TrimPrefix(endpoint, "/"))

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

// StreamParser handles streaming LLM responses and accumulates data
type StreamParser struct {
	accumulatedContent   string
	accumulatedArguments string
	isToolcall           bool
	finished             bool
	mutex                sync.Mutex // Add mutex for concurrent safety
}

// NewStreamParser creates a new stream parser
func NewStreamParser(isToolcall bool) *StreamParser {
	return &StreamParser{
		isToolcall: isToolcall,
	}
}

// ParseStreamChunk parses a single streaming chunk and accumulates data
func (sp *StreamParser) ParseStreamChunk(chunkData []byte) (*StreamChunkData, error) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	// Skip empty chunks
	if len(chunkData) == 0 {
		return &StreamChunkData{
			Content:    sp.accumulatedContent,
			Arguments:  sp.accumulatedArguments,
			IsToolcall: sp.isToolcall,
			Finished:   sp.finished,
		}, nil
	}

	// Handle SSE format (data: prefix)
	dataStr := string(chunkData)
	if strings.HasPrefix(dataStr, "data: ") {
		dataStr = strings.TrimPrefix(dataStr, "data: ")
		if strings.TrimSpace(dataStr) == "[DONE]" {
			sp.finished = true
			// When finished, try to parse positions from accumulated data
			positions := sp.tryParsePositions()
			return &StreamChunkData{
				Content:    sp.accumulatedContent,
				Arguments:  sp.accumulatedArguments,
				Positions:  positions,
				IsToolcall: sp.isToolcall,
				Finished:   sp.finished,
			}, nil
		}
		chunkData = []byte(dataStr)
	}

	// Parse the streaming chunk JSON
	var chunkObj map[string]interface{}
	if err := json.Unmarshal(chunkData, &chunkObj); err != nil {
		// If JSON parsing fails, return current state with error info and raw data
		return &StreamChunkData{
			Content:    sp.accumulatedContent,
			Arguments:  sp.accumulatedArguments,
			IsToolcall: sp.isToolcall,
			Finished:   sp.finished,
			Error:      fmt.Sprintf("JSON parse error: %v", err),
			Raw:        map[string]interface{}{"raw_data": string(chunkData)},
		}, nil
	}

	// Extract and accumulate data based on type
	if sp.isToolcall {
		sp.parseToolcallChunk(chunkObj)
	} else {
		sp.parseRegularChunk(chunkObj)
	}

	// Try to parse positions from current accumulated data
	// Only attempt parsing if stream is finished or we have very complete data
	var positions []SemanticPosition
	if sp.finished {
		positions = sp.tryParsePositions()
	} else {
		// For ongoing streams, only try parsing if data looks very complete
		positions = sp.tryParsePositionsConservative()
	}

	return &StreamChunkData{
		Content:    sp.accumulatedContent,
		Arguments:  sp.accumulatedArguments,
		Positions:  positions,
		IsToolcall: sp.isToolcall,
		Finished:   sp.finished,
	}, nil
}

// getAccumulatedData returns the relevant accumulated data based on type
func (sp *StreamParser) getAccumulatedData() string {
	if sp.isToolcall {
		return sp.accumulatedArguments
	}
	return sp.accumulatedContent
}

// tryParsePositions attempts to parse semantic positions from accumulated data
func (sp *StreamParser) tryParsePositions() []SemanticPosition {
	data := strings.TrimSpace(sp.getAccumulatedData())
	if len(data) < 20 { // Need substantial data
		return nil
	}

	if sp.isToolcall {
		return sp.tryParseToolcallPositions(data)
	}
	return sp.tryParseRegularPositions(data)
}

// tryParseToolcallPositions attempts to parse positions from toolcall arguments
func (sp *StreamParser) tryParseToolcallPositions(arguments string) []SemanticPosition {
	// Must look like JSON with segments structure
	if !strings.Contains(arguments, "segments") || !strings.Contains(arguments, "start_pos") {
		return nil
	}

	// Try to parse using TolerantJSONUnmarshal directly
	var args map[string]interface{}
	if err := TolerantJSONUnmarshal([]byte(arguments), &args); err != nil {
		return nil // Parsing failed, wait for more data
	}

	// Extract segments
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

		startPos := sp.toInt(segMap["start_pos"])
		endPos := sp.toInt(segMap["end_pos"])

		if startPos >= 0 && endPos > startPos {
			positions = append(positions, SemanticPosition{
				StartPos: startPos,
				EndPos:   endPos,
			})
		}
	}

	return positions
}

// tryParseRegularPositions attempts to parse positions from regular content
func (sp *StreamParser) tryParseRegularPositions(content string) []SemanticPosition {
	// Extract JSON array from content
	jsonStr := sp.extractJSONArray(content)
	if jsonStr == "" {
		return nil
	}

	// Try to parse using TolerantJSONUnmarshal directly
	var positions []SemanticPosition
	if err := TolerantJSONUnmarshal([]byte(jsonStr), &positions); err != nil {
		return nil // Parsing failed, wait for more data
	}

	return positions
}

// extractJSONArray extracts JSON array from text content
func (sp *StreamParser) extractJSONArray(text string) string {
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

// toInt safely converts interface{} to int
func (sp *StreamParser) toInt(value interface{}) int {
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

// TolerantJSONUnmarshal unmarshals JSON with basic error handling and repair
func TolerantJSONUnmarshal(data []byte, v interface{}) error {
	// First try normal unmarshaling
	if err := json.Unmarshal(data, v); err == nil {
		return nil
	}

	// Try basic JSON repair for common issues
	jsonStr := string(data)

	// Remove trailing commas
	jsonStr = strings.ReplaceAll(jsonStr, ",}", "}")
	jsonStr = strings.ReplaceAll(jsonStr, ",]", "]")

	// Fix missing commas between object key-value pairs (simple case)
	// This is a basic fix for: {"key": "value" "key2": "value2"}
	jsonStr = strings.ReplaceAll(jsonStr, "\" \"", "\", \"")

	// Try unmarshaling the repaired JSON
	return json.Unmarshal([]byte(jsonStr), v)
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
	return `You are an expert text analyst. Your task is to segment text based on SEMANTIC BOUNDARIES, not fixed character counts.

CRITICAL INSTRUCTIONS:
1. NEVER create segments with regular intervals or fixed character counts
2. ALWAYS prioritize natural semantic boundaries (topic changes, paragraph breaks, concept shifts)
3. Each segment should represent a complete thought, topic, or logical unit
4. Segment sizes should VARY NATURALLY based on content structure
5. Look for natural breakpoints: paragraph endings, topic transitions, section breaks
6. Avoid splitting sentences, related concepts, or coherent thoughts
7. Small segments (50-200 chars) are acceptable for short complete thoughts
8. Large segments (500-1200 chars) are acceptable for complex topics that shouldn't be split

WRONG APPROACH (DO NOT DO THIS):
- Creating segments every 100-150 characters regardless of content
- Splitting in the middle of sentences or concepts
- Using regular intervals like 0-130, 130-260, 260-390

CORRECT APPROACH:
- Find natural topic boundaries and paragraph breaks
- Keep related sentences together
- Vary segment sizes based on content structure
- Example: [{"start_pos": 0, "end_pos": 89}, {"start_pos": 89, "end_pos": 234}, {"start_pos": 234, "end_pos": 567}, {"start_pos": 567, "end_pos": 723}]

Output format: JSON array with start_pos and end_pos positions only (no text content)
[
  {"start_pos": 0, "end_pos": <natural_boundary>},
  {"start_pos": <natural_boundary>, "end_pos": <next_boundary>}
]

Remember: SEMANTIC COHERENCE is more important than size uniformity. Segments should feel natural to a human reader.`
}

// tryParsePositionsConservative attempts to parse positions only when data looks very complete
func (sp *StreamParser) tryParsePositionsConservative() []SemanticPosition {
	data := strings.TrimSpace(sp.getAccumulatedData())
	if len(data) < 30 { // Need minimum data
		return nil
	}

	if sp.isToolcall {
		return sp.tryParseToolcallPositionsConservative(data)
	}
	return sp.tryParseRegularPositionsConservative(data)
}

// tryParseToolcallPositionsConservative attempts to parse toolcall positions conservatively
func (sp *StreamParser) tryParseToolcallPositionsConservative(arguments string) []SemanticPosition {
	// Must contain complete segments structure and look finished
	if !strings.Contains(arguments, "segments") || !strings.Contains(arguments, "start_pos") {
		return nil
	}

	// Try to intelligently complete the JSON
	completedJSON := sp.smartCompleteToolcallJSON(arguments)
	if completedJSON == "" {
		return nil
	}

	// Try to parse using TolerantJSONUnmarshal
	var args map[string]interface{}
	if err := TolerantJSONUnmarshal([]byte(completedJSON), &args); err != nil {
		return nil
	}

	// Extract segments
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

		startPos := sp.toInt(segMap["start_pos"])
		endPos := sp.toInt(segMap["end_pos"])

		if startPos >= 0 && endPos > startPos {
			positions = append(positions, SemanticPosition{
				StartPos: startPos,
				EndPos:   endPos,
			})
		}
	}

	return positions
}

// tryParseRegularPositionsConservative attempts to parse regular positions conservatively
func (sp *StreamParser) tryParseRegularPositionsConservative(content string) []SemanticPosition {
	// Extract JSON array from content
	jsonStr := sp.extractJSONArray(content)
	if jsonStr == "" {
		return nil
	}

	// Try to intelligently complete the JSON
	completedJSON := sp.smartCompleteRegularJSON(jsonStr)
	if completedJSON == "" {
		return nil
	}

	// Try to parse using TolerantJSONUnmarshal
	var positions []SemanticPosition
	if err := TolerantJSONUnmarshal([]byte(completedJSON), &positions); err != nil {
		return nil
	}

	return positions
}

// smartCompleteToolcallJSON intelligently completes toolcall JSON
func (sp *StreamParser) smartCompleteToolcallJSON(arguments string) string {
	arguments = strings.TrimSpace(arguments)

	// Must start with { and contain segments
	if !strings.HasPrefix(arguments, "{") || !strings.Contains(arguments, "segments") {
		return ""
	}

	// If already complete, return as-is
	if strings.HasSuffix(arguments, "}]}") {
		return arguments
	}

	// Don't complete if it ends with a comma (more data expected)
	if strings.HasSuffix(arguments, ",") {
		return ""
	}

	// Count braces and brackets
	openBraces := strings.Count(arguments, "{") - strings.Count(arguments, "}")
	openBrackets := strings.Count(arguments, "[") - strings.Count(arguments, "]")

	// Only complete if reasonable number of unclosed brackets/braces
	if openBraces > 5 || openBrackets > 5 {
		return ""
	}

	// Must contain at least one complete position object
	if !strings.Contains(arguments, "start_pos") || !strings.Contains(arguments, "end_pos") {
		return ""
	}

	// Don't complete if the last position object looks incomplete
	// Look for pattern like: "end_pos": 25} to ensure we have a complete object
	if !strings.Contains(arguments, "}") {
		return ""
	}

	result := arguments

	// Close brackets first
	for i := 0; i < openBrackets && i < 3; i++ {
		result += "]"
	}

	// Close braces
	for i := 0; i < openBraces && i < 3; i++ {
		result += "}"
	}

	return result
}

// smartCompleteRegularJSON intelligently completes regular JSON array
func (sp *StreamParser) smartCompleteRegularJSON(jsonStr string) string {
	jsonStr = strings.TrimSpace(jsonStr)

	// Must start with [ and contain position structure
	if !strings.HasPrefix(jsonStr, "[") {
		return ""
	}

	// If already complete, return as-is
	if strings.HasSuffix(jsonStr, "]") {
		return jsonStr
	}

	// Don't complete if it ends with a comma (more data expected)
	if strings.HasSuffix(jsonStr, ",") {
		return ""
	}

	// Must contain at least one position structure
	if !strings.Contains(jsonStr, "start_pos") || !strings.Contains(jsonStr, "end_pos") {
		return ""
	}

	// Don't complete if the last position object looks incomplete
	// Look for pattern like: "end_pos": 50} to ensure we have a complete object
	if !strings.Contains(jsonStr, "}") {
		return ""
	}

	// Count brackets and braces
	openBrackets := strings.Count(jsonStr, "[") - strings.Count(jsonStr, "]")
	openBraces := strings.Count(jsonStr, "{") - strings.Count(jsonStr, "}")

	// Only complete if reasonable number of unclosed brackets/braces
	if openBrackets > 3 || openBraces > 5 {
		return ""
	}

	result := jsonStr

	// Close braces first (for objects)
	for i := 0; i < openBraces && i < 3; i++ {
		result += "}"
	}

	// Close brackets (for array)
	for i := 0; i < openBrackets && i < 2; i++ {
		result += "]"
	}

	return result
}
