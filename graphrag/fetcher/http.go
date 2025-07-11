package fetcher

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/http"
)

// HTTPOptions contains options for HTTP fetcher
type HTTPOptions struct {
	Headers   map[string]string `json:"headers,omitempty"`    // Custom headers to add to requests
	UserAgent string            `json:"user_agent,omitempty"` // Custom User-Agent header
	Timeout   time.Duration     `json:"timeout,omitempty"`    // Request timeout (default: 300s)
}

// HTTPFetcher implements the Fetcher interface for HTTP/HTTPS URLs
type HTTPFetcher struct {
	options *HTTPOptions
}

// NewHTTPFetcher creates a new HTTP fetcher instance
func NewHTTPFetcher(options ...*HTTPOptions) *HTTPFetcher {
	var opts *HTTPOptions
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	} else {
		opts = &HTTPOptions{}
	}

	// Set default values
	if opts.UserAgent == "" {
		opts.UserAgent = "GraphRAG-Fetcher/1.0"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 300 * time.Second // Default 5 minutes
	}

	return &HTTPFetcher{
		options: opts,
	}
}

// Fetch implements the Fetcher interface
func (f *HTTPFetcher) Fetch(ctx context.Context, url string, callback ...types.FetcherProgress) (string, string, error) {
	// Report start
	f.reportProgress(types.FetcherStatusPending, "Starting URL fetch", 0.0, url, 0, callback...)

	// Create HTTP request using gou/http package
	req := http.New(url)
	req.WithContext(ctx)

	// Set User-Agent
	req.SetHeader("User-Agent", f.options.UserAgent)

	// Add custom headers
	if f.options.Headers != nil {
		for key, value := range f.options.Headers {
			req.SetHeader(key, value)
		}
	}

	// Report progress
	f.reportProgress(types.FetcherStatusPending, "Sending HTTP request", 0.1, url, 0, callback...)

	// Send GET request
	resp := req.Get()
	if resp.Status != 200 {
		f.reportProgress(types.FetcherStatusError, fmt.Sprintf("HTTP %d: %s", resp.Status, resp.Message), 0.0, url, 0, callback...)
		return "", "", fmt.Errorf("HTTP request failed with status: %d %s", resp.Status, resp.Message)
	}

	// Get content type from response headers
	contentType := resp.Headers.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream" // Default
	}

	// Extract MIME type (remove charset and other parameters)
	mimeType := strings.Split(contentType, ";")[0]
	mimeType = strings.TrimSpace(mimeType)

	// Report progress
	f.reportProgress(types.FetcherStatusPending, "Reading response body", 0.5, url, 0, callback...)

	// Get response body
	var body []byte
	if resp.Data != nil {
		switch data := resp.Data.(type) {
		case []byte:
			body = data
		case string:
			body = []byte(data)
		default:
			// For other types, try to convert to string first
			body = []byte(fmt.Sprintf("%v", data))
		}
	}

	// Report progress
	f.reportProgress(types.FetcherStatusPending, "Processing content", 0.8, url, int64(len(body)), callback...)

	// Determine if content should be base64 encoded
	var content string
	if f.isBinaryContent(mimeType, body) {
		// Binary content - encode as base64
		content = base64.StdEncoding.EncodeToString(body)
	} else {
		// Text content - convert to string
		content = string(body)
	}

	// Report success
	f.reportProgress(types.FetcherStatusSuccess, "URL fetch completed", 1.0, url, int64(len(body)), callback...)

	return content, mimeType, nil
}

// isBinaryContent determines if content should be treated as binary
func (f *HTTPFetcher) isBinaryContent(mimeType string, body []byte) bool {
	// Text-based MIME types
	textTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/xhtml+xml",
		"application/rss+xml",
		"application/atom+xml",
	}

	// Check if it's a known text type
	for _, textType := range textTypes {
		if strings.HasPrefix(mimeType, textType) {
			return false
		}
	}

	// Check if content is valid UTF-8 text (simple heuristic)
	if len(body) > 0 {
		// Count null bytes and control characters
		nullBytes := 0
		maxCheck := len(body)
		if maxCheck > 512 {
			maxCheck = 512 // Check first 512 bytes
		}
		for _, b := range body[:maxCheck] {
			if b == 0 {
				nullBytes++
			}
		}
		// If more than 1% null bytes, likely binary
		if nullBytes > len(body)/100 {
			return true
		}
	}

	// Binary MIME types
	binaryTypes := []string{
		"image/", "video/", "audio/", "application/pdf",
		"application/msword", "application/vnd.", "application/zip",
		"application/octet-stream", "application/x-binary",
	}

	for _, binaryType := range binaryTypes {
		if strings.HasPrefix(mimeType, binaryType) {
			return true
		}
	}

	// Default to text for unknown types
	return false
}

// reportProgress reports progress to callbacks
func (f *HTTPFetcher) reportProgress(status types.FetcherStatus, message string, progress float64, url string, bytes int64, callbacks ...types.FetcherProgress) {
	if len(callbacks) == 0 {
		return
	}

	payload := types.FetcherPayload{
		Status:   status,
		Message:  message,
		Progress: progress,
		URL:      url,
		Bytes:    bytes,
	}

	for _, callback := range callbacks {
		if callback != nil {
			callback(status, payload)
		}
	}
}
