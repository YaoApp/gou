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

// ChunkOptions contains all the options for audio/video chunking operations.
// It specifies input file, output directory, and chunking parameters.
type ChunkOptions struct {
	Input                  string            `json:"input"`                    // Input file path
	OutputDir              string            `json:"output_dir"`               // Output directory for chunks
	OutputPrefix           string            `json:"output_prefix"`            // Prefix for output chunk files
	ChunkDuration          float64           `json:"chunk_duration"`           // Chunk duration in seconds
	SilenceThreshold       float64           `json:"silence_threshold"`        // Silence threshold in dB (e.g., -40)
	SilenceMinLength       float64           `json:"silence_min_length"`       // Minimum silence length in seconds to trigger a split
	Format                 string            `json:"format"`                   // Output format (wav, mp3, mp4, etc.)
	OverlapDuration        float64           `json:"overlap_duration"`         // Overlap between chunks in seconds
	EnableSilenceDetection bool              `json:"enable_silence_detection"` // Whether to use silence detection for chunking
	MaxChunkSize           int64             `json:"max_chunk_size"`           // Maximum chunk size in bytes (0 means no limit)
	Options                map[string]string `json:"options"`                  // Additional FFmpeg command line options
	OnProgress             ProgressCallback  `json:"-"`                        // Progress callback function
}

// ChunkInfo contains information about a generated chunk
type ChunkInfo struct {
	Index     int     `json:"index"`      // Chunk index (0-based)
	StartTime float64 `json:"start_time"` // Start time in seconds
	EndTime   float64 `json:"end_time"`   // End time in seconds
	Duration  float64 `json:"duration"`   // Duration in seconds
	FilePath  string  `json:"file_path"`  // Path to the chunk file
	FileSize  int64   `json:"file_size"`  // File size in bytes
	IsSilence bool    `json:"is_silence"` // Whether this chunk was created due to silence detection
}

// ChunkResult contains the results of a chunking operation
type ChunkResult struct {
	Chunks      []ChunkInfo `json:"chunks"`       // List of generated chunks
	TotalChunks int         `json:"total_chunks"` // Total number of chunks created
	TotalSize   int64       `json:"total_size"`   // Total size of all chunks in bytes
	OutputDir   string      `json:"output_dir"`   // Output directory path
}

// SystemInfo contains information about the system and available FFmpeg tools.
// It provides details about the operating system, FFmpeg versions, and hardware capabilities.
type SystemInfo struct {
	OS      string   `json:"os"`              // Operating system name
	GPUs    []string `json:"gpus"`            // List of available GPUs
	FFmpeg  string   `json:"ffmpeg_version"`  // FFmpeg version string
	FFprobe string   `json:"ffprobe_version"` // FFprobe version string
}

// MediaInfo contains information about a media file
type MediaInfo struct {
	Duration   float64 `json:"duration"`    // Duration in seconds
	Width      int     `json:"width"`       // Video width in pixels
	Height     int     `json:"height"`      // Video height in pixels
	Bitrate    string  `json:"bitrate"`     // Bitrate
	FrameRate  float64 `json:"frame_rate"`  // Frame rate (fps)
	AudioCodec string  `json:"audio_codec"` // Audio codec
	VideoCodec string  `json:"video_codec"` // Video codec
	FileSize   int64   `json:"file_size"`   // File size in bytes
}

// FFmpeg defines the interface for all FFmpeg wrapper implementations.
// It provides a unified API for media processing operations across different platforms.
type FFmpeg interface {
	// Initialize and configuration
	Init(config Config) error           // Initialize the FFmpeg wrapper with configuration
	GetConfig() Config                  // Get the current configuration
	GetSystemInfo() (SystemInfo, error) // Get system information and capabilities

	// Media information
	GetMediaInfo(ctx context.Context, inputFile string) (*MediaInfo, error) // Get media file information

	// Single operations
	Convert(ctx context.Context, options ConvertOptions) error // Convert a single media file
	Extract(ctx context.Context, options ExtractOptions) error // Extract content from a single media file

	// Chunking operations
	ChunkAudio(ctx context.Context, options ChunkOptions) (*ChunkResult, error) // Chunk audio file with silence detection
	ChunkVideo(ctx context.Context, options ChunkOptions) (*ChunkResult, error) // Chunk video file with silence detection

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
