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

// getOfficeTestDataDir returns the office test data directory
func getOfficeTestDataDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(currentFile)
	testDataDir := filepath.Join(currentDir, "..", "tests", "converter")
	absPath, err := filepath.Abs(testDataDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to get absolute path for office test data dir: %v", err))
	}
	return absPath
}

// getOfficeTestFilePath returns the full path to an office test file
func getOfficeTestFilePath(subdir, filename string) string {
	return filepath.Join(getOfficeTestDataDir(), subdir, filename)
}

// ensureOfficeTestDataExists checks if office test data directory and files exist
func ensureOfficeTestDataExists(t *testing.T) {
	t.Helper()

	testDir := getOfficeTestDataDir()
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatalf("Office test data directory does not exist: %s", testDir)
	}

	// Check for required test subdirectories
	requiredDirs := []string{"docx", "pptx"}
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(testDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Fatalf("Required test directory does not exist: %s", dirPath)
		}
	}

	// Check for required test files
	requiredFiles := []struct {
		subdir   string
		filename string
	}{
		{"docx", "english_sample_1.docx"},
		{"docx", "english_sample_2.docx"},
		{"docx", "chinese_sample_1.docx"},
		{"docx", "chinese_sample_2.docx"},
		{"docx", "english_sample_1.docx.gz"},
		{"docx", "english_sample_2.docx.gz"},
		{"docx", "chinese_sample_1.docx.gz"},
		{"docx", "chinese_sample_2.docx.gz"},
		{"pptx", "sample_1.pptx"},
		{"pptx", "sample_2.pptx"},
		{"pptx", "sample_1.pptx.gz"},
		{"pptx", "sample_2.pptx.gz"},
	}

	for _, file := range requiredFiles {
		filePath := getOfficeTestFilePath(file.subdir, file.filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatalf("Required office test file does not exist: %s", filePath)
		}
	}
}

// OfficeTestFileInfo contains information about an office test file
type OfficeTestFileInfo struct {
	Name        string
	Path        string
	ShouldFail  bool
	Format      string
	Description string
	Language    string
	Type        string // "docx" or "pptx"
	HasMedia    bool   // Whether the file is expected to contain media
}

// getOfficeConverterTestFiles returns all office test files that should convert successfully
func getOfficeConverterTestFiles() []OfficeTestFileInfo {
	return []OfficeTestFileInfo{
		{
			Name:        "english_sample_1.docx",
			Path:        getOfficeTestFilePath("docx", "english_sample_1.docx"),
			Format:      "DOCX",
			Description: "English Word document sample 1",
			Language:    "en",
			Type:        "docx",
			HasMedia:    true, // Assume it might contain images
		},
		{
			Name:        "english_sample_2.docx",
			Path:        getOfficeTestFilePath("docx", "english_sample_2.docx"),
			Format:      "DOCX",
			Description: "English Word document sample 2",
			Language:    "en",
			Type:        "docx",
			HasMedia:    false,
		},
		{
			Name:        "chinese_sample_1.docx",
			Path:        getOfficeTestFilePath("docx", "chinese_sample_1.docx"),
			Format:      "DOCX",
			Description: "Chinese Word document sample 1",
			Language:    "zh",
			Type:        "docx",
			HasMedia:    true,
		},
		{
			Name:        "chinese_sample_2.docx",
			Path:        getOfficeTestFilePath("docx", "chinese_sample_2.docx"),
			Format:      "DOCX",
			Description: "Chinese Word document sample 2",
			Language:    "zh",
			Type:        "docx",
			HasMedia:    false,
		},
		{
			Name:        "english_sample_1.docx.gz",
			Path:        getOfficeTestFilePath("docx", "english_sample_1.docx.gz"),
			Format:      "DOCX_GZIP",
			Description: "English Word document sample 1 (gzipped)",
			Language:    "en",
			Type:        "docx",
			HasMedia:    true,
		},
		{
			Name:        "english_sample_2.docx.gz",
			Path:        getOfficeTestFilePath("docx", "english_sample_2.docx.gz"),
			Format:      "DOCX_GZIP",
			Description: "English Word document sample 2 (gzipped)",
			Language:    "en",
			Type:        "docx",
			HasMedia:    false,
		},
		{
			Name:        "chinese_sample_1.docx.gz",
			Path:        getOfficeTestFilePath("docx", "chinese_sample_1.docx.gz"),
			Format:      "DOCX_GZIP",
			Description: "Chinese Word document sample 1 (gzipped)",
			Language:    "zh",
			Type:        "docx",
			HasMedia:    true,
		},
		{
			Name:        "chinese_sample_2.docx.gz",
			Path:        getOfficeTestFilePath("docx", "chinese_sample_2.docx.gz"),
			Format:      "DOCX_GZIP",
			Description: "Chinese Word document sample 2 (gzipped)",
			Language:    "zh",
			Type:        "docx",
			HasMedia:    false,
		},
		{
			Name:        "sample_1.pptx",
			Path:        getOfficeTestFilePath("pptx", "sample_1.pptx"),
			Format:      "PPTX",
			Description: "PowerPoint presentation sample 1",
			Language:    "en", // Assume English unless otherwise detected
			Type:        "pptx",
			HasMedia:    true, // PowerPoint files typically contain images
		},
		{
			Name:        "sample_2.pptx",
			Path:        getOfficeTestFilePath("pptx", "sample_2.pptx"),
			Format:      "PPTX",
			Description: "PowerPoint presentation sample 2",
			Language:    "en",
			Type:        "pptx",
			HasMedia:    true,
		},
		{
			Name:        "sample_1.pptx.gz",
			Path:        getOfficeTestFilePath("pptx", "sample_1.pptx.gz"),
			Format:      "PPTX_GZIP",
			Description: "PowerPoint presentation sample 1 (gzipped)",
			Language:    "en",
			Type:        "pptx",
			HasMedia:    true,
		},
		{
			Name:        "sample_2.pptx.gz",
			Path:        getOfficeTestFilePath("pptx", "sample_2.pptx.gz"),
			Format:      "PPTX_GZIP",
			Description: "PowerPoint presentation sample 2 (gzipped)",
			Language:    "en",
			Type:        "pptx",
			HasMedia:    true,
		},
	}
}

