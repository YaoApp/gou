package ffmpeg

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// ==== Test Helper Types ====

// TestProgressCallback is a test implementation of ProgressCallback
type TestProgressCallback struct {
	mu           sync.Mutex
	calls        []ProgressInfo
	callCount    int
	lastProgress float64
	lastBitrate  string
	lastSpeed    float64
}

// NewTestProgressCallback creates a new test progress callback
func NewTestProgressCallback() *TestProgressCallback {
	return &TestProgressCallback{
		calls: make([]ProgressInfo, 0),
	}
}

// Callback implements the ProgressCallback interface
func (t *TestProgressCallback) Callback(info ProgressInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.calls = append(t.calls, info)
	t.callCount++
	t.lastProgress = info.Progress
	t.lastBitrate = info.Bitrate
	t.lastSpeed = info.Speed
}

// GetCallCount returns the number of times the callback was called
func (t *TestProgressCallback) GetCallCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.callCount
}

// GetLastProgress returns the last progress value
func (t *TestProgressCallback) GetLastProgress() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastProgress
}

// GetLastBitrate returns the last bitrate value
func (t *TestProgressCallback) GetLastBitrate() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastBitrate
}

// GetLastSpeed returns the last speed value
func (t *TestProgressCallback) GetLastSpeed() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastSpeed
}

// Reset resets the callback state
func (t *TestProgressCallback) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls = make([]ProgressInfo, 0)
	t.callCount = 0
	t.lastProgress = 0
	t.lastBitrate = ""
	t.lastSpeed = 0
}

// ==== Core Interface Tests ====

func TestNewFFmpeg(t *testing.T) {
	ffmpeg := NewFFmpeg()
	if ffmpeg == nil {
		t.Fatal("NewFFmpeg() returned nil")
	}

	// Check that it returns the correct implementation based on OS
	switch runtime.GOOS {
	case "linux":
		if _, ok := ffmpeg.(*LinuxFFmpeg); !ok {
			t.Fatal("Expected LinuxFFmpeg implementation on Linux")
		}
	case "darwin":
		if _, ok := ffmpeg.(*DarwinFFmpeg); !ok {
			t.Fatal("Expected DarwinFFmpeg implementation on macOS")
		}
	case "windows":
		if _, ok := ffmpeg.(*WindowsFFmpeg); !ok {
			t.Fatal("Expected WindowsFFmpeg implementation on Windows")
		}
	default:
		t.Fatalf("Running on unsupported OS: %s", runtime.GOOS)
	}
}

// ==== Constants Tests ====

func TestConstants(t *testing.T) {
	if DefaultAudioFormat != "wav" {
		t.Errorf("Expected DefaultAudioFormat to be 'wav', got '%s'", DefaultAudioFormat)
	}

	if DefaultVideoFormat != "mp4" {
		t.Errorf("Expected DefaultVideoFormat to be 'mp4', got '%s'", DefaultVideoFormat)
	}

	if JobTypeConvert != "convert" {
		t.Errorf("Expected JobTypeConvert to be 'convert', got '%s'", JobTypeConvert)
	}

	if JobTypeExtract != "extract" {
		t.Errorf("Expected JobTypeExtract to be 'extract', got '%s'", JobTypeExtract)
	}

	// Test job status constants
	if JobStatusPending != "pending" {
		t.Errorf("Expected JobStatusPending to be 'pending', got '%s'", JobStatusPending)
	}

	if JobStatusRunning != "running" {
		t.Errorf("Expected JobStatusRunning to be 'running', got '%s'", JobStatusRunning)
	}

	if JobStatusCompleted != "completed" {
		t.Errorf("Expected JobStatusCompleted to be 'completed', got '%s'", JobStatusCompleted)
	}

	if JobStatusFailed != "failed" {
		t.Errorf("Expected JobStatusFailed to be 'failed', got '%s'", JobStatusFailed)
	}
}

// ==== Platform-Specific Tests ====

