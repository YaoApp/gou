package pdf

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// defaultPDF is the default PDF processor instance
var defaultPDF *PDF

func init() {
	process.RegisterGroup("pdf", map[string]process.Handler{
		"info":    ProcessInfo,
		"split":   ProcessSplit,
		"convert": ProcessConvert,
	})

	// Create default PDF instance with sensible defaults
	defaultPDF = New(Options{})
}

// ProcessInfo pdf.Info
// Get PDF file information including page count, metadata, and file size.
//
// Args:
//   - filePath string - Path to the PDF file
//
// Returns: *Info - PDF file information
//
// Usage:
//
//	var info = Process("pdf.Info", "/path/to/file.pdf")
//	// Returns: {"page_count": 10, "file_size": 12345, "title": "...", ...}
func ProcessInfo(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	filePath := process.ArgsString(0)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	info, err := defaultPDF.GetInfo(ctx, filePath)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return info
}

// ProcessSplit pdf.Split
// Split a PDF file into multiple files by page ranges.
//
// Args:
//   - filePath string - Path to the PDF file
//   - config map[string]interface{} - Split configuration
//
// Config fields:
//   - pages string - Page ranges, e.g. "1-3,5,7-10"
//   - pages_per_file int - Alternative: pages per output file
//   - output_dir string - Output directory (optional, default: auto temp dir)
//   - output_prefix string - Prefix for output files (optional)
//
// Returns: []string - Paths to output PDF files
//
// Note: If output_dir is not specified, files are created in OS temp directory.
// TS scripts without "system" fs access MUST provide output_dir explicitly.
// The caller is responsible for cleaning up the output directory when done.
//
// Usage:
//
//	var files = Process("pdf.Split", "/path/to/file.pdf", {"pages": "1-3,5,7-10", "output_dir": "/tmp/out"})
//	// Returns: ["/tmp/out/file_1-3.pdf", "/tmp/out/file_5.pdf", "/tmp/out/file_7-10.pdf"]
func ProcessSplit(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	filePath := process.ArgsString(0)
	configMap := process.ArgsMap(1)

	config := SplitConfig{}

	// Parse pages as page ranges
	if pages, ok := configMap["pages"].(string); ok && pages != "" {
		config.PageRanges = parsePageRanges(pages)
	}

	// Parse pages_per_file
	if pagesPerFile, ok := configMap["pages_per_file"]; ok {
		config.PagesPerFile = toInt(pagesPerFile)
	}

	// Parse output_dir
	if outputDir, ok := configMap["output_dir"].(string); ok && outputDir != "" {
		config.OutputDir = outputDir
	}

	// Parse output_prefix
	if outputPrefix, ok := configMap["output_prefix"].(string); ok {
		config.OutputPrefix = outputPrefix
	}

	// Auto-create temp dir if output_dir not specified
	if config.OutputDir == "" {
		tmpDir, err := os.MkdirTemp("", "pdf_split_*")
		if err != nil {
			exception.New(fmt.Sprintf("failed to create temp directory: %s", err.Error()), 500).Throw()
		}
		config.OutputDir = tmpDir
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	files, err := defaultPDF.Split(ctx, filePath, config)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return files
}

// ProcessConvert pdf.Convert
// Convert PDF pages to images (PNG/JPEG).
//
// Args:
//   - filePath string - Path to the PDF file
//   - config map[string]interface{} - Convert configuration (optional)
//
// Config fields:
//   - format string - Output format "png" or "jpeg" (default: "png")
//   - dpi int - Resolution (default: 150)
//   - pages string - Page range, e.g. "1-3" (default: all pages)
//   - output_dir string - Output directory (optional, default: auto temp dir)
//   - output_prefix string - Prefix for output files (optional)
//   - quality int - JPEG quality 1-100 (default: 95)
//
// Returns: []string - Paths to output image files
//
// Note: If output_dir is not specified, files are created in OS temp directory.
// TS scripts without "system" fs access MUST provide output_dir explicitly.
// The caller is responsible for cleaning up the output directory when done.
//
// Usage:
//
//	var images = Process("pdf.Convert", "/path/to/file.pdf", {"format": "png", "dpi": 150, "pages": "1-3"})
//	// Returns: ["/tmp/out/page_1.png", "/tmp/out/page_2.png", "/tmp/out/page_3.png"]
func ProcessConvert(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	filePath := process.ArgsString(0)

	config := ConvertConfig{
		Format: "png",
		DPI:    150,
	}

	// Parse optional config
	if process.NumOfArgs() > 1 {
		configMap := process.ArgsMap(1)

		if format, ok := configMap["format"].(string); ok && format != "" {
			config.Format = format
		}

		if dpi, ok := configMap["dpi"]; ok {
			config.DPI = toInt(dpi)
		}

		if quality, ok := configMap["quality"]; ok {
			config.Quality = toInt(quality)
		}

		if pages, ok := configMap["pages"].(string); ok {
			config.PageRange = pages
		}

		if outputDir, ok := configMap["output_dir"].(string); ok && outputDir != "" {
			config.OutputDir = outputDir
		}

		if outputPrefix, ok := configMap["output_prefix"].(string); ok {
			config.OutputPrefix = outputPrefix
		}
	}

	// Auto-create temp dir if output_dir not specified
	if config.OutputDir == "" {
		tmpDir, err := os.MkdirTemp("", "pdf_convert_*")
		if err != nil {
			exception.New(fmt.Sprintf("failed to create temp directory: %s", err.Error()), 500).Throw()
		}
		config.OutputDir = tmpDir
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	files, err := defaultPDF.Convert(ctx, filePath, config)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return files
}

// parsePageRanges parses a page range string like "1-3,5,7-10" into []string{"1-3", "5", "7-10"}
func parsePageRanges(pages string) []string {
	var ranges []string
	for _, part := range strings.Split(pages, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			ranges = append(ranges, trimmed)
		}
	}
	return ranges
}

// toInt converts various numeric types to int.
// Handles int, int64, float64, float32, and string (via strconv.Atoi).
func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	case string:
		n, _ := strconv.Atoi(val)
		return n
	default:
		return 0
	}
}