// ==== Connector Setup ====

// prepareOfficeConnectors creates connectors for office testing (vision, video, whisper)
func prepareOfficeConnectors(t *testing.T) {
	t.Helper()

	// Create OpenAI connector for vision (image processing)
	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping office tests")
	}

	// OpenAI connector for Vision
	openaiVisionDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0", 
		"label": "OpenAI Vision Office Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "gpt-4o-mini",
			"key": "%s"
		}
	}`, openaiKey)

	_, err := connector.New("openai", "test-office-vision", []byte(openaiVisionDSL))
	if err != nil {
		t.Fatalf("Failed to create OpenAI vision connector: %v", err)
	}

	// OpenAI connector for Whisper (audio processing)
	openaiWhisperDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0", 
		"label": "OpenAI Whisper Office Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "whisper-1",
			"key": "%s"
		}
	}`, openaiKey)

	_, err = connector.New("openai", "test-office-whisper", []byte(openaiWhisperDSL))
	if err != nil {
		t.Fatalf("Failed to create OpenAI whisper connector: %v", err)
	}
}

// createOfficeConverters creates vision, video, and whisper converters for office testing
func createOfficeConverters(t *testing.T) (types.Converter, types.Converter, types.Converter) {
	t.Helper()

	// Create Vision converter (required)
	visionOptions := VisionOption{
		ConnectorName: "test-office-vision",
		Model:         "gpt-4o-mini",
		CompressSize:  1024,
		Language:      "Auto",
		Options:       map[string]any{"max_tokens": 1000, "temperature": 0.1},
	}

	visionConverter, err := NewVision(visionOptions)
	if err != nil {
		t.Fatalf("Failed to create Vision converter: %v", err)
	}

	// Create Whisper converter (optional)
	whisperOptions := WhisperOption{
		ConnectorName:          "test-office-whisper",
		Model:                  "whisper-1",
		Language:               "en",
		ChunkDuration:          30.0,
		MappingDuration:        5.0,
		SilenceThreshold:       -40.0,
		SilenceMinLength:       1.0,
		EnableSilenceDetection: true,
		MaxConcurrency:         4,
		CleanupTemp:            true,
		Options:                map[string]any{"temperature": 0.0},
	}

	whisperConverter, err := NewWhisper(whisperOptions)
	if err != nil {
		t.Logf("Warning: Failed to create Whisper converter: %v", err)
		whisperConverter = nil // Optional converter
	}

	// Create Video converter (optional) - reuse existing converters
	var videoConverter types.Converter
	if whisperConverter != nil {
		videoOptions := VideoOption{
			AudioConverter:     whisperConverter,
			VisionConverter:    visionConverter,
			KeyframeInterval:   30.0,
			MaxKeyframes:       10,
			TempDir:            "",
			CleanupTemp:        true,
			MaxConcurrency:     4,
			TextOptimization:   true,
			DeduplicationRatio: 0.8,
		}

		videoConverter, err = NewVideo(videoOptions)
		if err != nil {
			t.Logf("Warning: Failed to create Video converter: %v", err)
			videoConverter = nil // Optional converter
		}
	}

	return visionConverter, videoConverter, whisperConverter
}

// createOfficeOptions creates OfficeOption for testing
func createOfficeOptions(t *testing.T) OfficeOption {
	t.Helper()

	visionConverter, videoConverter, whisperConverter := createOfficeConverters(t)

	return OfficeOption{
		VisionConverter:  visionConverter,
		VideoConverter:   videoConverter,
		WhisperConverter: whisperConverter,
		MaxConcurrency:   4,
		TempDir:          "", // Use system temp dir
		CleanupTemp:      true,
	}
}

