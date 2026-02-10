package ffmpeg

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// defaultFFmpeg is the default FFmpeg instance
var defaultFFmpeg FFmpeg

func init() {
	process.RegisterGroup("ffmpeg", map[string]process.Handler{
		"info":         ProcessInfo,
		"convert":      ProcessConvert,
		"extractaudio": ProcessExtractAudio,
		"chunkaudio":   ProcessChunkAudio,
	})

	// Create and initialize default FFmpeg instance
	defaultFFmpeg = NewFFmpeg()
	if err := defaultFFmpeg.Init(Config{
		MaxProcesses: 4,
		MaxThreads:   8,
	}); err != nil {
		// Log warning but don't panic - ffmpeg may not be installed
		log.Printf("Warning: FFmpeg initialization failed: %v", err)
	}
}

// ProcessInfo ffmpeg.Info
// Get media file information including duration, dimensions, codecs, and file size.
//
// Args:
//   - filePath string - Path to the media file
//
// Returns: *MediaInfo - Media file information
//
// Usage:
//
//	var info = Process("ffmpeg.Info", "/path/to/video.mp4")
//	// Returns: {"duration": 120.5, "width": 1920, "height": 1080, "audio_codec": "aac", ...}
func ProcessInfo(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	filePath := process.ArgsString(0)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	info, err := defaultFFmpeg.GetMediaInfo(ctx, filePath)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return info
}

// ProcessConvert ffmpeg.Convert
// Convert media file to another format.
//
// Args:
//   - inputPath string - Path to the input media file
//   - config map[string]interface{} - Convert configuration
//
// Config fields:
//   - format string - Output format (e.g. "mp3", "wav", "mp4")
//   - output_path string - Output file path (optional, default: temp file)
//   - bitrate string - Audio bitrate (e.g. "128k") (optional)
//   - sample_rate int - Audio sample rate (e.g. 16000) (optional)
//
// Returns: string - Path to the output file
//
// Note: If output_path is not specified, files are created in OS temp directory.
// TS scripts without "system" fs access MUST provide output_path explicitly.
//
// Usage:
//
//	var outputPath = Process("ffmpeg.Convert", "/path/to/input.mp4", {"format": "mp3", "bitrate": "128k"})
//	// Returns: "/tmp/out.mp3"
func ProcessConvert(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	inputPath := process.ArgsString(0)
	configMap := process.ArgsMap(1)

	opts := ConvertOptions{
		Input: inputPath,
	}

	// Parse format
	if format, ok := configMap["format"].(string); ok && format != "" {
		opts.Format = format
	} else {
		exception.New("format is required for ffmpeg.Convert", 400).Throw()
	}

	// Parse output_path
	if outputPath, ok := configMap["output_path"].(string); ok && outputPath != "" {
		opts.Output = outputPath
	} else {
		// Generate unique temp file path to avoid concurrent overwrites
		// Create and immediately remove the file to reserve a unique name for ffmpeg
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("ffmpeg_convert_*.%s", opts.Format))
		if err != nil {
			exception.New(fmt.Sprintf("failed to create temp file: %s", err.Error()), 500).Throw()
		}
		opts.Output = tmpFile.Name()
		tmpFile.Close()
		os.Remove(opts.Output) // ffmpeg needs the file to not exist
	}

	// Parse additional options (keys need "-" prefix for ffmpeg args)
	opts.Options = make(map[string]string)
	if bitrate, ok := configMap["bitrate"].(string); ok && bitrate != "" {
		opts.Options["-b:a"] = bitrate
	}
	if sampleRate, ok := configMap["sample_rate"]; ok {
		opts.Options["-ar"] = fmt.Sprintf("%d", toInt(sampleRate))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	err := defaultFFmpeg.Convert(ctx, opts)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return opts.Output
}

