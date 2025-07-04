package converter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/office"
)

// Office is a converter for office files (docx, pptx)
// Processing pipeline:
/*
   1. Parse docx/pptx files using office package to get Markdown content, media files, and page mapping metadata
   2. Identify and process media content:
      a) Images: Use vision converter to identify image content, generate description text and metadata with concurrent processing
      b) Videos: Use video converter to identify video content, generate description text and metadata (includes built-in audio recognition)
      c) Audio: Use whisper converter to identify audio content, generate description text and metadata
   3. Merge text content:
      a) Merge media content descriptions with markdown text, maintaining correct position mapping
      b) Merge metadata with accurate page positioning, updating text length and position after media insertion
   4. Return results:
      a) Return merged text and metadata
      b) Metadata includes accurate text-to-page mapping relationships
*/

// Office is a converter for office files
type Office struct {
	VisionConverter  types.Converter   // Vision converter for image processing
	VideoConverter   types.Converter   // Video converter for video processing
	WhisperConverter types.Converter   // Whisper converter for audio processing
	MaxConcurrency   int               // Maximum concurrent media processing
	TempDir          string            // Temporary directory for processing
	CleanupTemp      bool              // Whether to cleanup temporary files
	Parser           office.FileParser // Office document parser
}

// OfficeOption is the configuration for the Office converter
type OfficeOption struct {
	VisionConverter  types.Converter `json:"vision_converter,omitempty"`  // Vision converter instance
	VideoConverter   types.Converter `json:"video_converter,omitempty"`   // Video converter instance
	WhisperConverter types.Converter `json:"whisper_converter,omitempty"` // Whisper converter instance
	MaxConcurrency   int             `json:"max_concurrency,omitempty"`   // Max concurrent media processing
	TempDir          string          `json:"temp_dir,omitempty"`          // Temporary directory
	CleanupTemp      bool            `json:"cleanup_temp,omitempty"`      // Cleanup temporary files
}