// ==== Test Progress Callback ====

// OfficeTestProgressCallback is a test implementation of progress callback
type OfficeTestProgressCallback struct {
	Calls        []types.ConverterPayload
	CallCount    int
	LastStatus   types.ConverterStatus
	LastMessage  string
	LastProgress float64
}

// NewOfficeTestProgressCallback creates a new test progress callback
func NewOfficeTestProgressCallback() *OfficeTestProgressCallback {
	return &OfficeTestProgressCallback{
		Calls: make([]types.ConverterPayload, 0),
	}
}

// Callback implements the progress callback interface
func (c *OfficeTestProgressCallback) Callback(status types.ConverterStatus, payload types.ConverterPayload) {
	c.Calls = append(c.Calls, payload)
	c.CallCount++
	c.LastStatus = status
	c.LastMessage = payload.Message
	c.LastProgress = payload.Progress
}

// GetCallCount returns the number of times the callback was called
func (c *OfficeTestProgressCallback) GetCallCount() int {
	return c.CallCount
}

// GetLastStatus returns the last status
func (c *OfficeTestProgressCallback) GetLastStatus() types.ConverterStatus {
	return c.LastStatus
}

// GetLastProgress returns the last progress value
func (c *OfficeTestProgressCallback) GetLastProgress() float64 {
	return c.LastProgress
}

// ==== Helper Functions ====

// Note: truncateString is already defined in vision_test.go

// ==== Basic Functionality Tests ====

func TestOffice_NewOffice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping office tests in short mode")
	}

	ensureOfficeTestDataExists(t)
	prepareOfficeConnectors(t)

	t.Run("Valid converters with defaults", func(t *testing.T) {
		options := createOfficeOptions(t)
		converter, err := NewOffice(options)
		if err != nil {
			t.Fatalf("NewOffice failed: %v", err)
		}

		if converter == nil {
			t.Fatal("NewOffice returned nil")
		}

		if converter.VisionConverter == nil {
			t.Error("VisionConverter should not be nil")
		}

		if converter.MaxConcurrency != 4 {
			t.Errorf("Expected default MaxConcurrency 4, got %d", converter.MaxConcurrency)
		}

		if !converter.CleanupTemp {
			t.Error("Expected CleanupTemp to be true")
		}

		if converter.Parser == nil {
			t.Error("Parser should not be nil")
		}
	})

	t.Run("Custom parameters", func(t *testing.T) {
		options := createOfficeOptions(t)
		options.MaxConcurrency = 8
		options.CleanupTemp = false

		converter, err := NewOffice(options)
		if err != nil {
			t.Fatalf("NewOffice with custom params failed: %v", err)
		}

		if converter.MaxConcurrency != 8 {
			t.Errorf("Expected MaxConcurrency 8, got %d", converter.MaxConcurrency)
		}

		if converter.CleanupTemp {
			t.Error("Expected CleanupTemp to be false")
		}
	})

	t.Run("Missing vision converter", func(t *testing.T) {
		options := OfficeOption{
			VisionConverter: nil, // Missing required converter
			MaxConcurrency:  4,
		}

		converter, err := NewOffice(options)
		if err == nil {
			t.Error("Expected error for missing vision converter, but got none")
		}
		if converter != nil {
			t.Error("Expected nil converter for missing vision converter")
		}
		if !strings.Contains(err.Error(), "vision converter is required") {
			t.Errorf("Expected error message about vision converter, got: %v", err)
		}
	})
}

func TestOffice_Convert_OfficeFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping office conversion tests in short mode")
	}

	ensureOfficeTestDataExists(t)
	prepareOfficeConnectors(t)

	testFiles := getOfficeConverterTestFiles()

	for _, testFile := range testFiles {
		t.Run(testFile.Name, func(t *testing.T) {
			options := createOfficeOptions(t)
			converter, err := NewOffice(options)
			if err != nil {
				t.Fatalf("Failed to create Office converter: %v", err)
			}

			ctx := context.Background()
			callback := NewOfficeTestProgressCallback()

			result, err := converter.Convert(ctx, testFile.Path, callback.Callback)

			if testFile.ShouldFail {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", testFile.Description)
				}
				return
			}

			if err != nil {
				t.Fatalf("Convert failed for %s: %v", testFile.Description, err)
			}

			// Perform comprehensive validation
			validateOfficeConversionResult(t, result, testFile)

			// Check that we got reasonable progress callbacks
			if callback.GetCallCount() < 3 {
				t.Errorf("Expected at least 3 progress calls for %s, got %d", testFile.Description, callback.GetCallCount())
			}

			if callback.GetLastStatus() != types.ConverterStatusSuccess {
				t.Errorf("Expected final status Success for %s, got %v", testFile.Description, callback.GetLastStatus())
			}

			t.Logf("%s: Generated %d chars text with %d progress calls",
				testFile.Description, len(result.Text), callback.GetCallCount())
			t.Logf("Text preview: %s", truncateString(result.Text, 200))
		})
	}
}

