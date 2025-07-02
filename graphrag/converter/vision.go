package converter

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"os"
	"strings"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"

	// Import WebP decoder
	_ "golang.org/x/image/webp"
)

// Vision implements the Converter interface for vision conversion
type Vision struct {
	Connector    connector.Connector
	Model        string
	Prompt       string
	Options      map[string]any
	CompressSize int64
	Language     string // Output language parameter, default is "Auto"
}

// VisionOption is the option for the Vision instance
type VisionOption struct {
	ConnectorName string         `json:"connector,omitempty"`     // Connector name
	Model         string         `json:"model,omitempty"`         // Model name
	Prompt        string         `json:"prompt,omitempty"`        // Prompt template
	Options       map[string]any `json:"options,omitempty"`       // Options
	CompressSize  int64          `json:"compress_size,omitempty"` // Compress size
	Language      string         `json:"language,omitempty"`      // Output language, default is "Auto"
}

// NewVision creates a new Vision instance
func NewVision(option VisionOption) (*Vision, error) {
	c, err := connector.Select(option.ConnectorName)
	if err != nil {
		return nil, err
	}

	if !c.Is(connector.OPENAI) {
		return nil, errors.New("connector is not a openai connector")
	}

	prompt := option.Prompt
	if prompt == "" {
		prompt = defaultPromptTemplate()
	}

	compressSize := option.CompressSize
	if compressSize == 0 {
		compressSize = 1024 // The max width or height of the image is 1024px
	}

	language := option.Language
	if language == "" {
		language = "Auto" // Default to auto-detect language from image content
	}

	return &Vision{
		Connector:    c,
		Model:        option.Model,
		Prompt:       prompt,
		Options:      option.Options,
		CompressSize: compressSize,
		Language:     language,
	}, nil
}

// Convert converts a file to plain text by calling ConvertStream
func (v *Vision) Convert(ctx context.Context, file string, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	v.reportProgress(types.ConverterStatusPending, "Opening file", 0.0, callback...)

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

	v.reportProgress(types.ConverterStatusSuccess, "File conversion completed", 1.0, callback...)
	return result, nil
}

// ConvertStream converts an image stream to text description using vision AI
func (v *Vision) ConvertStream(ctx context.Context, stream io.ReadSeeker, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	v.reportProgress(types.ConverterStatusPending, "Starting image processing", 0.0, callback...)

	// Check if gzipped
	var reader io.Reader
	peekBuffer := make([]byte, 2)

	n, err := stream.Read(peekBuffer)
	if err != nil && err != io.EOF {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to peek stream: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to peek stream: %w", err)
	}

	// Reset stream position
	_, err = stream.Seek(0, io.SeekStart)
	if err != nil {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to reset stream: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to reset stream: %w", err)
	}

	// Check for gzip magic bytes (0x1f, 0x8b)
	if n >= 2 && peekBuffer[0] == 0x1f && peekBuffer[1] == 0x8b {
		v.reportProgress(types.ConverterStatusPending, "Detected gzip compression, decompressing...", 0.1, callback...)

		gzReader, err := gzip.NewReader(stream)
		if err != nil {
			v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to create gzip reader: %v", err), 0.0, callback...)
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	} else {
		reader = stream
	}

	// Read the entire stream into memory
	data, err := io.ReadAll(reader)
	if err != nil {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to read stream: %v", err), 0.0, callback...)
		return nil, fmt.Errorf("failed to read stream: %w", err)
	}

	v.reportProgress(types.ConverterStatusPending, "Validating image format", 0.2, callback...)

	// Validate and process image
	processedData, contentType, err := v.validateAndProcessImage(data)
	if err != nil {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Image validation failed: %v", err), 0.0, callback...)
		return nil, err
	}

	v.reportProgress(types.ConverterStatusPending, "Compressing image", 0.4, callback...)

	// Compress image if needed
	compressedData, err := v.compressImage(processedData, contentType)
	if err != nil {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("Image compression failed: %v", err), 0.0, callback...)
		return nil, err
	}

	v.reportProgress(types.ConverterStatusPending, "Converting to base64", 0.6, callback...)

	// Convert to base64
	base64Data := base64.StdEncoding.EncodeToString(compressedData)

	v.reportProgress(types.ConverterStatusPending, "Processing with LLM", 0.8, callback...)

	// Process with LLM
	result, err := v.processWithLLM(ctx, base64Data, contentType, callback...)
	if err != nil {
		v.reportProgress(types.ConverterStatusError, fmt.Sprintf("LLM processing failed: %v", err), 0.0, callback...)
		return nil, err
	}

	v.reportProgress(types.ConverterStatusSuccess, "Vision conversion completed", 1.0, callback...)
	return result, nil
}

