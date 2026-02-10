package ffmpeg

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yaoapp/gou/process"
)

// ==== Test Helper Functions ====

// getProcessTestDataDir returns the ffmpeg test data directory
func getProcessTestDataDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(currentFile)
	testDataDir := filepath.Join(currentDir, "tests")
	absPath, err := filepath.Abs(testDataDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to get absolute path for ffmpeg test data dir: %v", err))
	}
	return absPath
}

// ensureProcessTestDataExists checks if test data exists
func ensureProcessTestDataExists(t *testing.T) {
	t.Helper()
	testDir := getProcessTestDataDir()
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skipf("FFmpeg test data directory does not exist: %s", testDir)
	}
}

// isFFmpegAvailable checks if ffmpeg is available on the system
func isFFmpegAvailable(t *testing.T) bool {
	t.Helper()
	info, err := defaultFFmpeg.GetSystemInfo()
	if err != nil {
		return false
	}
	return info.FFmpeg != ""
}

// getProcessTestVideoFile returns a test MP4 video file path
func getProcessTestVideoFile() string {
	return filepath.Join(getProcessTestDataDir(), "sample_small.mp4")
}

// getProcessTestAudioFile returns a test MP3 audio file path
func getProcessTestAudioFile() string {
	return filepath.Join(getProcessTestDataDir(), "english_speech_test.mp3")
}

// getProcessTestSpeechVideoFile returns a test video with speech file path
func getProcessTestSpeechVideoFile() string {
	return filepath.Join(getProcessTestDataDir(), "english_speech_video.mp4")
}

// ==== Process Registration Tests ====

func TestProcessRegistration(t *testing.T) {
	// Verify that ffmpeg process handlers are registered
	handlers := []string{"ffmpeg.info", "ffmpeg.convert", "ffmpeg.extractaudio", "ffmpeg.chunkaudio"}
	for _, name := range handlers {
		p, err := process.Of(name, nil)
		if err != nil {
			t.Errorf("Process %s should be registered, got error: %v", name, err)
			continue
		}
		if p == nil {
			t.Errorf("Process %s returned nil", name)
		}
	}
}

// ==== ffmpeg.Info Tests ====

func TestProcessInfo_Video(t *testing.T) {
	ensureProcessTestDataExists(t)
	if !isFFmpegAvailable(t) {
		t.Skip("FFmpeg not available, skipping test")
	}

	testFile := getProcessTestVideoFile()
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	p, err := process.Of("ffmpeg.info", testFile)
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	result := p.Run()
	if result == nil {
		t.Fatal("ffmpeg.Info returned nil")
	}

	info, ok := result.(*MediaInfo)
	if !ok {
		t.Fatalf("Expected *MediaInfo, got %T", result)
	}

	if info.Duration <= 0 {
		t.Errorf("Expected Duration > 0, got %f", info.Duration)
	}

	if info.FileSize <= 0 {
		t.Errorf("Expected FileSize > 0, got %d", info.FileSize)
	}

	t.Logf("Video: duration=%.2fs, %dx%d, video=%s, audio=%s, size=%d",
		info.Duration, info.Width, info.Height,
		info.VideoCodec, info.AudioCodec, info.FileSize)
}

func TestProcessInfo_Audio(t *testing.T) {
	ensureProcessTestDataExists(t)
	if !isFFmpegAvailable(t) {
		t.Skip("FFmpeg not available, skipping test")
	}

	testFile := getProcessTestAudioFile()
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	p, err := process.Of("ffmpeg.info", testFile)
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	result := p.Run()
	if result == nil {
		t.Fatal("ffmpeg.Info returned nil")
	}

	info, ok := result.(*MediaInfo)
	if !ok {
		t.Fatalf("Expected *MediaInfo, got %T", result)
	}

	if info.Duration <= 0 {
		t.Errorf("Expected Duration > 0, got %f", info.Duration)
	}

	// Note: AudioCodec may not be populated depending on ffprobe output parsing
	if info.AudioCodec != "" {
		t.Logf("AudioCodec: %s", info.AudioCodec)
	}

	// Audio-only file should have no video dimensions
	t.Logf("Audio: duration=%.2fs, audio=%s, size=%d",
		info.Duration, info.AudioCodec, info.FileSize)
}

