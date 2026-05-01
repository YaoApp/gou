package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/gou/llm"
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
	host, apiKey, authMode := connectorCredentials(conn)
	if host == "" {
		return nil, fmt.Errorf("no host found in connector settings")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key is not set")
	}

	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	url := connector.BuildAPIURL(host, endpoint)

	r := http.New(url)
	setAuthHeader(r, authMode, apiKey)
	r.SetHeader("Content-Type", "application/json")
	r.WithContext(ctx)

	resp := r.Post(payload)
	if resp.Status != 200 {
		return nil, fmt.Errorf("request failed with status: %d, data: %v", resp.Status, resp.Data)
	}

	return resp.Data, nil
}

// PostLLMFile sends a POST request to LLM API with file upload using multipart/form-data
func PostLLMFile(ctx context.Context, conn connector.Connector, endpoint string, filePath string, formData map[string]interface{}) (interface{}, error) {
	host, apiKey, authMode := connectorCredentials(conn)
	if host == "" {
		return nil, fmt.Errorf("no host found in connector settings")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key is not set")
	}

	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	url := connector.BuildAPIURL(host, endpoint)

	r := http.New(url)
	setAuthHeader(r, authMode, apiKey)
	r.SetHeader("Content-Type", "multipart/form-data")
	r.WithContext(ctx)

	r.AddFile("file", filePath)

	resp := r.Post(formData)
	if resp.Status != 200 {
		return nil, fmt.Errorf("request failed with status: %d, data: %v", resp.Status, resp.Data)
	}

	return resp.Data, nil
}

// StreamLLM sends a streaming request to LLM service with real-time callback
func StreamLLM(ctx context.Context, conn connector.Connector, endpoint string, payload map[string]interface{}, callback func(data []byte) error) error {
	host, key, authMode := connectorCredentials(conn)
	if host == "" {
		return fmt.Errorf("no host found in connector settings")
	}

	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}

	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	url := connector.BuildAPIURL(host, endpoint)

	if key == "" {
		return fmt.Errorf("API key is not set")
	}

	if payload == nil {
		payload = make(map[string]interface{})
	}

	streamPayload := make(map[string]interface{})
	for k, v := range payload {
		streamPayload[k] = v
	}
	streamPayload["stream"] = true

	req := http.New(url).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "text/event-stream")
	setAuthHeader(req, authMode, key)

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

// connectorCredentials extracts host, key, and auth mode from a connector,
// preferring the typed LLMConnector interface over the Setting() map.
func connectorCredentials(conn connector.Connector) (host, key string, authMode llm.AuthMode) {
	authMode = llm.AuthBearer
	if lc, ok := conn.(llm.LLMConnector); ok {
		host = lc.GetURL()
		key = lc.GetKey()
		authMode = lc.GetAuthMode()
	}
	if host == "" || key == "" {
		setting := conn.Setting()
		if host == "" {
			host, _ = setting["host"].(string)
		}
		if key == "" {
			key, _ = setting["key"].(string)
		}
	}
	return
}

// setAuthHeader sets the appropriate authentication header based on AuthMode.
func setAuthHeader(r *http.Request, authMode llm.AuthMode, key string) {
	switch authMode {
	case llm.AuthAPIKey:
		r.SetHeader("api-key", key)
	case llm.AuthXAPIKey:
		r.SetHeader("x-api-key", key)
	default:
		r.SetHeader("Authorization", fmt.Sprintf("Bearer %s", key))
	}
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
