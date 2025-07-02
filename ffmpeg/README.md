# FFmpeg Package

A cross-platform Go wrapper for the FFmpeg command line tool supporting Linux, macOS (Darwin), and Windows (reserved for future implementation).

## Features

- **Cross-platform support**: Automatically selects the appropriate implementation based on runtime OS
- **Multi-threaded processing**: Configurable concurrency and thread limits
- **GPU acceleration**: Platform-specific hardware acceleration support
- **Progress callbacks**: Real-time progress monitoring
- **Batch processing**: Convert/extract multiple files in batches
- **Process management**: Timeout controls and process limits
- **Audio conversion**: Any format to WAV/MP3 with streaming support
- **Video conversion**: Any format to MP4 with streaming support
- **Audio extraction**: Extract audio from video files
- **Keyframe extraction**: Extract keyframes from video files

## Platform Support

- **Linux**: Standard FFmpeg with NVIDIA GPU detection
- **macOS**: VideoToolbox hardware acceleration for Apple Silicon and Intel Macs
- **Windows**: Reserved for future implementation

## Requirements

- FFmpeg 6+ installed on the system
- FFprobe (usually comes with FFmpeg)

## Installation

```bash
go get github.com/yaoapp/gou/ffmpeg
```

## Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/yaoapp/gou/ffmpeg"
)

func main() {
    // Create FFmpeg instance (automatically selects OS-specific implementation)
    ff := ffmpeg.NewFFmpeg()

    // Configure
    config := ffmpeg.Config{
        MaxProcesses:   4,
        MaxThreads:     8,
        EnableGPU:      true,
        MaxProcessTime: 5 * time.Minute,
        WorkDir:        "/tmp",
    }

    // Initialize
    err := ff.Init(config)
    if err != nil {
        log.Fatal(err)
    }
    defer ff.Close()

    ctx := context.Background()

    // Convert audio
    err = ff.Convert(ctx, ffmpeg.ConvertOptions{
        Input:  "input.mp3",
        Output: "output.wav",
        Format: "wav",
    })
    if err != nil {
        log.Printf("Convert failed: %v", err)
    }

    // Extract audio from video
    err = ff.Extract(ctx, ffmpeg.ExtractOptions{
        Input:  "video.mp4",
        Output: "audio.wav",
        Type:   "audio",
        Format: "wav",
    })
    if err != nil {
        log.Printf("Extract failed: %v", err)
    }
}
```

## Advanced Usage

### Progress Monitoring

```go
err = ff.Convert(ctx, ffmpeg.ConvertOptions{
    Input:  "input.mp4",
    Output: "output.wav",
    Format: "wav",
    OnProgress: func(info ffmpeg.ProgressInfo) {
        fmt.Printf("Progress: %.2f%% - Speed: %.2fx\n",
            info.Progress*100, info.Speed)
    },
})
```

### Batch Processing

```go
jobs := []ffmpeg.ConvertOptions{
    {Input: "file1.mp3", Output: "file1.wav", Format: "wav"},
    {Input: "file2.mp3", Output: "file2.wav", Format: "wav"},
    {Input: "file3.mp3", Output: "file3.wav", Format: "wav"},
}

err = ff.ConvertBatch(ctx, jobs)
```

### Custom Configuration

```go
config := ffmpeg.Config{
    FFmpegPath:     "/usr/local/bin/ffmpeg",
    FFprobePath:    "/usr/local/bin/ffprobe",
    WorkDir:        "/tmp/ffmpeg_work",
    MaxProcesses:   2,
    MaxThreads:     4,
    MaxProcessTime: 10 * time.Minute,
    EnableGPU:      true,
    GPUIndex:       0,  // Use specific GPU (0-based index, -1 for auto)
}
```

### Job Management

```go
// Add a job to the queue
job := ffmpeg.BatchJob{
    Type: ffmpeg.JobTypeConvert,
    Options: ffmpeg.ConvertOptions{
        Input:  "input.mp4",
        Output: "output.wav",
        Format: "wav",
    },
}

jobID := ff.AddJob(job)

// Check job status
retrievedJob, err := ff.GetJob(jobID)
if err == nil {
    fmt.Printf("Job Status: %s\n", retrievedJob.Status)
}

// Cancel a job
err = ff.CancelJob(jobID)
```

## System Information

```go
info, err := ff.GetSystemInfo()
if err == nil {
    fmt.Printf("OS: %s\n", info.OS)
    fmt.Printf("FFmpeg: %s\n", info.FFmpeg)
    fmt.Printf("FFprobe: %s\n", info.FFprobe)
    fmt.Printf("GPUs: %v\n", info.GPUs)
}
```

## Architecture

The package uses a factory pattern with runtime OS detection:

- `types.go`: Interface definitions and common types
- `linux.go`: Linux-specific implementation
- `darwin.go`: macOS-specific implementation
- `windows.go`: Windows placeholder implementation
- `ffmpeg.go`: Factory function and package documentation

The `NewFFmpeg()` function automatically returns the appropriate implementation based on `runtime.GOOS`.

## Error Handling

All methods return errors for proper error handling. Windows implementation currently returns "not yet supported" errors as placeholders.

## License

This package is part of the YaoApp project and follows the same license terms.
