package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// WindowsFFmpeg represents the Windows-specific implementation of FFmpeg wrapper.
type WindowsFFmpeg struct {
	config    Config
	processes map[string]*exec.Cmd
	jobs      map[string]BatchJob
	mu        sync.RWMutex
	jobMu     sync.RWMutex
}

// NewWindowsFFmpeg creates and returns a new Windows FFmpeg instance.
func NewWindowsFFmpeg() FFmpeg {
	return &WindowsFFmpeg{
		processes: make(map[string]*exec.Cmd),
		jobs:      make(map[string]BatchJob),
	}
}

// Init initializes the Windows FFmpeg wrapper with the provided configuration.
func (f *WindowsFFmpeg) Init(config Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if config.FFmpegPath == "" {
		config.FFmpegPath = "ffmpeg"
	}
	if config.FFprobePath == "" {
		config.FFprobePath = "ffprobe"
	}
	if config.WorkDir == "" {
		config.WorkDir = os.TempDir()
	}
	if config.MaxProcesses <= 0 {
		config.MaxProcesses = runtime.NumCPU()
	}
	if config.MaxThreads <= 0 {
		config.MaxThreads = runtime.NumCPU()
	}

	if err := f.verifyCommands(config); err != nil {
		return fmt.Errorf("command verification failed: %v", err)
	}

	f.config = config
	return nil
}

// GetConfig returns the current configuration.
func (f *WindowsFFmpeg) GetConfig() Config {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.config
}

// GetSystemInfo retrieves system information including OS and FFmpeg version.
func (f *WindowsFFmpeg) GetSystemInfo() (SystemInfo, error) {
	info := SystemInfo{OS: "windows"}

	if version, err := f.getVersion(f.config.FFmpegPath); err == nil {
		info.FFmpeg = version
	}
	if version, err := f.getVersion(f.config.FFprobePath); err == nil {
		info.FFprobe = version
	}
	return info, nil
}

// GetMediaInfo gets comprehensive information about a media file.
func (f *WindowsFFmpeg) GetMediaInfo(ctx context.Context, inputFile string) (*MediaInfo, error) {
	fileInfo, err := os.Stat(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	info := &MediaInfo{FileSize: fileInfo.Size()}

	cmd := exec.CommandContext(ctx, f.config.FFprobePath,
		"-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", inputFile)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %v", err)
	}

	outputStr := string(output)

	if strings.Contains(outputStr, `"duration"`) {
		for _, line := range strings.Split(outputStr, "\n") {
			if strings.Contains(line, `"duration"`) && strings.Contains(line, `:`) {
				parts := strings.Split(line, `"`)
				if len(parts) >= 4 {
					if dur, err := strconv.ParseFloat(strings.Trim(parts[3], ` ,"`), 64); err == nil {
						info.Duration = dur
						break
					}
				}
			}
		}
	}

	if strings.Contains(outputStr, `"codec_type": "video"`) {
		inVideo := false
		for _, line := range strings.Split(outputStr, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, `"codec_type": "video"`) {
				inVideo = true
				continue
			}
			if inVideo && strings.Contains(line, `}`) {
				inVideo = false
				continue
			}
			if inVideo {
				if strings.Contains(line, `"width"`) {
					parts := strings.Split(line, `:`)
					if len(parts) >= 2 {
						if w, err := strconv.Atoi(strings.Trim(parts[1], ` ,"`)); err == nil {
							info.Width = w
						}
					}
				} else if strings.Contains(line, `"height"`) {
					parts := strings.Split(line, `:`)
					if len(parts) >= 2 {
						if h, err := strconv.Atoi(strings.Trim(parts[1], ` ,"`)); err == nil {
							info.Height = h
						}
					}
				} else if strings.Contains(line, `"codec_name"`) {
					parts := strings.Split(line, `"`)
					if len(parts) >= 4 {
						info.VideoCodec = parts[3]
					}
				} else if strings.Contains(line, `"avg_frame_rate"`) {
					parts := strings.Split(line, `"`)
					if len(parts) >= 4 && strings.Contains(parts[3], "/") {
						fp := strings.Split(parts[3], "/")
						if len(fp) == 2 {
							if num, e1 := strconv.ParseFloat(fp[0], 64); e1 == nil {
								if den, e2 := strconv.ParseFloat(fp[1], 64); e2 == nil && den != 0 {
									info.FrameRate = num / den
								}
							}
						}
					}
				}
			}
		}
	}

	if strings.Contains(outputStr, `"codec_type": "audio"`) {
		inAudio := false
		for _, line := range strings.Split(outputStr, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, `"codec_type": "audio"`) {
				inAudio = true
				continue
			}
			if inAudio && strings.Contains(line, `}`) {
				inAudio = false
				continue
			}
			if inAudio && strings.Contains(line, `"codec_name"`) {
				parts := strings.Split(line, `"`)
				if len(parts) >= 4 {
					info.AudioCodec = parts[3]
					break
				}
			}
		}
	}

	if strings.Contains(outputStr, `"bit_rate"`) {
		for _, line := range strings.Split(outputStr, "\n") {
			if strings.Contains(line, `"bit_rate"`) {
				parts := strings.Split(line, `"`)
				if len(parts) >= 4 {
					info.Bitrate = parts[3]
					break
				}
			}
		}
	}

	return info, nil
}

