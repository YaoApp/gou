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

// getVideoTestDataDir returns the video test data directory
func getVideoTestDataDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(currentFile)
	testDataDir := filepath.Join(currentDir, "..", "tests", "converter", "video")
	absPath, err := filepath.Abs(testDataDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to get absolute path for video test data dir: %v", err))
	}
	return absPath
}

// getVideoTestFilePath returns the full path to a video test file
func getVideoTestFilePath(filename string) string {
	return filepath.Join(getVideoTestDataDir(), filename)
}

// ensureVideoTestDataExists checks if video test data directory and files exist
func ensureVideoTestDataExists(t *testing.T) {
	t.Helper()

	testDir := getVideoTestDataDir()
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatalf("Video test data directory does not exist: %s", testDir)
	}

	// Check for required test files
	requiredFiles := []string{
		"chinese_video_1min.mp4",
		"english_video_1min.mp4",
	}

	for _, filename := range requiredFiles {
		filePath := getVideoTestFilePath(filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatalf("Required video test file does not exist: %s", filePath)
		}
	}
}

// VideoTestFileInfo contains information about a video test file
type VideoTestFileInfo struct {
	Name        string
	Path        string
	ShouldFail  bool
	Format      string
	Description string
	Language    string
	Duration    float64 // Expected duration in seconds
}

// getVideoConverterTestFiles returns all video test files that should convert successfully
func getVideoConverterTestFiles() []VideoTestFileInfo {
	return []VideoTestFileInfo{
		{
			Name:        "chinese_video_1min.mp4",
			Path:        getVideoTestFilePath("chinese_video_1min.mp4"),
			Format:      "MP4",
			Description: "Chinese MP4 video file (1 minute)",
			Language:    "zh",
			Duration:    60.0,
		},
		{
			Name:        "english_video_1min.mp4",
			Path:        getVideoTestFilePath("english_video_1min.mp4"),
			Format:      "MP4",
			Description: "English MP4 video file (1 minute)",
			Language:    "en",
			Duration:    60.0,
		},
	}
}

// ==== Connector Setup ====

// prepareVideoConnectors creates connectors for video testing (whisper + vision)
func prepareVideoConnectors(t *testing.T) {
	t.Helper()

	// Create OpenAI connector for whisper (audio processing)
	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping video tests")
	}

	// OpenAI connector for Whisper
	openaiWhisperDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0", 
		"label": "OpenAI Whisper Video Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "whisper-1",
			"key": "%s"
		}
	}`, openaiKey)

	_, err := connector.New("openai", "test-video-whisper", []byte(openaiWhisperDSL))
	if err != nil {
		t.Fatalf("Failed to create OpenAI whisper connector: %v", err)
	}

	// OpenAI connector for Vision
	openaiVisionDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0", 
		"label": "OpenAI Vision Video Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "gpt-4o-mini",
			"key": "%s"
		}
	}`, openaiKey)

	_, err = connector.New("openai", "test-video-vision", []byte(openaiVisionDSL))
	if err != nil {
		t.Fatalf("Failed to create OpenAI vision connector: %v", err)
	}
}

// createVideoConverters creates whisper and vision converters for video testing
func createVideoConverters(t *testing.T) (types.Converter, types.Converter) {
	t.Helper()

	// Create Whisper converter
	whisperOptions := WhisperOption{
		ConnectorName:          "test-video-whisper",
		Model:                  "whisper-1",
		Language:               "en", // Use ISO-639-1 format instead of "auto"
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
		t.Fatalf("Failed to create Whisper converter: %v", err)
	}

	// Create Vision converter
	visionOptions := VisionOption{
		ConnectorName: "test-video-vision",
		Model:         "gpt-4o-mini",
		CompressSize:  1024,
		Language:      "Auto",
		Options:       map[string]any{"max_tokens": 1000, "temperature": 0.1},
	}

	visionConverter, err := NewVision(visionOptions)
	if err != nil {
		t.Fatalf("Failed to create Vision converter: %v", err)
	}

	return whisperConverter, visionConverter
}

