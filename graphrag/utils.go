package graphrag

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/graphrag/chunking"
	"github.com/yaoapp/gou/graphrag/converter"
	"github.com/yaoapp/gou/graphrag/embedding"
	"github.com/yaoapp/gou/graphrag/extraction/openai"
	"github.com/yaoapp/gou/graphrag/fetcher"
	"github.com/yaoapp/gou/graphrag/types"
)

// MakeUpsertCallback creates a new UpsertCallback
func MakeUpsertCallback(id string, chunks *[]*types.Chunk, cb types.UpsertProgress) types.UpsertCallback {

	// If the callback is nil, use the default callback
	if cb == nil {
		cb = func(id string, status types.UpsertProgressType, payload types.UpsertProgressPayload) {}
	}

	return types.UpsertCallback{
		Converter: func(status types.ConverterStatus, payload types.ConverterPayload) {
			// Report converter progress
			cb(id, types.UpsertProgressTypeConverter, types.UpsertProgressPayload{
				ID:       id,
				Progress: payload.Progress,
				Type:     types.UpsertProgressTypeConverter,
				Data: map[string]interface{}{
					"status":  status,
					"message": payload.Message,
				},
			})
		},
		Chunking: func(chunk *types.Chunk) error {
			// Add chunk to the slice if pointer is provided
			if chunks != nil {
				*chunks = append(*chunks, chunk)
			}

			// Report chunking progress
			cb(id, types.UpsertProgressTypeChunking, types.UpsertProgressPayload{
				ID:       id,
				Progress: 0, // Chunking progress is not easily measurable per chunk
				Type:     types.UpsertProgressTypeChunking,
				Data: map[string]interface{}{
					"chunk_id":    chunk.ID,
					"chunk_index": chunk.Index,
					"chunk_size":  len(chunk.Text),
					"chunk_depth": chunk.Depth,
				},
			})

			return nil
		},
		Embedding: func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
			// Report embedding progress
			progress := 0.0
			if payload.Total > 0 {
				progress = float64(payload.Current) / float64(payload.Total) * 100
			}
			cb(id, types.UpsertProgressTypeEmbedding, types.UpsertProgressPayload{
				ID:       id,
				Progress: progress,
				Type:     types.UpsertProgressTypeEmbedding,
				Data: map[string]interface{}{
					"status":  status,
					"message": payload.Message,
					"current": payload.Current,
					"total":   payload.Total,
				},
			})
		},
		Extraction: func(status types.ExtractionStatus, payload types.ExtractionPayload) {
			// Report extraction progress
			progress := 0.0
			if payload.Total > 0 {
				progress = float64(payload.Current) / float64(payload.Total) * 100
			}
			cb(id, types.UpsertProgressTypeExtraction, types.UpsertProgressPayload{
				ID:       id,
				Progress: progress,
				Type:     types.UpsertProgressTypeExtraction,
				Data: map[string]interface{}{
					"status":  status,
					"message": payload.Message,
					"current": payload.Current,
					"total":   payload.Total,
				},
			})
		},
		Fetcher: func(status types.FetcherStatus, payload types.FetcherPayload) {
			// Report fetcher progress
			cb(id, types.UpsertProgressTypeFetcher, types.UpsertProgressPayload{
				ID:       id,
				Progress: payload.Progress,
				Type:     types.UpsertProgressTypeFetcher,
				Data: map[string]interface{}{
					"status":  status,
					"message": payload.Message,
					"url":     payload.URL,
					"bytes":   payload.Bytes,
				},
			})
		},
	}
}

