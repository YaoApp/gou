package converter

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/ffmpeg"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// Whisper is a model for audio transcription.

// Whisper implements the Converter interface for audio transcription using OpenAI's Whisper API
type Whisper struct {
	Connector              connector.Connector
	Model                  string
	Options                map[string]any
	Language               string        // Target language for transcription
	ChunkDuration          float64       // Duration of each audio chunk in seconds
	MappingDuration        float64       // Duration for timeline mapping in seconds
	SilenceThreshold       float64       // Silence threshold in dB for chunk splitting
	SilenceMinLength       float64       // Minimum silence length in seconds
	EnableSilenceDetection bool          // Whether to use silence detection
	MaxConcurrency         int           // Maximum concurrent transcription requests
	TempDir                string        // Temporary directory for chunk files
	CleanupTemp            bool          // Whether to cleanup temp files after processing
	FFmpeg                 ffmpeg.FFmpeg // FFmpeg instance for audio processing
}

// WhisperOption is the option for the Whisper instance
type WhisperOption struct {
	ConnectorName          string         `json:"connector,omitempty"`                // Connector name
	Model                  string         `json:"model,omitempty"`                    // Model name (whisper-1, etc.)
	Options                map[string]any `json:"options,omitempty"`                  // Additional options
	Language               string         `json:"language,omitempty"`                 // Target language
	ChunkDuration          float64        `json:"chunk_duration,omitempty"`           // Chunk duration in seconds
	MappingDuration        float64        `json:"mapping_duration,omitempty"`         // Timeline mapping duration
	SilenceThreshold       float64        `json:"silence_threshold,omitempty"`        // Silence threshold in dB
	SilenceMinLength       float64        `json:"silence_min_length,omitempty"`       // Minimum silence length
	EnableSilenceDetection bool           `json:"enable_silence_detection,omitempty"` // Enable silence detection
	MaxConcurrency         int            `json:"max_concurrency,omitempty"`          // Max concurrent requests
	TempDir                string         `json:"temp_dir,omitempty"`                 // Temp directory
	CleanupTemp            bool           `json:"cleanup_temp,omitempty"`             // Cleanup temp files
}

// ChunkTranscription represents a transcription result for a single chunk
type ChunkTranscription struct {
	Index     int     `json:"index"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
	Text      string  `json:"text"`
	Language  string  `json:"language,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// TimelineMapping represents the timeline mapping for reconstruction
type TimelineMapping struct {
	Timestamp float64 `json:"timestamp"`
	Text      string  `json:"text"`
}

// NewWhisper creates a new Whisper instance
func NewWhisper(option WhisperOption) (*Whisper, error) {
	c, err := connector.Select(option.ConnectorName)
	if err != nil {
		return nil, err
	}

	if !c.Is(connector.OPENAI) {
		return nil, errors.New("connector is not an OpenAI connector")
	}

	// Set default values
	chunkDuration := option.ChunkDuration
	if chunkDuration == 0 {
		chunkDuration = 30.0 // Default 30 seconds per chunk
	}

	mappingDuration := option.MappingDuration
	if mappingDuration == 0 {
		mappingDuration = 5.0 // Default 5 seconds for timeline mapping
	}

	silenceThreshold := option.SilenceThreshold
	if silenceThreshold == 0 {
		silenceThreshold = -40.0 // Default -40dB silence threshold
	}

	silenceMinLength := option.SilenceMinLength
	if silenceMinLength == 0 {
		silenceMinLength = 1.0 // Default 1 second minimum silence
	}

	maxConcurrency := option.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = 4 // Default 4 concurrent requests
	}

	tempDir := option.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	model := option.Model
	if model == "" {
		model = "whisper-1" // Default Whisper model
	}

	language := option.Language
	if language == "" {
		language = "auto" // Default auto-detect language
	}

	// Initialize FFmpeg
	ffmpegInstance := ffmpeg.NewFFmpeg()
	ffmpegConfig := ffmpeg.Config{
		MaxProcesses: maxConcurrency,
		MaxThreads:   4,
		EnableGPU:    false, // Audio processing doesn't need GPU
		WorkDir:      tempDir,
	}

	if err := ffmpegInstance.Init(ffmpegConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize FFmpeg: %v", err)
	}

	return &Whisper{
		Connector:              c,
		Model:                  model,
		Options:                option.Options,
		Language:               language,
		ChunkDuration:          chunkDuration,
		MappingDuration:        mappingDuration,
		SilenceThreshold:       silenceThreshold,
		SilenceMinLength:       silenceMinLength,
		EnableSilenceDetection: option.EnableSilenceDetection,
		MaxConcurrency:         maxConcurrency,
		TempDir:                tempDir,
		CleanupTemp:            option.CleanupTemp,
		FFmpeg:                 ffmpegInstance,
	}, nil
}