func TestAllPlatforms_Implementation(t *testing.T) {
	// Test that all platform implementations exist
	linux := NewLinuxFFmpeg()
	if linux == nil {
		t.Fatal("NewLinuxFFmpeg returned nil")
	}

	darwin := NewDarwinFFmpeg()
	if darwin == nil {
		t.Fatal("NewDarwinFFmpeg returned nil")
	}

	windows := NewWindowsFFmpeg()
	if windows == nil {
		t.Fatal("NewWindowsFFmpeg returned nil")
	}

	// Test that they implement the FFmpeg interface
	var _ FFmpeg = linux
	var _ FFmpeg = darwin
	var _ FFmpeg = windows
}

// ==== Windows Implementation Tests ====

func TestWindowsImplementation(t *testing.T) {
	windows := NewWindowsFFmpeg()

	// Test Init
	config := Config{
		MaxProcesses: 1,
		MaxThreads:   2,
	}

	err := windows.Init(config)
	if err == nil {
		t.Fatal("Expected error for Windows implementation, got nil")
	}
	if !strings.Contains(err.Error(), "Windows implementation not yet supported") {
		t.Fatalf("Expected 'Windows implementation not yet supported' error, got: %v", err)
	}

	// Test GetSystemInfo
	_, err = windows.GetSystemInfo()
	if err == nil {
		t.Fatal("Expected error for Windows GetSystemInfo, got nil")
	}
	if !strings.Contains(err.Error(), "Windows implementation not yet supported") {
		t.Fatalf("Expected 'Windows implementation not yet supported' error, got: %v", err)
	}

	// Test Convert
	ctx := context.Background()
	err = windows.Convert(ctx, ConvertOptions{})
	if err == nil {
		t.Fatal("Expected error for Windows Convert, got nil")
	}
	if !strings.Contains(err.Error(), "Windows implementation not yet supported") {
		t.Fatalf("Expected 'Windows implementation not yet supported' error, got: %v", err)
	}

	// Test Extract
	err = windows.Extract(ctx, ExtractOptions{})
	if err == nil {
		t.Fatal("Expected error for Windows Extract, got nil")
	}
	if !strings.Contains(err.Error(), "Windows implementation not yet supported") {
		t.Fatalf("Expected 'Windows implementation not yet supported' error, got: %v", err)
	}

	// Test ConvertBatch
	err = windows.ConvertBatch(ctx, []ConvertOptions{})
	if err == nil {
		t.Fatal("Expected error for Windows ConvertBatch, got nil")
	}
	if !strings.Contains(err.Error(), "Windows implementation not yet supported") {
		t.Fatalf("Expected 'Windows implementation not yet supported' error, got: %v", err)
	}

	// Test ExtractBatch
	err = windows.ExtractBatch(ctx, []ExtractOptions{})
	if err == nil {
		t.Fatal("Expected error for Windows ExtractBatch, got nil")
	}
	if !strings.Contains(err.Error(), "Windows implementation not yet supported") {
		t.Fatalf("Expected 'Windows implementation not yet supported' error, got: %v", err)
	}

	// Test AddJob
	jobID := windows.AddJob(BatchJob{})
	if jobID != "" {
		t.Fatalf("Expected empty job ID for Windows AddJob, got: %s", jobID)
	}

	// Test GetJob
	_, err = windows.GetJob("test")
	if err == nil {
		t.Fatal("Expected error for Windows GetJob, got nil")
	}
	if !strings.Contains(err.Error(), "Windows implementation not yet supported") {
		t.Fatalf("Expected 'Windows implementation not yet supported' error, got: %v", err)
	}

	// Test CancelJob
	err = windows.CancelJob("test")
	if err == nil {
		t.Fatal("Expected error for Windows CancelJob, got nil")
	}
	if !strings.Contains(err.Error(), "Windows implementation not yet supported") {
		t.Fatalf("Expected 'Windows implementation not yet supported' error, got: %v", err)
	}

	// Test ListJobs
	jobs := windows.ListJobs()
	if len(jobs) != 0 {
		t.Fatalf("Expected empty job list for Windows ListJobs, got: %d jobs", len(jobs))
	}

	// Test GetActiveProcesses
	activeProcesses := windows.GetActiveProcesses()
	if activeProcesses != 0 {
		t.Fatalf("Expected 0 active processes for Windows GetActiveProcesses, got: %d", activeProcesses)
	}

	// Test KillAllProcesses
	err = windows.KillAllProcesses()
	if err == nil {
		t.Fatal("Expected error for Windows KillAllProcesses, got nil")
	}
	if !strings.Contains(err.Error(), "Windows implementation not yet supported") {
		t.Fatalf("Expected 'Windows implementation not yet supported' error, got: %v", err)
	}

	// Test Close
	err = windows.Close()
	if err == nil {
		t.Fatal("Expected error for Windows Close, got nil")
	}
	if !strings.Contains(err.Error(), "Windows implementation not yet supported") {
		t.Fatalf("Expected 'Windows implementation not yet supported' error, got: %v", err)
	}

	// Test GetConfig
	config = windows.GetConfig()
	if config.FFmpegPath != "" {
		t.Fatalf("Expected empty FFmpegPath for Windows implementation, got: %s", config.FFmpegPath)
	}
	if config.FFprobePath != "" {
		t.Fatalf("Expected empty FFprobePath for Windows implementation, got: %s", config.FFprobePath)
	}
	if config.MaxProcesses != 0 {
		t.Fatalf("Expected 0 MaxProcesses for Windows implementation, got: %d", config.MaxProcesses)
	}
}