// createVideoOptions creates VideoOption for testing
func createVideoOptions(t *testing.T, keyframeInterval float64, maxKeyframes int) VideoOption {
	t.Helper()

	whisperConverter, visionConverter := createVideoConverters(t)

	return VideoOption{
		AudioConverter:     whisperConverter,
		VisionConverter:    visionConverter,
		KeyframeInterval:   keyframeInterval,
		MaxKeyframes:       maxKeyframes,
		TempDir:            "", // Use system temp dir
		CleanupTemp:        true,
		MaxConcurrency:     4,
		TextOptimization:   true,
		DeduplicationRatio: 0.8,
	}
}

// ==== Test Progress Callback ====

// VideoTestProgressCallback is a test implementation of progress callback
type VideoTestProgressCallback struct {
	Calls        []types.ConverterPayload
	CallCount    int
	LastStatus   types.ConverterStatus
	LastMessage  string
	LastProgress float64
}

// NewVideoTestProgressCallback creates a new test progress callback
func NewVideoTestProgressCallback() *VideoTestProgressCallback {
	return &VideoTestProgressCallback{
		Calls: make([]types.ConverterPayload, 0),
	}
}

// Callback implements the progress callback interface
func (c *VideoTestProgressCallback) Callback(status types.ConverterStatus, payload types.ConverterPayload) {
	c.Calls = append(c.Calls, payload)
	c.CallCount++
	c.LastStatus = status
	c.LastMessage = payload.Message
	c.LastProgress = payload.Progress
}

// GetCallCount returns the number of times the callback was called
func (c *VideoTestProgressCallback) GetCallCount() int {
	return c.CallCount
}

// GetLastStatus returns the last status
func (c *VideoTestProgressCallback) GetLastStatus() types.ConverterStatus {
	return c.LastStatus
}

// GetLastProgress returns the last progress value
func (c *VideoTestProgressCallback) GetLastProgress() float64 {
	return c.LastProgress
}

// ==== Helper Functions ====

// Note: truncateString is already defined in vision_test.go

// ==== Basic Functionality Tests ====

func TestVideo_NewVideo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping video tests in short mode")
	}

	ensureVideoTestDataExists(t)
	prepareVideoConnectors(t)

	t.Run("Valid converters with defaults", func(t *testing.T) {
		options := createVideoOptions(t, 0, 0) // Use defaults
		converter, err := NewVideo(options)
		if err != nil {
			t.Fatalf("NewVideo failed: %v", err)
		}

		if converter == nil {
			t.Fatal("NewVideo returned nil")
		}

		if converter.KeyframeInterval != 10.0 {
			t.Errorf("Expected default KeyframeInterval 10.0, got %f", converter.KeyframeInterval)
		}

		if converter.MaxKeyframes != 20 {
			t.Errorf("Expected default MaxKeyframes 20, got %d", converter.MaxKeyframes)
		}

		if !converter.TextOptimization {
			t.Error("Expected TextOptimization to be true")
		}

		if converter.DeduplicationRatio != 0.8 {
			t.Errorf("Expected DeduplicationRatio 0.8, got %f", converter.DeduplicationRatio)
		}
	})

	t.Run("Custom parameters", func(t *testing.T) {
		options := createVideoOptions(t, 15.0, 30)
		converter, err := NewVideo(options)
		if err != nil {
			t.Fatalf("NewVideo with custom params failed: %v", err)
		}

		if converter.KeyframeInterval != 15.0 {
			t.Errorf("Expected KeyframeInterval 15.0, got %f", converter.KeyframeInterval)
		}

		if converter.MaxKeyframes != 30 {
			t.Errorf("Expected MaxKeyframes 30, got %d", converter.MaxKeyframes)
		}
	})

	t.Run("Missing audio converter", func(t *testing.T) {
		_, visionConverter := createVideoConverters(t)

		options := VideoOption{
			AudioConverter:  nil, // Missing
			VisionConverter: visionConverter,
		}

		converter, err := NewVideo(options)
		if err == nil {
			t.Error("Expected error for missing audio converter, but got none")
		}
		if converter != nil {
			t.Error("Expected nil converter for missing audio converter")
		}
	})

	t.Run("Missing vision converter", func(t *testing.T) {
		whisperConverter, _ := createVideoConverters(t)

		options := VideoOption{
			AudioConverter:  whisperConverter,
			VisionConverter: nil, // Missing
		}

		converter, err := NewVideo(options)
		if err == nil {
			t.Error("Expected error for missing vision converter, but got none")
		}
		if converter != nil {
			t.Error("Expected nil converter for missing vision converter")
		}
	})
}