// Convert converts a file to plain text by calling ConvertStream
func (w *Whisper) Convert(ctx context.Context, file string, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	w.reportProgress(types.ConverterStatusPending, "Opening file", 0.0, callback...)

	// Open the file
	f, err := os.Open(file)
	if err != nil {
		w.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to open file: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer f.Close()

	// Use ConvertStream to process the file
	result, err := w.ConvertStream(ctx, f, callback...)
	if err != nil {
		return nil, err
	}

	w.reportProgress(types.ConverterStatusSuccess, "File conversion completed", 1.0, callback...)
	return result, nil
}

// ConvertStream converts an audio stream to text using Whisper
func (w *Whisper) ConvertStream(ctx context.Context, stream io.ReadSeeker, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	w.reportProgress(types.ConverterStatusPending, "Starting audio processing", 0.0, callback...)

	// Check if gzipped and decompress if needed
	reader, err := w.handleGzipStream(stream)
	if err != nil {
		w.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to handle gzip stream: %v", err), 0.0, callback...)
		return nil, err
	}

	// Save stream to temporary file for processing
	tempFile, err := w.saveStreamToTempFile(reader)
	if err != nil {
		w.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to save stream: %v", err), 0.0, callback...)
		return nil, err
	}
	defer func() {
		if w.CleanupTemp {
			os.Remove(tempFile)
		}
	}()

	w.reportProgress(types.ConverterStatusPending, "Checking file type", 0.1, callback...)

	// Check file type and convert if necessary
	processedFile, mediaType, err := w.preprocessFile(ctx, tempFile)
	if err != nil {
		w.reportProgress(types.ConverterStatusError, fmt.Sprintf("File preprocessing failed: %v", err), 0.0, callback...)
		return nil, err
	}
	defer func() {
		if w.CleanupTemp && processedFile != tempFile {
			os.Remove(processedFile)
		}
	}()

	w.reportProgress(types.ConverterStatusPending, "Chunking audio", 0.2, callback...)

	// Chunk the audio file
	chunks, err := w.chunkAudio(ctx, processedFile)
	if err != nil {
		w.reportProgress(types.ConverterStatusError, fmt.Sprintf("Audio chunking failed: %v", err), 0.0, callback...)
		return nil, err
	}

	w.reportProgress(types.ConverterStatusPending, "Transcribing chunks", 0.4, callback...)

	// Transcribe chunks concurrently
	transcriptions, err := w.transcribeChunks(ctx, chunks, callback...)
	if err != nil {
		w.reportProgress(types.ConverterStatusError, fmt.Sprintf("Transcription failed: %v", err), 0.0, callback...)
		return nil, err
	}

	w.reportProgress(types.ConverterStatusPending, "Combining transcriptions", 0.9, callback...)

	// Combine transcriptions and create timeline mapping
	result := w.combineTranscriptions(transcriptions, mediaType)

	// Cleanup chunk files
	if w.CleanupTemp {
		for _, chunk := range chunks {
			os.Remove(chunk.FilePath)
		}
	}

	w.reportProgress(types.ConverterStatusSuccess, "Audio transcription completed", 1.0, callback...)
	return result, nil
}

// handleGzipStream checks if the stream is gzipped and decompresses if needed
func (w *Whisper) handleGzipStream(stream io.ReadSeeker) (io.Reader, error) {
	// Check if gzipped
	peekBuffer := make([]byte, 2)
	n, err := stream.Read(peekBuffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to peek stream: %w", err)
	}

	// Reset stream position
	_, err = stream.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to reset stream: %w", err)
	}

	// Check for gzip magic bytes (0x1f, 0x8b)
	if n >= 2 && peekBuffer[0] == 0x1f && peekBuffer[1] == 0x8b {
		gzReader, err := gzip.NewReader(stream)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return gzReader, nil
	}

	return stream, nil
}

