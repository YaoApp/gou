package ffmpeg

import (
	"context"
	"time"
)

// Config represents the configuration settings for FFmpeg wrapper.
// It contains all the necessary settings for FFmpeg operation including paths,
// concurrency limits, GPU settings, and process management options.
type Config struct {
	// Command paths
	FFmpegPath  string `json:"ffmpeg_path"`  // Path to the FFmpeg executable
	FFprobePath string `json:"ffprobe_path"` // Path to the FFprobe executable

	// Working directory
	WorkDir string `json:"work_dir"` // Directory where FFmpeg operations are performed

	// Concurrency settings
	MaxProcesses int `json:"max_processes"` // Maximum number of concurrent FFmpeg processes
	MaxThreads   int `json:"max_threads"`   // Maximum number of threads per FFmpeg process

	// Process limits
	MaxProcessTime time.Duration `json:"max_process_time"` // Maximum execution time per process (0 means no limit)

	// GPU settings
	EnableGPU bool `json:"enable_gpu"` // Whether to enable GPU acceleration
	GPUIndex  int  `json:"gpu_index"`  // GPU index to use (-1 means auto detect)
}

// ProgressInfo contains real-time progress information for FFmpeg operations.
// It provides detailed information about the current state of media processing.
type ProgressInfo struct {
	Duration    time.Duration `json:"duration"`     // Total duration of the media file
	CurrentTime time.Duration `json:"current_time"` // Current processing time position
	Progress    float64       `json:"progress"`     // Progress percentage (0.0 - 1.0)
	Speed       float64       `json:"speed"`        // Processing speed multiplier
	Bitrate     string        `json:"bitrate"`      // Current bitrate
	FPS         float64       `json:"fps"`          // Frames per second
}

// ProgressCallback defines the function signature for progress monitoring callbacks.
// It receives ProgressInfo updates during FFmpeg operations.
type ProgressCallback func(info ProgressInfo)

// ConvertOptions contains all the options for media conversion operations.
// It specifies input/output files, format settings, and processing options.
type ConvertOptions struct {
	Input      string            `json:"input"`   // Input file path
	Output     string            `json:"output"`  // Output file path
	Format     string            `json:"format"`  // Output format (wav, mp3, mp4, etc.)
	Quality    string            `json:"quality"` // Quality setting for the output
	Options    map[string]string `json:"options"` // Additional FFmpeg command line options
	Stream     bool              `json:"stream"`  // Whether to support streaming
	OnProgress ProgressCallback  `json:"-"`       // Progress callback function
}

// ExtractOptions contains all the options for media extraction operations.
// It specifies what type of content to extract and how to process it.
type ExtractOptions struct {
	Input      string            `json:"input"`   // Input file path
	Output     string            `json:"output"`  // Output file path
	Type       string            `json:"type"`    // Type of extraction (audio, keyframe)
	Format     string            `json:"format"`  // Output format
	Options    map[string]string `json:"options"` // Additional FFmpeg options
	Stream     bool              `json:"stream"`  // Whether to support streaming
	OnProgress ProgressCallback  `json:"-"`       // Progress callback function
}

// BatchJob represents a single job in the batch processing queue.
// It contains the job metadata and processing options.
type BatchJob struct {
	ID      string      `json:"id"`      // Unique job identifier
	Type    string      `json:"type"`    // Job type (convert, extract)
	Options interface{} `json:"options"` // Job options (ConvertOptions or ExtractOptions)
	Status  string      `json:"status"`  // Current job status
	Error   string      `json:"error"`   // Error message if job failed
}

// SystemInfo contains information about the system and available FFmpeg tools.
// It provides details about the operating system, FFmpeg versions, and hardware capabilities.
type SystemInfo struct {
	OS      string   `json:"os"`              // Operating system name
	GPUs    []string `json:"gpus"`            // List of available GPUs
	FFmpeg  string   `json:"ffmpeg_version"`  // FFmpeg version string
	FFprobe string   `json:"ffprobe_version"` // FFprobe version string
}

// FFmpeg defines the interface for all FFmpeg wrapper implementations.
// It provides a unified API for media processing operations across different platforms.
type FFmpeg interface {
	// Initialize and configuration
	Init(config Config) error           // Initialize the FFmpeg wrapper with configuration
	GetConfig() Config                  // Get the current configuration
	GetSystemInfo() (SystemInfo, error) // Get system information and capabilities

	// Single operations
	Convert(ctx context.Context, options ConvertOptions) error // Convert a single media file
	Extract(ctx context.Context, options ExtractOptions) error // Extract content from a single media file

	// Batch operations
	ConvertBatch(ctx context.Context, jobs []ConvertOptions) error // Convert multiple media files
	ExtractBatch(ctx context.Context, jobs []ExtractOptions) error // Extract from multiple media files

	// Job management
	AddJob(job BatchJob) string         // Add a job to the processing queue
	GetJob(id string) (BatchJob, error) // Get job status by ID
	CancelJob(id string) error          // Cancel a job by ID
	ListJobs() []BatchJob               // List all jobs in the queue

	// Process management
	GetActiveProcesses() int // Get the number of active processes
	KillAllProcesses() error // Kill all active processes

	// Cleanup
	Close() error // Close and cleanup resources
}

// Default formats for audio and video conversion
const (
	DefaultAudioFormat = "wav" // Default audio output format
	DefaultVideoFormat = "mp4" // Default video output format

	// Job types for batch processing
	JobTypeConvert = "convert" // Conversion job type
	JobTypeExtract = "extract" // Extraction job type

	// Job status constants
	JobStatusPending   = "pending"   // Job is waiting to be processed
	JobStatusRunning   = "running"   // Job is currently being processed
	JobStatusCompleted = "completed" // Job has completed successfully
	JobStatusFailed    = "failed"    // Job has failed with an error
)