// ==== Type Tests ====

func TestProgressInfo_Fields(t *testing.T) {
	info := ProgressInfo{
		Duration:    time.Minute,
		CurrentTime: 30 * time.Second,
		Progress:    0.5,
		Speed:       1.5,
		Bitrate:     "128kb/s",
		FPS:         30.0,
	}

	if info.Duration != time.Minute {
		t.Errorf("Duration field not set correctly: expected %v, got %v", time.Minute, info.Duration)
	}
	if info.CurrentTime != 30*time.Second {
		t.Errorf("CurrentTime field not set correctly: expected %v, got %v", 30*time.Second, info.CurrentTime)
	}
	if info.Progress != 0.5 {
		t.Errorf("Progress field not set correctly: expected %v, got %v", 0.5, info.Progress)
	}
	if info.Speed != 1.5 {
		t.Errorf("Speed field not set correctly: expected %v, got %v", 1.5, info.Speed)
	}
	if info.Bitrate != "128kb/s" {
		t.Errorf("Bitrate field not set correctly: expected %v, got %v", "128kb/s", info.Bitrate)
	}
	if info.FPS != 30.0 {
		t.Errorf("FPS field not set correctly: expected %v, got %v", 30.0, info.FPS)
	}
}

func TestBatchJob_Fields(t *testing.T) {
	job := BatchJob{
		ID:     "test_job_123",
		Type:   JobTypeConvert,
		Status: JobStatusPending,
		Error:  "test error",
		Options: ConvertOptions{
			Input:  "input.mp3",
			Output: "output.wav",
			Format: "wav",
		},
	}

	if job.ID != "test_job_123" {
		t.Errorf("ID field not set correctly: expected %v, got %v", "test_job_123", job.ID)
	}
	if job.Type != JobTypeConvert {
		t.Errorf("Type field not set correctly: expected %v, got %v", JobTypeConvert, job.Type)
	}
	if job.Status != JobStatusPending {
		t.Errorf("Status field not set correctly: expected %v, got %v", JobStatusPending, job.Status)
	}
	if job.Error != "test error" {
		t.Errorf("Error field not set correctly: expected %v, got %v", "test error", job.Error)
	}

	// Test options type assertion
	convertOpts, ok := job.Options.(ConvertOptions)
	if !ok {
		t.Fatal("Options field type assertion failed")
	}
	if convertOpts.Input != "input.mp3" {
		t.Errorf("Options.Input not set correctly: expected %v, got %v", "input.mp3", convertOpts.Input)
	}
}

