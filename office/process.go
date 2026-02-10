package office

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// defaultParser is the default Office parser instance
var defaultParser *Parser

func init() {
	process.RegisterGroup("office", map[string]process.Handler{
		"parse":      ProcessParse,
		"parsebytes": ProcessParseBytes,
	})

	// Create default parser instance
	defaultParser = NewParser()
}

// ProcessParseResult is the result returned by office process handlers.
// Media files are saved to output_dir; media[].path contains the file path.
type ProcessParseResult struct {
	Markdown string              `json:"markdown"`
	Metadata *Metadata           `json:"metadata"`
	Media    []ProcessMediaEntry `json:"media"`
}

// ProcessMediaEntry represents a media entry with file path (no base64).
type ProcessMediaEntry struct {
	ID       string `json:"id"`
	Type     string `json:"type"`     // "image", "video", "audio"
	Format   string `json:"format"`   // "png", "jpeg", "gif", etc.
	Filename string `json:"filename"` // Original filename
	Path     string `json:"path"`     // OS file path where media is saved
}

// ProcessParse office.Parse
// Parse an Office document (DOCX/PPTX) from a file path.
// Embedded media is extracted and saved as files to output_dir.
//
// Args:
//   - filePath string - Path to the Office document
//   - config map[string]interface{} - Optional configuration
//
// Config fields:
//   - output_dir string - Directory for extracted media (optional, default: auto temp dir)
//
// Returns: *ProcessParseResult - Parsed document with markdown, metadata, and media file paths
//
// Note: If output_dir is not specified, a temp directory is created automatically.
// The caller is responsible for cleaning up the output directory when done.
//
// Usage:
//
//	var result = Process("office.Parse", "/path/to/document.docx", {"output_dir": "/data/tmp"})
//	// Returns: {"markdown": "...", "metadata": {...}, "media": [{"path": "/data/tmp/image1.png", ...}]}
func ProcessParse(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	filePath := process.ArgsString(0)

	// Parse optional config for output_dir
	outputDir := ""
	if process.NumOfArgs() > 1 {
		configMap := process.ArgsMap(1)
		if dir, ok := configMap["output_dir"].(string); ok && dir != "" {
			outputDir = dir
		}
	}

	// Read file from disk
	data, err := os.ReadFile(filePath)
	if err != nil {
		exception.New(fmt.Sprintf("failed to read file %s: %s", filePath, err.Error()), 500).Throw()
	}

	// Parse the document
	result, err := defaultParser.Parse(data)
	if err != nil {
		exception.New(fmt.Sprintf("failed to parse office document: %s", err.Error()), 500).Throw()
	}

	return toProcessParseResult(result, outputDir)
}

// ProcessParseBytes office.ParseBytes
// Parse an Office document from base64-encoded data (for in-memory files).
// Embedded media is extracted and saved as files to output_dir.
//
// Args:
//   - data string - Base64-encoded file content
//   - config map[string]interface{} - Optional configuration
//
// Config fields:
//   - output_dir string - Directory for extracted media (optional, default: auto temp dir)
//
// Returns: *ProcessParseResult - Parsed document with markdown, metadata, and media file paths
//
// Note: The base64 decode + parse will hold the entire file in memory.
// For very large files (>100MB), prefer using office.Parse with a file path instead.
// If output_dir is not specified, a temp directory is created automatically.
// The caller is responsible for cleaning up the output directory when done.
//
// Usage:
//
//	var result = Process("office.ParseBytes", "UEsDBBQAAAAIAA...", {"output_dir": "/data/tmp"})
//	// Returns: {"markdown": "...", "metadata": {...}, "media": [{"path": "/data/tmp/image1.png", ...}]}
func ProcessParseBytes(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	dataStr := process.ArgsString(0)

	// Parse optional config for output_dir
	outputDir := ""
	if process.NumOfArgs() > 1 {
		configMap := process.ArgsMap(1)
		if dir, ok := configMap["output_dir"].(string); ok && dir != "" {
			outputDir = dir
		}
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		exception.New(fmt.Sprintf("failed to decode base64 data: %s", err.Error()), 400).Throw()
	}

	// Parse the document
	result, err := defaultParser.Parse(data)
	if err != nil {
		exception.New(fmt.Sprintf("failed to parse office document: %s", err.Error()), 500).Throw()
	}

	return toProcessParseResult(result, outputDir)
}

// toProcessParseResult converts a ParseResult to ProcessParseResult.
// Media content is written to files in outputDir instead of base64-encoding.
func toProcessParseResult(result *ParseResult, outputDir string) *ProcessParseResult {
	if result == nil {
		return nil
	}

	// Auto-create temp dir if output_dir not specified and there are media files
	if outputDir == "" && len(result.Media) > 0 {
		tmpDir, err := os.MkdirTemp("", "office_media_*")
		if err != nil {
			exception.New(fmt.Sprintf("failed to create temp directory: %s", err.Error()), 500).Throw()
		}
		outputDir = tmpDir
	}

	// Ensure output dir exists
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			exception.New(fmt.Sprintf("failed to create output directory: %s", err.Error()), 500).Throw()
		}
	}

	// Convert media entries: write content to files, return paths
	mediaEntries := make([]ProcessMediaEntry, 0, len(result.Media))
	for _, m := range result.Media {
		// Build output file path
		filename := m.Filename
		if filename == "" {
			filename = fmt.Sprintf("%s.%s", m.ID, m.Format)
		}
		filePath := filepath.Join(outputDir, filename)

		// Write media content to file (skip on error)
		if err := os.WriteFile(filePath, m.Content, 0644); err != nil {
			log.Warn("failed to write media file %s: %s", filePath, err.Error())
			continue
		}

		mediaEntries = append(mediaEntries, ProcessMediaEntry{
			ID:       m.ID,
			Type:     m.Type,
			Format:   m.Format,
			Filename: m.Filename,
			Path:     filePath,
		})
	}

	return &ProcessParseResult{
		Markdown: result.Markdown,
		Metadata: result.Metadata,
		Media:    mediaEntries,
	}
}