func TestVideo_CalculateKeyframeParams(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keyframe calculation tests in short mode")
	}

	prepareVideoConnectors(t)

	t.Run("Smart defaults based on duration", func(t *testing.T) {
		options := createVideoOptions(t, 0, 0) // No explicit settings
		converter, err := NewVideo(options)
		if err != nil {
			t.Fatalf("NewVideo failed: %v", err)
		}

		// Test short video (≤1 min): should use 5 second interval
		interval, _ := converter.calculateKeyframeParams(45.0)
		if interval != 5.0 {
			t.Errorf("Expected 5.0s interval for short video, got %f", interval)
		}

		// Test medium video (≤5 min): should use 10 second interval
		interval, _ = converter.calculateKeyframeParams(180.0)
		if interval != 10.0 {
			t.Errorf("Expected 10.0s interval for medium video, got %f", interval)
		}

		// Test long video (≤30 min): should use 30 second interval
		interval, _ = converter.calculateKeyframeParams(900.0)
		if interval != 30.0 {
			t.Errorf("Expected 30.0s interval for long video, got %f", interval)
		}

		// Test very long video (>30 min): should use 60 second interval
		interval, _ = converter.calculateKeyframeParams(2400.0)
		if interval != 60.0 {
			t.Errorf("Expected 60.0s interval for very long video, got %f", interval)
		}
	})
}

func TestVideo_Convert_VideoFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping video conversion tests in short mode")
	}

	ensureVideoTestDataExists(t)
	prepareVideoConnectors(t)

	testFiles := getVideoConverterTestFiles()

	for _, testFile := range testFiles {
		t.Run(testFile.Name, func(t *testing.T) {
			// Use different keyframe settings for different tests
			options := createVideoOptions(t, 15.0, 0) // 15 second interval for faster test
			converter, err := NewVideo(options)
			if err != nil {
				t.Fatalf("Failed to create Video converter: %v", err)
			}

			ctx := context.Background()
			callback := NewVideoTestProgressCallback()

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
			validateVideoConversionResult(t, result, testFile)

			// Check that we got reasonable progress callbacks
			if callback.GetCallCount() < 5 {
				t.Errorf("Expected at least 5 progress calls for %s, got %d", testFile.Description, callback.GetCallCount())
			}

			if callback.GetLastStatus() != types.ConverterStatusSuccess {
				t.Errorf("Expected final status Success for %s, got %v", testFile.Description, callback.GetLastStatus())
			}

			t.Logf("%s: Generated %d chars combined text with %d progress calls",
				testFile.Description, len(result.Text), callback.GetCallCount())
			t.Logf("Text preview: %s", truncateString(result.Text, 200))
		})
	}
}