// validateAndProcessImage validates image format and converts unsupported formats to JPEG
func (v *Vision) validateAndProcessImage(data []byte) ([]byte, string, error) {
	// Detect image format
	contentType := mime.TypeByExtension(getImageFormat(data))
	if contentType == "" {
		// Try to detect by content
		contentType = detectImageType(data)
	}

	// Check if it's an image
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", errors.New("file is not an image")
	}

	// Check supported formats: PNG, JPEG, WEBP, GIF
	supportedFormats := map[string]bool{
		"image/png":  true,
		"image/jpeg": true,
		"image/webp": true,
		"image/gif":  true,
	}

	if supportedFormats[contentType] {
		return data, contentType, nil
	}

	// Convert unsupported format to JPEG
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode image as JPEG: %w", err)
	}

	return buf.Bytes(), "image/jpeg", nil
}

// compressImage compresses the image based on CompressSize
func (v *Vision) compressImage(data []byte, contentType string) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	maxSize := int(v.CompressSize)

	// Check if compression is needed
	if width <= maxSize && height <= maxSize {
		return data, nil
	}

	// Calculate new dimensions maintaining aspect ratio
	var newWidth, newHeight int
	if width > height {
		newWidth = maxSize
		newHeight = int(float64(height) * (float64(maxSize) / float64(width)))
	} else {
		newHeight = maxSize
		newWidth = int(float64(width) * (float64(maxSize) / float64(height)))
	}

	// Create new image with new dimensions using simple scaling
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) * float64(width) / float64(newWidth))
			srcY := int(float64(y) * float64(height) / float64(newHeight))
			newImg.Set(x, y, img.At(srcX, srcY))
		}
	}

	// Encode with the same format
	var buf bytes.Buffer
	switch contentType {
	case "image/png":
		err = png.Encode(&buf, newImg)
	case "image/jpeg":
		err = jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 85})
	case "image/gif":
		err = gif.Encode(&buf, newImg, nil)
	case "image/webp":
		// For WebP, we'll convert to JPEG since Go's webp package is read-only
		err = jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 85})
	default:
		err = jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode compressed image: %w", err)
	}

	return buf.Bytes(), nil
}

// processWithLLM sends the base64 image to LLM and returns the description
func (v *Vision) processWithLLM(ctx context.Context, base64Data, contentType string, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	// Get model from Vision settings or connector
	model := v.Model
	if model == "" {
		setting := v.Connector.Setting()
		if connectorModel, ok := setting["model"].(string); ok && connectorModel != "" {
			model = connectorModel
		} else {
			model = "gpt-4o-mini" // Default model for vision
		}
	}

	// Generate the actual prompt with language instruction
	actualPrompt := v.generatePrompt()

	// Prepare messages for vision API
	messages := []map[string]interface{}{
		{
			"role":    "system",
			"content": actualPrompt,
		},
		{
			"role": "user",
			"content": []map[string]interface{}{
				{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": fmt.Sprintf("data:%s;base64,%s", contentType, base64Data),
					},
				},
			},
		},
	}

	// Prepare payload
	payload := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"max_tokens":  1000,
		"temperature": 0.1,
	}

	// Add any additional options
	if v.Options != nil {
		for k, val := range v.Options {
			payload[k] = val
		}
	}

	// Collect response content directly
	var resultContent strings.Builder

	// Stream callback to collect text content
	streamCallback := func(data []byte) error {
		if len(data) == 0 {
			return nil
		}

		// Parse SSE streaming data to extract content
		content := extractContentFromStreamChunk(data)
		if content != "" {
			resultContent.WriteString(content)
			// Report streaming progress with real-time content
			currentText := resultContent.String()
			v.reportProgress(types.ConverterStatusPending, currentText, 0.9, callback...)
		}

		return nil
	}

	// Make streaming request
	err := utils.StreamLLM(ctx, v.Connector, "chat/completions", payload, streamCallback)
	if err != nil {
		return nil, fmt.Errorf("streaming request failed: %w", err)
	}

	// Get the accumulated text description
	description := strings.TrimSpace(resultContent.String())
	if description == "" {
		return nil, errors.New("no description received from LLM")
	}

	// Create metadata with vision-specific information
	metadata := map[string]interface{}{
		"source_type":        "vision",
		"content_type":       contentType,
		"model":              model,
		"language":           v.Language,
		"compress_size":      v.CompressSize,
		"description_length": len(description),
	}

	return &types.ConvertResult{
		Text:     description,
		Metadata: metadata,
	}, nil
}

