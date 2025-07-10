package converter

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/mcp"
)

// TestMCPConverter tests the MCP converter functionality
func TestMCPConverter(t *testing.T) {
	// Check if MCP_CLIENT_TEST_SSE_URL is set
	testURL := os.Getenv("MCP_CLIENT_TEST_SSE_URL")
	if testURL == "" {
		t.Skip("MCP_CLIENT_TEST_SSE_URL not set, skipping integration test")
	}

	// Get authorization token
	authToken := os.Getenv("MCP_CLIENT_TEST_SSE_AUTHORIZATION_TOKEN")
	if authToken == "" {
		authToken = "Bearer 123456" // Default token
	}

	// Create MCP client DSL
	clientDSL := fmt.Sprintf(`{
		"id": "test_client",
		"name": "Test MCP Client",
		"version": "1.0.0",
		"transport": "sse",
		"url": "%s",
		"authorization_token": "%s",
		"timeout": "30s",
		"enable_sampling": false,
		"enable_roots": false,
		"enable_elicitation": false
	}`, testURL, authToken)

	// Load client from source
	client, err := mcp.LoadClientSource(clientDSL, "test_client")
	if err != nil {
		t.Fatalf("Failed to load MCP client: %v", err)
	}

	// Connect to the client
	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to MCP client: %v", err)
	}
	defer client.Disconnect(ctx)

	// Initialize the client
	_, err = client.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize MCP client: %v", err)
	}

	// Create MCP converter options
	options := &MCPOptions{
		ID:   "test_client",
		Tool: "echo",
		ArgumentsMapping: map[string]string{
			"message": "{{data_uri}}",
		},
		ResultMapping: map[string]string{
			"text": "{{content.0.text}}",
			"foo":  "bar", // Static value that will be added to metadata
		},
		NotificationMapping: map[string]string{
			"status":   "pending",
			"message":  "{{notification.params.message}}",
			"progress": "{{notification.params.progress}}",
		},
	}

	converter, err := NewMCP(options)
	if err != nil {
		t.Fatalf("Failed to create MCP converter: %v", err)
	}

	// Test Convert method
	testFile := createTempFile(t, "Hello, World! This is a test.")
	defer os.Remove(testFile)

	// Track progress calls
	progressCalls := []types.ConverterPayload{}
	progressCallback := func(status types.ConverterStatus, payload types.ConverterPayload) {
		progressCalls = append(progressCalls, payload)
		t.Logf("Progress: %s - %s (%.2f%%)", status, payload.Message, payload.Progress*100)
	}

	result, err := converter.Convert(ctx, testFile, progressCallback)
	if err != nil {
		t.Fatalf("Failed to convert: %v", err)
	}

	// Validate result
	if result == nil {
		t.Fatal("Converter result is nil")
	}

	if result.Text == "" {
		t.Error("Converter result text is empty")
	}

	if result.Metadata == nil {
		t.Error("Converter result metadata is nil")
	}

	t.Logf("Convert result text: '%s'", result.Text)
	t.Logf("Convert result metadata: %+v", result.Metadata)

	// Validate progress calls
	if len(progressCalls) == 0 {
		t.Error("No progress callbacks were called")
	}

	// Test ConvertStream method
	testFile2 := createTempFile(t, "Stream test content")
	defer os.Remove(testFile2)

	file, err := os.Open(testFile2)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer file.Close()

	progressCalls2 := []types.ConverterPayload{}
	progressCallback2 := func(status types.ConverterStatus, payload types.ConverterPayload) {
		progressCalls2 = append(progressCalls2, payload)
		t.Logf("Stream Progress: %s - %s (%.2f%%)", status, payload.Message, payload.Progress*100)
	}

	result2, err := converter.ConvertStream(ctx, file, progressCallback2)
	if err != nil {
		t.Fatalf("Failed to convert stream: %v", err)
	}

	// Validate stream result
	if result2 == nil {
		t.Fatal("Stream converter result is nil")
	}

	if result2.Text == "" {
		t.Error("Stream converter result text is empty")
	}

	if result2.Metadata == nil {
		t.Error("Stream converter result metadata is nil")
	}

	t.Logf("Stream convert result text: %s", result2.Text)
	t.Logf("Stream convert result metadata: %+v", result2.Metadata)

	// Validate progress calls for stream
	if len(progressCalls2) == 0 {
		t.Error("No progress callbacks were called for stream")
	}
}

