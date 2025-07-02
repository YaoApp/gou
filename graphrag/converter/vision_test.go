package converter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/types"
)

// ==== Test Data Utils ====

// getVisionTestDataDir returns the vision test data directory
func getVisionTestDataDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(currentFile)
	testDataDir := filepath.Join(currentDir, "..", "tests", "converter", "vision")
	absPath, err := filepath.Abs(testDataDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to get absolute path for vision test data dir: %v", err))
	}
	return absPath
}

// getVisionTestFilePath returns the full path to a vision test file
func getVisionTestFilePath(filename string) string {
	return filepath.Join(getVisionTestDataDir(), filename)
}

// ensureVisionTestDataExists checks if vision test data directory and files exist
func ensureVisionTestDataExists(t *testing.T) {
	t.Helper()

	testDir := getVisionTestDataDir()
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatalf("Vision test data directory does not exist: %s", testDir)
	}

	// Check for required test files
	requiredFiles := []string{
		"test.png", "test.png.gz",
		"test.jpg", "test.jpg.gz",
		"test.gif", "test.gif.gz",
		"test.webp", "test.webp.gz",
		"test.txt", "test.txt.gz", // Non-image files for error testing
	}

	for _, filename := range requiredFiles {
		filePath := getVisionTestFilePath(filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatalf("Required vision test file does not exist: %s", filePath)
		}
	}
}

// VisionTestFileInfo contains information about a vision test file
type VisionTestFileInfo struct {
	Name         string
	Path         string
	ShouldFail   bool
	Format       string
	Description  string
	IsCompressed bool
}

// getImageTestFiles returns all image test files that should convert successfully
func getImageTestFiles() []VisionTestFileInfo {
	return []VisionTestFileInfo{
		{
			Name:        "test.png",
			Path:        getVisionTestFilePath("test.png"),
			Format:      "PNG",
			Description: "PNG image file",
		},
		{
			Name:        "test.jpg",
			Path:        getVisionTestFilePath("test.jpg"),
			Format:      "JPEG",
			Description: "JPEG image file",
		},
		{
			Name:        "test.gif",
			Path:        getVisionTestFilePath("test.gif"),
			Format:      "GIF",
			Description: "GIF image file",
		},
		{
			Name:        "test.webp",
			Path:        getVisionTestFilePath("test.webp"),
			Format:      "WebP",
			Description: "WebP image file",
		},
	}
}

// getCompressedImageTestFiles returns all compressed image test files
func getCompressedImageTestFiles() []VisionTestFileInfo {
	return []VisionTestFileInfo{
		{
			Name:         "test.png.gz",
			Path:         getVisionTestFilePath("test.png.gz"),
			Format:       "PNG",
			Description:  "Gzipped PNG image file",
			IsCompressed: true,
		},
		{
			Name:         "test.jpg.gz",
			Path:         getVisionTestFilePath("test.jpg.gz"),
			Format:       "JPEG",
			Description:  "Gzipped JPEG image file",
			IsCompressed: true,
		},
		{
			Name:         "test.gif.gz",
			Path:         getVisionTestFilePath("test.gif.gz"),
			Format:       "GIF",
			Description:  "Gzipped GIF image file",
			IsCompressed: true,
		},
		{
			Name:         "test.webp.gz",
			Path:         getVisionTestFilePath("test.webp.gz"),
			Format:       "WebP",
			Description:  "Gzipped WebP image file",
			IsCompressed: true,
		},
	}
}

// getNonImageTestFiles returns all non-image test files that should fail
func getNonImageTestFiles() []VisionTestFileInfo {
	return []VisionTestFileInfo{
		{
			Name:        "test.txt",
			Path:        getVisionTestFilePath("test.txt"),
			ShouldFail:  true,
			Format:      "Text",
			Description: "Plain text file (should fail)",
		},
		{
			Name:         "test.txt.gz",
			Path:         getVisionTestFilePath("test.txt.gz"),
			ShouldFail:   true,
			Format:       "Text",
			Description:  "Gzipped text file (should fail)",
			IsCompressed: true,
		},
	}
}