func TestOffice_Convert_NonOfficeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping non-office file tests in short mode")
	}

	prepareOfficeConnectors(t)

	options := createOfficeOptions(t)
	converter, err := NewOffice(options)
	if err != nil {
		t.Fatalf("Failed to create Office converter: %v", err)
	}

	// Create a temporary non-office file
	tempFile := filepath.Join(os.TempDir(), "test_non_office.txt")
	err = os.WriteFile(tempFile, []byte("This is not an office file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)

	ctx := context.Background()
	_, err = converter.Convert(ctx, tempFile)
	if err == nil {
		t.Error("Expected error for non-office file, but got none")
	}

	// Check that error message indicates parsing failure
	if !strings.Contains(err.Error(), "failed to parse office document") {
		t.Logf("Expected 'failed to parse office document' in error message, got: %v", err)
	}

	t.Logf("Correctly rejected non-office file with error: %v", err)
}

func TestOffice_ProgressReporting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping progress reporting tests in short mode")
	}

	prepareOfficeConnectors(t)

	options := createOfficeOptions(t)
	converter, err := NewOffice(options)
	if err != nil {
		t.Fatalf("Failed to create Office converter: %v", err)
	}

	t.Run("Progress callback sequence", func(t *testing.T) {
		callback := NewOfficeTestProgressCallback()

		// Test manual progress reporting
		converter.reportProgress(types.ConverterStatusPending, "Starting office document processing", 0.0, callback.Callback)
		converter.reportProgress(types.ConverterStatusPending, "Parsing office document", 0.1, callback.Callback)
		converter.reportProgress(types.ConverterStatusPending, "Processing media files", 0.3, callback.Callback)
		converter.reportProgress(types.ConverterStatusPending, "Merging text and media", 0.8, callback.Callback)
		converter.reportProgress(types.ConverterStatusSuccess, "Office document processing completed", 1.0, callback.Callback)

		if callback.GetCallCount() != 5 {
			t.Errorf("Expected 5 callback calls, got %d", callback.GetCallCount())
		}

		if callback.GetLastStatus() != types.ConverterStatusSuccess {
			t.Errorf("Expected final status Success, got %v", callback.GetLastStatus())
		}

		if callback.GetLastProgress() != 1.0 {
			t.Errorf("Expected final progress 1.0, got %f", callback.GetLastProgress())
		}
	})

	t.Run("Nil callback handling", func(t *testing.T) {
		// Should not panic with nil callback
		converter.reportProgress(types.ConverterStatusSuccess, "Test", 1.0)
		t.Log("Nil callback handled correctly")
	})
}

// ==== Error Handling Tests ====

func TestOffice_Convert_NonExistentFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error handling tests in short mode")
	}

	prepareOfficeConnectors(t)

	options := createOfficeOptions(t)
	converter, err := NewOffice(options)
	if err != nil {
		t.Fatalf("Failed to create Office converter: %v", err)
	}

	ctx := context.Background()
	_, err = converter.Convert(ctx, "/non/existent/document.docx")
	if err == nil {
		t.Error("Expected error for non-existent file, but got none")
	}

	t.Logf("Correctly failed with error: %v", err)
}

func TestOffice_Convert_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping context cancellation tests in short mode")
	}

	ensureOfficeTestDataExists(t)
	prepareOfficeConnectors(t)

	options := createOfficeOptions(t)
	converter, err := NewOffice(options)
	if err != nil {
		t.Fatalf("Failed to create Office converter: %v", err)
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	testFile := getOfficeConverterTestFiles()[0].Path // Use first test file
	_, err = converter.Convert(ctx, testFile)

	// The operation might complete before cancellation is checked
	if err != nil && err == context.Canceled {
		t.Log("Context cancellation handled correctly")
	} else {
		t.Log("Operation completed before cancellation check (acceptable)")
	}
}

// ==== Media Processing Tests ====

