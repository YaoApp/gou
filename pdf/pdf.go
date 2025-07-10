package pdf

import (
	"context"
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// PDF handles PDF operations
type PDF struct {
	convertTool ConvertTool
	toolPath    string
	cmd         Command
}

// New creates a new PDF processor
func New(opts Options) *PDF {
	// Prepare custom tool paths
	customPaths := make(ToolPaths)
	if opts.ToolPath != "" && opts.ConvertTool != "" {
		customPaths[opts.ConvertTool] = opts.ToolPath
	}

	// Create command with custom tool paths
	cmd := NewCommand(customPaths)

	pdf := &PDF{
		convertTool: ToolPdftoppm, // Default tool
		cmd:         cmd,
	}

	// Set conversion tool preference
	if opts.ConvertTool != "" {
		pdf.convertTool = opts.ConvertTool
	}

	// Set tool path (from command driver)
	pdf.toolPath = cmd.GetToolPath(pdf.convertTool)

	return pdf
}

// GetInfo reads PDF file information
func (p *PDF) GetInfo(ctx context.Context, filePath string) (*Info, error) {
	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Get page count using pdfcpu
	pageCount, err := api.PageCountFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	info := &Info{
		FilePath:  filePath,
		FileSize:  fileInfo.Size(),
		PageCount: pageCount,
		Metadata:  make(map[string]string),
	}

	// Extract metadata using pdfcpu's public API
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for metadata extraction: %w", err)
	}
	defer file.Close()

	// Use pdfcpu's Properties function to extract metadata
	properties, err := api.Properties(file, nil)
	if err != nil {
		// If metadata extraction fails, just log and continue with basic info
		// This is not a fatal error
		info.Metadata["extraction_error"] = err.Error()
	} else {
		// Copy properties to metadata
		for key, value := range properties {
			info.Metadata[key] = value
		}
	}

	return info, nil
}

// Split splits a PDF file according to the configuration
func (p *PDF) Split(ctx context.Context, filePath string, config SplitConfig) ([]string, error) {
	return p.splitPDF(ctx, filePath, config)
}

// Convert converts PDF to images
func (p *PDF) Convert(ctx context.Context, filePath string, config ConvertConfig) ([]string, error) {
	// Validate and set defaults
	if err := p.validateConvertConfig(&config); err != nil {
		return nil, err
	}

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get PDF info
	info, err := p.GetInfo(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get PDF info: %w", err)
	}

	// Check if the configured tool is available
	if !p.cmd.IsAvailable(p.convertTool) {
		return nil, fmt.Errorf("conversion tool %s is not available", p.convertTool)
	}

	// Convert based on the configured tool using the new Command interface
	switch p.convertTool {
	case ToolPdftoppm:
		return p.cmd.ConvertWithPdftoppm(ctx, filePath, config, info.PageCount)
	case ToolMutool:
		return p.cmd.ConvertWithMutool(ctx, filePath, config, info.PageCount)
	case ToolImageMagick:
		return p.cmd.ConvertWithImageMagick(ctx, filePath, config, info.PageCount)
	default:
		return nil, fmt.Errorf("unsupported conversion tool: %s", p.convertTool)
	}
}
