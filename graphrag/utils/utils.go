package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

	// Clean and normalize endpoint
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}

	// Ensure endpoint starts with /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	// Special handling for OpenAI API
	if host == "https://api.openai.com" && !strings.HasPrefix(endpoint, "/v1") {
		endpoint = "/v1" + endpoint
	}

	// Build full URL - fix the double slash issue
	host = strings.TrimSuffix(host, "/")
	url := host + endpoint

	// Get API key
	key, ok := setting["key"].(string)
	if !ok || key == "" {
		return fmt.Errorf("API key is not set")
	}

	// Validate payload
	if payload == nil {
		payload = make(map[string]interface{})
	}

	// Add stream parameter to payload
	streamPayload := make(map[string]interface{})
	for k, v := range payload {
		streamPayload[k] = v
	}
	streamPayload["stream"] = true

	// Create HTTP request with proper headers
	req := http.New(url).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", key)).
		SetHeader("Accept", "text/event-stream")

	// Stream handler function with better error handling
	streamHandler := func(data []byte) int {
		// Skip empty data
		if len(data) == 0 {
			return http.HandlerReturnOk
		}

		// Call user callback with error handling
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