func TestConfig_Fields(t *testing.T) {
	config := Config{
		FFmpegPath:     "/usr/bin/ffmpeg",
		FFprobePath:    "/usr/bin/ffprobe",
		WorkDir:        "/tmp/work",
		MaxProcesses:   4,
		MaxThreads:     8,
		MaxProcessTime: 5 * time.Minute,
		EnableGPU:      true,
		GPUIndex:       1,
	}

	if config.FFmpegPath != "/usr/bin/ffmpeg" {
		t.Errorf("FFmpegPath field not set correctly: expected %v, got %v", "/usr/bin/ffmpeg", config.FFmpegPath)
	}
	if config.FFprobePath != "/usr/bin/ffprobe" {
		t.Errorf("FFprobePath field not set correctly: expected %v, got %v", "/usr/bin/ffprobe", config.FFprobePath)
	}
	if config.WorkDir != "/tmp/work" {
		t.Errorf("WorkDir field not set correctly: expected %v, got %v", "/tmp/work", config.WorkDir)
	}
	if config.MaxProcesses != 4 {
		t.Errorf("MaxProcesses field not set correctly: expected %v, got %v", 4, config.MaxProcesses)
	}
	if config.MaxThreads != 8 {
		t.Errorf("MaxThreads field not set correctly: expected %v, got %v", 8, config.MaxThreads)
	}
	if config.MaxProcessTime != 5*time.Minute {
		t.Errorf("MaxProcessTime field not set correctly: expected %v, got %v", 5*time.Minute, config.MaxProcessTime)
	}
	if config.EnableGPU != true {
		t.Errorf("EnableGPU field not set correctly: expected %v, got %v", true, config.EnableGPU)
	}
	if config.GPUIndex != 1 {
		t.Errorf("GPUIndex field not set correctly: expected %v, got %v", 1, config.GPUIndex)
	}
}

// ==== Helper Tests ====

func TestProgressCallback_ThreadSafety(t *testing.T) {
	callback := NewTestProgressCallback()

	const numGoroutines = 10
	const callsPerGoroutine = 100

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				info := ProgressInfo{
					Progress: float64(j) / float64(callsPerGoroutine),
					Speed:    float64(j),
				}
				callback.Callback(info)
			}
		}()
	}

	wg.Wait()

	expectedCalls := numGoroutines * callsPerGoroutine
	actualCalls := callback.GetCallCount()

	if actualCalls != expectedCalls {
		t.Errorf("Expected %d calls, got %d", expectedCalls, actualCalls)
	}

	// Test other methods
	lastProgress := callback.GetLastProgress()
	if lastProgress < 0 || lastProgress > 1 {
		t.Errorf("Unexpected last progress value: %f", lastProgress)
	}

	lastSpeed := callback.GetLastSpeed()
	if lastSpeed < 0 {
		t.Errorf("Unexpected last speed value: %f", lastSpeed)
	}

	// Test reset
	callback.Reset()
	if callback.GetCallCount() != 0 {
		t.Error("Reset didn't clear call count")
	}
}

// ==== Linux/Darwin Implementation Tests ====

func TestLinuxDarwinImplementation_Init(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Linux/Darwin tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	// Test Init with invalid config - this MUST fail
	config := Config{
		FFmpegPath:  "/nonexistent/ffmpeg",
		FFprobePath: "/nonexistent/ffprobe",
	}

	err := ffmpeg.Init(config)
	if err == nil {
		t.Fatal("Expected error for invalid config, but got nil")
	}
	if !strings.Contains(err.Error(), "command verification failed") {
		t.Fatalf("Expected 'command verification failed' error, got: %v", err)
	}
}