func TestProcessInfo_AllTestFiles(t *testing.T) {
	ensureProcessTestDataExists(t)
	if !isFFmpegAvailable(t) {
		t.Skip("FFmpeg not available, skipping test")
	}

	testFiles := []struct {
		name string
		path string
	}{
		{"sample_small.mp4", filepath.Join(getProcessTestDataDir(), "sample_small.mp4")},
		{"sample_1mb.mp4", filepath.Join(getProcessTestDataDir(), "sample_1mb.mp4")},
		{"english_speech_test.mp3", filepath.Join(getProcessTestDataDir(), "english_speech_test.mp3")},
		{"chinese_speech_test.mp3", filepath.Join(getProcessTestDataDir(), "chinese_speech_test.mp3")},
		{"english_speech_test.wav", filepath.Join(getProcessTestDataDir(), "english_speech_test.wav")},
		{"chinese_short.ogg", filepath.Join(getProcessTestDataDir(), "chinese_short.ogg")},
		{"english_speech_video.mp4", filepath.Join(getProcessTestDataDir(), "english_speech_video.mp4")},
		{"chinese_speech_video.mp4", filepath.Join(getProcessTestDataDir(), "chinese_speech_video.mp4")},
	}

	for _, tf := range testFiles {
		t.Run(tf.name, func(t *testing.T) {
			if _, err := os.Stat(tf.path); os.IsNotExist(err) {
				t.Skipf("Test file not found: %s", tf.path)
				return
			}

			p, err := process.Of("ffmpeg.info", tf.path)
			if err != nil {
				t.Fatalf("Failed to create process: %v", err)
			}

			result := p.Run()
			if result == nil {
				t.Fatal("ffmpeg.Info returned nil")
			}

			info, ok := result.(*MediaInfo)
			if !ok {
				t.Fatalf("Expected *MediaInfo, got %T", result)
			}

			if info.Duration <= 0 {
				t.Errorf("Expected Duration > 0, got %f", info.Duration)
			}

			t.Logf("%s: duration=%.2fs, %dx%d, video=%s, audio=%s",
				tf.name, info.Duration, info.Width, info.Height,
				info.VideoCodec, info.AudioCodec)
		})
	}
}

func TestProcessInfo_NonExistentFile(t *testing.T) {
	if !isFFmpegAvailable(t) {
		t.Skip("FFmpeg not available, skipping test")
	}

	p, err := process.Of("ffmpeg.info", "/non/existent/video.mp4")
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic for non-existent file, but got none")
		}
		t.Logf("Correctly panicked with: %v", r)
	}()

	p.Run()
}

// ==== ffmpeg.Convert Tests ====