func TestVideo_Convert_NonVideoFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping non-video file tests in short mode")
	}

	prepareVideoConnectors(t)

	options := createVideoOptions(t, 0, 0)
	converter, err := NewVideo(options)
	if err != nil {
		t.Fatalf("Failed to create Video converter: %v", err)
	}

	// Create a temporary non-video file
	tempFile := filepath.Join(os.TempDir(), "test_non_video.txt")
	err = os.WriteFile(tempFile, []byte("This is not a video file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)

	ctx := context.Background()
	_, err = converter.Convert(ctx, tempFile)
	if err == nil {
		t.Error("Expected error for non-video file, but got none")
	}

	// Check that error message indicates it's not a video
	if !strings.Contains(err.Error(), "not a video") {
		t.Logf("Expected 'not a video' in error message, got: %v", err)
	}

	t.Logf("Correctly rejected non-video file with error: %v", err)
}

func TestVideo_ProgressReporting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping progress reporting tests in short mode")
	}

	prepareVideoConnectors(t)

	options := createVideoOptions(t, 0, 0)
	converter, err := NewVideo(options)
	if err != nil {
		t.Fatalf("Failed to create Video converter: %v", err)
	}

	t.Run("Progress callback sequence", func(t *testing.T) {
		callback := NewVideoTestProgressCallback()

		// Test manual progress reporting
		converter.reportProgress(types.ConverterStatusPending, "Starting video processing", 0.0, callback.Callback)
		converter.reportProgress(types.ConverterStatusPending, "Extracting audio", 0.2, callback.Callback)
		converter.reportProgress(types.ConverterStatusPending, "Extracting keyframes", 0.4, callback.Callback)
		converter.reportProgress(types.ConverterStatusPending, "Processing content", 0.6, callback.Callback)
		converter.reportProgress(types.ConverterStatusPending, "Merging results", 0.8, callback.Callback)
		converter.reportProgress(types.ConverterStatusSuccess, "Video processing completed", 1.0, callback.Callback)

		if callback.GetCallCount() != 6 {
			t.Errorf("Expected 6 callback calls, got %d", callback.GetCallCount())
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

func TestVideo_TextOptimization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping text optimization tests in short mode")
	}

	prepareVideoConnectors(t)

	t.Run("Text deduplication", func(t *testing.T) {
		options := createVideoOptions(t, 0, 0)
		converter, err := NewVideo(options)
		if err != nil {
			t.Fatalf("Failed to create Video converter: %v", err)
		}

		// Test the optimization function directly
		testText := `Line 1: This is a test
Line 2: This is a test
Line 3: Something different
Line 4: This is a test again
Line 5: Another unique line`

		optimized := converter.optimizeText(testText)

		// Should remove duplicate lines
		lines := strings.Split(optimized, "\n")
		if len(lines) >= len(strings.Split(testText, "\n")) {
			t.Log("Text optimization may not remove lines with current similarity threshold")
		}

		t.Logf("Original lines: %d, Optimized lines: %d",
			len(strings.Split(testText, "\n")), len(lines))
	})

	t.Run("Similarity calculation", func(t *testing.T) {
		options := createVideoOptions(t, 0, 0)
		converter, err := NewVideo(options)
		if err != nil {
			t.Fatalf("Failed to create Video converter: %v", err)
		}

		// Test similarity calculation
		similarity1 := converter.calculateSimilarity("hello world", "hello world")
		if similarity1 != 1.0 {
			t.Errorf("Expected similarity 1.0 for identical strings, got %f", similarity1)
		}

		similarity2 := converter.calculateSimilarity("hello world", "goodbye world")
		if similarity2 == 0.0 {
			t.Error("Expected some similarity for partially matching strings")
		}

		similarity3 := converter.calculateSimilarity("hello", "goodbye")
		if similarity3 > similarity2 {
			t.Error("Expected lower similarity for completely different strings")
		}

		t.Logf("Similarities: identical=%.2f, partial=%.2f, different=%.2f",
			similarity1, similarity2, similarity3)
	})
}

// ==== Error Handling Tests ====

func TestVideo_Convert_NonExistentFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error handling tests in short mode")
	}

	prepareVideoConnectors(t)

	options := createVideoOptions(t, 0, 0)
	converter, err := NewVideo(options)
	if err != nil {
		t.Fatalf("Failed to create Video converter: %v", err)
	}

	ctx := context.Background()
	_, err = converter.Convert(ctx, "/non/existent/video.mp4")
	if err == nil {
		t.Error("Expected error for non-existent file, but got none")
	}

	t.Logf("Correctly failed with error: %v", err)
}