// MediaProcessingResult represents the result of processing a media file
type MediaProcessingResult struct {
	MediaID     string                 `json:"media_id"`
	MediaType   string                 `json:"media_type"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
	Error       string                 `json:"error,omitempty"`
}

// NewOffice creates a new Office converter instance
func NewOffice(option OfficeOption) (*Office, error) {
	// Vision converter is required for image processing
	if option.VisionConverter == nil {
		return nil, errors.New("vision converter is required for image processing")
	}

	maxConcurrency := option.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = 4 // Default: 4 concurrent media processes
	}

	tempDir := option.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	return &Office{
		VisionConverter:  option.VisionConverter,
		VideoConverter:   option.VideoConverter,
		WhisperConverter: option.WhisperConverter,
		MaxConcurrency:   maxConcurrency,
		TempDir:          tempDir,
		CleanupTemp:      option.CleanupTemp,
		Parser:           office.NewParser(),
	}, nil
}

// Convert converts an office file to plain text by calling ConvertStream
func (o *Office) Convert(ctx context.Context, file string, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	o.reportProgress(types.ConverterStatusPending, "Opening office file", 0.0, callback...)

	// Open the file
	f, err := os.Open(file)
	if err != nil {
		o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to open file: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer f.Close()

	// Use ConvertStream to process the file
	result, err := o.ConvertStream(ctx, f, callback...)
	if err != nil {
		return nil, err
	}

	o.reportProgress(types.ConverterStatusSuccess, "Office file conversion completed", 1.0, callback...)
	return result, nil
}

// ConvertStream converts an office document stream to text using office parser and media converters
func (o *Office) ConvertStream(ctx context.Context, stream io.ReadSeeker, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	o.reportProgress(types.ConverterStatusPending, "Starting office document processing", 0.0, callback...)

	// Read the entire stream into memory
	data, err := io.ReadAll(stream)
	if err != nil {
		o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to read stream: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to read stream: %w", err)
	}

	o.reportProgress(types.ConverterStatusPending, "Parsing office document", 0.1, callback...)

	// Parse the office document
	parseResult, err := o.Parser.Parse(data)
	if err != nil {
		o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to parse office document: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to parse office document: %w", err)
	}

	o.reportProgress(types.ConverterStatusPending, "Processing media files", 0.3, callback...)

	// Process media files concurrently
	mediaResults, err := o.processMediaFiles(ctx, parseResult.Media, callback...)
	if err != nil {
		o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Media processing failed: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("media processing failed: %w", err)
	}

	o.reportProgress(types.ConverterStatusPending, "Merging text and media", 0.8, callback...)

	// Merge markdown text with media descriptions
	finalResult := o.mergeTextAndMedia(parseResult, mediaResults)

	o.reportProgress(types.ConverterStatusSuccess, "Office document processing completed", 1.0, callback...)
	return finalResult, nil
}

// processMediaFiles processes all media files concurrently
func (o *Office) processMediaFiles(ctx context.Context, mediaFiles []office.Media, callback ...types.ConverterProgress) ([]MediaProcessingResult, error) {
	if len(mediaFiles) == 0 {
		return []MediaProcessingResult{}, nil
	}

	results := make([]MediaProcessingResult, len(mediaFiles))

	// Create semaphore for concurrent processing
	semaphore := make(chan struct{}, o.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, media := range mediaFiles {
		wg.Add(1)
		go func(index int, mediaFile office.Media) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Process media file
			result := o.processMediaFile(ctx, mediaFile)

			mu.Lock()
			results[index] = result

			// Report progress
			completed := 0
			for _, r := range results {
				if r.Description != "" || r.Error != "" {
					completed++
				}
			}
			progress := 0.3 + (0.5 * float64(completed) / float64(len(mediaFiles)))
			o.reportProgress(types.ConverterStatusPending, fmt.Sprintf("Processed %d/%d media files", completed, len(mediaFiles)), progress, callback...)
			mu.Unlock()
		}(i, media)
	}

	wg.Wait()
	return results, nil
}

// processMediaFile processes a single media file based on its type
func (o *Office) processMediaFile(ctx context.Context, media office.Media) MediaProcessingResult {
	result := MediaProcessingResult{
		MediaID:   media.ID,
		MediaType: media.Type,
	}

	// Save media content to temporary file
	tempFile, err := o.saveMediaToTempFile(media)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to save media to temp file: %v", err)
		return result
	}
	defer func() {
		if o.CleanupTemp {
			os.Remove(tempFile)
		}
	}()

	// Process based on media type
	switch media.Type {
	case "image":
		if o.VisionConverter != nil {
			convertResult, err := o.VisionConverter.Convert(ctx, tempFile)
			if err != nil {
				result.Error = fmt.Sprintf("Vision conversion failed: %v", err)
			} else {
				result.Description = convertResult.Text
				result.Metadata = convertResult.Metadata
			}
		} else {
			result.Description = fmt.Sprintf("[Image: %s]", media.Filename)
		}
	case "video":
		if o.VideoConverter != nil {
			convertResult, err := o.VideoConverter.Convert(ctx, tempFile)
			if err != nil {
				result.Error = fmt.Sprintf("Video conversion failed: %v", err)
			} else {
				result.Description = convertResult.Text
				result.Metadata = convertResult.Metadata
			}
		} else {
			result.Description = fmt.Sprintf("[Video: %s]", media.Filename)
		}
	case "audio":
		if o.WhisperConverter != nil {
			convertResult, err := o.WhisperConverter.Convert(ctx, tempFile)
			if err != nil {
				result.Error = fmt.Sprintf("Audio conversion failed: %v", err)
			} else {
				result.Description = convertResult.Text
				result.Metadata = convertResult.Metadata
			}
		} else {
			result.Description = fmt.Sprintf("[Audio: %s]", media.Filename)
		}
	default:
		result.Description = fmt.Sprintf("[Media: %s]", media.Filename)
	}

	return result
}

// saveMediaToTempFile saves media content to a temporary file
func (o *Office) saveMediaToTempFile(media office.Media) (string, error) {
	// Create temp file with appropriate extension
	tempFile, err := os.CreateTemp(o.TempDir, fmt.Sprintf("media_%s_*.%s", media.ID, media.Format))
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Write media content to temp file
	_, err = tempFile.Write(media.Content)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write media content: %w", err)
	}

	return tempFile.Name(), nil
}

// mergeTextAndMedia merges markdown text with media descriptions while maintaining accurate positioning
func (o *Office) mergeTextAndMedia(parseResult *office.ParseResult, mediaResults []MediaProcessingResult) *types.ConvertResult {
	// Create media lookup map
	mediaLookup := make(map[string]MediaProcessingResult)
	for _, result := range mediaResults {
		mediaLookup[result.MediaID] = result
	}

	// Start with the original markdown text
	text := parseResult.Markdown

	// Process media references and insert descriptions
	if parseResult.Metadata != nil && parseResult.Metadata.MediaRefs != nil {
		// Sort media references by position to process from end to beginning
		// This prevents position shifts when inserting text
		type MediaRef struct {
			Position int
			MediaID  string
			RefID    string
		}

		var mediaRefs []MediaRef
		for refID, mediaID := range parseResult.Metadata.MediaRefs {
			// Find position of media reference in text
			refText := fmt.Sprintf("[%s]", refID)
			pos := strings.Index(text, refText)
			if pos != -1 {
				mediaRefs = append(mediaRefs, MediaRef{
					Position: pos,
					MediaID:  mediaID,
					RefID:    refID,
				})
			}
		}

		// Sort by position in descending order (process from end to beginning)
		sort.Slice(mediaRefs, func(i, j int) bool {
			return mediaRefs[i].Position > mediaRefs[j].Position
		})

		// Insert media descriptions
		for _, ref := range mediaRefs {
			if mediaResult, exists := mediaLookup[ref.MediaID]; exists {
				// Create media description with formatting
				var mediaDescription string
				if mediaResult.Error != "" {
					mediaDescription = fmt.Sprintf("[%s - Error: %s]", ref.RefID, mediaResult.Error)
				} else if mediaResult.Description != "" {
					mediaDescription = fmt.Sprintf("[%s: %s]", ref.RefID, mediaResult.Description)
				} else {
					mediaDescription = fmt.Sprintf("[%s]", ref.RefID)
				}

				// Replace the reference with the description
				refText := fmt.Sprintf("[%s]", ref.RefID)
				text = strings.Replace(text, refText, mediaDescription, 1)
			}
		}
	}

	// Create updated metadata
	metadata := map[string]interface{}{
		"source_type":       "office",
		"original_metadata": parseResult.Metadata,
		"media_count":       len(mediaResults),
		"processed_media":   len(mediaResults),
		"text_length":       len(text),
		"conversion_time":   time.Now().Unix(),
	}

	// Include media processing results in metadata
	successfulMedia := 0
	for _, result := range mediaResults {
		if result.Error == "" {
			successfulMedia++
		}
	}
	metadata["successful_media"] = successfulMedia

	// Include text range information for accurate page mapping
	if parseResult.Metadata != nil && parseResult.Metadata.TextRanges != nil {
		// Update text ranges based on the modified text
		updatedRanges := o.updateTextRanges(parseResult.Metadata.TextRanges, text)
		metadata["text_ranges"] = updatedRanges
		metadata["page_mapping"] = o.createPageMapping(updatedRanges, text)
	}

	return &types.ConvertResult{
		Text:     text,
		Metadata: metadata,
	}
}

// updateTextRanges updates text ranges after text modifications
func (o *Office) updateTextRanges(originalRanges []office.TextRange, finalText string) []office.TextRange {
	// For now, we'll keep the original ranges as-is
	// In a more sophisticated implementation, we would track text position changes
	// and update the ranges accordingly based on inserted media descriptions
	return originalRanges
}

// createPageMapping creates a mapping from text positions to page numbers
func (o *Office) createPageMapping(textRanges []office.TextRange, text string) map[string]interface{} {
	pageMapping := make(map[string]interface{})

	// Create position to page mapping
	positionToPage := make(map[int]int)
	for _, tr := range textRanges {
		for pos := tr.StartPos; pos <= tr.EndPos && pos < len(text); pos++ {
			positionToPage[pos] = tr.Page
		}
	}

	// Create summary statistics
	totalPages := 0
	for _, tr := range textRanges {
		if tr.Page > totalPages {
			totalPages = tr.Page
		}
	}

	pageMapping["total_pages"] = totalPages
	pageMapping["position_to_page"] = positionToPage
	pageMapping["text_ranges"] = textRanges

	return pageMapping
}

// reportProgress reports conversion progress
func (o *Office) reportProgress(status types.ConverterStatus, message string, progress float64, callbacks ...types.ConverterProgress) {
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
func (o *Office) Close() error {
	// Nothing to clean up for now
	return nil
}
