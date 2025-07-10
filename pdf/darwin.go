package pdf

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// MacOSDriver implements Command for macOS systems
type MacOSDriver struct {
	toolPaths ToolPaths
}

// NewMacOSDriver creates a new macOS driver with optional custom tool paths
func NewMacOSDriver(customPaths ToolPaths) *MacOSDriver {
	// Default tool paths for macOS
	defaultPaths := ToolPaths{
		ToolPdftoppm:    "pdftoppm", // brew install poppler
		ToolMutool:      "mutool",   // brew install mupdf-tools
		ToolImageMagick: "magick",   // brew install imagemagick
	}

	// Merge custom paths with defaults
	finalPaths := make(ToolPaths)
	for tool, path := range defaultPaths {
		finalPaths[tool] = path
	}
	for tool, path := range customPaths {
		if path != "" {
			finalPaths[tool] = path
		}
	}

	return &MacOSDriver{
		toolPaths: finalPaths,
	}
}

// IsAvailable checks if a specific tool is available on macOS
func (d *MacOSDriver) IsAvailable(tool ConvertTool) bool {
	toolPath := d.GetToolPath(tool)
	if toolPath == "" {
		return false
	}
	_, err := exec.LookPath(toolPath)
	return err == nil
}

// GetToolPath returns the executable path for a specific tool on macOS
func (d *MacOSDriver) GetToolPath(tool ConvertTool) string {
	if path, exists := d.toolPaths[tool]; exists {
		return path
	}
	return ""
}

// ConvertWithPdftoppm converts PDF to images using pdftoppm on macOS
func (d *MacOSDriver) ConvertWithPdftoppm(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error) {
	return d.executePdftoppm(ctx, filePath, config, pageCount)
}

// ConvertWithMutool converts PDF to images using mutool on macOS
func (d *MacOSDriver) ConvertWithMutool(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error) {
	return d.executeMutool(ctx, filePath, config, pageCount)
}

// ConvertWithImageMagick converts PDF to images using ImageMagick on macOS
func (d *MacOSDriver) ConvertWithImageMagick(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error) {
	return d.executeImageMagick(ctx, filePath, config, pageCount)
}

// executePdftoppm executes pdftoppm command on macOS
func (d *MacOSDriver) executePdftoppm(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error) {
	toolPath := d.GetToolPath(ToolPdftoppm)
	if toolPath == "" {
		return nil, fmt.Errorf("pdftoppm tool not found")
	}

	// Map format to pdftoppm format flag
	var formatFlag string
	switch config.Format {
	case "png":
		formatFlag = "-png"
	case "jpg", "jpeg":
		formatFlag = "-jpeg"
	case "tiff":
		formatFlag = "-tiff"
	default:
		formatFlag = "-png" // Default to PNG
	}

	args := []string{
		formatFlag,
		"-r", fmt.Sprintf("%d", config.DPI),
	}

	// Parse page range
	startPage, endPage := d.parsePageRange(config.PageRange, pageCount)

	// Add page range if specified
	if startPage > 0 {
		args = append(args, "-f", fmt.Sprintf("%d", startPage))
	}
	if endPage > 0 {
		args = append(args, "-l", fmt.Sprintf("%d", endPage))
	}

	// Add input file and output prefix
	outputPrefix := filepath.Join(config.OutputDir, config.OutputPrefix)
	args = append(args, filePath, outputPrefix)

	cmd := exec.CommandContext(ctx, toolPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pdftoppm failed: %w, command: %s %v, output: %s", err, toolPath, args, string(output))
	}

	return d.buildOutputPaths(config, startPage, endPage)
}