// saveStreamToTempFile saves the stream to a temporary file
func (w *Whisper) saveStreamToTempFile(reader io.Reader) (string, error) {
	tempFile, err := os.CreateTemp(w.TempDir, "whisper_input_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, reader)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to copy stream to temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// preprocessFile checks file type and converts if necessary
func (w *Whisper) preprocessFile(ctx context.Context, filePath string) (string, string, error) {
	// Detect file type
	mediaType, err := w.detectFileType(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to detect file type: %w", err)
	}

	// Check if it's a supported media type
	if !w.isSupportedMediaType(mediaType) {
		return "", "", fmt.Errorf("unsupported file type: %s", mediaType)
	}

	// If it's a video file, extract audio
	if strings.HasPrefix(mediaType, "video/") {
		audioFile, err := w.extractAudioFromVideo(ctx, filePath)
		if err != nil {
			return "", "", fmt.Errorf("failed to extract audio from video: %w", err)
		}
		return audioFile, "audio/wav", nil
	}

	// If it's not WAV format, convert to WAV
	if mediaType != "audio/wav" && mediaType != "audio/x-wav" {
		wavFile, err := w.convertToWav(ctx, filePath)
		if err != nil {
			return "", "", fmt.Errorf("failed to convert to WAV: %w", err)
		}
		return wavFile, "audio/wav", nil
	}

	return filePath, mediaType, nil
}

// detectFileType detects the file type using mime type detection
func (w *Whisper) detectFileType(filePath string) (string, error) {
	// First try by extension
	ext := strings.ToLower(filepath.Ext(filePath))
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		return mimeType, nil
	}

	// Try to detect by content
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}

	// Check common audio/video file signatures
	if w.isAudioFile(buffer) {
		return "audio/unknown", nil
	}
	if w.isVideoFile(buffer) {
		return "video/unknown", nil
	}

	return "application/octet-stream", nil
}

// isSupportedMediaType checks if the media type is supported
func (w *Whisper) isSupportedMediaType(mediaType string) bool {
	supportedTypes := []string{
		"audio/", "video/",
	}

	for _, supportedType := range supportedTypes {
		if strings.HasPrefix(mediaType, supportedType) {
			return true
		}
	}

	return false
}

// isAudioFile checks if the file is an audio file based on magic bytes
func (w *Whisper) isAudioFile(buffer []byte) bool {
	// WAV
	if len(buffer) >= 12 && string(buffer[0:4]) == "RIFF" && string(buffer[8:12]) == "WAVE" {
		return true
	}
	// MP3 with ID3 tag
	if len(buffer) >= 3 && string(buffer[0:3]) == "ID3" {
		return true
	}
	// MP3 without ID3 tag
	if len(buffer) >= 3 && buffer[0] == 0xFF && (buffer[1]&0xE0) == 0xE0 {
		return true
	}
	// FLAC
	if len(buffer) >= 4 && string(buffer[0:4]) == "fLaC" {
		return true
	}
	// OGG
	if len(buffer) >= 4 && string(buffer[0:4]) == "OggS" {
		return true
	}
	// AAC
	if len(buffer) >= 4 && buffer[0] == 0xFF && (buffer[1]&0xF6) == 0xF0 {
		return true
	}
	return false
}

