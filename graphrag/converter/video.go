package converter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/ffmpeg"
	"github.com/yaoapp/gou/graphrag/types"
)

// Video is a converter for video files
// Processing pipeline:
/*
   1. Check if the input file/stream is a video file
   2. If not a video file, return error
   3. If it's a video file, preprocess the file: a. Extract audio b. Extract keyframe images. Use ffmpeg for processing
   4. Text conversion:
         a. Use whisper by default to transcribe audio to text, or specify a converter consistent with whisper
		 b. Use vision by default to describe keyframe images, generating text and metadata, or specify a converter consistent with vision
	5. Text merging:
         a. Merge the transcribed text from audio and the descriptive text from keyframe images
		 b. Optimize the merged text by removing duplicate content and optimizing text format
	6. Return the merged text and metadata
	7. Metadata should be consistent with whisper for easy merging
	8. Concurrency requirements: a. whisper itself supports slicing and concurrent calls, pass parameters as needed, b. vision needs concurrent processing
*/

// Video implements the Converter interface for video file processing
type Video struct {
	AudioConverter     types.Converter // Audio converter (whisper by default)
	VisionConverter    types.Converter // Vision converter for keyframe processing
	KeyframeInterval   float64         // Interval between keyframes in seconds
	MaxKeyframes       int             // Maximum number of keyframes to extract
	TempDir            string          // Temporary directory for processing
	CleanupTemp        bool            // Whether to cleanup temporary files
	MaxConcurrency     int             // Maximum concurrent vision processing
	FFmpeg             ffmpeg.FFmpeg   // FFmpeg instance for video processing
	TextOptimization   bool            // Whether to optimize merged text
	DeduplicationRatio float64         // Ratio for text deduplication (0.0-1.0)
}

// VideoOption is the configuration for the Video converter
type VideoOption struct {
	AudioConverter     types.Converter `json:"audio_converter,omitempty"`     // Audio converter instance
	VisionConverter    types.Converter `json:"vision_converter,omitempty"`    // Vision converter instance
	KeyframeInterval   float64         `json:"keyframe_interval,omitempty"`   // Keyframe extraction interval
	MaxKeyframes       int             `json:"max_keyframes,omitempty"`       // Maximum keyframes to extract
	TempDir            string          `json:"temp_dir,omitempty"`            // Temporary directory
	CleanupTemp        bool            `json:"cleanup_temp,omitempty"`        // Cleanup temporary files
	MaxConcurrency     int             `json:"max_concurrency,omitempty"`     // Max concurrent vision processing
	TextOptimization   bool            `json:"text_optimization,omitempty"`   // Enable text optimization
	DeduplicationRatio float64         `json:"deduplication_ratio,omitempty"` // Text deduplication ratio
}

