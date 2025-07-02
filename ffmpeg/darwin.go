package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// DarwinFFmpeg represents the macOS-specific implementation of FFmpeg wrapper.
// It provides full FFmpeg functionality with VideoToolbox hardware acceleration support.
type DarwinFFmpeg struct {
	config    Config               // FFmpeg configuration
	processes map[string]*exec.Cmd // Active processes map
	jobs      map[string]BatchJob  // Job queue
	mu        sync.RWMutex         // Mutex for processes map
	jobMu     sync.RWMutex         // Mutex for jobs map
}

// NewDarwinFFmpeg creates and returns a new macOS FFmpeg instance.
// Initializes the internal maps for process and job management.
func NewDarwinFFmpeg() FFmpeg {
	return &DarwinFFmpeg{
		processes: make(map[string]*exec.Cmd),
		jobs:      make(map[string]BatchJob),
	}
}

// Init initializes the macOS FFmpeg wrapper with the provided configuration.
// Automatically detects common FFmpeg installation paths on macOS and sets default values.
func (f *DarwinFFmpeg) Init(config Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Set default paths if not provided
	if config.FFmpegPath == "" {
		// Check common macOS locations
		paths := []string{
			"ffmpeg",
			"/usr/local/bin/ffmpeg",
			"/opt/homebrew/bin/ffmpeg",
			"/usr/bin/ffmpeg",
		}
		for _, path := range paths {
			if _, err := exec.LookPath(path); err == nil {
				config.FFmpegPath = path
				break
			}
		}
		if config.FFmpegPath == "" {
			config.FFmpegPath = "ffmpeg"
		}
	}

	if config.FFprobePath == "" {
		// Check common macOS locations
		paths := []string{
			"ffprobe",
			"/usr/local/bin/ffprobe",
			"/opt/homebrew/bin/ffprobe",
			"/usr/bin/ffprobe",
		}
		for _, path := range paths {
			if _, err := exec.LookPath(path); err == nil {
				config.FFprobePath = path
				break
			}
		}
		if config.FFprobePath == "" {
			config.FFprobePath = "ffprobe"
		}
	}

	// Set default work directory
	if config.WorkDir == "" {
		config.WorkDir = "/tmp"
	}

	// Set default values
	if config.MaxProcesses <= 0 {
		config.MaxProcesses = runtime.NumCPU()
	}
	if config.MaxThreads <= 0 {
		config.MaxThreads = runtime.NumCPU()
	}

	// Verify FFmpeg and FFprobe exist
	if err := f.verifyCommands(config); err != nil {
		return fmt.Errorf("command verification failed: %v", err)
	}

	f.config = config
	return nil
}

// GetConfig returns the current configuration of the macOS FFmpeg instance.
func (f *DarwinFFmpeg) GetConfig() Config {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.config
}

// GetSystemInfo retrieves system information including OS, FFmpeg version, and available GPUs.
// Detects Apple Silicon GPUs and AMD/Intel graphics cards available on macOS.
func (f *DarwinFFmpeg) GetSystemInfo() (SystemInfo, error) {
	info := SystemInfo{
		OS: "darwin",
	}

	// Get FFmpeg version
	if version, err := f.getVersion(f.config.FFmpegPath); err == nil {
		info.FFmpeg = version
	}

	// Get FFprobe version
	if version, err := f.getVersion(f.config.FFprobePath); err == nil {
		info.FFprobe = version
	}

	// Detect GPUs
	gpus, _ := f.detectGPUs()
	info.GPUs = gpus

	return info, nil
}

// Convert performs media file conversion using the specified options.
// Supports VideoToolbox hardware acceleration on macOS with Apple Silicon and Intel Macs.
func (f *DarwinFFmpeg) Convert(ctx context.Context, options ConvertOptions) error {
	if f.GetActiveProcesses() >= f.config.MaxProcesses {
		return fmt.Errorf("maximum processes (%d) reached", f.config.MaxProcesses)
	}

	args := f.buildConvertArgs(options)
	return f.executeCommand(ctx, args, options.OnProgress)
}