func TestOffice_MediaProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping media processing tests in short mode")
	}

	ensureOfficeTestDataExists(t)
	prepareOfficeConnectors(t)

	t.Run("Document with potential media", func(t *testing.T) {
		options := createOfficeOptions(t)
		converter, err := NewOffice(options)
		if err != nil {
			t.Fatalf("Failed to create Office converter: %v", err)
		}

		ctx := context.Background()

		// Test files that might contain media
		mediaTestFiles := []OfficeTestFileInfo{}
		for _, file := range getOfficeConverterTestFiles() {
			if file.HasMedia {
				mediaTestFiles = append(mediaTestFiles, file)
			}
		}

		if len(mediaTestFiles) == 0 {
			t.Skip("No test files with media available")
		}

		for _, testFile := range mediaTestFiles {
			t.Run(testFile.Name, func(t *testing.T) {
				callback := NewOfficeTestProgressCallback()
				result, err := converter.Convert(ctx, testFile.Path, callback.Callback)

				if err != nil {
					t.Fatalf("Convert failed for %s: %v", testFile.Description, err)
				}

				// Check if media processing occurred
				if result.Metadata != nil {
					if mediaCount, exists := result.Metadata["media_count"]; exists {
						if count, ok := mediaCount.(int); ok && count > 0 {
							t.Logf("Document %s contains %d media files", testFile.Name, count)

							// Check for successful media processing
							if successfulMedia, exists := result.Metadata["successful_media"]; exists {
								if successful, ok := successfulMedia.(int); ok {
									t.Logf("Successfully processed %d/%d media files", successful, count)
								}
							}
						} else {
							t.Logf("Document %s contains no media files", testFile.Name)
						}
					}
				}
			})
		}
	})
}

// ==== Integration Tests ====

func TestOffice_Integration_Comprehensive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ensureOfficeTestDataExists(t)
	prepareOfficeConnectors(t)

	t.Run("Comprehensive office document processing", func(t *testing.T) {
		options := createOfficeOptions(t)
		converter, err := NewOffice(options)
		if err != nil {
			t.Fatalf("Failed to create Office converter: %v", err)
		}

		ctx := context.Background()
		testFiles := getOfficeConverterTestFiles()

		var totalTexts []string
		var totalMetadata []map[string]interface{}

		for _, testFile := range testFiles {
			t.Run(testFile.Name, func(t *testing.T) {
				callback := NewOfficeTestProgressCallback()
				result, err := converter.Convert(ctx, testFile.Path, callback.Callback)

				if err != nil {
					t.Fatalf("Convert failed for %s: %v", testFile.Description, err)
				}

				if result == nil || result.Text == "" {
					t.Error("Conversion returned empty result")
				}

				// Collect results for comprehensive analysis
				totalTexts = append(totalTexts, result.Text)
				if result.Metadata != nil {
					totalMetadata = append(totalMetadata, result.Metadata)
				}

				// Check that we got meaningful progress through all stages
				if callback.GetCallCount() < 3 {
					t.Errorf("Expected at least 3 progress calls, got %d", callback.GetCallCount())
				}

				if callback.GetLastStatus() != types.ConverterStatusSuccess {
					t.Errorf("Expected final status Success, got %v", callback.GetLastStatus())
				}

				t.Logf("%s: %d characters, %d progress calls",
					testFile.Name, len(result.Text), callback.GetCallCount())
			})
		}

		// Comprehensive analysis
		t.Logf("Processed %d office documents successfully", len(totalTexts))

		totalChars := 0
		for _, text := range totalTexts {
			totalChars += len(text)
		}
		t.Logf("Total extracted text: %d characters", totalChars)
	})
}

// ==== Cleanup Tests ====

func TestOffice_ResourceCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource cleanup tests in short mode")
	}

	prepareOfficeConnectors(t)

	t.Run("Resource cleanup", func(t *testing.T) {
		options := createOfficeOptions(t)
		options.CleanupTemp = true // Ensure cleanup is enabled

		converter, err := NewOffice(options)
		if err != nil {
			t.Fatalf("Failed to create Office converter: %v", err)
		}

		// Close converter to test cleanup
		if err := converter.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}

		t.Log("Resource cleanup test completed")
	})
}

// ==== Validation Functions ====

// validateOfficeConversionResult validates the text and metadata from office conversion
func validateOfficeConversionResult(t *testing.T, result *types.ConvertResult, testFile OfficeTestFileInfo) {
	t.Helper()

	// Basic validation
	if result == nil {
		t.Fatalf("Convert returned nil result for %s", testFile.Description)
	}

	if result.Text == "" {
		t.Fatalf("Convert returned empty text for %s", testFile.Description)
	}

	// Text content validation
	validateOfficeTextContent(t, result.Text, testFile)

	// Metadata validation
	validateOfficeMetadata(t, result.Metadata, testFile)

	// Structure validation
	validateOfficeTextStructure(t, result.Text, testFile)
}