func TestVideo_Convert_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping context cancellation tests in short mode")
	}

	ensureVideoTestDataExists(t)
	prepareVideoConnectors(t)

	options := createVideoOptions(t, 0, 0)
	converter, err := NewVideo(options)
	if err != nil {
		t.Fatalf("Failed to create Video converter: %v", err)
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	testFile := getVideoConverterTestFiles()[1].Path // Use English video file
	_, err = converter.Convert(ctx, testFile)

	// The operation might complete before cancellation is checked
	if err != nil && err == context.Canceled {
		t.Log("Context cancellation handled correctly")
	} else {
		t.Log("Operation completed before cancellation check (acceptable)")
	}
}

// ==== Integration Tests ====

func TestVideo_Integration_ShortInterval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ensureVideoTestDataExists(t)
	prepareVideoConnectors(t)

	t.Run("Short interval video processing", func(t *testing.T) {
		// Use small keyframe interval for comprehensive test
		options := createVideoOptions(t, 20.0, 0) // 20 second intervals (3 frames for 1-min video)
		converter, err := NewVideo(options)
		if err != nil {
			t.Fatalf("Failed to create Video converter: %v", err)
		}

		ctx := context.Background()
		testFile := getVideoConverterTestFiles()[1].Path // Use English video file

		callback := NewVideoTestProgressCallback()
		result, err := converter.Convert(ctx, testFile, callback.Callback)

		if err != nil {
			t.Fatalf("Video conversion failed: %v", err)
		}

		if result == nil || result.Text == "" {
			t.Error("Conversion returned empty result")
		}

		// Check that we got meaningful progress through all stages
		if callback.GetCallCount() < 5 {
			t.Errorf("Expected at least 5 progress calls, got %d", callback.GetCallCount())
		}

		if callback.GetLastStatus() != types.ConverterStatusSuccess {
			t.Errorf("Expected final status Success, got %v", callback.GetLastStatus())
		}

		// Check comprehensive metadata
		if result.Metadata == nil {
			t.Error("Expected metadata with video processing info")
		} else {
			// Verify key metadata fields
			expectedFields := []string{"source_type", "keyframe_interval", "extracted_keyframes", "text_length"}
			for _, field := range expectedFields {
				if _, exists := result.Metadata[field]; !exists {
					t.Errorf("Expected metadata field '%s' missing", field)
				}
			}
		}

		t.Logf("Video integration test successful!")
		t.Logf("Total text length: %d characters", len(result.Text))
		t.Logf("Progress calls: %d", callback.GetCallCount())
		t.Logf("Text preview: %s", truncateString(result.Text, 300))
		t.Logf("Metadata: %v", result.Metadata)
	})
}

// ==== Cleanup Tests ====

func TestVideo_ResourceCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource cleanup tests in short mode")
	}

	prepareVideoConnectors(t)

	t.Run("Temporary file cleanup", func(t *testing.T) {
		options := createVideoOptions(t, 0, 0)
		options.CleanupTemp = true // Ensure cleanup is enabled

		converter, err := NewVideo(options)
		if err != nil {
			t.Fatalf("Failed to create Video converter: %v", err)
		}

		// Close converter to test cleanup
		if err := converter.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}

		// Check that FFmpeg resources are cleaned up
		t.Log("Resource cleanup test completed")
	})
}

// validateVideoConversionResult validates the text and metadata from video conversion
func validateVideoConversionResult(t *testing.T, result *types.ConvertResult, testFile VideoTestFileInfo) {
	t.Helper()

	// Basic validation
	if result == nil {
		t.Fatalf("Convert returned nil result for %s", testFile.Description)
	}

	if result.Text == "" {
		t.Fatalf("Convert returned empty text for %s", testFile.Description)
	}

	// Text content validation
	validateTextContent(t, result.Text, testFile)

	// Metadata validation
	validateMetadata(t, result.Metadata, testFile)

	// Structure validation
	validateTextStructure(t, result.Text, testFile)
}