// Convert performs media file conversion.
func (f *WindowsFFmpeg) Convert(ctx context.Context, options ConvertOptions) error {
	if f.GetActiveProcesses() >= f.config.MaxProcesses {
		return fmt.Errorf("maximum processes (%d) reached", f.config.MaxProcesses)
	}
	return f.executeCommand(ctx, f.buildConvertArgs(options), options.OnProgress)
}

// Extract extracts media content from input files.
func (f *WindowsFFmpeg) Extract(ctx context.Context, options ExtractOptions) error {
	if f.GetActiveProcesses() >= f.config.MaxProcesses {
		return fmt.Errorf("maximum processes (%d) reached", f.config.MaxProcesses)
	}
	return f.executeCommand(ctx, f.buildExtractArgs(options), options.OnProgress)
}

// ConvertBatch performs batch conversion.
func (f *WindowsFFmpeg) ConvertBatch(ctx context.Context, jobs []ConvertOptions) error {
	for _, job := range jobs {
		if err := f.Convert(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

// ExtractBatch performs batch extraction.
func (f *WindowsFFmpeg) ExtractBatch(ctx context.Context, jobs []ExtractOptions) error {
	for _, job := range jobs {
		if err := f.Extract(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

// ChunkAudio chunks audio file with optional silence detection.
func (f *WindowsFFmpeg) ChunkAudio(ctx context.Context, options ChunkOptions) (*ChunkResult, error) {
	if f.GetActiveProcesses() >= f.config.MaxProcesses {
		return nil, fmt.Errorf("maximum processes (%d) reached", f.config.MaxProcesses)
	}
	return f.performChunking(ctx, options, "audio")
}

// ChunkVideo chunks video file with optional silence detection.
func (f *WindowsFFmpeg) ChunkVideo(ctx context.Context, options ChunkOptions) (*ChunkResult, error) {
	if f.GetActiveProcesses() >= f.config.MaxProcesses {
		return nil, fmt.Errorf("maximum processes (%d) reached", f.config.MaxProcesses)
	}
	return f.performChunking(ctx, options, "video")
}

// AddJob adds a new job to the processing queue.
func (f *WindowsFFmpeg) AddJob(job BatchJob) string {
	f.jobMu.Lock()
	defer f.jobMu.Unlock()
	if job.ID == "" {
		job.ID = fmt.Sprintf("job_%d", time.Now().UnixNano())
	}
	job.Status = JobStatusPending
	f.jobs[job.ID] = job
	return job.ID
}

// GetJob retrieves a job by ID.
func (f *WindowsFFmpeg) GetJob(id string) (BatchJob, error) {
	f.jobMu.RLock()
	defer f.jobMu.RUnlock()
	job, exists := f.jobs[id]
	if !exists {
		return BatchJob{}, fmt.Errorf("job %s not found", id)
	}
	return job, nil
}

// CancelJob cancels a job by ID.
func (f *WindowsFFmpeg) CancelJob(id string) error {
	f.jobMu.Lock()
	defer f.jobMu.Unlock()
	job, exists := f.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}
	job.Status = JobStatusFailed
	job.Error = "cancelled"
	f.jobs[id] = job
	return nil
}

// ListJobs returns all jobs.
func (f *WindowsFFmpeg) ListJobs() []BatchJob {
	f.jobMu.RLock()
	defer f.jobMu.RUnlock()
	jobs := make([]BatchJob, 0, len(f.jobs))
	for _, job := range f.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetActiveProcesses returns the number of active processes.
func (f *WindowsFFmpeg) GetActiveProcesses() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.processes)
}

// KillAllProcesses terminates all active processes.
func (f *WindowsFFmpeg) KillAllProcesses() error {
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

// Close cleans up resources.
func (f *WindowsFFmpeg) Close() error {
	return f.KillAllProcesses()
}

func (f *WindowsFFmpeg) verifyCommands(config Config) error {
	if _, err := exec.LookPath(config.FFmpegPath); err != nil {
		return fmt.Errorf("ffmpeg not found at %s: %v", config.FFmpegPath, err)
	}
	if _, err := exec.LookPath(config.FFprobePath); err != nil {
		return fmt.Errorf("ffprobe not found at %s: %v", config.FFprobePath, err)
	}
	return nil
}

func (f *WindowsFFmpeg) getVersion(command string) (string, error) {
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

func (f *WindowsFFmpeg) buildConvertArgs(options ConvertOptions) []string {
	args := []string{"-i", options.Input, "-threads", fmt.Sprintf("%d", f.config.MaxThreads)}
	if f.config.EnableGPU {
		args = append(args, "-hwaccel", "auto")
	}
	if options.Format != "" {
		args = append(args, "-f", options.Format)
	}
	if options.Quality != "" {
		args = append(args, "-q:v", options.Quality)
	}
	for key, value := range options.Options {
		args = append(args, key, value)
	}
	args = append(args, "-progress", "pipe:1", options.Output)
	return args
}

func (f *WindowsFFmpeg) buildExtractArgs(options ExtractOptions) []string {
	args := []string{"-i", options.Input, "-threads", fmt.Sprintf("%d", f.config.MaxThreads)}
	switch options.Type {
	case "audio":
		args = append(args, "-vn")
		if options.Format != "" {
			args = append(args, "-f", options.Format)
		}
	case "keyframe":
		args = append(args, "-an", "-vf", "select='eq(pict_type,I)'")
	}
	for key, value := range options.Options {
		args = append(args, key, value)
	}
	args = append(args, "-progress", "pipe:1", options.Output)
	return args
}

func (f *WindowsFFmpeg) executeCommand(ctx context.Context, args []string, onProgress ProgressCallback) error {
	cmd := exec.CommandContext(ctx, f.config.FFmpegPath, args...)
	cmd.Dir = f.config.WorkDir

	if f.config.MaxProcessTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, f.config.MaxProcessTime)
		defer cancel()
		cmd = exec.CommandContext(ctx, f.config.FFmpegPath, args...)
		cmd.Dir = f.config.WorkDir
	}

	processID := fmt.Sprintf("proc_%d", time.Now().UnixNano())
	f.mu.Lock()
	f.processes[processID] = cmd
	f.mu.Unlock()

	defer func() {
		f.mu.Lock()
		delete(f.processes, processID)
		f.mu.Unlock()
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg execution failed: %v", err)
	}
	return nil
}

func (f *WindowsFFmpeg) performChunking(ctx context.Context, options ChunkOptions, mediaType string) (*ChunkResult, error) {
	if err := os.MkdirAll(options.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	duration, err := f.getMediaDuration(options.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to get media duration: %v", err)
	}

	var chunks []ChunkInfo
	if options.EnableSilenceDetection && mediaType == "audio" {
		chunks, err = f.chunkWithSilenceDetection(ctx, options, duration)
	} else {
		chunks, err = f.chunkWithFixedDuration(ctx, options, duration, mediaType)
	}
	if err != nil {
		return nil, err
	}

	var totalSize int64
	for i, chunk := range chunks {
		if stat, err := os.Stat(chunk.FilePath); err == nil {
			chunks[i].FileSize = stat.Size()
			totalSize += chunks[i].FileSize
		}
	}

	return &ChunkResult{
		Chunks:      chunks,
		TotalChunks: len(chunks),
		TotalSize:   totalSize,
		OutputDir:   options.OutputDir,
	}, nil
}

func (f *WindowsFFmpeg) chunkWithFixedDuration(ctx context.Context, options ChunkOptions, totalDuration float64, mediaType string) ([]ChunkInfo, error) {
	var chunks []ChunkInfo
	idx := 0
	for start := 0.0; start < totalDuration; start += options.ChunkDuration {
		end := start + options.ChunkDuration
		if end > totalDuration {
			end = totalDuration
		}
		actualEnd := end
		if options.OverlapDuration > 0 && end < totalDuration {
			actualEnd = end + options.OverlapDuration
			if actualEnd > totalDuration {
				actualEnd = totalDuration
			}
		}

		chunkPath := filepath.Join(options.OutputDir, fmt.Sprintf("%s_%04d.%s", options.OutputPrefix, idx, options.Format))
		args := f.buildChunkArgs(options, start, actualEnd, chunkPath, mediaType)
		if err := f.executeCommand(ctx, args, options.OnProgress); err != nil {
			return nil, fmt.Errorf("failed to create chunk %d: %v", idx, err)
		}
		chunks = append(chunks, ChunkInfo{Index: idx, StartTime: start, EndTime: end, Duration: end - start, FilePath: chunkPath})
		idx++
	}
	return chunks, nil
}

func (f *WindowsFFmpeg) chunkWithSilenceDetection(ctx context.Context, options ChunkOptions, totalDuration float64) ([]ChunkInfo, error) {
	periods, err := f.detectSilencePeriods(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to detect silence periods: %v", err)
	}

	var chunks []ChunkInfo
	idx := 0
	lastEnd := 0.0

	for _, p := range periods {
		if p.Start > lastEnd && p.Start-lastEnd >= 0.5 {
			chunkPath := filepath.Join(options.OutputDir, fmt.Sprintf("%s_%04d.%s", options.OutputPrefix, idx, options.Format))
			args := f.buildChunkArgs(options, lastEnd, p.Start, chunkPath, "audio")
			if err := f.executeCommand(ctx, args, options.OnProgress); err != nil {
				return nil, fmt.Errorf("failed to create chunk %d: %v", idx, err)
			}
			chunks = append(chunks, ChunkInfo{Index: idx, StartTime: lastEnd, EndTime: p.Start, Duration: p.Start - lastEnd, FilePath: chunkPath})
			idx++
		}
		lastEnd = p.End
	}

	if lastEnd < totalDuration && totalDuration-lastEnd >= 0.5 {
		chunkPath := filepath.Join(options.OutputDir, fmt.Sprintf("%s_%04d.%s", options.OutputPrefix, idx, options.Format))
		args := f.buildChunkArgs(options, lastEnd, totalDuration, chunkPath, "audio")
		if err := f.executeCommand(ctx, args, options.OnProgress); err != nil {
			return nil, fmt.Errorf("failed to create final chunk %d: %v", idx, err)
		}
		chunks = append(chunks, ChunkInfo{Index: idx, StartTime: lastEnd, EndTime: totalDuration, Duration: totalDuration - lastEnd, FilePath: chunkPath})
	}

	return chunks, nil
}

func (f *WindowsFFmpeg) detectSilencePeriods(ctx context.Context, options ChunkOptions) ([]SilencePeriod, error) {
	args := []string{
		"-i", options.Input,
		"-af", fmt.Sprintf("silencedetect=noise=%fdB:d=%f", options.SilenceThreshold, options.SilenceMinLength),
		"-f", "null", "-",
	}
	cmd := exec.CommandContext(ctx, f.config.FFmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("silence detection failed: %v", err)
	}
	return f.parseSilencePeriods(string(output)), nil
}

func (f *WindowsFFmpeg) parseSilencePeriods(output string) []SilencePeriod {
	var periods []SilencePeriod
	var currentStart float64 = -1
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "silence_start:") {
			parts := strings.Split(line, "silence_start:")
			if len(parts) > 1 {
				if start, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					currentStart = start
				}
			}
		} else if strings.Contains(line, "silence_end:") && currentStart >= 0 {
			parts := strings.Split(line, "silence_end:")
			if len(parts) > 1 {
				timeStr := strings.TrimSpace(strings.Split(parts[1], " ")[0])
				if end, err := strconv.ParseFloat(timeStr, 64); err == nil {
					periods = append(periods, SilencePeriod{Start: currentStart, End: end})
					currentStart = -1
				}
			}
		}
	}
	return periods
}

func (f *WindowsFFmpeg) getMediaDuration(inputFile string) (float64, error) {
	cmd := exec.Command(f.config.FFprobePath, "-v", "quiet", "-show_entries", "format=duration", "-of", "csv=p=0", inputFile)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	durationStr := strings.TrimSpace(string(output))
	return strconv.ParseFloat(durationStr, 64)
}

func (f *WindowsFFmpeg) buildChunkArgs(options ChunkOptions, startTime, endTime float64, outputPath, mediaType string) []string {
	args := []string{
		"-ss", fmt.Sprintf("%.3f", startTime),
		"-i", options.Input,
		"-t", fmt.Sprintf("%.3f", endTime-startTime),
		"-threads", fmt.Sprintf("%d", f.config.MaxThreads),
	}
	if f.config.EnableGPU {
		args = append(args, "-hwaccel", "auto")
	}
	if mediaType == "audio" {
		args = append(args, "-vn")
	}
	if options.Format != "" {
		args = append(args, "-f", options.Format)
	}
	for key, value := range options.Options {
		args = append(args, key, value)
	}
	if mediaType == "audio" && options.Format == "wav" {
		args = append(args, "-acodec", "pcm_s16le")
	}
	args = append(args, outputPath)
	return args
}