// ProcessExtractAudio ffmpeg.ExtractAudio
// Extract audio track from a video file.
//
// Args:
//   - inputPath string - Path to the video file
//   - config map[string]interface{} - Extract configuration (optional)
//
// Config fields:
//   - format string - Output audio format (default: "mp3")
//   - output_path string - Output file path (optional, default: temp file)
//   - bitrate string - Audio bitrate (e.g. "128k") (optional)
//   - sample_rate int - Audio sample rate (e.g. 16000) (optional)
//
// Returns: string - Path to the extracted audio file
//
// Note: If output_path is not specified, files are created in OS temp directory.
// TS scripts without "system" fs access MUST provide output_path explicitly.
//
// Usage:
//
//	var audioPath = Process("ffmpeg.ExtractAudio", "/path/to/video.mp4", {"format": "mp3"})
//	// Returns: "/tmp/audio.mp3"
func ProcessExtractAudio(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	inputPath := process.ArgsString(0)

	format := "mp3"
	outputPath := ""
	options := make(map[string]string)

	// Parse optional config
	if process.NumOfArgs() > 1 {
		configMap := process.ArgsMap(1)

		if f, ok := configMap["format"].(string); ok && f != "" {
			format = f
		}
		if op, ok := configMap["output_path"].(string); ok && op != "" {
			outputPath = op
		}
		if bitrate, ok := configMap["bitrate"].(string); ok && bitrate != "" {
			options["-b:a"] = bitrate
		}
		if sampleRate, ok := configMap["sample_rate"]; ok {
			options["-ar"] = fmt.Sprintf("%d", toInt(sampleRate))
		}
	}

	// Generate unique output path if not specified
	if outputPath == "" {
		// Create and immediately remove the file to reserve a unique name for ffmpeg
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("ffmpeg_audio_*.%s", format))
		if err != nil {
			exception.New(fmt.Sprintf("failed to create temp file: %s", err.Error()), 500).Throw()
		}
		outputPath = tmpFile.Name()
		tmpFile.Close()
		os.Remove(outputPath) // ffmpeg needs the file to not exist
	}

	opts := ExtractOptions{
		Input:   inputPath,
		Output:  outputPath,
		Type:    "audio",
		Format:  format,
		Options: options,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	err := defaultFFmpeg.Extract(ctx, opts)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return outputPath
}

// ProcessChunkAudio ffmpeg.ChunkAudio
// Split audio file into chunks using silence detection.
//
// Args:
//   - inputPath string - Path to the audio file
//   - config map[string]interface{} - Chunk configuration (optional)
//
// Config fields:
//   - max_duration float64 - Max chunk duration in seconds (default: 600 = 10 min)
//   - max_size int64 - Max chunk size in bytes (default: 25000000 = 25MB)
//   - silence_threshold float64 - Silence detection threshold in dB (default: -40)
//   - output_dir string - Output directory for chunks (optional, default: auto temp dir)
//   - format string - Output format (default: "mp3")
//
// Returns: *ChunkResult - Chunk information including paths, durations, sizes
//
// Note: If output_dir is not specified, files are created in OS temp directory.
// TS scripts without "system" fs access MUST provide output_dir explicitly.
// The caller is responsible for cleaning up the output directory when done.
//
// Usage:
//
//	var result = Process("ffmpeg.ChunkAudio", "/path/to/audio.mp3", {"max_duration": 600})
//	// Returns: {"chunks": [...], "total_chunks": 3, "total_size": 12345, "output_dir": "/tmp/..."}
func ProcessChunkAudio(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	inputPath := process.ArgsString(0)

	opts := ChunkOptions{
		Input:                  inputPath,
		ChunkDuration:          600,      // Default: 10 minutes
		MaxChunkSize:           25000000, // Default: 25MB for Whisper
		SilenceThreshold:       -40,
		SilenceMinLength:       1.0,
		EnableSilenceDetection: true,
		Format:                 "mp3",
		OutputPrefix:           "chunk",
	}

	// Parse optional config
	if process.NumOfArgs() > 1 {
		configMap := process.ArgsMap(1)

		if maxDuration, ok := configMap["max_duration"]; ok {
			opts.ChunkDuration = toFloat64(maxDuration)
		}
		if maxSize, ok := configMap["max_size"]; ok {
			opts.MaxChunkSize = toInt64(maxSize)
		}
		if threshold, ok := configMap["silence_threshold"]; ok {
			opts.SilenceThreshold = toFloat64(threshold)
		}
		if outputDir, ok := configMap["output_dir"].(string); ok && outputDir != "" {
			opts.OutputDir = outputDir
		}
		if format, ok := configMap["format"].(string); ok && format != "" {
			opts.Format = format
		}
	}

	// Set default output_dir if not specified
	if opts.OutputDir == "" {
		tmpDir, err := os.MkdirTemp("", "ffmpeg_chunk_*")
		if err != nil {
			exception.New(fmt.Sprintf("failed to create temp directory: %s", err.Error()), 500).Throw()
		}
		opts.OutputDir = tmpDir
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	result, err := defaultFFmpeg.ChunkAudio(ctx, opts)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return result
}

// toInt converts various numeric types to int.
// Handles int, int64, float64, float32, and string (via strconv.Atoi).
func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	case string:
		n, _ := strconv.Atoi(val)
		return n
	default:
		return 0
	}
}

// toInt64 converts various numeric types to int64.
// Handles int, int64, float64, float32, and string.
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int64:
		return val
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	case string:
		n, _ := strconv.ParseInt(val, 10, 64)
		return n
	default:
		return 0
	}
}

// toFloat64 converts various numeric types to float64.
// Handles float64, float32, int, int64, and string.
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		n, _ := strconv.ParseFloat(val, 64)
		return n
	default:
		return 0
	}
}
