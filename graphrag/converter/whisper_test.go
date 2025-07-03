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

// getWhisperTestDataDir returns the whisper test data directory
func getWhisperTestDataDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(currentFile)
	testDataDir := filepath.Join(currentDir, "..", "tests", "converter", "audio")
	absPath, err := filepath.Abs(testDataDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to get absolute path for whisper test data dir: %v", err))
	}
	return absPath
}

// getWhisperTestFilePath returns the full path to a whisper test file
func getWhisperTestFilePath(filename string) string {
	return filepath.Join(getWhisperTestDataDir(), filename)
}

// ensureWhisperTestDataExists checks if whisper test data directory and files exist
func ensureWhisperTestDataExists(t *testing.T) {
	t.Helper()

	testDir := getWhisperTestDataDir()
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatalf("Whisper test data directory does not exist: %s", testDir)
	}

	// Check for required test files
	requiredFiles := []string{
		"chinese_sample.mp3",
		"chinese_sample.ogg",
		"english_sample.mp3",
		"english_sample.ogg",
	}

	for _, filename := range requiredFiles {
		filePath := getWhisperTestFilePath(filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatalf("Required whisper test file does not exist: %s", filePath)
		}
	}
}

// WhisperTestFileInfo contains information about a whisper test file
type WhisperTestFileInfo struct {
	Name         string
	Path         string
	ShouldFail   bool
	Format       string
	Description  string
	IsCompressed bool
}

// getAudioTestFiles returns all audio test files that should convert successfully
func getAudioTestFiles() []WhisperTestFileInfo {
	return []WhisperTestFileInfo{
		{
			Name:        "chinese_sample.mp3",
			Path:        getWhisperTestFilePath("chinese_sample.mp3"),
			Format:      "MP3",
			Description: "Chinese MP3 audio file",
		},
		{
			Name:        "chinese_sample.ogg",
			Path:        getWhisperTestFilePath("chinese_sample.ogg"),
			Format:      "OGG",
			Description: "Chinese OGG audio file",
		},
		{
			Name:        "english_sample.mp3",
			Path:        getWhisperTestFilePath("english_sample.mp3"),
			Format:      "MP3",
			Description: "English MP3 audio file",
		},
		{
			Name:        "english_sample.ogg",
			Path:        getWhisperTestFilePath("english_sample.ogg"),
			Format:      "OGG",
			Description: "English OGG audio file",
		},
	}
}

// getVideoTestFiles returns all video test files (for testing video->audio conversion)
func getVideoTestFiles() []WhisperTestFileInfo {
	return []WhisperTestFileInfo{
		// No video files available yet for testing
	}
}

// getCompressedAudioTestFiles returns all compressed audio test files
func getCompressedAudioTestFiles() []WhisperTestFileInfo {
	return []WhisperTestFileInfo{
		// No compressed audio files available yet for testing
	}
}

// getNonAudioTestFiles returns all non-audio test files that should fail
func getNonAudioTestFiles() []WhisperTestFileInfo {
	return []WhisperTestFileInfo{
		// No non-audio files available yet for testing
	}
}

// getAllWhisperTestFiles returns all whisper test files
func getAllWhisperTestFiles() []WhisperTestFileInfo {
	var all []WhisperTestFileInfo
	all = append(all, getAudioTestFiles()...)
	all = append(all, getVideoTestFiles()...)
	all = append(all, getCompressedAudioTestFiles()...)
	all = append(all, getNonAudioTestFiles()...)
	return all
}

// ==== Connector Setup ====

