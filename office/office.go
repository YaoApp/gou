package office

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ParseResult represents the result of parsing an Office document
type ParseResult struct {
	Markdown string    `json:"markdown"`
	Metadata *Metadata `json:"metadata"`
	Media    []Media   `json:"media"`
}

// Metadata contains document metadata and text range information
type Metadata struct {
	Title      string            `json:"title"`
	Author     string            `json:"author"`
	Subject    string            `json:"subject"`
	Keywords   string            `json:"keywords"`
	Pages      int               `json:"pages"`
	TextRanges []TextRange       `json:"text_ranges"`
	MediaRefs  map[string]string `json:"media_refs"` // Media reference mapping
}

// TextRange represents a text range with position and page information
type TextRange struct {
	StartPos int    `json:"start_pos"`
	EndPos   int    `json:"end_pos"`
	Page     int    `json:"page"`
	Type     string `json:"type"` // text, heading, list, etc.
}

// Media represents a media file within the document
type Media struct {
	ID       string `json:"id"`
	Type     string `json:"type"`     // image, video, audio
	Format   string `json:"format"`   // png, jpg, mp4, etc.
	Content  []byte `json:"content"`  // File content
	Filename string `json:"filename"` // Original filename
	RefID    string `json:"ref_id"`   // Reference ID
}

// FileParser interface for Office document parsing
type FileParser interface {
	Parse(data []byte) (*ParseResult, error)
	GetSupportedFormats() []string
}

// Parser handles Office document parsing
type Parser struct {
	zipReader *zip.ReadCloser
	files     map[string]*zip.File
}

// NewParser creates a new Office parser instance
func NewParser() *Parser {
	return &Parser{
		files: make(map[string]*zip.File),
	}
}

// Parse parses an Office document from byte data
func (p *Parser) Parse(data []byte) (*ParseResult, error) {
	// Create ZIP reader
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %v", err)
	}

	// Build file mapping
	p.files = make(map[string]*zip.File)
	for _, file := range zipReader.File {
		p.files[file.Name] = file
	}

	// Detect document type
	docType := p.detectDocumentType()

	switch docType {
	case "docx":
		return p.parseDocx()
	case "pptx":
		return p.parsePptx()
	default:
		return nil, fmt.Errorf("unsupported document type: %s", docType)
	}
}

// detectDocumentType detects the type of Office document
func (p *Parser) detectDocumentType() string {
	// Check for specific files to determine document type
	if _, exists := p.files["word/document.xml"]; exists {
		return "docx"
	}
	if _, exists := p.files["ppt/presentation.xml"]; exists {
		return "pptx"
	}
	return "unknown"
}

// readFile reads a file from the ZIP archive
func (p *Parser) readFile(filename string) ([]byte, error) {
	file, exists := p.files[filename]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", filename)
	}

	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filename, err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// extractMedia extracts media files from the specified directory
func (p *Parser) extractMedia(mediaDir string) ([]Media, error) {
	media := make([]Media, 0) // Initialize as empty slice, not nil

	for filename := range p.files {
		if !strings.HasPrefix(filename, mediaDir) {
			continue
		}

		// Skip .rels files
		if strings.Contains(filename, ".rels") {
			continue
		}

		// Read media file content
		content, err := p.readFile(filename)
		if err != nil {
			continue
		}

		// Determine media type and format
		mediaType, format := p.getMediaTypeAndFormat(filename)
		if mediaType == "" {
			continue
		}

		media = append(media, Media{
			ID:       p.generateMediaID(filename),
			Type:     mediaType,
			Format:   format,
			Content:  content,
			Filename: filepath.Base(filename),
			RefID:    p.generateRefID(filename),
		})
	}

	return media, nil
}

// getMediaTypeAndFormat determines media type and format from filename
func (p *Parser) getMediaTypeAndFormat(filename string) (string, string) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tiff", ".webp":
		return "image", ext[1:]
	case ".mp4", ".avi", ".mov", ".wmv", ".flv":
		return "video", ext[1:]
	case ".mp3", ".wav", ".aac", ".flac":
		return "audio", ext[1:]
	default:
		return "", ""
	}
}

// generateMediaID generates a unique media ID
func (p *Parser) generateMediaID(filename string) string {
	return fmt.Sprintf("media_%s", strings.ReplaceAll(filepath.Base(filename), ".", "_"))
}

// generateRefID generates a reference ID for media
func (p *Parser) generateRefID(filename string) string {
	return fmt.Sprintf("ref_%s", filepath.Base(filename))
}

// GetSupportedFormats returns the list of supported document formats
func (p *Parser) GetSupportedFormats() []string {
	return []string{"docx", "pptx"}
}