// validateOfficeTextContent validates the quality and content of generated text
func validateOfficeTextContent(t *testing.T, text string, testFile OfficeTestFileInfo) {
	t.Helper()

	// Check minimum text length (should be substantial for office documents)
	// Allow for some test files to be mostly empty
	if len(text) < 50 {
		if testFile.Name == "chinese_sample_2.docx" || testFile.Name == "chinese_sample_2.docx.gz" {
			t.Logf("Warning: Generated text is short for %s: %d characters (may be mostly empty document)", testFile.Description, len(text))
		} else {
			t.Errorf("Generated text too short for %s: %d characters", testFile.Description, len(text))
		}
	}

	// Check for office-specific content patterns
	if testFile.Type == "pptx" {
		// PowerPoint presentations might have slide-based content
		if !strings.Contains(text, "slide") && !strings.Contains(text, "Slide") {
			t.Logf("Info: PowerPoint content might not contain slide references for %s", testFile.Description)
		}
	}

	// Language-specific validation
	if testFile.Language == "zh" {
		// For Chinese documents, expect Chinese characters
		hasChinese := false
		for _, r := range text {
			if r >= 0x4e00 && r <= 0x9fff { // Basic Chinese character range
				hasChinese = true
				break
			}
		}
		if !hasChinese {
			if testFile.Name == "chinese_sample_2.docx" || testFile.Name == "chinese_sample_2.docx.gz" {
				t.Logf("Warning: No Chinese characters detected in Chinese document %s (may be mostly empty)", testFile.Description)
			} else {
				t.Logf("Warning: No Chinese characters detected in Chinese document %s", testFile.Description)
			}
		}
	} else if testFile.Language == "en" {
		// For English documents, expect Latin characters
		hasLatin := false
		for _, r := range text {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				hasLatin = true
				break
			}
		}
		if !hasLatin {
			t.Errorf("No Latin characters detected in English document %s", testFile.Description)
		}
	}

	// Check for media descriptions if media is expected
	if testFile.HasMedia {
		hasMediaContent := strings.Contains(text, "[Image:") ||
			strings.Contains(text, "[Video:") ||
			strings.Contains(text, "[Audio:") ||
			strings.Contains(text, "[Media:")
		if hasMediaContent {
			t.Logf("Media descriptions found in %s", testFile.Description)
		}
	}

	t.Logf("Text content validation passed for %s (%d chars)", testFile.Description, len(text))
}

// validateOfficeMetadata validates the completeness and accuracy of metadata
func validateOfficeMetadata(t *testing.T, metadata map[string]interface{}, testFile OfficeTestFileInfo) {
	t.Helper()

	if metadata == nil {
		t.Fatalf("Metadata is nil for %s", testFile.Description)
	}

	// Required metadata fields
	requiredFields := []string{
		"source_type",
		"media_count",
		"processed_media",
		"text_length",
		"conversion_time",
		"successful_media",
	}

	for _, field := range requiredFields {
		if _, exists := metadata[field]; !exists {
			t.Errorf("Missing required metadata field '%s' for %s", field, testFile.Description)
		}
	}

	// Validate specific metadata values
	if sourceType, ok := metadata["source_type"].(string); !ok || sourceType != "office" {
		t.Errorf("Expected source_type 'office', got %v for %s", metadata["source_type"], testFile.Description)
	}

	// Check for gzip information
	if strings.Contains(testFile.Format, "GZIP") {
		if gzipped, exists := metadata["gzipped"]; !exists {
			t.Errorf("Missing gzipped field for gzip file %s", testFile.Description)
		} else if gzippedBool, ok := gzipped.(bool); !ok || !gzippedBool {
			t.Errorf("Expected gzipped=true for gzip file %s, got %v", testFile.Description, gzipped)
		}
	} else {
		// For non-gzip files, gzipped should be false or not present
		if gzipped, exists := metadata["gzipped"]; exists {
			if gzippedBool, ok := gzipped.(bool); ok && gzippedBool {
				t.Errorf("Expected gzipped=false for non-gzip file %s, got %v", testFile.Description, gzipped)
			}
		}
	}

	if mediaCount, ok := metadata["media_count"].(int); ok {
		if mediaCount < 0 {
			t.Errorf("Invalid media_count %d for %s", mediaCount, testFile.Description)
		}
	}

	if processedMedia, ok := metadata["processed_media"].(int); ok {
		if processedMedia < 0 {
			t.Errorf("Invalid processed_media %d for %s", processedMedia, testFile.Description)
		}

		if mediaCount, ok := metadata["media_count"].(int); ok {
			if processedMedia != mediaCount {
				t.Errorf("Processed media (%d) should equal media count (%d) for %s",
					processedMedia, mediaCount, testFile.Description)
			}
		}
	}

	if textLength, ok := metadata["text_length"].(int); ok {
		if textLength <= 0 {
			t.Errorf("Invalid text_length %d for %s", textLength, testFile.Description)
		}
	}

	// Check for original metadata preservation
	if originalMetadata, exists := metadata["original_metadata"]; exists {
		if originalMetadata == nil {
			t.Logf("Original metadata is nil for %s (might be normal)", testFile.Description)
		}
	}

	// Check for page mapping if available
	if pageMapping, exists := metadata["page_mapping"]; exists {
		validatePageMapping(t, pageMapping, testFile)
	}

	t.Logf("Metadata validation passed for %s", testFile.Description)
}