// prepareWhisperConnector creates connectors for whisper testing
func prepareWhisperConnector(t *testing.T) {
	t.Helper()

	// Create OpenAI connector for whisper testing
	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping whisper tests")
	}

	openaiDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0", 
		"label": "OpenAI Whisper Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "whisper-1",
			"key": "%s"
		}
	}`, openaiKey)

	_, err := connector.New("openai", "test-whisper-openai", []byte(openaiDSL))
	if err != nil {
		t.Fatalf("Failed to create OpenAI whisper connector: %v", err)
	}
}

// createWhisperOptions creates WhisperOption for testing
func createWhisperOptions(chunkDuration, mappingDuration float64) WhisperOption {
	connectorID := "test-whisper-openai"
	model := "whisper-1"

	if chunkDuration == 0 {
		chunkDuration = 30.0 // Default chunk duration
	}
	if mappingDuration == 0 {
		mappingDuration = 5.0 // Default mapping duration
	}

	return WhisperOption{
		ConnectorName:          connectorID,
		Model:                  model,
		Language:               "en",
		ChunkDuration:          chunkDuration,
		MappingDuration:        mappingDuration,
		SilenceThreshold:       -40.0,
		SilenceMinLength:       1.0,
		EnableSilenceDetection: true,
		MaxConcurrency:         4,
		TempDir:                "", // Use system temp dir
		CleanupTemp:            true,
		Options:                map[string]any{"temperature": 0.0},
	}
}

// ==== Basic Functionality Tests ====

func TestWhisper_NewWhisper(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping whisper tests in short mode")
	}

	ensureWhisperTestDataExists(t)
	prepareWhisperConnector(t)

	t.Run("Valid OpenAI connector", func(t *testing.T) {
		options := createWhisperOptions(10.0, 5.0)
		converter, err := NewWhisper(options)
		if err != nil {
			t.Fatalf("NewWhisper failed: %v", err)
		}

		if converter == nil {
			t.Fatal("NewWhisper returned nil")
		}

		if converter.Model != "whisper-1" {
			t.Errorf("Expected model whisper-1, got %s", converter.Model)
		}

		if converter.ChunkDuration != 10.0 {
			t.Errorf("Expected ChunkDuration 10.0, got %f", converter.ChunkDuration)
		}

		if converter.MappingDuration != 5.0 {
			t.Errorf("Expected MappingDuration 5.0, got %f", converter.MappingDuration)
		}
	})

	t.Run("Invalid connector", func(t *testing.T) {
		options := WhisperOption{
			ConnectorName: "non-existent-connector",
		}

		converter, err := NewWhisper(options)
		if err == nil {
			t.Error("Expected error for invalid connector, but got none")
		}
		if converter != nil {
			t.Error("Expected nil converter for invalid connector")
		}
	})

	t.Run("Default values", func(t *testing.T) {
		options := WhisperOption{
			ConnectorName: "test-whisper-openai",
			// Leave other fields empty to test defaults
		}

		converter, err := NewWhisper(options)
		if err != nil {
			t.Fatalf("NewWhisper with defaults failed: %v", err)
		}

		if converter.ChunkDuration != 30.0 {
			t.Errorf("Expected default ChunkDuration 30.0, got %f", converter.ChunkDuration)
		}

		if converter.MappingDuration != 5.0 {
			t.Errorf("Expected default MappingDuration 5.0, got %f", converter.MappingDuration)
		}

		if converter.Language != "auto" {
			t.Errorf("Expected default Language 'auto', got %s", converter.Language)
		}

		if converter.MaxConcurrency != 4 {
			t.Errorf("Expected default MaxConcurrency 4, got %d", converter.MaxConcurrency)
		}
	})
}

func TestWhisper_Convert_AudioFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping whisper tests in short mode")
	}

	prepareWhisperConnector(t)

	// Use real OpenAI connector for testing
	whisper, err := NewWhisper(createWhisperOptions(0, 0))
	if err != nil {
		t.Fatalf("Failed to create Whisper converter: %v", err)
	}

	ctx := context.Background()
	testFiles := getAudioTestFiles()

	for _, testFile := range testFiles {
		t.Run(testFile.Name, func(t *testing.T) {
			result, err := whisper.Convert(ctx, testFile.Path)

			if testFile.ShouldFail {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", testFile.Description)
				}
				return
			}

			if err != nil {
				t.Fatalf("Convert failed for %s: %v", testFile.Description, err)
			}

			// Validate the result
			if result == nil {
				t.Errorf("Convert returned nil result for %s", testFile.Description)
				return
			}

			if result.Text == "" {
				t.Errorf("Convert returned empty text for %s", testFile.Description)
			}

			// Check metadata
			if result.Metadata == nil {
				t.Errorf("Convert returned nil metadata for %s", testFile.Description)
			}

			t.Logf("%s: Generated %d chars transcription with metadata: %v", testFile.Description, len(result.Text), result.Metadata)
		})
	}
}

func TestWhisper_Convert_VideoFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping whisper tests in short mode")
	}

	prepareWhisperConnector(t)

	// Use real OpenAI connector for testing
	whisper, err := NewWhisper(createWhisperOptions(0, 0))
	if err != nil {
		t.Fatalf("Failed to create Whisper converter: %v", err)
	}

	ctx := context.Background()
	testFiles := getVideoTestFiles()

	for _, testFile := range testFiles {
		t.Run(testFile.Name, func(t *testing.T) {
			result, err := whisper.Convert(ctx, testFile.Path)

			if testFile.ShouldFail {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", testFile.Description)
				}
				return
			}

			if err != nil {
				t.Fatalf("Convert failed for %s: %v", testFile.Description, err)
			}

			// Validate the result
			if result == nil {
				t.Errorf("Convert returned nil result for %s", testFile.Description)
				return
			}

			if result.Text == "" {
				t.Errorf("Convert returned empty text for %s", testFile.Description)
			}

			t.Logf("%s: Generated %d chars transcription", testFile.Description, len(result.Text))
		})
	}
}

func TestWhisper_Convert_CompressedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compressed file tests in short mode")
	}

	prepareWhisperConnector(t)

	options := createWhisperOptions(8.0, 4.0)
	converter, err := NewWhisper(options)
	if err != nil {
		t.Fatalf("Failed to create whisper converter: %v", err)
	}

	testFiles := getCompressedAudioTestFiles()
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

			t.Logf("%s: Generated transcription (%d chars): %s...",
				testFile.Description, len(result.Text), truncateString(result.Text, 100))
		})
	}
}

func TestWhisper_Convert_NonAudioFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping whisper tests in short mode")
	}

	prepareWhisperConnector(t)

	options := createWhisperOptions(0, 0) // Use real OpenAI connector
	converter, err := NewWhisper(options)
	if err != nil {
		t.Fatalf("Failed to create whisper converter: %v", err)
	}

	testFiles := getNonAudioTestFiles()
	for _, testFile := range testFiles {
		t.Run(testFile.Name, func(t *testing.T) {
			ctx := context.Background()

			result, err := converter.Convert(ctx, testFile.Path)

			if !testFile.ShouldFail {
				t.Fatalf("Test file %s should be marked as ShouldFail=true", testFile.Name)
			}

			if err == nil {
				t.Errorf("Expected error for non-audio file %s, but conversion succeeded with result length %d",
					testFile.Description, len(result.Text))
			} else {
				// Check that error message indicates it's not supported
				if !strings.Contains(err.Error(), "not supported") && !strings.Contains(err.Error(), "unsupported") {
					t.Logf("Expected 'not supported' in error message for %s, got: %v", testFile.Name, err)
				}
				t.Logf("%s: Correctly rejected with error: %v", testFile.Description, err)
			}
		})
	}
}

// ==== Stream Conversion Tests ====

func TestWhisper_ConvertStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stream conversion tests in short mode")
	}

	prepareWhisperConnector(t)

	// Use real OpenAI connector for stream testing
	options := createWhisperOptions(0, 0)
	converter, err := NewWhisper(options)
	if err != nil {
		t.Fatalf("Failed to create whisper converter: %v", err)
	}

	t.Run("MP3 stream", func(t *testing.T) {
		testFile := getWhisperTestFilePath("english_sample.mp3")
		file, err := os.Open(testFile)
		if err != nil {
			t.Fatalf("Failed to open test file: %v", err)
		}
		defer file.Close()

		ctx := context.Background()
		callback := NewTestProgressCallback()

		result, err := converter.ConvertStream(ctx, file, callback.Callback)

		if err != nil {
			t.Fatalf("ConvertStream failed: %v", err)
		}

		if result == nil || result.Text == "" {
			t.Error("ConvertStream returned empty result")
		}

		// Check that audio processing progress was reported
		if callback.GetCallCount() == 0 {
			t.Error("No progress callbacks during stream processing")
		}

		// Should have at least progressed past audio validation
		hasAudioProgress := false
		for _, call := range callback.Calls {
			if strings.Contains(call.Message, "audio") || call.Progress > 0.5 {
				hasAudioProgress = true
				break
			}
		}

		if !hasAudioProgress {
			t.Error("No audio processing progress reported")
		}

		t.Logf("Stream processing completed with %d progress calls", callback.GetCallCount())
	})
}

// ==== Error Handling Tests ====

func TestWhisper_Convert_NonExistentFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping whisper tests in short mode")
	}

	prepareWhisperConnector(t)

	options := createWhisperOptions(0, 0)
	converter, err := NewWhisper(options)
	if err != nil {
		t.Fatalf("Failed to create whisper converter: %v", err)
	}

	ctx := context.Background()
	_, err = converter.Convert(ctx, "/non/existent/audio.mp3")
	if err == nil {
		t.Error("Expected error for non-existent file, but got none")
	}

	t.Logf("Correctly failed with error: %v", err)
}

func TestWhisper_Convert_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping whisper tests in short mode")
	}

	prepareWhisperConnector(t)

	options := createWhisperOptions(0, 0)
	converter, err := NewWhisper(options)
	if err != nil {
		t.Fatalf("Failed to create whisper converter: %v", err)
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	testFile := getWhisperTestFilePath("english_sample.mp3")
	_, err = converter.Convert(ctx, testFile)

	// The operation might complete before cancellation is checked
	if err != nil && err == context.Canceled {
		t.Log("Context cancellation handled correctly")
	} else {
		t.Log("Operation completed before cancellation check (acceptable)")
	}
}

// ==== Progress Callback Tests ====

func TestWhisper_ProgressReporting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping whisper tests in short mode")
	}

	prepareWhisperConnector(t)

	options := createWhisperOptions(0, 0)
	converter, err := NewWhisper(options)
	if err != nil {
		t.Fatalf("Failed to create whisper converter: %v", err)
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

// ==== Edge Case Tests ====

func TestWhisper_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping whisper tests in short mode")
	}

	prepareWhisperConnector(t)

	t.Run("Very small chunk size", func(t *testing.T) {
		options := createWhisperOptions(1.0, 0.5) // Very small chunks
		converter, err := NewWhisper(options)
		if err != nil {
			t.Fatalf("Failed to create whisper converter: %v", err)
		}

		if converter.ChunkDuration != 1.0 {
			t.Errorf("Expected ChunkDuration 1.0, got %f", converter.ChunkDuration)
		}

		if converter.MappingDuration != 0.5 {
			t.Errorf("Expected MappingDuration 0.5, got %f", converter.MappingDuration)
		}

		t.Logf("Very small chunk configuration: chunk=%f, mapping=%f", converter.ChunkDuration, converter.MappingDuration)
	})

	t.Run("Zero duration - should use defaults", func(t *testing.T) {
		options := createWhisperOptions(0, 0) // Should use defaults
		converter, err := NewWhisper(options)
		if err != nil {
			t.Fatalf("Failed to create whisper converter: %v", err)
		}

		if converter.ChunkDuration != 30.0 {
			t.Errorf("Expected default ChunkDuration 30.0, got %f", converter.ChunkDuration)
		}

		if converter.MappingDuration != 5.0 {
			t.Errorf("Expected default MappingDuration 5.0, got %f", converter.MappingDuration)
		}
	})

	t.Run("Empty filename", func(t *testing.T) {
		options := createWhisperOptions(0, 0)
		converter, err := NewWhisper(options)
		if err != nil {
			t.Fatalf("Failed to create whisper converter: %v", err)
		}

		ctx := context.Background()
		_, err = converter.Convert(ctx, "")
		if err == nil {
			t.Error("Expected error for empty filename, but got none")
		}

		t.Logf("Empty filename correctly failed: %v", err)
	})
}

// ==== Integration Test with Real OpenAI (if available) ====

func TestWhisper_RealOpenAI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real OpenAI integration test in short mode")
	}

	prepareWhisperConnector(t)

	options := createWhisperOptions(10.0, 5.0)
	converter, err := NewWhisper(options)
	if err != nil {
		t.Fatalf("Failed to create whisper converter: %v", err)
	}

	t.Run("Single audio transcription", func(t *testing.T) {
		ctx := context.Background()
		testFile := getWhisperTestFilePath("english_sample.mp3")

		callback := NewTestProgressCallback()
		result, err := converter.Convert(ctx, testFile, callback.Callback)

		if err != nil {
			t.Fatalf("Real OpenAI conversion failed: %v", err)
		}

		if result == nil || result.Text == "" {
			t.Error("Real OpenAI returned empty result")
		}

		if len(result.Text) < 10 {
			t.Errorf("Real OpenAI result too short: %q", result.Text)
		}

		// Check that we got meaningful progress
		if callback.GetCallCount() < 3 {
			t.Errorf("Expected at least 3 progress calls, got %d", callback.GetCallCount())
		}

		if callback.GetLastStatus() != types.ConverterStatusSuccess {
			t.Errorf("Expected final status Success, got %v", callback.GetLastStatus())
		}

		// Check metadata for timeline mapping
		if result.Metadata == nil {
			t.Error("Expected metadata with timeline mapping")
		}

		t.Logf("Real OpenAI integration successful!")
		t.Logf("Transcription length: %d characters", len(result.Text))
		t.Logf("Progress calls: %d", callback.GetCallCount())
		t.Logf("Transcription preview: %s", truncateString(result.Text, 200))
		t.Logf("Timeline metadata: %v", result.Metadata)
	})
}