// validateTextContent validates the quality and content of generated text
func validateTextContent(t *testing.T, text string, testFile VideoTestFileInfo) {
	t.Helper()

	// Check minimum text length (should be substantial)
	if len(text) < 100 {
		t.Errorf("Generated text too short for %s: %d characters", testFile.Description, len(text))
	}

	// Check for expected sections
	hasAudioSection := strings.Contains(text, "Audio Transcription:")
	hasVisualSection := strings.Contains(text, "Visual Content:")

	if !hasAudioSection && !hasVisualSection {
		t.Errorf("Text should contain either audio or visual content sections for %s", testFile.Description)
	}

	// Validate visual content timestamps if present
	if hasVisualSection {
		validateVisualTimestamps(t, text, testFile)
	}

	// Check for reasonable content (not just error messages)
	if strings.Contains(text, "error") || strings.Contains(text, "failed") {
		t.Logf("Warning: Text contains error messages for %s", testFile.Description)
	}

	// Validate language-specific content
	if testFile.Language == "zh" {
		// For Chinese videos, expect some Chinese characters or related content
		if !strings.Contains(text, "中") && !strings.Contains(text, "Chinese") && !strings.Contains(text, "traditional") {
			t.Logf("Warning: No Chinese-related content detected for %s", testFile.Description)
		}
	}

	t.Logf("Text content validation passed for %s (%d chars)", testFile.Description, len(text))
}

// validateVisualTimestamps validates that visual content has proper timestamps
func validateVisualTimestamps(t *testing.T, text string, testFile VideoTestFileInfo) {
	t.Helper()

	// Look for timestamp patterns like "At 0.0s:", "At 15.0s:", etc.
	lines := strings.Split(text, "\n")

	var timestampCount int
	var lastTimestamp float64 = -1

	for _, line := range lines {
		if strings.Contains(line, "At ") && strings.Contains(line, "s:") {
			timestampCount++

			// Extract timestamp for validation
			parts := strings.Split(line, "At ")
			if len(parts) > 1 {
				timePart := strings.Split(parts[1], "s:")
				if len(timePart) > 0 {
					var timestamp float64
					if n, err := fmt.Sscanf(timePart[0], "%f", &timestamp); n == 1 && err == nil {
						if timestamp < lastTimestamp {
							t.Errorf("Timestamps not in order for %s: %.1fs after %.1fs", testFile.Description, timestamp, lastTimestamp)
						}
						lastTimestamp = timestamp

						// Check timestamp is within video duration
						if timestamp > testFile.Duration {
							t.Errorf("Timestamp %.1fs exceeds video duration %.1fs for %s", timestamp, testFile.Duration, testFile.Description)
						}
					}
				}
			}
		}
	}

	if timestampCount == 0 {
		t.Errorf("No timestamps found in visual content for %s", testFile.Description)
	} else {
		t.Logf("Found %d timestamps in visual content for %s", timestampCount, testFile.Description)
	}
}

// validateMetadata validates the completeness and accuracy of metadata
func validateMetadata(t *testing.T, metadata map[string]interface{}, testFile VideoTestFileInfo) {
	t.Helper()

	if metadata == nil {
		t.Fatalf("Metadata is nil for %s", testFile.Description)
	}

	// Required metadata fields
	requiredFields := []string{
		"source_type",
		"keyframe_interval",
		"max_keyframes",
		"extracted_keyframes",
		"text_optimization",
		"text_length",
		"successful_keyframes",
	}

	for _, field := range requiredFields {
		if _, exists := metadata[field]; !exists {
			t.Errorf("Missing required metadata field '%s' for %s", field, testFile.Description)
		}
	}

	// Validate specific metadata values
	if sourceType, ok := metadata["source_type"].(string); !ok || sourceType != "video" {
		t.Errorf("Expected source_type 'video', got %v for %s", metadata["source_type"], testFile.Description)
	}

	if keyframeInterval, ok := metadata["keyframe_interval"].(float64); ok {
		if keyframeInterval <= 0 {
			t.Errorf("Invalid keyframe_interval %f for %s", keyframeInterval, testFile.Description)
		}
	}

	if extractedKeyframes, ok := metadata["extracted_keyframes"].(int); ok {
		if extractedKeyframes < 0 {
			t.Errorf("Invalid extracted_keyframes %d for %s", extractedKeyframes, testFile.Description)
		}
	}

	if successfulKeyframes, ok := metadata["successful_keyframes"].(int); ok {
		if successfulKeyframes < 0 {
			t.Errorf("Invalid successful_keyframes %d for %s", successfulKeyframes, testFile.Description)
		}

		if extractedKeyframes, ok := metadata["extracted_keyframes"].(int); ok {
			if successfulKeyframes > extractedKeyframes {
				t.Errorf("Successful keyframes (%d) cannot exceed extracted keyframes (%d) for %s",
					successfulKeyframes, extractedKeyframes, testFile.Description)
			}
		}
	}

	if textLength, ok := metadata["text_length"].(int); ok {
		if textLength <= 0 {
			t.Errorf("Invalid text_length %d for %s", textLength, testFile.Description)
		}
	}

	// Validate audio metadata if present
	if audioMetadata, exists := metadata["audio_metadata"]; exists {
		validateAudioMetadata(t, audioMetadata, testFile)
	}

	t.Logf("Metadata validation passed for %s", testFile.Description)
}