// Extract extracts media content (audio or keyframes) from input files.
// Supports audio extraction and keyframe extraction with configurable output formats.
func (f *DarwinFFmpeg) Extract(ctx context.Context, options ExtractOptions) error {
	if f.GetActiveProcesses() >= f.config.MaxProcesses {
		return fmt.Errorf("maximum processes (%d) reached", f.config.MaxProcesses)
	}

	args := f.buildExtractArgs(options)
	return f.executeCommand(ctx, args, options.OnProgress)
}

// ConvertBatch performs batch conversion of multiple media files.
// Processes each conversion job sequentially with proper error handling.
func (f *DarwinFFmpeg) ConvertBatch(ctx context.Context, jobs []ConvertOptions) error {
	// Implementation for batch conversion
	for _, job := range jobs {
		if err := f.Convert(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

// ExtractBatch performs batch extraction from multiple media files.
// Processes each extraction job sequentially with proper error handling.
func (f *DarwinFFmpeg) ExtractBatch(ctx context.Context, jobs []ExtractOptions) error {
	// Implementation for batch extraction
	for _, job := range jobs {
		if err := f.Extract(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

// AddJob adds a new job to the processing queue and returns the job ID.
// Generates a unique job ID if not provided and sets initial status to pending.
func (f *DarwinFFmpeg) AddJob(job BatchJob) string {
	f.jobMu.Lock()
	defer f.jobMu.Unlock()

	if job.ID == "" {
		job.ID = fmt.Sprintf("job_%d", time.Now().UnixNano())
	}
	job.Status = JobStatusPending
	f.jobs[job.ID] = job
	return job.ID
}

// GetJob retrieves a job by its ID from the processing queue.
// Returns an error if the job with the specified ID is not found.
func (f *DarwinFFmpeg) GetJob(id string) (BatchJob, error) {
	f.jobMu.RLock()
	defer f.jobMu.RUnlock()

	job, exists := f.jobs[id]
	if !exists {
		return BatchJob{}, fmt.Errorf("job %s not found", id)
	}
	return job, nil
}

// CancelJob cancels a running or queued job by its ID.
// If the job is currently running, attempts to terminate the associated process.
func (f *DarwinFFmpeg) CancelJob(id string) error {
	f.jobMu.Lock()
	defer f.jobMu.Unlock()

	job, exists := f.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	if job.Status == JobStatusRunning {
		// Cancel the running process
		// Implementation depends on process tracking
	}

	job.Status = JobStatusFailed
	job.Error = "cancelled"
	f.jobs[id] = job
	return nil
}

// ListJobs returns a list of all jobs in the processing queue.
// Provides a snapshot of all jobs regardless of their current status.
func (f *DarwinFFmpeg) ListJobs() []BatchJob {
	f.jobMu.RLock()
	defer f.jobMu.RUnlock()

	jobs := make([]BatchJob, 0, len(f.jobs))
	for _, job := range f.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetActiveProcesses returns the number of currently active FFmpeg processes.
// Used for process limit enforcement and monitoring.
func (f *DarwinFFmpeg) GetActiveProcesses() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.processes)
}

// KillAllProcesses terminates all active FFmpeg processes.
// Forcefully kills all running processes and clears the process map.
func (f *DarwinFFmpeg) KillAllProcesses() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, cmd := range f.processes {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
	f.processes = make(map[string]*exec.Cmd)
	return nil
}

// Close cleans up resources and closes the FFmpeg wrapper.
// Terminates all active processes and performs cleanup operations.
func (f *DarwinFFmpeg) Close() error {
	return f.KillAllProcesses()
}

// Helper methods

// verifyCommands checks if FFmpeg and FFprobe commands are available in the system PATH.
func (f *DarwinFFmpeg) verifyCommands(config Config) error {
	if _, err := exec.LookPath(config.FFmpegPath); err != nil {
		return fmt.Errorf("ffmpeg not found at %s: %v", config.FFmpegPath, err)
	}
	if _, err := exec.LookPath(config.FFprobePath); err != nil {
		return fmt.Errorf("ffprobe not found at %s: %v", config.FFprobePath, err)
	}
	return nil
}

// getVersion retrieves the version information from the specified FFmpeg command.
func (f *DarwinFFmpeg) getVersion(command string) (string, error) {
	cmd := exec.Command(command, "-version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	return "", fmt.Errorf("unable to parse version")
}

// detectGPUs detects available GPUs on the macOS system.
// Supports Apple Silicon Metal GPUs and AMD/Intel discrete graphics cards.
func (f *DarwinFFmpeg) detectGPUs() ([]string, error) {
	var gpus []string

	// Check for Metal support (Apple Silicon)
	if runtime.GOARCH == "arm64" {
		cmd := exec.Command("system_profiler", "SPDisplaysDataType")
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Chipset Model:") {
					gpus = append(gpus, strings.TrimSpace(line))
				}
			}
		}
	}

	// Check for AMD GPUs
	cmd := exec.Command("system_profiler", "SPDisplaysDataType")
	if output, err := cmd.Output(); err == nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "AMD") || strings.Contains(outputStr, "Radeon") {
			lines := strings.Split(outputStr, "\n")
			for _, line := range lines {
				if strings.Contains(line, "Chipset Model:") &&
					(strings.Contains(line, "AMD") || strings.Contains(line, "Radeon")) {
					gpus = append(gpus, strings.TrimSpace(line))
				}
			}
		}
	}

	return gpus, nil
}

// buildConvertArgs constructs FFmpeg command arguments for media conversion on macOS.
func (f *DarwinFFmpeg) buildConvertArgs(options ConvertOptions) []string {
	args := []string{
		"-i", options.Input,
		"-threads", fmt.Sprintf("%d", f.config.MaxThreads),
	}

	// Add GPU acceleration if enabled
	if f.config.EnableGPU {
		// Use VideoToolbox for macOS hardware acceleration
		if runtime.GOARCH == "arm64" {
			// Apple Silicon
			args = append(args, "-hwaccel", "videotoolbox")
		} else {
			// Intel Mac
			args = append(args, "-hwaccel", "auto")
		}
	}

	// Add format-specific options
	if options.Format != "" {
		args = append(args, "-f", options.Format)
	}

	// Add quality settings
	if options.Quality != "" {
		args = append(args, "-q:v", options.Quality)
	}

	// Add custom options
	for key, value := range options.Options {
		args = append(args, key, value)
	}

	// Add progress reporting
	args = append(args, "-progress", "pipe:1")

	args = append(args, options.Output)
	return args
}

// buildExtractArgs constructs FFmpeg command arguments for media extraction on macOS.
func (f *DarwinFFmpeg) buildExtractArgs(options ExtractOptions) []string {
	args := []string{
		"-i", options.Input,
		"-threads", fmt.Sprintf("%d", f.config.MaxThreads),
	}

	switch options.Type {
	case "audio":
		args = append(args, "-vn") // no video
		if options.Format != "" {
			args = append(args, "-f", options.Format)
		}
	case "keyframe":
		args = append(args, "-an", "-vf", "select='eq(pict_type,I)'")
	}

	// Add custom options
	for key, value := range options.Options {
		args = append(args, key, value)
	}

	// Add progress reporting
	args = append(args, "-progress", "pipe:1")

	args = append(args, options.Output)
	return args
}

// executeCommand executes an FFmpeg command with proper process management and monitoring.
func (f *DarwinFFmpeg) executeCommand(ctx context.Context, args []string, onProgress ProgressCallback) error {
	// Create command with context
	cmd := exec.CommandContext(ctx, f.config.FFmpegPath, args...)
	cmd.Dir = f.config.WorkDir

	// Set up process timeout if configured
	if f.config.MaxProcessTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, f.config.MaxProcessTime)
		defer cancel()
		cmd = exec.CommandContext(ctx, f.config.FFmpegPath, args...)
		cmd.Dir = f.config.WorkDir
	}

	// Track the process
	processID := fmt.Sprintf("proc_%d", time.Now().UnixNano())
	f.mu.Lock()
	f.processes[processID] = cmd
	f.mu.Unlock()

	defer func() {
		f.mu.Lock()
		delete(f.processes, processID)
		f.mu.Unlock()
	}()

	// Set up progress monitoring if callback provided
	if onProgress != nil {
		// Implementation for progress monitoring
		// This would involve parsing FFmpeg's progress output
	}

	// Execute command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg execution failed: %v", err)
	}

	return nil
}