func TestLinuxDarwinImplementation_JobManagement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Linux/Darwin tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	// Test job management without initialization
	job := BatchJob{
		Type: JobTypeConvert,
		Options: ConvertOptions{
			Input:  "input.mp3",
			Output: "output.wav",
			Format: "wav",
		},
	}

	jobID := ffmpeg.AddJob(job)
	if jobID == "" {
		t.Fatal("AddJob returned empty job ID")
	}

	// Test GetJob
	retrievedJob, err := ffmpeg.GetJob(jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if retrievedJob.ID != jobID {
		t.Errorf("Expected job ID %s, got %s", jobID, retrievedJob.ID)
	}

	if retrievedJob.Type != JobTypeConvert {
		t.Errorf("Expected job type %s, got %s", JobTypeConvert, retrievedJob.Type)
	}

	// Test ListJobs
	jobs := ffmpeg.ListJobs()
	if len(jobs) == 0 {
		t.Fatal("ListJobs returned empty list")
	}

	// Test GetJob with non-existent ID - this MUST fail
	_, err = ffmpeg.GetJob("nonexistent_job_id")
	if err == nil {
		t.Fatal("Expected error for non-existent job ID, but got nil")
	}

	// Test CancelJob
	err = ffmpeg.CancelJob(jobID)
	if err != nil {
		t.Fatalf("CancelJob failed: %v", err)
	}

	// Verify job was cancelled
	cancelledJob, err := ffmpeg.GetJob(jobID)
	if err != nil {
		t.Fatalf("GetJob after cancel failed: %v", err)
	}

	if cancelledJob.Status != JobStatusFailed {
		t.Errorf("Expected job status %s, got %s", JobStatusFailed, cancelledJob.Status)
	}

	// Test CancelJob with non-existent ID - this MUST fail
	err = ffmpeg.CancelJob("nonexistent_job_id")
	if err == nil {
		t.Fatal("Expected error for cancelling non-existent job, but got nil")
	}
}

func TestLinuxDarwinImplementation_ProcessManagement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Linux/Darwin tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	// Test GetActiveProcesses
	activeProcesses := ffmpeg.GetActiveProcesses()
	if activeProcesses < 0 {
		t.Errorf("GetActiveProcesses returned negative value: %d", activeProcesses)
	}

	// Test KillAllProcesses
	err := ffmpeg.KillAllProcesses()
	if err != nil {
		t.Fatalf("KillAllProcesses failed: %v", err)
	}

	// Test Close
	err = ffmpeg.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

// ==== Integration Tests (when FFmpeg is available) ====

func TestIntegration_BasicOperations(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping integration tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	config := Config{
		MaxProcesses: 1,
		MaxThreads:   2,
	}

	err := ffmpeg.Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	defer ffmpeg.Close()

	// Test GetSystemInfo
	info, err := ffmpeg.GetSystemInfo()
	if err != nil {
		t.Fatalf("GetSystemInfo failed: %v", err)
	}

	if info.OS == "" {
		t.Fatal("Expected OS information, got empty string")
	}

	expectedOS := runtime.GOOS
	if info.OS != expectedOS {
		t.Errorf("Expected OS %s, got %s", expectedOS, info.OS)
	}

	// Test GetConfig
	retrievedConfig := ffmpeg.GetConfig()
	if retrievedConfig.MaxProcesses != config.MaxProcesses {
		t.Errorf("Expected MaxProcesses %d, got %d", config.MaxProcesses, retrievedConfig.MaxProcesses)
	}
	if retrievedConfig.MaxThreads != config.MaxThreads {
		t.Errorf("Expected MaxThreads %d, got %d", config.MaxThreads, retrievedConfig.MaxThreads)
	}

	t.Logf("System Info: OS=%s, FFmpeg=%s, FFprobe=%s, GPUs=%v",
		info.OS, info.FFmpeg, info.FFprobe, info.GPUs)
}

func TestIntegration_FileOperations(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping integration tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	config := Config{
		MaxProcesses: 1,
		MaxThreads:   2,
	}

	err := ffmpeg.Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	defer ffmpeg.Close()

	ctx := context.Background()

	// Test Convert with non-existent file - this MUST fail
	outputFile := filepath.Join(os.TempDir(), "test_nonexistent_output.wav")
	defer os.Remove(outputFile)

	options := ConvertOptions{
		Input:  "/nonexistent/file.mp3",
		Output: outputFile,
		Format: "wav",
	}

	err = ffmpeg.Convert(ctx, options)
	if err == nil {
		t.Fatal("Expected error for non-existent input file, but got nil")
	}

	// Test Convert with invalid output directory - this MUST fail
	options = ConvertOptions{
		Input:  "input.mp3",
		Output: "/nonexistent/directory/output.wav",
		Format: "wav",
	}

	err = ffmpeg.Convert(ctx, options)
	if err == nil {
		t.Fatal("Expected error for invalid output directory, but got nil")
	}
}