func TestProcessConvert(t *testing.T) {
	ensureProcessTestDataExists(t)
	if !isFFmpegAvailable(t) {
		t.Skip("FFmpeg not available, skipping test")
	}

	testFile := getProcessTestAudioFile()
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	t.Run("Convert MP3 to WAV", func(t *testing.T) {
		outputPath := filepath.Join(os.TempDir(), "gou_ffmpeg_process_test_convert.wav")
		defer os.Remove(outputPath)

		config := map[string]interface{}{
			"format":      "wav",
			"output_path": outputPath,
		}

		p, err := process.Of("ffmpeg.convert", testFile, config)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		if result == nil {
			t.Fatal("ffmpeg.Convert returned nil")
		}

		resultPath, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", result)
		}

		if resultPath != outputPath {
			t.Errorf("Expected output path %s, got %s", outputPath, resultPath)
		}

		// Verify output file exists
		fi, err := os.Stat(resultPath)
		if err != nil {
			t.Fatalf("Output file does not exist: %v", err)
		}

		if fi.Size() <= 0 {
			t.Error("Output file is empty")
		}

		t.Logf("Converted MP3 to WAV: %s (%d bytes)", resultPath, fi.Size())
	})

	t.Run("Convert with bitrate and sample rate", func(t *testing.T) {
		outputPath := filepath.Join(os.TempDir(), "gou_ffmpeg_process_test_convert_opts.wav")
		defer os.Remove(outputPath)

		config := map[string]interface{}{
			"format":      "wav",
			"output_path": outputPath,
			"sample_rate": 16000,
		}

		p, err := process.Of("ffmpeg.convert", testFile, config)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		resultPath, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", result)
		}

		fi, err := os.Stat(resultPath)
		if err != nil {
			t.Fatalf("Output file does not exist: %v", err)
		}

		t.Logf("Converted with sample_rate=16000: %s (%d bytes)", resultPath, fi.Size())
	})

	t.Run("Convert with auto output path", func(t *testing.T) {
		config := map[string]interface{}{
			"format": "wav",
		}

		p, err := process.Of("ffmpeg.convert", testFile, config)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		resultPath, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", result)
		}

		defer os.Remove(resultPath)

		if !strings.HasSuffix(resultPath, ".wav") {
			t.Errorf("Expected .wav extension, got %s", resultPath)
		}

		fi, err := os.Stat(resultPath)
		if err != nil {
			t.Fatalf("Output file does not exist: %v", err)
		}

		t.Logf("Converted with auto path: %s (%d bytes)", resultPath, fi.Size())
	})
}

func TestProcessConvert_NonExistentFile(t *testing.T) {
	if !isFFmpegAvailable(t) {
		t.Skip("FFmpeg not available, skipping test")
	}

	config := map[string]interface{}{
		"format": "wav",
	}

	p, err := process.Of("ffmpeg.convert", "/non/existent/audio.mp3", config)
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic for non-existent file, but got none")
		}
		t.Logf("Correctly panicked with: %v", r)
	}()

	p.Run()
}

// ==== ffmpeg.ExtractAudio Tests ====

func TestProcessExtractAudio(t *testing.T) {
	ensureProcessTestDataExists(t)
	if !isFFmpegAvailable(t) {
		t.Skip("FFmpeg not available, skipping test")
	}

	testFile := getProcessTestSpeechVideoFile()
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	t.Run("Extract audio from video as MP3", func(t *testing.T) {
		outputPath := filepath.Join(os.TempDir(), "gou_ffmpeg_process_test_extract.mp3")
		defer os.Remove(outputPath)

		config := map[string]interface{}{
			"format":      "mp3",
			"output_path": outputPath,
		}

		p, err := process.Of("ffmpeg.extractaudio", testFile, config)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		if result == nil {
			t.Fatal("ffmpeg.ExtractAudio returned nil")
		}

		resultPath, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", result)
		}

		if resultPath != outputPath {
			t.Errorf("Expected output path %s, got %s", outputPath, resultPath)
		}

		fi, err := os.Stat(resultPath)
		if err != nil {
			t.Fatalf("Output file does not exist: %v", err)
		}

		if fi.Size() <= 0 {
			t.Error("Output file is empty")
		}

		t.Logf("Extracted audio: %s (%d bytes)", resultPath, fi.Size())
	})

	t.Run("Extract audio with default format", func(t *testing.T) {
		outputPath := filepath.Join(os.TempDir(), "gou_ffmpeg_process_test_extract_default.mp3")
		defer os.Remove(outputPath)

		// No config - should use default format (mp3)
		p, err := process.Of("ffmpeg.extractaudio", testFile)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		resultPath, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", result)
		}

		defer os.Remove(resultPath)

		fi, err := os.Stat(resultPath)
		if err != nil {
			t.Fatalf("Output file does not exist: %v", err)
		}

		t.Logf("Extracted audio with defaults: %s (%d bytes)", resultPath, fi.Size())
	})

	t.Run("Extract audio as WAV", func(t *testing.T) {
		outputPath := filepath.Join(os.TempDir(), "gou_ffmpeg_process_test_extract.wav")
		defer os.Remove(outputPath)

		config := map[string]interface{}{
			"format":      "wav",
			"output_path": outputPath,
			"sample_rate": 16000,
		}

		p, err := process.Of("ffmpeg.extractaudio", testFile, config)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		resultPath, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", result)
		}

		fi, err := os.Stat(resultPath)
		if err != nil {
			t.Fatalf("Output file does not exist: %v", err)
		}

		t.Logf("Extracted audio as WAV: %s (%d bytes)", resultPath, fi.Size())
	})
}

