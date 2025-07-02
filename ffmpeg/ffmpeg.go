// Package ffmpeg provides a cross-platform wrapper for the FFmpeg command line tool.
//
// This package supports Linux, macOS (Darwin), and Windows (reserved for future implementation).
// The implementation uses runtime OS detection to provide platform-specific optimizations:
//
//   - Linux: Uses standard FFmpeg with NVIDIA GPU support detection
//   - macOS: Uses VideoToolbox hardware acceleration for Apple Silicon and Intel Macs
//   - Windows: Reserved for future implementation
//
// Features:
//   - Multi-threaded processing with configurable concurrency
//   - GPU acceleration support (platform-dependent)
//   - Progress callback support
//   - Batch processing capabilities
//   - Process management with timeouts
//   - Audio conversion (any format to WAV/MP3)
//   - Video conversion (any format to MP4)
//   - Audio extraction from video
//   - Keyframe extraction from video
//   - Streaming support
//
// Usage:
//
//	ffmpeg := NewFFmpeg()
//	config := Config{
//		MaxProcesses: 4,
//		MaxThreads:   8,
//		EnableGPU:    true,
//	}
//	err := ffmpeg.Init(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer ffmpeg.Close()
//
//	// Convert audio
//	err = ffmpeg.Convert(ctx, ConvertOptions{
//		Input:  "input.mp3",
//		Output: "output.wav",
//		Format: "wav",
//	})
//
//	// Extract audio from video
//	err = ffmpeg.Extract(ctx, ExtractOptions{
//		Input:  "video.mp4",
//		Output: "audio.wav",
//		Type:   "audio",
//		Format: "wav",
//	})
//
// The package requires FFmpeg 6+ to be installed on the system.
package ffmpeg

import (
	"fmt"
	"runtime"
)

// NewFFmpeg creates a new FFmpeg instance based on the current operating system
func NewFFmpeg() FFmpeg {
	switch runtime.GOOS {
	case "linux":
		return NewLinuxFFmpeg()
	case "darwin":
		return NewDarwinFFmpeg()
	case "windows":
		return NewWindowsFFmpeg()
	default:
		panic(fmt.Sprintf("unsupported operating system: %s", runtime.GOOS))
	}
}