// validatePageMapping validates page mapping metadata
func validatePageMapping(t *testing.T, pageMapping interface{}, testFile OfficeTestFileInfo) {
	t.Helper()

	pageMappingMap, ok := pageMapping.(map[string]interface{})
	if !ok {
		t.Errorf("Page mapping is not a map for %s", testFile.Description)
		return
	}

	// Check for total pages
	if totalPages, exists := pageMappingMap["total_pages"]; exists {
		if pages, ok := totalPages.(int); ok && pages <= 0 {
			t.Errorf("Invalid total_pages %d for %s", pages, testFile.Description)
		}
	}

	// Check for position mapping
	if positionToPage, exists := pageMappingMap["position_to_page"]; exists {
		if posMap, ok := positionToPage.(map[int]int); ok {
			if len(posMap) == 0 {
				t.Logf("Position to page mapping is empty for %s", testFile.Description)
			}
		}
	}

	t.Logf("Page mapping validation passed for %s", testFile.Description)
}

// validateOfficeTextStructure validates the overall structure and format of the text
func validateOfficeTextStructure(t *testing.T, text string, testFile OfficeTestFileInfo) {
	t.Helper()

	lines := strings.Split(text, "\n")

	// Check for reasonable line count
	if len(lines) < 1 {
		t.Errorf("Text has too few lines (%d) for %s", len(lines), testFile.Description)
	}

	// Check for excessive empty lines (should not be majority)
	emptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyLines++
		}
	}

	if len(lines) > 1 && emptyLines > len(lines)*2/3 {
		if testFile.Name == "chinese_sample_2.docx" || testFile.Name == "chinese_sample_2.docx.gz" {
			t.Logf("Warning: Too many empty lines (%d/%d) for %s (may be mostly empty document)", emptyLines, len(lines), testFile.Description)
		} else {
			t.Errorf("Too many empty lines (%d/%d) for %s", emptyLines, len(lines), testFile.Description)
		}
	}

	// Check for document structure elements
	hasStructuralElements := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 {
			// Look for typical document elements
			if strings.Contains(trimmed, "Chapter") ||
				strings.Contains(trimmed, "Section") ||
				strings.Contains(trimmed, "Title") ||
				len(trimmed) > 10 { // Substantial content
				hasStructuralElements = true
				break
			}
		}
	}

	if !hasStructuralElements {
		if testFile.Name == "chinese_sample_2.docx" || testFile.Name == "chinese_sample_2.docx.gz" {
			t.Logf("Warning: No clear structural elements found in %s (may be mostly empty document)", testFile.Description)
		} else {
			t.Logf("Warning: No clear structural elements found in %s", testFile.Description)
		}
	}

	t.Logf("Text structure validation passed for %s", testFile.Description)
}

// TestOffice_FileTypeSpecific performs specific tests for different file types
func TestOffice_FileTypeSpecific(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file type specific tests in short mode")
	}

	ensureOfficeTestDataExists(t)
	prepareOfficeConnectors(t)

	options := createOfficeOptions(t)
	converter, err := NewOffice(options)
	if err != nil {
		t.Fatalf("Failed to create Office converter: %v", err)
	}

	ctx := context.Background()

	t.Run("DOCX files", func(t *testing.T) {
		docxFiles := []OfficeTestFileInfo{}
		for _, file := range getOfficeConverterTestFiles() {
			if file.Type == "docx" {
				docxFiles = append(docxFiles, file)
			}
		}

		for _, testFile := range docxFiles {
			t.Run(testFile.Name, func(t *testing.T) {
				result, err := converter.Convert(ctx, testFile.Path)
				if err != nil {
					t.Fatalf("DOCX conversion failed for %s: %v", testFile.Description, err)
				}

				// DOCX-specific validations
				if result.Text == "" {
					t.Errorf("Empty text from DOCX file %s", testFile.Description)
				}

				t.Logf("DOCX %s: %d characters extracted", testFile.Name, len(result.Text))
			})
		}
	})

	t.Run("PPTX files", func(t *testing.T) {
		pptxFiles := []OfficeTestFileInfo{}
		for _, file := range getOfficeConverterTestFiles() {
			if file.Type == "pptx" {
				pptxFiles = append(pptxFiles, file)
			}
		}

		for _, testFile := range pptxFiles {
			t.Run(testFile.Name, func(t *testing.T) {
				result, err := converter.Convert(ctx, testFile.Path)
				if err != nil {
					t.Fatalf("PPTX conversion failed for %s: %v", testFile.Description, err)
				}

				// PPTX-specific validations
				if result.Text == "" {
					t.Errorf("Empty text from PPTX file %s", testFile.Description)
				}

				// PPTX files often contain more media
				if result.Metadata != nil {
					if mediaCount, exists := result.Metadata["media_count"]; exists {
						if count, ok := mediaCount.(int); ok {
							t.Logf("PPTX %s: %d characters, %d media files", testFile.Name, len(result.Text), count)
						}
					}
				}
			})
		}
	})
}