// TestMCPConverterArgumentsMapping tests the arguments mapping functionality
func TestMCPConverterArgumentsMapping(t *testing.T) {
	// Test getArguments method
	options := &MCPOptions{
		ArgumentsMapping: map[string]string{
			"message": "{{data_uri}}",
			"static":  "test_value",
		},
	}

	converter := &MCP{
		ArgumentsMapping: options.ArgumentsMapping,
	}

	testDataURI := "data:text/plain;base64,SGVsbG8gV29ybGQ="
	arguments, err := converter.getArguments(testDataURI)
	if err != nil {
		t.Fatalf("Failed to get arguments: %v", err)
	}

	if arguments["message"] != testDataURI {
		t.Errorf("Expected message to be %s, got %s", testDataURI, arguments["message"])
	}

	if arguments["static"] != "test_value" {
		t.Errorf("Expected static to be 'test_value', got %s", arguments["static"])
	}

	t.Logf("Arguments mapping result: %+v", arguments)
}

// TestMCPConverterResultMapping tests the result mapping functionality
func TestMCPConverterResultMapping(t *testing.T) {
	// Test getResult method
	options := &MCPOptions{
		ResultMapping: map[string]string{
			"text": "{{result}}",
			"foo":  "bar", // Static value that will be added to metadata
		},
	}

	converter := &MCP{
		ResultMapping: options.ResultMapping,
	}

	testResult := "Hello World"

	result, err := converter.getResult(testResult)
	if err != nil {
		t.Fatalf("Failed to get result: %v", err)
	}

	if result.Text != "Hello World" {
		t.Errorf("Expected text to be 'Hello World', got %s", result.Text)
	}

	if result.Metadata == nil {
		t.Fatal("Expected metadata to be set")
	}

	if result.Metadata["foo"] != "bar" {
		t.Errorf("Expected metadata['foo'] to be 'bar', got %v", result.Metadata["foo"])
	}

	t.Logf("Result mapping result: Text='%s', Metadata=%+v", result.Text, result.Metadata)
}

// TestMCPConverterDataURI tests the data URI creation functionality
func TestMCPConverterDataURI(t *testing.T) {
	converter := &MCP{}

	// Test with text content
	testContent := []byte("Hello, World!")
	dataURI, err := converter.createDataURI("test.txt", testContent)
	if err != nil {
		t.Fatalf("Failed to create data URI: %v", err)
	}

	if !strings.HasPrefix(dataURI, "data:") {
		t.Errorf("Expected data URI to start with 'data:', got %s", dataURI)
	}

	if !strings.Contains(dataURI, "text/plain") {
		t.Errorf("Expected data URI to contain 'text/plain', got %s", dataURI)
	}

	if !strings.Contains(dataURI, "base64,") {
		t.Errorf("Expected data URI to contain 'base64,', got %s", dataURI)
	}

	t.Logf("Data URI: %s", dataURI)

	// Test with binary content
	binaryContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	dataURI2, err := converter.createDataURI("test.png", binaryContent)
	if err != nil {
		t.Fatalf("Failed to create data URI for binary content: %v", err)
	}

	if !strings.Contains(dataURI2, "image/png") {
		t.Errorf("Expected data URI to contain 'image/png', got %s", dataURI2)
	}

	t.Logf("Binary Data URI: %s", dataURI2)
}

// Helper function to create a temporary file with content
func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "mcp_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
}