// KeyframeInfo represents information about an extracted keyframe
type KeyframeInfo struct {
	Index       int     `json:"index"`
	Timestamp   float64 `json:"timestamp"`
	FilePath    string  `json:"file_path"`
	Description string  `json:"description,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// NewVideo creates a new Video converter instance
func NewVideo(option VideoOption) (*Video, error) {
	// Validate required converters
	if option.AudioConverter == nil {
		return nil, errors.New("audio converter is required")
	}
	if option.VisionConverter == nil {
		return nil, errors.New("vision converter is required")
	}

	// Set default values for keyframe extraction
	// Priority logic:
	// 1. If KeyframeInterval is set, it takes priority (MaxKeyframes used as limit)
	// 2. If only MaxKeyframes is set, interval is calculated from video duration
	// 3. If both are set, KeyframeInterval takes priority
	// 4. If neither is set, smart defaults are used based on video duration
	keyframeInterval := option.KeyframeInterval
	if keyframeInterval == 0 {
		keyframeInterval = 10.0 // Default: extract keyframe every 10 seconds (will be overridden by smart logic)
	}

	maxKeyframes := option.MaxKeyframes
	if maxKeyframes == 0 {
		maxKeyframes = 20 // Default: maximum 20 keyframes (will be overridden by smart logic)
	}

	tempDir := option.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	maxConcurrency := option.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = 4 // Default: 4 concurrent vision processes
	}

	deduplicationRatio := option.DeduplicationRatio
	if deduplicationRatio == 0 {
		deduplicationRatio = 0.8 // Default: 80% similarity threshold for deduplication
	}

	// Initialize FFmpeg
	ffmpegInstance := ffmpeg.NewFFmpeg()
	ffmpegConfig := ffmpeg.Config{
		MaxProcesses: maxConcurrency,
		MaxThreads:   4,
		EnableGPU:    true, // Enable GPU for video processing
		WorkDir:      tempDir,
	}

	if err := ffmpegInstance.Init(ffmpegConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize FFmpeg: %v", err)
	}

	return &Video{
		AudioConverter:     option.AudioConverter,
		VisionConverter:    option.VisionConverter,
		KeyframeInterval:   keyframeInterval,
		MaxKeyframes:       maxKeyframes,
		TempDir:            tempDir,
		CleanupTemp:        option.CleanupTemp,
		MaxConcurrency:     maxConcurrency,
		FFmpeg:             ffmpegInstance,
		TextOptimization:   option.TextOptimization,
		DeduplicationRatio: deduplicationRatio,
	}, nil
}

// Convert converts a video file to plain text by calling ConvertStream
func (v *Video) Convert(ctx context.Context, file string, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	v.reportProgress(types.ConverterStatusPending, "Opening video file", 0.0, callback...)

	// Open the file
	f, err := os.Open(file)
	if err != nil {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to open file: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer f.Close()

	// Use ConvertStream to process the file
	result, err := v.ConvertStream(ctx, f, callback...)
	if err != nil {
		return nil, err
	}

	v.reportProgress(types.ConverterStatusSuccess, "Video conversion completed", 1.0, callback...)
	return result, nil
}

// ConvertStream converts a video stream to text using audio transcription and keyframe analysis
func (v *Video) ConvertStream(ctx context.Context, stream io.ReadSeeker, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	v.reportProgress(types.ConverterStatusPending, "Starting video processing", 0.0, callback...)

	// Save stream to temporary file for processing
	tempFile, err := v.saveStreamToTempFile(stream)
	if err != nil {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to save stream: %v", err), 0.0, callback...)
		return nil, err
	}
	defer func() {
		if v.CleanupTemp {
			os.Remove(tempFile)
		}
	}()

	v.reportProgress(types.ConverterStatusPending, "Validating video file", 0.05, callback...)

	// Check if file is a video
	if !v.isVideoFile(tempFile) {
		v.reportProgress(types.ConverterStatusError, "File is not a video", 0.0, callback...)
		return nil, errors.New("file is not a video file")
	}

	v.reportProgress(types.ConverterStatusPending, "Extracting audio", 0.1, callback...)

	// Extract audio from video
	audioFile, err := v.extractAudio(ctx, tempFile)
	if err != nil {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to extract audio: %v", err), 0.0, callback...)
		return nil, err
	}
	defer func() {
		if v.CleanupTemp {
			os.Remove(audioFile)
		}
	}()

	v.reportProgress(types.ConverterStatusPending, "Extracting keyframes", 0.2, callback...)

	// Extract keyframes from video
	keyframes, err := v.extractKeyframes(ctx, tempFile)
	if err != nil {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to extract keyframes: %v", err), 0.0, callback...)
		return nil, err
	}
	defer func() {
		if v.CleanupTemp {
			for _, kf := range keyframes {
				os.Remove(kf.FilePath)
			}
		}
	}()

	v.reportProgress(types.ConverterStatusPending, "Processing audio and keyframes", 0.3, callback...)

	// Process audio and keyframes concurrently
	var wg sync.WaitGroup
	var audioResult *types.ConvertResult
	var keyframeResults []KeyframeInfo
	var audioErr, keyframeErr error

	// Process audio transcription
	wg.Add(1)
	go func() {
		defer wg.Done()
		audioProgressCallback := func(status types.ConverterStatus, payload types.ConverterPayload) {
			// Scale audio progress to 30%-60% of total progress
			progress := 0.3 + (payload.Progress * 0.3)
			v.reportProgress(status, fmt.Sprintf("Audio: %s", payload.Message), progress, callback...)
		}
		audioResult, audioErr = v.AudioConverter.Convert(ctx, audioFile, audioProgressCallback)
	}()

	// Process keyframes with vision
	wg.Add(1)
	go func() {
		defer wg.Done()
		keyframeProgressCallback := func(status types.ConverterStatus, payload types.ConverterPayload) {
			// Scale keyframe progress to 30%-60% of total progress
			progress := 0.3 + (payload.Progress * 0.3)
			v.reportProgress(status, fmt.Sprintf("Keyframes: %s", payload.Message), progress, callback...)
		}
		keyframeResults, keyframeErr = v.processKeyframes(ctx, keyframes, keyframeProgressCallback)
	}()

	wg.Wait()

	// Check for errors
	if audioErr != nil {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Audio processing failed: %v", audioErr), 0.0, callback...)
		return nil, fmt.Errorf("audio processing failed: %w", audioErr)
	}
	if keyframeErr != nil {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Keyframe processing failed: %v", keyframeErr), 0.0, callback...)
		return nil, fmt.Errorf("keyframe processing failed: %w", keyframeErr)
	}

	v.reportProgress(types.ConverterStatusPending, "Merging results", 0.8, callback...)

	// Merge audio and keyframe results
	result := v.mergeResults(audioResult, keyframeResults)

	v.reportProgress(types.ConverterStatusSuccess, "Video processing completed", 1.0, callback...)
	return result, nil
}

// saveStreamToTempFile saves the stream to a temporary file
func (v *Video) saveStreamToTempFile(stream io.ReadSeeker) (string, error) {
	tempFile, err := os.CreateTemp(v.TempDir, "video_input_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, stream)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to copy stream to temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// isVideoFile checks if the file is a video file
func (v *Video) isVideoFile(filePath string) bool {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	videoExtensions := []string{".mp4", ".avi", ".mov", ".mkv", ".wmv", ".flv", ".webm", ".m4v", ".3gp"}

	for _, videoExt := range videoExtensions {
		if ext == videoExt {
			return true
		}
	}

	// Check file magic bytes
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 12)
	_, err = file.Read(buffer)
	if err != nil {
		return false
	}

	// Check common video file signatures
	// MP4/MOV
	if len(buffer) >= 8 && (string(buffer[4:8]) == "ftyp" || string(buffer[4:8]) == "moov") {
		return true
	}
	// AVI
	if len(buffer) >= 12 && string(buffer[0:4]) == "RIFF" && string(buffer[8:12]) == "AVI " {
		return true
	}
	// MKV
	if len(buffer) >= 4 && buffer[0] == 0x1A && buffer[1] == 0x45 && buffer[2] == 0xDF && buffer[3] == 0xA3 {
		return true
	}

	return false
}

// extractAudio extracts audio from the video file
func (v *Video) extractAudio(ctx context.Context, videoPath string) (string, error) {
	outputFile := filepath.Join(v.TempDir, fmt.Sprintf("extracted_audio_%d.wav", time.Now().UnixNano()))

	err := v.FFmpeg.Extract(ctx, ffmpeg.ExtractOptions{
		Input:  videoPath,
		Output: outputFile,
		Type:   "audio",
		Format: "wav",
	})

	if err != nil {
		return "", fmt.Errorf("failed to extract audio: %w", err)
	}

	return outputFile, nil
}

// extractKeyframes extracts keyframes from the video file
func (v *Video) extractKeyframes(ctx context.Context, videoPath string) ([]KeyframeInfo, error) {
	keyframesDir := filepath.Join(v.TempDir, fmt.Sprintf("keyframes_%d", time.Now().UnixNano()))
	err := os.MkdirAll(keyframesDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyframes directory: %w", err)
	}

	// Get video information to determine optimal keyframe extraction strategy
	mediaInfo, err := v.FFmpeg.GetMediaInfo(ctx, videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	// Calculate optimal extraction parameters based on user settings and video duration
	actualInterval, actualMaxFrames := v.calculateKeyframeParams(mediaInfo.Duration)

	// Build FFmpeg options for keyframe extraction
	options := map[string]string{
		"-vf":       fmt.Sprintf("fps=1/%f,scale=640:480", actualInterval), // Extract one frame every N seconds
		"-q:v":      "2",                                                   // High quality
		"-frames:v": fmt.Sprintf("%d", actualMaxFrames),                    // Limit number of frames
	}

	extractOptions := ffmpeg.ExtractOptions{
		Input:   videoPath,
		Output:  filepath.Join(keyframesDir, "keyframe_%03d.jpg"),
		Type:    "keyframe",
		Format:  "image2",
		Options: options,
	}

	err = v.FFmpeg.Extract(ctx, extractOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to extract keyframes: %w", err)
	}

	// List extracted keyframes
	files, err := os.ReadDir(keyframesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read keyframes directory: %w", err)
	}

	var keyframes []KeyframeInfo
	for i, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".jpg") {
			keyframes = append(keyframes, KeyframeInfo{
				Index:     i,
				Timestamp: float64(i) * actualInterval,
				FilePath:  filepath.Join(keyframesDir, file.Name()),
			})
		}
	}

	return keyframes, nil
}

// calculateKeyframeParams calculates optimal keyframe extraction parameters
func (v *Video) calculateKeyframeParams(videoDuration float64) (interval float64, maxFrames int) {
	originalInterval := v.KeyframeInterval
	originalMaxFrames := v.MaxKeyframes

	// Priority logic:
	// 1. If both are set, interval takes priority
	// 2. If only interval is set, use it (with maxFrames as upper limit)
	// 3. If only maxFrames is set, calculate interval based on video duration
	// 4. If neither is set, use defaults

	// Check if user explicitly set KeyframeInterval (different from constructor default)
	intervalExplicitlySet := originalInterval != 10.0 // Our default is 10.0
	maxFramesExplicitlySet := originalMaxFrames != 20 // Our default is 20

	if intervalExplicitlySet {
		// Interval is explicitly set, use it as priority
		interval = originalInterval
		// Calculate how many frames this would generate
		estimatedFrames := int(videoDuration/interval) + 1
		// Use maxFrames as upper limit
		if estimatedFrames > originalMaxFrames {
			maxFrames = originalMaxFrames
		} else {
			maxFrames = estimatedFrames
		}
	} else if maxFramesExplicitlySet {
		// Only maxFrames is set, calculate interval based on video duration
		maxFrames = originalMaxFrames
		interval = videoDuration / float64(maxFrames-1) // -1 to include both start and end
		// Ensure interval is not too small (minimum 1 second)
		if interval < 1.0 {
			interval = 1.0
		}
	} else {
		// Neither explicitly set, use smart defaults based on video duration
		if videoDuration <= 60 {
			// Short video (≤1 min): 5 second interval
			interval = 5.0
		} else if videoDuration <= 300 {
			// Medium video (≤5 min): 10 second interval
			interval = 10.0
		} else if videoDuration <= 1800 {
			// Long video (≤30 min): 30 second interval
			interval = 30.0
		} else {
			// Very long video (>30 min): 60 second interval
			interval = 60.0
		}

		// Calculate frames based on the chosen interval
		maxFrames = int(videoDuration/interval) + 1
		// Cap at reasonable maximum
		if maxFrames > 50 {
			maxFrames = 50
			interval = videoDuration / 49 // Recalculate interval to fit 50 frames
		}
	}

	// Final sanity checks
	if interval < 1.0 {
		interval = 1.0
	}
	if maxFrames < 1 {
		maxFrames = 1
	}
	if maxFrames > 100 { // Hard limit to prevent excessive frame extraction
		maxFrames = 100
	}

	return interval, maxFrames
}

// processKeyframes processes keyframes with vision converter concurrently
func (v *Video) processKeyframes(ctx context.Context, keyframes []KeyframeInfo, callback ...types.ConverterProgress) ([]KeyframeInfo, error) {
	results := make([]KeyframeInfo, len(keyframes))
	copy(results, keyframes)

	// Create semaphore for concurrent processing
	semaphore := make(chan struct{}, v.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, keyframe := range keyframes {
		wg.Add(1)
		go func(index int, kf KeyframeInfo) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Process keyframe with vision converter
			visionResult, err := v.VisionConverter.Convert(ctx, kf.FilePath)

			mu.Lock()
			if err != nil {
				results[index].Error = err.Error()
			} else {
				results[index].Description = visionResult.Text
			}

			// Report progress
			completed := 0
			for _, r := range results {
				if r.Description != "" || r.Error != "" {
					completed++
				}
			}
			progress := float64(completed) / float64(len(results))
			v.reportProgress(types.ConverterStatusPending, fmt.Sprintf("Processed %d/%d keyframes", completed, len(results)), progress, callback...)
			mu.Unlock()
		}(i, keyframe)
	}

	wg.Wait()
	return results, nil
}

// mergeResults merges audio transcription and keyframe descriptions
func (v *Video) mergeResults(audioResult *types.ConvertResult, keyframeResults []KeyframeInfo) *types.ConvertResult {
	var mergedText strings.Builder

	// Add audio transcription
	if audioResult != nil && audioResult.Text != "" {
		mergedText.WriteString("Audio Transcription:\n")
		mergedText.WriteString(audioResult.Text)
		mergedText.WriteString("\n\n")
	}

	// Add keyframe descriptions
	if len(keyframeResults) > 0 {
		mergedText.WriteString("Visual Content:\n")
		for _, kf := range keyframeResults {
			if kf.Description != "" {
				mergedText.WriteString(fmt.Sprintf("At %.1fs: %s\n", kf.Timestamp, kf.Description))
			}
		}
	}

	text := mergedText.String()

	// Apply text optimization if enabled
	if v.TextOptimization {
		text = v.optimizeText(text)
	}

	// Create combined metadata
	metadata := map[string]interface{}{
		"source_type":         "video",
		"keyframe_interval":   v.KeyframeInterval,
		"max_keyframes":       v.MaxKeyframes,
		"extracted_keyframes": len(keyframeResults),
		"text_optimization":   v.TextOptimization,
		"text_length":         len(text),
	}

	// Include audio metadata if available
	if audioResult != nil && audioResult.Metadata != nil {
		metadata["audio_metadata"] = audioResult.Metadata
	}

	// Include keyframe processing info
	successfulKeyframes := 0
	for _, kf := range keyframeResults {
		if kf.Description != "" {
			successfulKeyframes++
		}
	}
	metadata["successful_keyframes"] = successfulKeyframes

	return &types.ConvertResult{
		Text:     text,
		Metadata: metadata,
	}
}

// optimizeText optimizes the merged text by removing duplicates and improving format
func (v *Video) optimizeText(text string) string {
	// Simple text optimization: remove excessive whitespace and deduplicate similar content
	lines := strings.Split(text, "\n")
	var optimized []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Simple deduplication: check if similar line already exists
		isDuplicate := false
		for _, existing := range optimized {
			if v.calculateSimilarity(line, existing) > v.DeduplicationRatio {
				isDuplicate = true
				break
			}
		}

		if !isDuplicate {
			optimized = append(optimized, line)
		}
	}

	return strings.Join(optimized, "\n")
}

// calculateSimilarity calculates text similarity ratio (simple implementation)
func (v *Video) calculateSimilarity(text1, text2 string) float64 {
	if len(text1) == 0 || len(text2) == 0 {
		return 0.0
	}

	// Simple similarity calculation based on common words
	words1 := strings.Fields(strings.ToLower(text1))
	words2 := strings.Fields(strings.ToLower(text2))

	wordMap1 := make(map[string]bool)
	for _, word := range words1 {
		wordMap1[word] = true
	}

	commonWords := 0
	for _, word := range words2 {
		if wordMap1[word] {
			commonWords++
		}
	}

	totalWords := len(words1) + len(words2)
	if totalWords == 0 {
		return 0.0
	}

	return float64(commonWords*2) / float64(totalWords)
}

// reportProgress reports conversion progress
func (v *Video) reportProgress(status types.ConverterStatus, message string, progress float64, callbacks ...types.ConverterProgress) {
	if len(callbacks) == 0 {
		return
	}

	payload := types.ConverterPayload{
		Status:   status,
		Message:  message,
		Progress: progress,
	}

	for _, callback := range callbacks {
		if callback != nil {
			callback(status, payload)
		}
	}
}

// Close cleans up resources
func (v *Video) Close() error {
	if v.FFmpeg != nil {
		return v.FFmpeg.Close()
	}
	return nil
}