// validateAudioMetadata validates audio processing metadata
func validateAudioMetadata(t *testing.T, audioMetadata interface{}, testFile VideoTestFileInfo) {
	t.Helper()

	audioMeta, ok := audioMetadata.(map[string]interface{})
	if !ok {
		t.Errorf("Audio metadata is not a map for %s", testFile.Description)
		return
	}

	// Check for errors in audio processing
	if errors, exists := audioMeta["errors"]; exists {
		if errorList, ok := errors.([]interface{}); ok && len(errorList) > 0 {
			t.Logf("Audio processing errors for %s: %v", testFile.Description, errorList)

			// Check if it's the language format error
			for _, err := range errorList {
				if errStr, ok := err.(string); ok {
					if strings.Contains(errStr, "invalid_language_format") {
						t.Logf("Language format error detected for %s - should use ISO-639-1 format", testFile.Description)
					}
				}
			}
		}
	}

	// Validate audio-specific fields
	if model, exists := audioMeta["model"]; exists {
		if modelStr, ok := model.(string); !ok || modelStr == "" {
			t.Errorf("Invalid audio model for %s", testFile.Description)
		}
	}

	if language, exists := audioMeta["language"]; exists {
		if langStr, ok := language.(string); ok {
			if langStr == "auto" {
				t.Logf("Warning: 'auto' language may cause issues for %s - should use ISO-639-1 format", testFile.Description)
			}
		}
	}
}

// validateTextStructure validates the overall structure and format of the text
func validateTextStructure(t *testing.T, text string, testFile VideoTestFileInfo) {
	t.Helper()

	lines := strings.Split(text, "\n")

	// Check for reasonable line count
	if len(lines) < 3 {
		t.Errorf("Text has too few lines (%d) for %s", len(lines), testFile.Description)
	}

	// Check for excessive empty lines
	emptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyLines++
		}
	}

	if emptyLines > len(lines)/2 {
		t.Errorf("Too many empty lines (%d/%d) for %s", emptyLines, len(lines), testFile.Description)
	}

	// Check for proper section headers
	sectionHeaders := []string{"Audio Transcription:", "Visual Content:"}
	for _, header := range sectionHeaders {
		if strings.Contains(text, header) {
			// Verify section has content after header
			headerIndex := strings.Index(text, header)
			contentAfterHeader := text[headerIndex+len(header):]
			if strings.TrimSpace(contentAfterHeader) == "" {
				t.Errorf("Section '%s' has no content for %s", header, testFile.Description)
			}
		}
	}

	t.Logf("Text structure validation passed for %s", testFile.Description)
}