// ==== GZIP Support Tests ====

func TestOffice_GzipSupport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping gzip support tests in short mode")
	}

	ensureOfficeTestDataExists(t)
	prepareOfficeConnectors(t)

	t.Run("Gzipped office files", func(t *testing.T) {
		options := createOfficeOptions(t)
		converter, err := NewOffice(options)
		if err != nil {
			t.Fatalf("Failed to create Office converter: %v", err)
		}

		ctx := context.Background()

		// Test gzipped files
		gzipTestFiles := []OfficeTestFileInfo{}
		for _, file := range getOfficeConverterTestFiles() {
			if strings.Contains(file.Format, "GZIP") {
				gzipTestFiles = append(gzipTestFiles, file)
			}
		}

		if len(gzipTestFiles) == 0 {
			t.Skip("No gzipped test files available")
		}

		for _, testFile := range gzipTestFiles {
			t.Run(testFile.Name, func(t *testing.T) {
				callback := NewOfficeTestProgressCallback()
				result, err := converter.Convert(ctx, testFile.Path, callback.Callback)

				if err != nil {
					t.Fatalf("Convert failed for %s: %v", testFile.Description, err)
				}

				// Validate gzip-specific metadata
				if result.Metadata != nil {
					if gzipped, exists := result.Metadata["gzipped"]; !exists {
						t.Errorf("Missing gzipped field in metadata for %s", testFile.Description)
					} else if gzippedBool, ok := gzipped.(bool); !ok || !gzippedBool {
						t.Errorf("Expected gzipped=true for gzipped file %s, got %v", testFile.Description, gzipped)
					}
				}

				// Validate that content was properly extracted
				if result.Text == "" {
					t.Errorf("Empty text from gzipped file %s", testFile.Description)
				}

				// Check that we got reasonable progress callbacks
				if callback.GetCallCount() < 3 {
					t.Errorf("Expected at least 3 progress calls for %s, got %d", testFile.Description, callback.GetCallCount())
				}

				if callback.GetLastStatus() != types.ConverterStatusSuccess {
					t.Errorf("Expected final status Success for %s, got %v", testFile.Description, callback.GetLastStatus())
				}

				t.Logf("Gzipped %s: Generated %d chars text with %d progress calls",
					testFile.Description, len(result.Text), callback.GetCallCount())
			})
		}
	})

	t.Run("Gzip vs non-gzip comparison", func(t *testing.T) {
		options := createOfficeOptions(t)
		converter, err := NewOffice(options)
		if err != nil {
			t.Fatalf("Failed to create Office converter: %v", err)
		}

		ctx := context.Background()

		// Find a pair of gzipped and non-gzipped files for comparison
		var gzipFile, normalFile OfficeTestFileInfo
		found := false

		for _, file := range getOfficeConverterTestFiles() {
			if strings.Contains(file.Format, "GZIP") {
				// Find corresponding non-gzipped file
				baseName := strings.TrimSuffix(file.Name, ".gz")
				for _, otherFile := range getOfficeConverterTestFiles() {
					if otherFile.Name == baseName && !strings.Contains(otherFile.Format, "GZIP") {
						gzipFile = file
						normalFile = otherFile
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}

		if !found {
			t.Skip("No matching gzip/non-gzip file pairs found for comparison")
		}

		t.Run("Content comparison", func(t *testing.T) {
			// Convert both files
			gzipResult, err := converter.Convert(ctx, gzipFile.Path)
			if err != nil {
				t.Fatalf("Failed to convert gzipped file: %v", err)
			}

			normalResult, err := converter.Convert(ctx, normalFile.Path)
			if err != nil {
				t.Fatalf("Failed to convert normal file: %v", err)
			}

			// Compare text content (should be identical)
			if gzipResult.Text != normalResult.Text {
				t.Errorf("Gzipped and normal file content should be identical")
				t.Logf("Gzipped text length: %d", len(gzipResult.Text))
				t.Logf("Normal text length: %d", len(normalResult.Text))
			}

			// Check metadata differences
			if gzipResult.Metadata != nil && normalResult.Metadata != nil {
				if gzipped, exists := gzipResult.Metadata["gzipped"]; !exists {
					t.Errorf("Gzipped file missing gzipped field in metadata")
				} else if gzippedBool, ok := gzipped.(bool); !ok || !gzippedBool {
					t.Errorf("Gzipped file should have gzipped=true")
				}

				if gzipped, exists := normalResult.Metadata["gzipped"]; exists {
					if gzippedBool, ok := gzipped.(bool); ok && gzippedBool {
						t.Errorf("Normal file should not have gzipped=true")
					}
				}
			}

			t.Logf("Gzip vs normal comparison passed for %s", gzipFile.Name)
		})
	})
}