// isVideoFile checks if the file is a video file based on magic bytes
func (w *Whisper) isVideoFile(buffer []byte) bool {
	// MP4
	if len(buffer) >= 12 && (string(buffer[4:8]) == "ftyp" || string(buffer[4:8]) == "moov") {
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
	// MOV (QuickTime)
	if len(buffer) >= 8 && string(buffer[4:8]) == "ftypqt" {
		return true
	}
	return false
}

// extractAudioFromVideo extracts audio from video file
func (w *Whisper) extractAudioFromVideo(ctx context.Context, videoPath string) (string, error) {
	outputFile := filepath.Join(w.TempDir, fmt.Sprintf("extracted_audio_%d.wav", time.Now().UnixNano()))

	err := w.FFmpeg.Extract(ctx, ffmpeg.ExtractOptions{
		Input:  videoPath,
		Output: outputFile,
		Type:   "audio",
		Format: "wav",
	})

	if err != nil {
		return "", err
	}

	return outputFile, nil
}

// convertToWav converts audio file to WAV format
func (w *Whisper) convertToWav(ctx context.Context, inputPath string) (string, error) {
	outputFile := filepath.Join(w.TempDir, fmt.Sprintf("converted_audio_%d.wav", time.Now().UnixNano()))

	err := w.FFmpeg.Convert(ctx, ffmpeg.ConvertOptions{
		Input:  inputPath,
		Output: outputFile,
		Format: "wav",
	})

	if err != nil {
		return "", err
	}

	return outputFile, nil
}

// chunkAudio chunks the audio file into smaller pieces
func (w *Whisper) chunkAudio(ctx context.Context, audioPath string) ([]ffmpeg.ChunkInfo, error) {
	chunksDir := filepath.Join(w.TempDir, fmt.Sprintf("chunks_%d", time.Now().UnixNano()))

	chunkOptions := ffmpeg.ChunkOptions{
		Input:                  audioPath,
		OutputDir:              chunksDir,
		OutputPrefix:           "chunk",
		ChunkDuration:          w.ChunkDuration,
		SilenceThreshold:       w.SilenceThreshold,
		SilenceMinLength:       w.SilenceMinLength,
		Format:                 "wav",
		EnableSilenceDetection: w.EnableSilenceDetection,
	}

	result, err := w.FFmpeg.ChunkAudio(ctx, chunkOptions)
	if err != nil {
		return nil, err
	}

	return result.Chunks, nil
}

// transcribeChunks transcribes all audio chunks concurrently
func (w *Whisper) transcribeChunks(ctx context.Context, chunks []ffmpeg.ChunkInfo, callback ...types.ConverterProgress) ([]ChunkTranscription, error) {
	transcriptions := make([]ChunkTranscription, len(chunks))

	// Create a semaphore to limit concurrent requests
	semaphore := make(chan struct{}, w.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, chunk := range chunks {
		wg.Add(1)
		go func(index int, chunkInfo ffmpeg.ChunkInfo) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			transcription := ChunkTranscription{
				Index:     index,
				StartTime: chunkInfo.StartTime,
				EndTime:   chunkInfo.EndTime,
			}

			// Transcribe the chunk
			text, language, err := w.transcribeChunk(ctx, chunkInfo.FilePath)
			if err != nil {
				transcription.Error = err.Error()
			} else {
				transcription.Text = text
				transcription.Language = language
			}

			// Update results
			mu.Lock()
			transcriptions[index] = transcription
			// Report progress
			completed := 0
			for _, t := range transcriptions {
				if t.Text != "" || t.Error != "" {
					completed++
				}
			}
			progress := 0.4 + (0.5 * float64(completed) / float64(len(chunks)))
			w.reportProgress(types.ConverterStatusPending, fmt.Sprintf("Transcribed %d/%d chunks", completed, len(chunks)), progress, callback...)
			mu.Unlock()
		}(i, chunk)
	}

	wg.Wait()
	return transcriptions, nil
}

// transcribeChunk transcribes a single audio chunk
func (w *Whisper) transcribeChunk(ctx context.Context, chunkPath string) (string, string, error) {
	// Prepare the form data (without file - that's handled separately)
	formData := map[string]interface{}{
		"model": w.Model,
	}

	// Add language if specified
	if w.Language != "" {
		formData["language"] = w.Language
	}

	// Add additional options
	if w.Options != nil {
		for k, v := range w.Options {
			formData[k] = v
		}
	}

	// Make the request to Whisper API using file upload
	response, err := utils.PostLLMFile(ctx, w.Connector, "audio/transcriptions", chunkPath, formData)
	if err != nil {
		return "", "", fmt.Errorf("transcription request failed: %w", err)
	}

	// Parse the response
	if responseMap, ok := response.(map[string]interface{}); ok {
		text := ""
		language := ""

		if textVal, ok := responseMap["text"].(string); ok {
			text = textVal
		}

		if langVal, ok := responseMap["language"].(string); ok {
			language = langVal
		}

		return text, language, nil
	}

	return "", "", fmt.Errorf("unexpected response format")
}

// combineTranscriptions combines all transcriptions into a single result
func (w *Whisper) combineTranscriptions(transcriptions []ChunkTranscription, mediaType string) *types.ConvertResult {
	var fullText strings.Builder
	var timelineMappings []TimelineMapping
	var errors []string

	for _, transcription := range transcriptions {
		if transcription.Error != "" {
			errors = append(errors, fmt.Sprintf("Chunk %d: %s", transcription.Index, transcription.Error))
			continue
		}

		if transcription.Text != "" {
			if fullText.Len() > 0 {
				fullText.WriteString(" ")
			}
			fullText.WriteString(transcription.Text)

			// Create timeline mappings at specified intervals
			for timestamp := transcription.StartTime; timestamp < transcription.EndTime; timestamp += w.MappingDuration {
				timelineMappings = append(timelineMappings, TimelineMapping{
					Timestamp: timestamp,
					Text:      transcription.Text,
				})
			}
		}
	}

	// Create metadata
	metadata := map[string]interface{}{
		"source_type":       "whisper",
		"media_type":        mediaType,
		"model":             w.Model,
		"language":          w.Language,
		"chunk_duration":    w.ChunkDuration,
		"mapping_duration":  w.MappingDuration,
		"total_chunks":      len(transcriptions),
		"timeline_mappings": timelineMappings,
		"silence_detection": w.EnableSilenceDetection,
		"text_length":       fullText.Len(),
	}

	if len(errors) > 0 {
		metadata["errors"] = errors
	}

	return &types.ConvertResult{
		Text:     fullText.String(),
		Metadata: metadata,
	}
}

// reportProgress reports conversion progress
func (w *Whisper) reportProgress(status types.ConverterStatus, message string, progress float64, callbacks ...types.ConverterProgress) {
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
func (w *Whisper) Close() error {
	if w.FFmpeg != nil {
		return w.FFmpeg.Close()
	}
	return nil
}