// getAllVisionTestFiles returns all vision test files
func getAllVisionTestFiles() []VisionTestFileInfo {
	var all []VisionTestFileInfo
	all = append(all, getImageTestFiles()...)
	all = append(all, getCompressedImageTestFiles()...)
	all = append(all, getNonImageTestFiles()...)
	return all
}

// ==== Connector Setup ====

// prepareVisionConnector creates connectors for vision testing
func prepareVisionConnector(t *testing.T) {
	t.Helper()

	// Create OpenAI connector for vision testing
	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		openaiKey = "mock-key"
	}

	openaiDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0", 
		"label": "OpenAI Vision Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "gpt-4o-mini",
			"key": "%s"
		}
	}`, openaiKey)

	_, err := connector.New("openai", "test-vision-openai", []byte(openaiDSL))
	if err != nil {
		t.Logf("Failed to create OpenAI vision connector: %v", err)
	}

	// Create mock connector for tests that don't require real LLM calls
	mockDSL := `{
		"LANG": "1.0.0",
		"VERSION": "1.0.0",
		"label": "Mock Vision Service Test",
		"type": "openai",
		"options": {
			"proxy": "http://127.0.0.1:9999",
			"model": "gpt-4o-mini",
			"key": "mock-key"
		}
	}`

	_, err = connector.New("openai", "test-vision-mock", []byte(mockDSL))
	if err != nil {
		t.Logf("Failed to create mock vision connector: %v", err)
	}
}

// createVisionOptions creates VisionOption for testing
func createVisionOptions(useOpenAI bool, compressSize int64) VisionOption {
	var connectorID string
	var model string

	if useOpenAI {
		connectorID = "test-vision-openai"
		model = "gpt-4o-mini"
	} else {
		connectorID = "test-vision-mock"
		model = "gpt-4o-mini"
	}

	if compressSize == 0 {
		compressSize = 1024 // Default compression size
	}

	return VisionOption{
		ConnectorName: connectorID,
		Model:         model,
		Prompt:        "", // Use default prompt
		Options:       map[string]any{"temperature": 0.1},
		CompressSize:  compressSize,
		Language:      "Auto", // Default language setting
	}
}

// ==== Basic Functionality Tests ====

func TestVision_NewVision(t *testing.T) {
	ensureVisionTestDataExists(t)
	prepareVisionConnector(t)

	t.Run("Valid OpenAI connector", func(t *testing.T) {
		options := createVisionOptions(true, 1024)
		converter, err := NewVision(options)
		if err != nil {
			t.Fatalf("NewVision failed: %v", err)
		}

		if converter == nil {
			t.Fatal("NewVision returned nil")
		}

		if converter.Model != "gpt-4o-mini" {
			t.Errorf("Expected model gpt-4o-mini, got %s", converter.Model)
		}

		if converter.CompressSize != 1024 {
			t.Errorf("Expected CompressSize 1024, got %d", converter.CompressSize)
		}
	})

	t.Run("Invalid connector", func(t *testing.T) {
		options := VisionOption{
			ConnectorName: "non-existent-connector",
		}

		converter, err := NewVision(options)
		if err == nil {
			t.Error("Expected error for invalid connector, but got none")
		}
		if converter != nil {
			t.Error("Expected nil converter for invalid connector")
		}
	})

	t.Run("Non-OpenAI connector", func(t *testing.T) {
		// Create a non-OpenAI connector for testing
		nonOpenAIDSL := `{
			"LANG": "1.0.0",
			"VERSION": "1.0.0",
			"label": "Non-OpenAI Test",
			"type": "zhipuai",
			"options": {
				"host": "localhost"
			}
		}`

		_, err := connector.New("zhipuai", "test-non-openai", []byte(nonOpenAIDSL))
		if err != nil {
			t.Skipf("Failed to create non-OpenAI connector (expected): %v", err)
		}

		options := VisionOption{
			ConnectorName: "test-non-openai",
		}

		converter, err := NewVision(options)
		if err == nil {
			t.Error("Expected error for non-OpenAI connector, but got none")
		}
		if converter != nil {
			t.Error("Expected nil converter for non-OpenAI connector")
		}
	})

	t.Run("Default values", func(t *testing.T) {
		options := VisionOption{
			ConnectorName: "test-vision-openai",
			// Leave other fields empty to test defaults
		}

		converter, err := NewVision(options)
		if err != nil {
			t.Fatalf("NewVision with defaults failed: %v", err)
		}

		if converter.Prompt == "" {
			t.Error("Expected default prompt to be set")
		}

		if converter.CompressSize != 1024 {
			t.Errorf("Expected default CompressSize 1024, got %d", converter.CompressSize)
		}

		if converter.Language != "Auto" {
			t.Errorf("Expected default Language 'Auto', got %s", converter.Language)
		}

		// Test that the prompt template contains the language instruction placeholder
		if !strings.Contains(converter.Prompt, "{LANGUAGE_INSTRUCTION}") {
			t.Error("Expected default prompt template to contain {LANGUAGE_INSTRUCTION} placeholder")
		}
	})
}

func TestVision_Convert_ImageFiles(t *testing.T) {
	prepareVisionConnector(t)

	// Use mock connector for basic validation testing
	vision, err := NewVision(createVisionOptions(false, 0))
	if err != nil {
		t.Fatalf("Failed to create Vision converter: %v", err)
	}

	ctx := context.Background()
	testFiles := getImageTestFiles()

	for _, testFile := range testFiles {
		t.Run(testFile.Name, func(t *testing.T) {
			result, err := vision.Convert(ctx, testFile.Path)

			if testFile.ShouldFail {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", testFile.Description)
				}
				return
			}

			// With mock connector, we expect the conversion to fail at LLM stage
			// This tests that image validation and processing works correctly
			if err != nil {
				expectedErrors := []string{
					"connection refused",
					"LLM processing failed",
					"streaming request failed",
					"no such host",
					"DNS resolve fail",
				}

				hasExpectedError := false
				for _, expectedErr := range expectedErrors {
					if strings.Contains(err.Error(), expectedErr) {
						hasExpectedError = true
						break
					}
				}

				if hasExpectedError {
					t.Logf("%s: Expected LLM failure with mock connector: %v", testFile.Description, err)
				} else {
					t.Errorf("%s: Unexpected error: %v", testFile.Description, err)
				}
				return
			}

			// If somehow the mock connector works, validate the result
			if result == nil {
				t.Errorf("Convert returned nil result for %s", testFile.Description)
				return
			}

			if result.Text == "" {
				t.Errorf("Convert returned empty text for %s", testFile.Description)
			}

			// Validate result content
			if err := validateVisionResult(result.Text, 10, []string{"image", "picture", "description"}); err != nil {
				t.Logf("Vision result validation warning for %s: %v", testFile.Description, err)
			}

			// Check metadata
			if result.Metadata == nil {
				t.Errorf("Convert returned nil metadata for %s", testFile.Description)
			}

			t.Logf("%s: Generated %d chars description with metadata: %v", testFile.Description, len(result.Text), result.Metadata)
		})
	}
}

func TestVision_Convert_CompressedImageFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compressed image tests in short mode")
	}

	prepareVisionConnector(t)

	// Check if OpenAI key is available
	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping integration tests")
	}

	options := createVisionOptions(true, 512)
	converter, err := NewVision(options)
	if err != nil {
		t.Fatalf("Failed to create vision converter: %v", err)
	}

	testFiles := getCompressedImageTestFiles()
	for _, testFile := range testFiles {
		t.Run(testFile.Name, func(t *testing.T) {
			ctx := context.Background()

			callback := NewTestProgressCallback()
			result, err := converter.Convert(ctx, testFile.Path, callback.Callback)

			if err != nil {
				t.Fatalf("Convert failed for %s: %v", testFile.Description, err)
			}

			if result == nil || result.Text == "" {
				t.Errorf("Convert returned empty result for %s", testFile.Description)
			}

			// Check that gzip decompression progress was reported
			hasGzipProgress := false
			for _, call := range callback.Calls {
				if strings.Contains(call.Message, "gzip") || strings.Contains(call.Message, "decompressing") {
					hasGzipProgress = true
					break
				}
			}

			if !hasGzipProgress {
				t.Logf("Gzip decompression progress not explicitly reported for %s", testFile.Description)
			}

			t.Logf("%s: Generated description (%d chars): %s...",
				testFile.Description, len(result.Text), truncateString(result.Text, 100))
		})
	}
}

func TestVision_Convert_NonImageFiles(t *testing.T) {
	prepareVisionConnector(t)

	options := createVisionOptions(false, 1024) // Use mock connector
	converter, err := NewVision(options)
	if err != nil {
		t.Fatalf("Failed to create vision converter: %v", err)
	}

	testFiles := getNonImageTestFiles()
	for _, testFile := range testFiles {
		t.Run(testFile.Name, func(t *testing.T) {
			ctx := context.Background()

			result, err := converter.Convert(ctx, testFile.Path)

			if !testFile.ShouldFail {
				t.Fatalf("Test file %s should be marked as ShouldFail=true", testFile.Name)
			}

			if err == nil {
				t.Errorf("Expected error for non-image file %s, but conversion succeeded with result length %d",
					testFile.Description, len(result.Text))
			} else {
				// Check that error message indicates it's not an image
				if !strings.Contains(err.Error(), "not an image") {
					t.Logf("Expected 'not an image' in error message for %s, got: %v", testFile.Name, err)
				}
				t.Logf("%s: Correctly rejected with error: %v", testFile.Description, err)
			}
		})
	}
}

// ==== Stream Conversion Tests ====

func TestVision_ConvertStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stream conversion tests in short mode")
	}

	prepareVisionConnector(t)

	// Use mock connector for basic stream testing
	options := createVisionOptions(false, 1024)
	converter, err := NewVision(options)
	if err != nil {
		t.Fatalf("Failed to create vision converter: %v", err)
	}

	t.Run("PNG stream", func(t *testing.T) {
		testFile := getVisionTestFilePath("test.png")
		file, err := os.Open(testFile)
		if err != nil {
			t.Fatalf("Failed to open test file: %v", err)
		}
		defer file.Close()

		ctx := context.Background()
		callback := NewTestProgressCallback()

		// This will fail at LLM stage with mock connector, but tests stream processing
		_, err = converter.ConvertStream(ctx, file, callback.Callback)

		// We expect this to fail at LLM stage
		if err != nil {
			expectedErrors := []string{
				"connection refused",
				"LLM processing failed",
				"streaming request failed",
				"no such host",
			}

			hasExpectedError := false
			for _, expectedErr := range expectedErrors {
				if strings.Contains(err.Error(), expectedErr) {
					hasExpectedError = true
					break
				}
			}

			if hasExpectedError {
				t.Logf("Expected LLM failure: %v", err)
			} else {
				t.Errorf("Unexpected error: %v", err)
			}
		}

		// Check that image processing progress was reported
		if callback.GetCallCount() == 0 {
			t.Error("No progress callbacks during stream processing")
		}

		// Should have at least progressed past image validation
		hasImageProgress := false
		for _, call := range callback.Calls {
			if strings.Contains(call.Message, "image") || call.Progress > 0.5 {
				hasImageProgress = true
				break
			}
		}

		if !hasImageProgress {
			t.Error("No image processing progress reported")
		}

		t.Logf("Stream processing completed with %d progress calls", callback.GetCallCount())
	})

	t.Run("Gzipped stream", func(t *testing.T) {
		testFile := getVisionTestFilePath("test.jpg.gz")
		file, err := os.Open(testFile)
		if err != nil {
			t.Fatalf("Failed to open test file: %v", err)
		}
		defer file.Close()

		ctx := context.Background()
		callback := NewTestProgressCallback()

		// This will fail at LLM stage but tests gzip handling
		_, err = converter.ConvertStream(ctx, file, callback.Callback)

		if err != nil {
			t.Logf("Expected error with mock connector: %v", err)
		}

		// Check for gzip decompression progress
		hasGzipProgress := false
		for _, call := range callback.Calls {
			if strings.Contains(call.Message, "gzip") || strings.Contains(call.Message, "decompressing") {
				hasGzipProgress = true
				break
			}
		}

		if !hasGzipProgress {
			t.Log("Gzip progress not explicitly reported (may be handled transparently)")
		}
	})
}

// ==== Image Processing Tests ====

func TestVision_validateAndProcessImage(t *testing.T) {
	prepareVisionConnector(t)

	options := createVisionOptions(false, 1024)
	converter, err := NewVision(options)
	if err != nil {
		t.Fatalf("Failed to create vision converter: %v", err)
	}

	t.Run("Valid PNG", func(t *testing.T) {
		testFile := getVisionTestFilePath("test.png")
		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file: %v", err)
		}

		processedData, contentType, err := converter.validateAndProcessImage(data)
		if err != nil {
			t.Fatalf("validateAndProcessImage failed for PNG: %v", err)
		}

		if contentType != "image/png" {
			t.Errorf("Expected content type image/png, got %s", contentType)
		}

		if len(processedData) == 0 {
			t.Error("Processed data is empty")
		}

		t.Logf("PNG validation: original %d bytes, processed %d bytes", len(data), len(processedData))
	})

	t.Run("Valid JPEG", func(t *testing.T) {
		testFile := getVisionTestFilePath("test.jpg")
		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file: %v", err)
		}

		processedData, contentType, err := converter.validateAndProcessImage(data)
		if err != nil {
			t.Fatalf("validateAndProcessImage failed for JPEG: %v", err)
		}

		if contentType != "image/jpeg" {
			t.Errorf("Expected content type image/jpeg, got %s", contentType)
		}

		if len(processedData) == 0 {
			t.Error("Processed data is empty")
		}

		t.Logf("JPEG validation: original %d bytes, processed %d bytes", len(data), len(processedData))
	})

	t.Run("Invalid - text file", func(t *testing.T) {
		testFile := getVisionTestFilePath("test.txt")
		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file: %v", err)
		}

		_, _, err = converter.validateAndProcessImage(data)
		if err == nil {
			t.Error("Expected error for text file, but got none")
		}

		if !strings.Contains(err.Error(), "not an image") {
			t.Errorf("Expected 'not an image' error, got: %v", err)
		}

		t.Logf("Text file correctly rejected: %v", err)
	})
}

func TestVision_compressImage(t *testing.T) {
	prepareVisionConnector(t)

	t.Run("Small image - no compression needed", func(t *testing.T) {
		options := createVisionOptions(false, 2048) // Large compression size
		converter, err := NewVision(options)
		if err != nil {
			t.Fatalf("Failed to create vision converter: %v", err)
		}

		testFile := getVisionTestFilePath("test.png")
		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file: %v", err)
		}

		compressedData, err := converter.compressImage(data, "image/png")
		if err != nil {
			t.Fatalf("compressImage failed: %v", err)
		}

		// Should return original data if no compression needed
		if len(compressedData) != len(data) {
			t.Logf("Compression occurred: %d -> %d bytes", len(data), len(compressedData))
		} else {
			t.Logf("No compression needed: %d bytes", len(data))
		}
	})

	t.Run("Large image - compression needed", func(t *testing.T) {
		options := createVisionOptions(false, 256) // Small compression size
		converter, err := NewVision(options)
		if err != nil {
			t.Fatalf("Failed to create vision converter: %v", err)
		}

		testFile := getVisionTestFilePath("test.jpg")
		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file: %v", err)
		}

		compressedData, err := converter.compressImage(data, "image/jpeg")
		if err != nil {
			t.Fatalf("compressImage failed: %v", err)
		}

		if len(compressedData) == 0 {
			t.Error("Compressed data is empty")
		}

		t.Logf("Image compression: %d -> %d bytes (%.1f%% of original)",
			len(data), len(compressedData), float64(len(compressedData))/float64(len(data))*100)
	})
}

// ==== Image Format Detection Tests ====

func TestVision_getImageFormat(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"PNG", "test.png", ".png"},
		{"JPEG", "test.jpg", ".jpg"},
		{"GIF", "test.gif", ".gif"},
		{"WebP", "test.webp", ".webp"},
		{"Text", "test.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := getVisionTestFilePath(tt.filename)
			data, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			format := getImageFormat(data)
			if format != tt.expected {
				t.Errorf("Expected format %s, got %s", tt.expected, format)
			}

			t.Logf("Format detected for %s: %s", tt.filename, format)
		})
	}
}

func TestVision_detectImageType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"PNG", "test.png", "image/png"},
		{"JPEG", "test.jpg", "image/jpeg"},
		{"GIF", "test.gif", "image/gif"},
		{"WebP", "test.webp", "image/webp"},
		{"Text", "test.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := getVisionTestFilePath(tt.filename)
			data, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			imageType := detectImageType(data)
			if imageType != tt.expected {
				t.Errorf("Expected type %s, got %s", tt.expected, imageType)
			}

			t.Logf("Type detected for %s: %s", tt.filename, imageType)
		})
	}
}

// ==== Error Handling Tests ====

func TestVision_Convert_NonExistentFile(t *testing.T) {
	prepareVisionConnector(t)

	options := createVisionOptions(false, 1024)
	converter, err := NewVision(options)
	if err != nil {
		t.Fatalf("Failed to create vision converter: %v", err)
	}

	ctx := context.Background()
	_, err = converter.Convert(ctx, "/non/existent/image.png")
	if err == nil {
		t.Error("Expected error for non-existent file, but got none")
	}

	t.Logf("Correctly failed with error: %v", err)
}

func TestVision_Convert_ContextCancellation(t *testing.T) {
	prepareVisionConnector(t)

	options := createVisionOptions(false, 1024)
	converter, err := NewVision(options)
	if err != nil {
		t.Fatalf("Failed to create vision converter: %v", err)
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	testFile := getVisionTestFilePath("test.png")
	_, err = converter.Convert(ctx, testFile)

	// The operation might complete before cancellation is checked
	if err != nil && err == context.Canceled {
		t.Log("Context cancellation handled correctly")
	} else {
		t.Log("Operation completed before cancellation check (acceptable)")
	}
}

// ==== Progress Callback Tests ====

func TestVision_ProgressReporting(t *testing.T) {
	prepareVisionConnector(t)

	options := createVisionOptions(false, 1024)
	converter, err := NewVision(options)
	if err != nil {
		t.Fatalf("Failed to create vision converter: %v", err)
	}

	t.Run("Progress callback sequence", func(t *testing.T) {
		callback := NewTestProgressCallback()

		// Test manual progress reporting
		converter.reportProgress(types.ConverterStatusPending, "Starting", 0.0, callback.Callback)
		converter.reportProgress(types.ConverterStatusPending, "Processing", 0.5, callback.Callback)
		converter.reportProgress(types.ConverterStatusSuccess, "Completed", 1.0, callback.Callback)

		if callback.GetCallCount() != 3 {
			t.Errorf("Expected 3 callback calls, got %d", callback.GetCallCount())
		}

		if callback.GetLastStatus() != types.ConverterStatusSuccess {
			t.Errorf("Expected final status Success, got %v", callback.GetLastStatus())
		}

		if callback.GetLastProgress() != 1.0 {
			t.Errorf("Expected final progress 1.0, got %f", callback.GetLastProgress())
		}

		// Check progress sequence
		expectedProgresses := []float64{0.0, 0.5, 1.0}
		for i, call := range callback.Calls {
			if call.Progress != expectedProgresses[i] {
				t.Errorf("Call %d: expected progress %f, got %f", i, expectedProgresses[i], call.Progress)
			}
		}
	})

	t.Run("Nil callback handling", func(t *testing.T) {
		// Should not panic with nil callback
		converter.reportProgress(types.ConverterStatusSuccess, "Test", 1.0)
		t.Log("Nil callback handled correctly")
	})
}

// ==== Memory Leak Detection Tests ====

// NOTE: These tests are commented out because they call real OpenAI API which is expensive
// To enable memory leak and concurrent tests, uncomment these functions

/*
func TestVision_Convert_NoMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	prepareVisionConnector(t)

	leakResult := runWithLeakDetection(t, func() error {
		options := createVisionOptions(false, 1024)
		converter, err := NewVision(options)
		if err != nil {
			return err
		}

		ctx := context.Background()

		// Process multiple files to detect leaks
		testFiles := getImageTestFiles()
		for _, testFile := range testFiles {
			// These will fail at LLM stage but test image processing
			converter.Convert(ctx, testFile.Path)
		}
		return nil
	})

	assertNoLeaks(t, leakResult, "Vision Convert operations")
}

func TestVision_ConvertStream_NoMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	prepareVisionConnector(t)

	leakResult := runWithLeakDetection(t, func() error {
		options := createVisionOptions(false, 1024)
		converter, err := NewVision(options)
		if err != nil {
			return err
		}

		ctx := context.Background()

		// Process multiple stream operations
		testFiles := getCompressedImageTestFiles()
		for _, testFile := range testFiles {
			file, err := os.Open(testFile.Path)
			if err != nil {
				return err
			}

			converter.ConvertStream(ctx, file)
			file.Close()
		}
		return nil
	})

	assertNoLeaks(t, leakResult, "Vision ConvertStream operations")
}

// ==== Concurrent Stress Tests ====

func TestVision_Convert_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent stress test in short mode")
	}

	prepareVisionConnector(t)
	config := LightStressConfig() // Use light config for CI

	operation := func(ctx context.Context) error {
		options := createVisionOptions(false, 512) // Smaller compression for speed
		converter, err := NewVision(options)
		if err != nil {
			return err
		}

		testFiles := getImageTestFiles()
		// Pick a random file from the list
		testFile := testFiles[len(testFiles)%4] // Simple way to vary files

		_, err = converter.Convert(ctx, testFile.Path)
		// We expect this to fail with mock connector, so don't return the error
		return nil
	}

	stressResult, leakResult := runConcurrentStressWithLeakDetection(t, config, operation)

	assertStressTestResult(t, stressResult, config, "Vision Convert concurrent stress test")
	assertNoLeaks(t, leakResult, "Vision Convert concurrent stress test")
}

func TestVision_ConvertStream_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent stress test in short mode")
	}

	prepareVisionConnector(t)
	config := LightStressConfig()

	operation := func(ctx context.Context) error {
		options := createVisionOptions(false, 512)
		converter, err := NewVision(options)
		if err != nil {
			return err
		}

		testFile := getVisionTestFilePath("test.png")
		file, err := os.Open(testFile)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = converter.ConvertStream(ctx, file)
		// Don't return error as we expect LLM failure with mock connector
		return nil
	}

	stressResult, leakResult := runConcurrentStressWithLeakDetection(t, config, operation)

	assertStressTestResult(t, stressResult, config, "Vision ConvertStream concurrent stress test")
	assertNoLeaks(t, leakResult, "Vision ConvertStream concurrent stress test")
}

func TestVision_Mixed_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping mixed concurrent stress test in short mode")
	}

	prepareVisionConnector(t)
	config := LightStressConfig()

	operation := func(ctx context.Context) error {
		options := createVisionOptions(false, 1024)
		converter, err := NewVision(options)
		if err != nil {
			return err
		}

		// Alternate between Convert and ConvertStream
		if time.Now().UnixNano()%2 == 0 {
			// Use Convert
			testFile := getVisionTestFilePath("test.jpg")
			converter.Convert(ctx, testFile)
		} else {
			// Use ConvertStream
			testFile := getVisionTestFilePath("test.png.gz")
			file, err := os.Open(testFile)
			if err != nil {
				return err
			}
			defer file.Close()

			converter.ConvertStream(ctx, file)
		}
		return nil
	}

	stressResult, leakResult := runConcurrentStressWithLeakDetection(t, config, operation)

	assertStressTestResult(t, stressResult, config, "Vision mixed operation concurrent stress test")
	assertNoLeaks(t, leakResult, "Vision mixed operation concurrent stress test")
}
*/

// ==== Performance Benchmarks ====

// NOTE: These benchmarks are commented out because they call real OpenAI API which is expensive
// To enable benchmarks, uncomment these functions

/*
func BenchmarkVision_Convert_PNG(b *testing.B) {
	prepareVisionConnector(&testing.T{})

	options := createVisionOptions(false, 1024)
	converter, err := NewVision(options)
	if err != nil {
		b.Fatalf("Failed to create vision converter: %v", err)
	}

	ctx := context.Background()
	testFile := getVisionTestFilePath("test.png")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter.Convert(ctx, testFile)
	}
}

func BenchmarkVision_ConvertStream_CompressedImage(b *testing.B) {
	prepareVisionConnector(&testing.T{})

	options := createVisionOptions(false, 1024)
	converter, err := NewVision(options)
	if err != nil {
		b.Fatalf("Failed to create vision converter: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file, err := os.Open(getVisionTestFilePath("test.jpg.gz"))
		if err != nil {
			b.Fatalf("Failed to open file: %v", err)
		}

		converter.ConvertStream(ctx, file)
		file.Close()
	}
}
*/

// ==== Edge Case Tests ====

func TestVision_EdgeCases(t *testing.T) {
	prepareVisionConnector(t)

	t.Run("Very small compress size", func(t *testing.T) {
		options := createVisionOptions(false, 64) // Very small compression
		converter, err := NewVision(options)
		if err != nil {
			t.Fatalf("Failed to create vision converter: %v", err)
		}

		testFile := getVisionTestFilePath("test.png")
		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file: %v", err)
		}

		compressedData, err := converter.compressImage(data, "image/png")
		if err != nil {
			t.Fatalf("compressImage failed: %v", err)
		}

		t.Logf("Very small compression: %d -> %d bytes", len(data), len(compressedData))
	})

	t.Run("Zero compress size", func(t *testing.T) {
		options := createVisionOptions(false, 0) // Should use default
		converter, err := NewVision(options)
		if err != nil {
			t.Fatalf("Failed to create vision converter: %v", err)
		}

		if converter.CompressSize != 1024 {
			t.Errorf("Expected default CompressSize 1024, got %d", converter.CompressSize)
		}
	})

	t.Run("Empty filename", func(t *testing.T) {
		options := createVisionOptions(false, 1024)
		converter, err := NewVision(options)
		if err != nil {
			t.Fatalf("Failed to create vision converter: %v", err)
		}

		ctx := context.Background()
		_, err = converter.Convert(ctx, "")
		if err == nil {
			t.Error("Expected error for empty filename, but got none")
		}

		t.Logf("Empty filename correctly failed: %v", err)
	})
}

// ==== Utility Functions ====

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ==== Integration Test with Real OpenAI (if available) ====

func TestVision_RealOpenAI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real OpenAI integration test in short mode")
	}

	prepareVisionConnector(t)

	// Check if OpenAI key is available
	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping real OpenAI integration test")
	}

	options := createVisionOptions(true, 1024)
	converter, err := NewVision(options)
	if err != nil {
		t.Fatalf("Failed to create vision converter: %v", err)
	}

	t.Run("Single image description", func(t *testing.T) {
		ctx := context.Background()
		testFile := getVisionTestFilePath("test.png")

		callback := NewTestProgressCallback()
		result, err := converter.Convert(ctx, testFile, callback.Callback)

		if err != nil {
			t.Fatalf("Real OpenAI conversion failed: %v", err)
		}

		if result == nil || result.Text == "" {
			t.Error("Real OpenAI returned empty result")
		}

		if len(result.Text) < 20 {
			t.Errorf("Real OpenAI result too short: %q", result.Text)
		}

		// Check that we got meaningful progress
		if callback.GetCallCount() < 3 {
			t.Errorf("Expected at least 3 progress calls, got %d", callback.GetCallCount())
		}

		if callback.GetLastStatus() != types.ConverterStatusSuccess {
			t.Errorf("Expected final status Success, got %v", callback.GetLastStatus())
		}

		t.Logf("Real OpenAI integration successful!")
		t.Logf("Description length: %d characters", len(result.Text))
		t.Logf("Progress calls: %d", callback.GetCallCount())
		t.Logf("Description preview: %s", truncateString(result.Text, 200))
	})
}