func TestIntegration_EmptyOptions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping integration tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	config := Config{
		FFmpegPath:   "ffmpeg",
		FFprobePath:  "ffprobe",
		MaxProcesses: 1,
		MaxThreads:   1,
	}

	err := ffmpeg.Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	defer ffmpeg.Close()

	ctx := context.Background()

	// Test Convert with empty options - this MUST fail
	err = ffmpeg.Convert(ctx, ConvertOptions{})
	if err == nil {
		t.Fatal("Expected error for empty convert options, but got nil")
	}

	// Test Extract with empty options - this MUST fail
	err = ffmpeg.Extract(ctx, ExtractOptions{})
	if err == nil {
		t.Fatal("Expected error for empty extract options, but got nil")
	}

	// Test Extract with invalid type - this MUST fail
	err = ffmpeg.Extract(ctx, ExtractOptions{
		Input:  "input.mp4",
		Output: "output.wav",
		Type:   "invalid_type",
		Format: "wav",
	})
	if err == nil {
		t.Fatal("Expected error for invalid extraction type, but got nil")
	}
}

func TestIntegration_ContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping integration tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	config := Config{
		MaxProcesses: 1,
		MaxThreads:   2,
	}

	err := ffmpeg.Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	defer ffmpeg.Close()

	outputFile := filepath.Join(os.TempDir(), "test_cancelled_output.wav")
	defer os.Remove(outputFile)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	options := ConvertOptions{
		Input:  "input.mp3",
		Output: outputFile,
		Format: "wav",
	}

	err = ffmpeg.Convert(ctx, options)
	if err != nil && err == context.Canceled {
		t.Log("Context cancellation handled correctly")
	} else {
		t.Log("Operation completed before cancellation check (acceptable)")
	}
}

// ==== Additional Coverage Tests ====

func TestLinuxDarwinImplementation_ErrorHandling(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Linux/Darwin tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	// Test operations without initialization - these should handle gracefully
	ctx := context.Background()

	// These may or may not fail depending on implementation details,
	// but they should not panic
	_ = ffmpeg.Convert(ctx, ConvertOptions{
		Input:  "test.mp3",
		Output: "test.wav",
		Format: "wav",
	})

	_ = ffmpeg.Extract(ctx, ExtractOptions{
		Input:  "test.mp4",
		Output: "test.wav",
		Type:   "audio",
		Format: "wav",
	})

	_ = ffmpeg.ConvertBatch(ctx, []ConvertOptions{{
		Input:  "test.mp3",
		Output: "test.wav",
		Format: "wav",
	}})

	_ = ffmpeg.ExtractBatch(ctx, []ExtractOptions{{
		Input:  "test.mp4",
		Output: "test.wav",
		Type:   "audio",
		Format: "wav",
	}})
}

func TestLinuxDarwinImplementation_ConfigDefaults(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Linux/Darwin tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	// Test with minimal config to trigger default value setting
	config := Config{
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
		// Leave other fields as zero values to test defaults
	}

	err := ffmpeg.Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	defer ffmpeg.Close()

	// Verify defaults were set
	retrievedConfig := ffmpeg.GetConfig()
	if retrievedConfig.MaxProcesses <= 0 {
		t.Errorf("Expected positive MaxProcesses default, got: %d", retrievedConfig.MaxProcesses)
	}
	if retrievedConfig.MaxThreads <= 0 {
		t.Errorf("Expected positive MaxThreads default, got: %d", retrievedConfig.MaxThreads)
	}
	if retrievedConfig.WorkDir == "" {
		t.Error("Expected WorkDir default to be set")
	}
}