// executeMutool executes mutool command on macOS
func (d *MacOSDriver) executeMutool(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error) {
	toolPath := d.GetToolPath(ToolMutool)
	if toolPath == "" {
		return nil, fmt.Errorf("mutool tool not found")
	}

	outputPrefix := filepath.Join(config.OutputDir, config.OutputPrefix)
	args := []string{
		"convert",
		"-F", config.Format,
		"-O", fmt.Sprintf("resolution=%d", config.DPI),
		"-o", outputPrefix + "-%d." + config.Format,
	}

	// Parse page range
	startPage, endPage := d.parsePageRange(config.PageRange, pageCount)

	// Add input file first
	args = append(args, filePath)

	// Add page range if specified
	if startPage > 0 && endPage > 0 {
		if startPage == endPage {
			args = append(args, fmt.Sprintf("%d", startPage))
		} else {
			args = append(args, fmt.Sprintf("%d-%d", startPage, endPage))
		}
	}

	cmd := exec.CommandContext(ctx, toolPath, args...)
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("mutool failed: %w", err)
	}

	return d.buildOutputPaths(config, startPage, endPage)
}

// executeImageMagick executes ImageMagick convert command on macOS
func (d *MacOSDriver) executeImageMagick(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error) {
	toolPath := d.GetToolPath(ToolImageMagick)
	if toolPath == "" {
		return nil, fmt.Errorf("ImageMagick convert tool not found")
	}

	args := []string{
		"-density", fmt.Sprintf("%d", config.DPI),
	}

	// Parse page range
	startPage, endPage := d.parsePageRange(config.PageRange, pageCount)

	// Add input file with page range if specified
	inputFile := filePath
	if startPage > 0 && endPage > 0 {
		if startPage == endPage {
			inputFile = fmt.Sprintf("%s[%d]", filePath, startPage-1) // ImageMagick uses 0-based indexing
		} else {
			inputFile = fmt.Sprintf("%s[%d-%d]", filePath, startPage-1, endPage-1)
		}
	}

	// Add quality for JPEG
	if config.Format == "jpg" || config.Format == "jpeg" {
		args = append(args, "-quality", fmt.Sprintf("%d", config.Quality))
	}

	outputPrefix := filepath.Join(config.OutputDir, config.OutputPrefix)
	args = append(args, inputFile, outputPrefix+"-%d."+config.Format)

	cmd := exec.CommandContext(ctx, toolPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("convert failed: %w, command: %s %v, output: %s", err, toolPath, args, string(output))
	}

	return d.buildOutputPathsImageMagick(config, startPage, endPage)
}

// parsePageRange parses page range string like "1-5" or "all"
func (d *MacOSDriver) parsePageRange(pageRange string, pageCount int) (int, int) {
	if pageRange == "" || pageRange == "all" {
		return 1, pageCount
	}

	if strings.Contains(pageRange, "-") {
		parts := strings.Split(pageRange, "-")
		if len(parts) == 2 {
			startPage, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			endPage, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil {
				return startPage, endPage
			}
		}
	} else {
		// Single page
		page, err := strconv.Atoi(strings.TrimSpace(pageRange))
		if err == nil {
			return page, page
		}
	}

	// Default to all pages
	return 1, pageCount
}

// buildOutputPaths builds the list of output file paths for macOS
func (d *MacOSDriver) buildOutputPaths(config ConvertConfig, startPage, endPage int) ([]string, error) {
	var paths []string

	if startPage <= 0 {
		startPage = 1
	}
	if endPage <= 0 {
		endPage = startPage
	}

	for i := startPage; i <= endPage; i++ {
		filename := fmt.Sprintf("%s-%d.%s", config.OutputPrefix, i, config.Format)
		path := filepath.Join(config.OutputDir, filename)
		paths = append(paths, path)
	}

	return paths, nil
}

// buildOutputPathsImageMagick builds the list of output file paths for ImageMagick (0-based indexing)
func (d *MacOSDriver) buildOutputPathsImageMagick(config ConvertConfig, startPage, endPage int) ([]string, error) {
	var paths []string

	if startPage <= 0 {
		startPage = 1
	}
	if endPage <= 0 {
		endPage = startPage
	}

	// ImageMagick uses 0-based indexing for output files
	for i := startPage; i <= endPage; i++ {
		filename := fmt.Sprintf("%s-%d.%s", config.OutputPrefix, i-1, config.Format)
		path := filepath.Join(config.OutputDir, filename)
		paths = append(paths, path)
	}

	return paths, nil
}