func TestProcessExtractAudio_NonExistentFile(t *testing.T) {
	if !isFFmpegAvailable(t) {
		t.Skip("FFmpeg not available, skipping test")
	}

	p, err := process.Of("ffmpeg.extractaudio", "/non/existent/video.mp4")
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic for non-existent file, but got none")
		}
		t.Logf("Correctly panicked with: %v", r)
	}()

	p.Run()
}

// ==== ffmpeg.ChunkAudio Tests ====

func TestProcessChunkAudio(t *testing.T) {
	ensureProcessTestDataExists(t)
	if !isFFmpegAvailable(t) {
		t.Skip("FFmpeg not available, skipping test")
	}

	// Use a longer audio file for chunking
	testFile := filepath.Join(getProcessTestDataDir(), "english_sample.wav")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		// Fall back to shorter file
		testFile = getProcessTestAudioFile()
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Skipf("No audio test file found")
		}
	}

	t.Run("Chunk audio with short duration", func(t *testing.T) {
		outputDir := filepath.Join(os.TempDir(), "gou_ffmpeg_process_test_chunk")
		defer os.RemoveAll(outputDir)

		config := map[string]interface{}{
			"max_duration":      10.0,
			"silence_threshold": -40.0,
			"output_dir":        outputDir,
		}

		p, err := process.Of("ffmpeg.chunkaudio", testFile, config)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		if result == nil {
			t.Fatal("ffmpeg.ChunkAudio returned nil")
		}

		chunkResult, ok := result.(*ChunkResult)
		if !ok {
			t.Fatalf("Expected *ChunkResult, got %T", result)
		}

		if chunkResult.TotalChunks <= 0 {
			t.Errorf("Expected TotalChunks > 0, got %d", chunkResult.TotalChunks)
		}

		if len(chunkResult.Chunks) == 0 {
			t.Error("Expected at least 1 chunk")
		}

		// Verify each chunk file exists
		for _, chunk := range chunkResult.Chunks {
			if _, err := os.Stat(chunk.FilePath); os.IsNotExist(err) {
				t.Errorf("Chunk file does not exist: %s", chunk.FilePath)
			}
			t.Logf("Chunk %d: %.2f-%.2fs (%.2fs), %d bytes, path=%s",
				chunk.Index, chunk.StartTime, chunk.EndTime, chunk.Duration,
				chunk.FileSize, filepath.Base(chunk.FilePath))
		}

		t.Logf("Total chunks: %d, Total size: %d bytes",
			chunkResult.TotalChunks, chunkResult.TotalSize)
	})

	t.Run("Chunk audio with default config", func(t *testing.T) {
		p, err := process.Of("ffmpeg.chunkaudio", testFile)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		if result == nil {
			t.Fatal("ffmpeg.ChunkAudio returned nil")
		}

		chunkResult, ok := result.(*ChunkResult)
		if !ok {
			t.Fatalf("Expected *ChunkResult, got %T", result)
		}

		// Cleanup
		if chunkResult.OutputDir != "" {
			defer os.RemoveAll(chunkResult.OutputDir)
		}

		t.Logf("Default config: %d chunks, %d bytes total",
			chunkResult.TotalChunks, chunkResult.TotalSize)
	})
}

