package pdf

import (
	"context"
	"errors"
	"time"
)

// ErrNotImplemented is returned when a feature is not implemented for the current platform
var ErrNotImplemented = errors.New("feature not implemented for this platform")

// ConvertTool defines supported PDF to image conversion tools
type ConvertTool string

const (
	// ToolPdftoppm is the poppler-utils tool
	ToolPdftoppm ConvertTool = "pdftoppm" // poppler-utils
	// ToolMutool is the mupdf-tools tool
	ToolMutool ConvertTool = "mutool" // mupdf-tools
	// ToolImageMagick is the ImageMagick convert tool
	ToolImageMagick ConvertTool = "imagemagick" // ImageMagick convert
)

// Info contains information about a PDF file
type Info struct {
	FilePath     string            `json:"file_path"`
	FileSize     int64             `json:"file_size"`
	PageCount    int               `json:"page_count"`
	Title        string            `json:"title,omitempty"`
	Author       string            `json:"author,omitempty"`
	Subject      string            `json:"subject,omitempty"`
	Creator      string            `json:"creator,omitempty"`
	Producer     string            `json:"producer,omitempty"`
	CreationDate time.Time         `json:"creation_date,omitempty"`
	ModDate      time.Time         `json:"mod_date,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// SplitConfig defines how to split a PDF file
type SplitConfig struct {
	OutputDir    string   `json:"output_dir"`
	OutputPrefix string   `json:"output_prefix"`
	PageRanges   []string `json:"page_ranges,omitempty"`    // e.g., ["1-5", "6-10", "11-15"]
	PagesPerFile int      `json:"pages_per_file,omitempty"` // Alternative to PageRanges
}

// ConvertConfig defines how to convert PDF to images
type ConvertConfig struct {
	OutputDir    string `json:"output_dir"`
	OutputPrefix string `json:"output_prefix"`
	Format       string `json:"format"`     // "png", "jpg", "jpeg"
	DPI          int    `json:"dpi"`        // Resolution, default 150
	Quality      int    `json:"quality"`    // JPEG quality 1-100, default 95
	PageRange    string `json:"page_range"` // e.g., "1-5" or "all"
}

// Options defines configuration for PDF processor
type Options struct {
	ConvertTool ConvertTool `json:"convert_tool,omitempty"` // Preferred conversion tool
	ToolPath    string      `json:"tool_path,omitempty"`    // Custom path to the tool executable
}

// CommandExecutor defines interface for executing system commands
type CommandExecutor interface {
	// Execute runs a command with given arguments
	Execute(ctx context.Context, command string, args []string) error

	// IsCommandAvailable checks if a command is available
	IsCommandAvailable(command string) bool

	// GetDefaultToolPaths returns default paths for conversion tools
	GetDefaultToolPaths() map[ConvertTool]string
}

// Parser defines the contract for PDF processing operations
type Parser interface {
	// GetInfo reads PDF file information
	GetInfo(ctx context.Context, filePath string) (*Info, error)

	// Split splits a PDF file according to the configuration
	Split(ctx context.Context, filePath string, config SplitConfig) ([]string, error)

	// Convert converts PDF to images
	Convert(ctx context.Context, filePath string, config ConvertConfig) ([]string, error)
}
