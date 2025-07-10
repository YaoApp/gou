package pdf

import (
	"context"
	"fmt"
	"runtime"
)

// Command defines interface for PDF conversion operations
type Command interface {
	// IsAvailable checks if a specific tool is available
	IsAvailable(tool ConvertTool) bool

	// GetToolPath returns the executable path for a specific tool
	GetToolPath(tool ConvertTool) string

	// ConvertWithPdftoppm converts PDF to images using pdftoppm
	ConvertWithPdftoppm(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error)

	// ConvertWithMutool converts PDF to images using mutool
	ConvertWithMutool(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error)

	// ConvertWithImageMagick converts PDF to images using ImageMagick
	ConvertWithImageMagick(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error)
}

// ToolPaths contains custom paths for conversion tools
type ToolPaths map[ConvertTool]string

// NewCommand creates a platform-specific command driver with optional custom tool paths
func NewCommand(toolPaths ...ToolPaths) Command {
	// Merge all provided tool paths
	customPaths := make(ToolPaths)
	for _, paths := range toolPaths {
		for tool, path := range paths {
			customPaths[tool] = path
		}
	}

	switch runtime.GOOS {
	case "darwin":
		return NewMacOSDriver(customPaths)
	case "linux":
		return NewLinuxDriver(customPaths)
	case "windows":
		return NewWindowsDriver(customPaths)
	default:
		// Fallback to Linux implementation for other Unix-like systems
		return NewLinuxDriver(customPaths)
	}
}

// validateConvertConfig validates and sets defaults for convert configuration
func (p *PDF) validateConvertConfig(config *ConvertConfig) error {
	if config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}
	if config.OutputPrefix == "" {
		config.OutputPrefix = "page"
	}
	if config.Format == "" {
		config.Format = "png"
	}
	if config.DPI <= 0 {
		config.DPI = 150
	}
	if config.Quality <= 0 {
		config.Quality = 95
	}
	return nil
}

// The conversion methods are now implemented by the platform-specific drivers
// in linux.go, darwin.go, and windows.go files
