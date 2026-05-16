package pdf

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// WindowsDriver implements Command for Windows systems
type WindowsDriver struct {
	toolPaths ToolPaths
}

// NewWindowsDriver creates a new Windows driver with optional custom tool paths
func NewWindowsDriver(customPaths ToolPaths) *WindowsDriver {
	defaultPaths := ToolPaths{
		ToolPdftoppm:    "pdftoppm",
		ToolMutool:      "mutool",
		ToolImageMagick: "magick",
	}

	finalPaths := make(ToolPaths)
	for tool, path := range defaultPaths {
		finalPaths[tool] = path
	}
	for tool, path := range customPaths {
		if path != "" {
			finalPaths[tool] = path
		}
	}

	return &WindowsDriver{toolPaths: finalPaths}
}

// IsAvailable checks if a specific tool is available on Windows
func (d *WindowsDriver) IsAvailable(tool ConvertTool) bool {
	toolPath := d.GetToolPath(tool)
	if toolPath == "" {
		return false
	}
	_, err := exec.LookPath(toolPath)
	return err == nil
}

// GetToolPath returns the executable path for a specific tool on Windows
func (d *WindowsDriver) GetToolPath(tool ConvertTool) string {
	if path, exists := d.toolPaths[tool]; exists {
		return path
	}
	return ""
}

// ConvertWithPdftoppm converts PDF to images using pdftoppm on Windows
func (d *WindowsDriver) ConvertWithPdftoppm(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error) {
	toolPath := d.GetToolPath(ToolPdftoppm)
	if toolPath == "" {
		return nil, fmt.Errorf("pdftoppm tool not found")
	}

	var formatFlag string
	switch config.Format {
	case "png":
		formatFlag = "-png"
	case "jpg", "jpeg":
		formatFlag = "-jpeg"
	case "tiff":
		formatFlag = "-tiff"
	default:
		formatFlag = "-png"
	}

	args := []string{formatFlag, "-r", fmt.Sprintf("%d", config.DPI)}

	startPage, endPage := d.parsePageRange(config.PageRange, pageCount)
	if startPage > 0 {
		args = append(args, "-f", fmt.Sprintf("%d", startPage))
	}
	if endPage > 0 {
		args = append(args, "-l", fmt.Sprintf("%d", endPage))
	}

	outputPrefix := filepath.Join(config.OutputDir, config.OutputPrefix)
	args = append(args, filePath, outputPrefix)

	cmd := exec.CommandContext(ctx, toolPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pdftoppm failed: %w, command: %s %v, output: %s", err, toolPath, args, string(output))
	}

	return d.buildOutputPaths(config, startPage, endPage)
}

// ConvertWithMutool converts PDF to images using mutool on Windows
func (d *WindowsDriver) ConvertWithMutool(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error) {
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

	startPage, endPage := d.parsePageRange(config.PageRange, pageCount)
	args = append(args, filePath)
	if startPage > 0 && endPage > 0 {
		if startPage == endPage {
			args = append(args, fmt.Sprintf("%d", startPage))
		} else {
			args = append(args, fmt.Sprintf("%d-%d", startPage, endPage))
		}
	}

	cmd := exec.CommandContext(ctx, toolPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("mutool failed: %w, command: %s %v, output: %s", err, toolPath, args, string(output))
	}

	return d.buildOutputPaths(config, startPage, endPage)
}

// ConvertWithImageMagick converts PDF to images using ImageMagick on Windows
func (d *WindowsDriver) ConvertWithImageMagick(ctx context.Context, filePath string, config ConvertConfig, pageCount int) ([]string, error) {
	toolPath := d.GetToolPath(ToolImageMagick)
	if toolPath == "" {
		return nil, fmt.Errorf("ImageMagick tool not found")
	}

	args := []string{"convert", "-density", fmt.Sprintf("%d", config.DPI)}

	startPage, endPage := d.parsePageRange(config.PageRange, pageCount)
	inputFile := filePath
	if startPage > 0 && endPage > 0 {
		if startPage == endPage {
			inputFile = fmt.Sprintf("%s[%d]", filePath, startPage-1)
		} else {
			inputFile = fmt.Sprintf("%s[%d-%d]", filePath, startPage-1, endPage-1)
		}
	}

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

func (d *WindowsDriver) parsePageRange(pageRange string, pageCount int) (int, int) {
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
		page, err := strconv.Atoi(strings.TrimSpace(pageRange))
		if err == nil {
			return page, page
		}
	}
	return 1, pageCount
}

func (d *WindowsDriver) buildOutputPaths(config ConvertConfig, startPage, endPage int) ([]string, error) {
	var paths []string
	if startPage <= 0 {
		startPage = 1
	}
	if endPage <= 0 {
		endPage = startPage
	}
	for i := startPage; i <= endPage; i++ {
		filename := fmt.Sprintf("%s-%d.%s", config.OutputPrefix, i, config.Format)
		paths = append(paths, filepath.Join(config.OutputDir, filename))
	}
	return paths, nil
}

func (d *WindowsDriver) buildOutputPathsImageMagick(config ConvertConfig, startPage, endPage int) ([]string, error) {
	var paths []string
	if startPage <= 0 {
		startPage = 1
	}
	if endPage <= 0 {
		endPage = startPage
	}
	for i := startPage; i <= endPage; i++ {
		filename := fmt.Sprintf("%s-%d.%s", config.OutputPrefix, i-1, config.Format)
		paths = append(paths, filepath.Join(config.OutputDir, filename))
	}
	return paths, nil
}