func TestProcessChunkAudio_NonExistentFile(t *testing.T) {
	if !isFFmpegAvailable(t) {
		t.Skip("FFmpeg not available, skipping test")
	}

	p, err := process.Of("ffmpeg.chunkaudio", "/non/existent/audio.mp3")
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic for non-existent file, but got none")
		}
		t.Logf("Correctly panicked with: %v", r)
	}()

	p.Run()
}

// ==== Comprehensive Integration Test ====

func TestProcessFFmpeg_Comprehensive(t *testing.T) {
	ensureProcessTestDataExists(t)
	if !isFFmpegAvailable(t) {
		t.Skip("FFmpeg not available, skipping test")
	}

	speechVideo := getProcessTestSpeechVideoFile()
	if _, err := os.Stat(speechVideo); os.IsNotExist(err) {
		t.Skipf("Speech video test file not found: %s", speechVideo)
	}

	// Step 1: Get video info
	t.Run("Pipeline: Info → ExtractAudio → Convert → ChunkAudio", func(t *testing.T) {
		// Step 1: Get info
		p, err := process.Of("ffmpeg.info", speechVideo)
		if err != nil {
			t.Fatalf("Failed to create info process: %v", err)
		}

		result := p.Run()
		info, ok := result.(*MediaInfo)
		if !ok {
			t.Fatalf("Expected *MediaInfo, got %T", result)
		}

		t.Logf("Step 1 - Info: duration=%.2fs, %dx%d", info.Duration, info.Width, info.Height)

		// Step 2: Extract audio
		audioOutput := filepath.Join(os.TempDir(), "gou_ffmpeg_comprehensive_audio.mp3")
		defer os.Remove(audioOutput)

		p, err = process.Of("ffmpeg.extractaudio", speechVideo, map[string]interface{}{
			"format":      "mp3",
			"output_path": audioOutput,
		})
		if err != nil {
			t.Fatalf("Failed to create extract process: %v", err)
		}

		result = p.Run()
		extractedPath, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", result)
		}

		fi, err := os.Stat(extractedPath)
		if err != nil {
			t.Fatalf("Extracted audio file not found: %v", err)
		}

		t.Logf("Step 2 - ExtractAudio: %s (%d bytes)", extractedPath, fi.Size())

		// Step 3: Convert extracted audio to WAV
		wavOutput := filepath.Join(os.TempDir(), "gou_ffmpeg_comprehensive_audio.wav")
		defer os.Remove(wavOutput)

		p, err = process.Of("ffmpeg.convert", extractedPath, map[string]interface{}{
			"format":      "wav",
			"output_path": wavOutput,
			"sample_rate": 16000,
		})
		if err != nil {
			t.Fatalf("Failed to create convert process: %v", err)
		}

		result = p.Run()
		convertedPath, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", result)
		}

		fi, err = os.Stat(convertedPath)
		if err != nil {
			t.Fatalf("Converted audio file not found: %v", err)
		}

		t.Logf("Step 3 - Convert: %s (%d bytes)", convertedPath, fi.Size())

		// Step 4: Chunk the audio
		chunkDir := filepath.Join(os.TempDir(), "gou_ffmpeg_comprehensive_chunks")
		defer os.RemoveAll(chunkDir)

		p, err = process.Of("ffmpeg.chunkaudio", extractedPath, map[string]interface{}{
			"max_duration":      30.0,
			"silence_threshold": -40.0,
			"output_dir":        chunkDir,
		})
		if err != nil {
			t.Fatalf("Failed to create chunk process: %v", err)
		}

		result = p.Run()
		chunkResult, ok := result.(*ChunkResult)
		if !ok {
			t.Fatalf("Expected *ChunkResult, got %T", result)
		}

		t.Logf("Step 4 - ChunkAudio: %d chunks, %d bytes total",
			chunkResult.TotalChunks, chunkResult.TotalSize)

		for _, chunk := range chunkResult.Chunks {
			t.Logf("  Chunk %d: %.2f-%.2fs (%.2fs), %d bytes",
				chunk.Index, chunk.StartTime, chunk.EndTime, chunk.Duration, chunk.FileSize)
		}
	})
}