// DetectConverter detects the converter to use for the file
func DetectConverter(file string) (types.Converter, error) {
	if file == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Get file extension and convert to lowercase
	ext := strings.ToLower(filepath.Ext(file))
	filename := strings.ToLower(filepath.Base(file))

	// Check for compressed files (.gz suffix)
	isGzipped := strings.HasSuffix(filename, ".gz")
	if isGzipped {
		// Remove .gz and get the actual file extension
		nameWithoutGz := strings.TrimSuffix(filename, ".gz")
		ext = filepath.Ext(nameWithoutGz)
	}

	switch ext {
	// Image files - use Vision converter
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tiff", ".svg":
		return converter.NewVision(converter.VisionOption{
			ConnectorName: "openai", // Default connector, should be configured elsewhere
			Model:         "gpt-4o-mini",
			CompressSize:  1024,
			Language:      "Auto",
			Options:       map[string]any{"max_tokens": 1000, "temperature": 0.1},
		})

	// PDF files - use OCR converter
	case ".pdf":
		// Need Vision converter for OCR
		visionConverter, err := converter.NewVision(converter.VisionOption{
			ConnectorName: "openai",
			Model:         "gpt-4o-mini",
			CompressSize:  1024,
			Language:      "Auto",
			Options:       map[string]any{"max_tokens": 1000, "temperature": 0.1},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create vision converter for OCR: %w", err)
		}

		return converter.NewOCR(converter.OCROption{
			Vision:         visionConverter,
			Mode:           converter.OCRModeConcurrent,
			MaxConcurrency: 4,
			CompressSize:   1024,
			ForceImageMode: true, // 关键：PDF需要强制图像模式
		})

	// Office documents - use Office converter
	case ".doc", ".docx", ".ppt", ".pptx":
		// Need Vision converter for Office documents
		visionConverter, err := converter.NewVision(converter.VisionOption{
			ConnectorName: "openai",
			Model:         "gpt-4o-mini",
			CompressSize:  1024,
			Language:      "Auto",
			Options:       map[string]any{"max_tokens": 1000, "temperature": 0.1},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create vision converter for Office: %w", err)
		}

		return converter.NewOffice(converter.OfficeOption{
			VisionConverter: visionConverter,
			MaxConcurrency:  4,
			CleanupTemp:     true,
		})

	// Video files - use Video converter
	case ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".mkv":
		// Need both Vision and Whisper converters for Video
		visionConverter, err := converter.NewVision(converter.VisionOption{
			ConnectorName: "openai",
			Model:         "gpt-4o-mini",
			CompressSize:  1024,
			Language:      "Auto",
			Options:       map[string]any{"max_tokens": 1000, "temperature": 0.1},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create vision converter for Video: %w", err)
		}

		whisperConverter, err := converter.NewWhisper(converter.WhisperOption{
			ConnectorName:          "openai",
			Model:                  "whisper-1",
			Language:               "en",
			ChunkDuration:          30.0,
			MappingDuration:        5.0,
			SilenceThreshold:       -40.0,
			SilenceMinLength:       1.0,
			EnableSilenceDetection: true,
			MaxConcurrency:         4,
			CleanupTemp:            true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create whisper converter for Video: %w", err)
		}

		return converter.NewVideo(converter.VideoOption{
			AudioConverter:     whisperConverter,
			VisionConverter:    visionConverter,
			KeyframeInterval:   30.0,
			MaxKeyframes:       10,
			TempDir:            "",
			CleanupTemp:        true,
			MaxConcurrency:     4,
			TextOptimization:   true,
			DeduplicationRatio: 0.8,
		})

	// Audio files - use Whisper converter
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma", ".m4a":
		return converter.NewWhisper(converter.WhisperOption{
			ConnectorName:          "openai",
			Model:                  "whisper-1",
			Language:               "auto",
			ChunkDuration:          30.0,
			MappingDuration:        5.0,
			SilenceThreshold:       -40.0,
			SilenceMinLength:       1.0,
			EnableSilenceDetection: true,
			MaxConcurrency:         4,
			CleanupTemp:            true,
		})

	// Plain text files - use UTF8 converter
	case ".txt", ".md", ".html", ".xml", ".rtf", ".json", ".csv", ".log",
		".go", ".py", ".java", ".c", ".cpp", ".cs", ".js", ".ts", ".php", ".rb", ".sh", ".yml", ".yaml":
		return converter.NewUTF8(), nil

	default:
		// For unknown file types, try UTF8 converter as fallback
		return converter.NewUTF8(), nil
	}
}

// DetectChunking detects the chunking function to use for the file
func DetectChunking(file string) (types.Chunking, error) {
	if file == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Return structured chunker directly - Go interface system will handle the compatibility
	chunker := chunking.NewStructuredChunker()
	return chunker, nil
}

// DetectExtractor detects the extractor function to use for the file
func DetectExtractor(file string) (types.Extraction, error) {
	if file == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Get the chunking type to determine extraction approach
	chunkingType := types.GetChunkingTypeFromFilename(file)

	// For most content types, use OpenAI extraction with default settings
	// Different content types might benefit from different extraction strategies
	switch chunkingType {
	case types.ChunkingTypeCode:
		// For code files, use lower temperature for more precise extraction
		return openai.NewOpenai(openai.Options{
			ConnectorName: "openai", // Default connector
			Concurrent:    3,        // Lower concurrency for code
			Model:         "gpt-4o-mini",
			Temperature:   0.0, // Very low temperature for code
			MaxTokens:     4000,
			Toolcall:      nil, // Use default (true)
			RetryAttempts: 3,
		})

	case types.ChunkingTypeJSON:
		// For JSON files, use structured extraction
		return openai.NewOpenai(openai.Options{
			ConnectorName: "openai",
			Concurrent:    3,
			Model:         "gpt-4o-mini",
			Temperature:   0.0, // Low temperature for structured data
			MaxTokens:     4000,
			Toolcall:      nil,
			RetryAttempts: 3,
		})

	case types.ChunkingTypeVideo, types.ChunkingTypeAudio, types.ChunkingTypeImage:
		// For media files, use higher concurrency but standard settings
		return openai.NewOpenai(openai.Options{
			ConnectorName: "openai",
			Concurrent:    5,
			Model:         "gpt-4o-mini",
			Temperature:   0.1,
			MaxTokens:     4000,
			Toolcall:      nil,
			RetryAttempts: 3,
		})

	case types.ChunkingTypePDF, types.ChunkingTypeWord:
		// For document files, use standard settings with higher token limit
		return openai.NewOpenai(openai.Options{
			ConnectorName: "openai",
			Concurrent:    4,
			Model:         "gpt-4o-mini",
			Temperature:   0.1,
			MaxTokens:     6000, // Higher token limit for documents
			Toolcall:      nil,
			RetryAttempts: 3,
		})

	default:
		// Default extraction settings for text and other types
		return openai.NewOpenai(openai.Options{
			ConnectorName: "openai",
			Concurrent:    5,
			Model:         "gpt-4o-mini",
			Temperature:   0.1,
			MaxTokens:     4000,
			Toolcall:      nil,
			RetryAttempts: 3,
		})
	}
}

// DetectEmbedding detects the embedding function to use for the file
func DetectEmbedding(file string) (types.Embedding, error) {
	if file == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Get the chunking type to determine embedding approach
	chunkingType := types.GetChunkingTypeFromFilename(file)

	// Different content types might benefit from different embedding models
	switch chunkingType {
	case types.ChunkingTypeCode:
		// For code files, use standard OpenAI embedding with higher concurrency
		return embedding.NewOpenai(embedding.OpenaiOptions{
			ConnectorName: "openai",
			Concurrent:    8, // Higher concurrency for code
			Dimension:     1536,
			Model:         "text-embedding-3-small", // Good for code
		})

	case types.ChunkingTypeJSON, types.ChunkingTypeCSV:
		// For structured data, use standard embedding
		return embedding.NewOpenai(embedding.OpenaiOptions{
			ConnectorName: "openai",
			Concurrent:    8,
			Dimension:     1536,
			Model:         "text-embedding-3-small",
		})

	case types.ChunkingTypeVideo, types.ChunkingTypeAudio, types.ChunkingTypeImage:
		// For media files, use standard embedding but lower concurrency
		return embedding.NewOpenai(embedding.OpenaiOptions{
			ConnectorName: "openai",
			Concurrent:    6,
			Dimension:     1536,
			Model:         "text-embedding-3-small",
		})

	case types.ChunkingTypePDF, types.ChunkingTypeWord:
		// For document files, use standard settings
		return embedding.NewOpenai(embedding.OpenaiOptions{
			ConnectorName: "openai",
			Concurrent:    10,
			Dimension:     1536,
			Model:         "text-embedding-3-small",
		})

	default:
		// Default embedding settings for text and other types
		return embedding.NewOpenai(embedding.OpenaiOptions{
			ConnectorName: "openai",
			Concurrent:    10,
			Dimension:     1536,
			Model:         "text-embedding-3-small",
		})
	}
}

// DetectFetcher detects the fetcher to use for the URL
//
// Supported fetchers:
// - HTTP/HTTPS: Uses optimized gou/http package with DNS optimization and connection pooling
// - MCP: Uses MCP client for custom URL fetching (requires manual configuration)
//
// For MCP fetcher, use fetcher.NewMCP() with appropriate MCPOptions
func DetectFetcher(url string) (types.Fetcher, error) {
	if url == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// HTTP/HTTPS URLs - use optimized HTTP fetcher
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return fetcher.NewHTTPFetcher(), nil
	}

	// Future implementations could support:
	// - FTP: file transfer protocol
	// - file://: local file system
	// - s3://: Amazon S3
	// - gs://: Google Cloud Storage

	return nil, fmt.Errorf("unsupported URL scheme: %s", url)
}