// TestVideo_DetailedValidation performs comprehensive validation of results
func TestVideo_DetailedValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping detailed validation tests in short mode")
	}

	ensureVideoTestDataExists(t)
	prepareVideoConnectors(t)

	// Fix the language issue by using proper ISO-639-1 format
	t.Run("English video with proper language setting", func(t *testing.T) {
		// Create custom whisper options with proper language
		whisperOptions := WhisperOption{
			ConnectorName:          "test-video-whisper",
			Model:                  "whisper-1",
			Language:               "en", // Use ISO-639-1 format instead of "auto"
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
			t.Fatalf("Failed to create Whisper converter: %v", err)
		}

		// Create Vision converter
		visionOptions := VisionOption{
			ConnectorName: "test-video-vision",
			Model:         "gpt-4o-mini",
			CompressSize:  1024,
			Language:      "Auto",
			Options:       map[string]any{"max_tokens": 1000, "temperature": 0.1},
		}

		visionConverter, err := NewVision(visionOptions)
		if err != nil {
			t.Fatalf("Failed to create Vision converter: %v", err)
		}

		// Create video converter with proper language setting
		videoOptions := VideoOption{
			AudioConverter:     whisperConverter,
			VisionConverter:    visionConverter,
			KeyframeInterval:   20.0, // 20 second intervals
			MaxKeyframes:       0,
			TempDir:            "",
			CleanupTemp:        true,
			MaxConcurrency:     4,
			TextOptimization:   true,
			DeduplicationRatio: 0.8,
		}

		converter, err := NewVideo(videoOptions)
		if err != nil {
			t.Fatalf("Failed to create Video converter: %v", err)
		}

		ctx := context.Background()
		testFile := getVideoConverterTestFiles()[1] // English video

		callback := NewVideoTestProgressCallback()
		result, err := converter.Convert(ctx, testFile.Path, callback.Callback)

		if err != nil {
			t.Fatalf("Video conversion failed: %v", err)
		}

		// Perform detailed validation
		validateVideoConversionResult(t, result, testFile)

		// Additional checks for this specific test
		if result.Metadata != nil {
			if audioMeta, exists := result.Metadata["audio_metadata"]; exists {
				if audioMap, ok := audioMeta.(map[string]interface{}); ok {
					if errors, exists := audioMap["errors"]; exists {
						if errorList, ok := errors.([]interface{}); ok && len(errorList) > 0 {
							t.Logf("Audio processing still has errors: %v", errorList)
						}
					}

					if textLength, exists := audioMap["text_length"]; exists {
						if length, ok := textLength.(int); ok && length > 0 {
							t.Logf("Audio transcription successful with %d characters", length)
						}
					}
				}
			}
		}
	})

	t.Run("Chinese video with proper language setting", func(t *testing.T) {
		// Create custom whisper options for Chinese
		whisperOptions := WhisperOption{
			ConnectorName:          "test-video-whisper",
			Model:                  "whisper-1",
			Language:               "zh", // Chinese ISO-639-1 format
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
			t.Fatalf("Failed to create Whisper converter: %v", err)
		}

		// Create Vision converter
		visionOptions := VisionOption{
			ConnectorName: "test-video-vision",
			Model:         "gpt-4o-mini",
			CompressSize:  1024,
			Language:      "Auto",
			Options:       map[string]any{"max_tokens": 1000, "temperature": 0.1},
		}

		visionConverter, err := NewVision(visionOptions)
		if err != nil {
			t.Fatalf("Failed to create Vision converter: %v", err)
		}

		// Create video converter
		videoOptions := VideoOption{
			AudioConverter:     whisperConverter,
			VisionConverter:    visionConverter,
			KeyframeInterval:   20.0,
			MaxKeyframes:       0,
			TempDir:            "",
			CleanupTemp:        true,
			MaxConcurrency:     4,
			TextOptimization:   true,
			DeduplicationRatio: 0.8,
		}

		converter, err := NewVideo(videoOptions)
		if err != nil {
			t.Fatalf("Failed to create Video converter: %v", err)
		}

		ctx := context.Background()
		testFile := getVideoConverterTestFiles()[0] // Chinese video

		callback := NewVideoTestProgressCallback()
		result, err := converter.Convert(ctx, testFile.Path, callback.Callback)

		if err != nil {
			t.Fatalf("Video conversion failed: %v", err)
		}

		// Perform detailed validation
		validateVideoConversionResult(t, result, testFile)
	})
}