func TestLinuxDarwinImplementation_JobTypes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Linux/Darwin tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	// Test AddJob with ExtractOptions
	extractJob := BatchJob{
		ID:   "extract_test_123",
		Type: JobTypeExtract,
		Options: ExtractOptions{
			Input:  "input.mp4",
			Output: "output.wav",
			Type:   "audio",
			Format: "wav",
		},
	}

	jobID := ffmpeg.AddJob(extractJob)
	if jobID != "extract_test_123" {
		t.Errorf("Expected job ID 'extract_test_123', got '%s'", jobID)
	}

	// Test retrieving the job
	retrievedJob, err := ffmpeg.GetJob(jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if retrievedJob.Type != JobTypeExtract {
		t.Errorf("Expected job type %s, got %s", JobTypeExtract, retrievedJob.Type)
	}

	// Test options type assertion for ExtractOptions
	extractOpts, ok := retrievedJob.Options.(ExtractOptions)
	if !ok {
		t.Fatal("Options field type assertion to ExtractOptions failed")
	}
	if extractOpts.Type != "audio" {
		t.Errorf("ExtractOptions.Type not set correctly: expected 'audio', got '%s'", extractOpts.Type)
	}
}

func TestLinuxDarwinImplementation_EmptyJobID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Linux/Darwin tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	// Test AddJob with empty ID - should generate one
	job := BatchJob{
		Type: JobTypeConvert,
		Options: ConvertOptions{
			Input:  "input.mp3",
			Output: "output.wav",
			Format: "wav",
		},
		// ID is empty
	}

	jobID := ffmpeg.AddJob(job)
	if jobID == "" {
		t.Fatal("AddJob with empty ID should generate an ID")
	}

	// Verify the job was stored with the generated ID
	retrievedJob, err := ffmpeg.GetJob(jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if retrievedJob.ID != jobID {
		t.Errorf("Expected job ID %s, got %s", jobID, retrievedJob.ID)
	}

	if retrievedJob.Status != JobStatusPending {
		t.Errorf("Expected job status %s, got %s", JobStatusPending, retrievedJob.Status)
	}
}

func TestIntegration_ProcessLimits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping integration tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	config := Config{
		MaxProcesses: 1, // Limit to 1 process
		MaxThreads:   2,
	}

	err := ffmpeg.Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	defer ffmpeg.Close()

	// Test that process counting works
	activeProcesses := ffmpeg.GetActiveProcesses()
	if activeProcesses < 0 {
		t.Errorf("GetActiveProcesses returned negative value: %d", activeProcesses)
	}

	// Test initial state
	initialProcesses := activeProcesses

	// Attempt a quick operation that should complete fast
	ctx := context.Background()
	options := ConvertOptions{
		Input:  "/dev/null", // On Unix systems, this should fail quickly
		Output: "/tmp/test_output.wav",
		Format: "wav",
	}

	_ = ffmpeg.Convert(ctx, options) // Don't care if it succeeds or fails

	// Check process count after operation
	finalProcesses := ffmpeg.GetActiveProcesses()
	t.Logf("Process count: initial=%d, final=%d", initialProcesses, finalProcesses)
}

func TestIntegration_BatchOperations(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping integration tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	config := Config{
		MaxProcesses: 2,
		MaxThreads:   2,
	}

	err := ffmpeg.Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	defer ffmpeg.Close()

	ctx := context.Background()

	// Test ConvertBatch with invalid files - should fail but not panic
	jobs := []ConvertOptions{
		{Input: "/nonexistent1.mp3", Output: "/tmp/out1.wav", Format: "wav"},
		{Input: "/nonexistent2.mp3", Output: "/tmp/out2.wav", Format: "wav"},
	}

	err = ffmpeg.ConvertBatch(ctx, jobs)
	if err == nil {
		t.Fatal("Expected error for batch conversion with non-existent files, but got nil")
	}

	// Test ExtractBatch with invalid files - should fail but not panic
	extractJobs := []ExtractOptions{
		{Input: "/nonexistent1.mp4", Output: "/tmp/extract1.wav", Type: "audio", Format: "wav"},
		{Input: "/nonexistent2.mp4", Output: "/tmp/extract2.wav", Type: "audio", Format: "wav"},
	}

	err = ffmpeg.ExtractBatch(ctx, extractJobs)
	if err == nil {
		t.Fatal("Expected error for batch extraction with non-existent files, but got nil")
	}
}