// extractContentFromStreamChunk extracts content from SSE streaming chunk
func extractContentFromStreamChunk(data []byte) string {
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			jsonData := strings.TrimPrefix(line, "data: ")
			if jsonData == "[DONE]" {
				continue
			}

			// Simple JSON parsing for content extraction
			if strings.Contains(jsonData, `"content"`) {
				// Extract content between quotes after "content":
				start := strings.Index(jsonData, `"content":"`)
				if start != -1 {
					start += len(`"content":"`)
					end := strings.Index(jsonData[start:], `"`)
					if end != -1 {
						return jsonData[start : start+end]
					}
				}
			}
		}
	}
	return ""
}

// getImageFormat detects image format from data
func getImageFormat(data []byte) string {
	if len(data) < 12 {
		return ""
	}

	// Check PNG signature
	if bytes.Equal(data[0:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return ".png"
	}

	// Check JPEG signature
	if bytes.Equal(data[0:3], []byte{0xFF, 0xD8, 0xFF}) {
		return ".jpg"
	}

	// Check GIF signature
	if bytes.Equal(data[0:6], []byte{0x47, 0x49, 0x46, 0x38, 0x37, 0x61}) ||
		bytes.Equal(data[0:6], []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}) {
		return ".gif"
	}

	// Check WebP signature
	if bytes.Equal(data[0:4], []byte{0x52, 0x49, 0x46, 0x46}) &&
		bytes.Equal(data[8:12], []byte{0x57, 0x45, 0x42, 0x50}) {
		return ".webp"
	}

	return ""
}

// detectImageType detects image MIME type from data
func detectImageType(data []byte) string {
	format := getImageFormat(data)
	switch format {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}

// reportProgress reports conversion progress
func (v *Vision) reportProgress(status types.ConverterStatus, message string, progress float64, callbacks ...types.ConverterProgress) {
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

// defaultPromptTemplate returns the default prompt template for the Vision instance
func defaultPromptTemplate() string {
	return `Please provide a comprehensive and detailed description of this image. Include:

1. Overall scene and setting
2. Main objects, people, or subjects present
3. Colors, lighting, and visual style
4. Any text, symbols, or signs visible
5. Spatial relationships and layout
6. Actions, expressions, or movements
7. Background and foreground elements
8. Any other notable details or characteristics

{LANGUAGE_INSTRUCTION}

Describe what you see clearly and objectively, providing enough detail for someone who cannot see the image to understand its content.`
}

// generatePrompt generates the actual prompt based on the template and language setting
func (v *Vision) generatePrompt() string {
	template := v.Prompt
	languageInstruction := v.getLanguageInstruction()

	// Replace the language instruction placeholder
	return strings.Replace(template, "{LANGUAGE_INSTRUCTION}", languageInstruction, 1)
}

// getLanguageInstruction returns the language instruction based on the Language setting
func (v *Vision) getLanguageInstruction() string {
	if v.Language == "Auto" || v.Language == "" {
		return "Please respond in the most appropriate language based on the image's content (e.g., use Chinese if the image contains Chinese text, English if it contains English text, etc.)."
	}
	// For specific languages, directly use the user input
	return fmt.Sprintf("Please respond in %s.", v.Language)
}