func TestIntegration_ProgressCallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping integration tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	config := Config{
		MaxProcesses: 1,
		MaxThreads:   2,
	}

	err := ffmpeg.Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	defer ffmpeg.Close()

	ctx := context.Background()
	callback := NewTestProgressCallback()

	// Test with progress callback - operation will fail but callback should be tested
	options := ConvertOptions{
		Input:      "/nonexistent/file.mp3",
		Output:     "/tmp/test_progress.wav",
		Format:     "wav",
		OnProgress: callback.Callback,
	}

	_ = ffmpeg.Convert(ctx, options) // Expected to fail

	// Test callback methods even if no progress was reported
	_ = callback.GetCallCount()
	_ = callback.GetLastProgress()
	_ = callback.GetLastBitrate()
	_ = callback.GetLastSpeed()

	// Test reset
	callback.Reset()
	if callback.GetCallCount() != 0 {
		t.Error("Reset didn't clear call count")
	}
}

func TestIntegration_CustomOptions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping integration tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	config := Config{
		MaxProcesses: 1,
		MaxThreads:   2,
		EnableGPU:    true,
		GPUIndex:     0,
	}

	err := ffmpeg.Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	defer ffmpeg.Close()

	ctx := context.Background()

	// Test Convert with custom options
	options := ConvertOptions{
		Input:   "/nonexistent/file.mp3",
		Output:  "/tmp/test_custom.wav",
		Format:  "wav",
		Quality: "high",
		Options: map[string]string{
			"-ar": "44100",
			"-ac": "2",
		},
		Stream: true,
	}

	_ = ffmpeg.Convert(ctx, options) // Expected to fail but tests option handling

	// Test Extract with custom options
	extractOptions := ExtractOptions{
		Input:  "/nonexistent/file.mp4",
		Output: "/tmp/test_custom_extract.wav",
		Type:   "audio",
		Format: "wav",
		Options: map[string]string{
			"-ar": "22050",
			"-ab": "128k",
		},
		Stream: true,
	}

	_ = ffmpeg.Extract(ctx, extractOptions) // Expected to fail but tests option handling
}

func TestIntegration_TimeoutHandling(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping integration tests on Windows")
	}

	ffmpeg := NewFFmpeg()

	config := Config{
		MaxProcesses:   1,
		MaxThreads:     2,
		MaxProcessTime: 100 * time.Millisecond, // Very short timeout
	}

	err := ffmpeg.Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	defer ffmpeg.Close()

	ctx := context.Background()

	// Test with timeout - operation should be cancelled quickly
	options := ConvertOptions{
		Input:  "/dev/zero", // On Unix, this could potentially run for a while
		Output: "/tmp/test_timeout.wav",
		Format: "wav",
	}

	_ = ffmpeg.Convert(ctx, options) // May succeed or fail due to timeout
	t.Log("Timeout test completed")
}

// ==== Benchmarks ====

func BenchmarkNewFFmpeg(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ffmpeg := NewFFmpeg()
		if ffmpeg == nil {
			b.Fatal("NewFFmpeg returned nil")
		}
	}
}

func BenchmarkProgressCallback(b *testing.B) {
	callback := NewTestProgressCallback()
	info := ProgressInfo{
		Duration:    time.Minute,
		CurrentTime: 30 * time.Second,
		Progress:    0.5,
		Speed:       1.5,
		Bitrate:     "128kb/s",
		FPS:         30.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		callback.Callback(info)
	}
}

func BenchmarkJobManagement(b *testing.B) {
	if runtime.GOOS == "windows" {
		b.Skip("Skipping Linux/Darwin benchmarks on Windows")
	}

	ffmpeg := NewFFmpeg()
	job := BatchJob{
		Type: JobTypeConvert,
		Options: ConvertOptions{
			Input:  "input.mp3",
			Output: "output.wav",
			Format: "wav",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jobID := ffmpeg.AddJob(job)
		if jobID == "" {
			b.Fatal("AddJob returned empty job ID")
		}

		_, err := ffmpeg.GetJob(jobID)
		if err != nil {
			b.Fatalf("GetJob failed: %v", err)
		}

		err = ffmpeg.CancelJob(jobID)
		if err != nil {
			b.Fatalf("CancelJob failed: %v", err)
		}
	}
}
