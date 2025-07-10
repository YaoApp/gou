# PDF Processing Library

A cross-platform Go library for PDF processing operations: info extraction, splitting, and conversion to images.

## Installation

```bash
go get github.com/yaoapp/gou/pdf
```

## System Dependencies

### macOS

```bash
# Install via Homebrew
brew install poppler      # for pdftoppm
brew install mupdf-tools   # for mutool
brew install imagemagick   # for convert
```

### Linux (Ubuntu/Debian)

```bash
# Install via apt
sudo apt install poppler-utils  # for pdftoppm
sudo apt install mupdf-tools    # for mutool
sudo apt install imagemagick    # for convert
```

### Linux (CentOS/RHEL)

```bash
# Install via yum/dnf
sudo yum install poppler-utils   # for pdftoppm
sudo yum install mupdf           # for mutool
sudo yum install ImageMagick     # for convert
```

### Windows

Windows support is not yet implemented. Contributions welcome.

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/yaoapp/gou/pdf"
)

func main() {
    // Create PDF processor
    p := pdf.New(pdf.Options{
        ConvertTool: pdf.ToolMutool,
        ToolPath:    "/usr/local/bin/mutool", // optional custom path
    })

    ctx := context.Background()

    // Get PDF info
    info, err := p.GetInfo(ctx, "document.pdf")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Pages: %d\n", info.PageCount)

    // Split PDF
    files, err := p.Split(ctx, "document.pdf", pdf.SplitConfig{
        OutputDir:    "./output",
        OutputPrefix: "page",
        PageRanges:   []string{"1-5", "6-10"}, // or use PagesPerFile: 5
    })
    if err != nil {
        panic(err)
    }

    // Convert to images
    images, err := p.Convert(ctx, "document.pdf", pdf.ConvertConfig{
        OutputDir:    "./images",
        OutputPrefix: "page",
        Format:       "png",
        DPI:          150,
        PageRange:    "1-10",
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Generated %d images\n", len(images))
}
```

### Available Tools

- `pdf.ToolPdftoppm` - Uses poppler-utils (recommended for Linux)
- `pdf.ToolMutool` - Uses MuPDF tools (recommended for macOS)
- `pdf.ToolImageMagick` - Uses ImageMagick convert

### Configuration Options

```go
// Create with custom tool and path
p := pdf.New(pdf.Options{
    ConvertTool: pdf.ToolPdftoppm,
    ToolPath:    "/custom/path/to/pdftoppm",
})

// Split configuration
splitConfig := pdf.SplitConfig{
    OutputDir:    "./output",
    OutputPrefix: "chapter",
    PageRanges:   []string{"1-10", "11-20"},  // specific ranges
    // OR
    PagesPerFile: 5,                          // pages per file
}

// Convert configuration
convertConfig := pdf.ConvertConfig{
    OutputDir:    "./images",
    OutputPrefix: "slide",
    Format:       "jpg",        // png, jpg, jpeg
    DPI:          300,          // resolution
    Quality:      95,           // JPEG quality (1-100)
    PageRange:    "all",        // "all", "1-5", "3"
}
```

## API Reference

### `New(opts Options) *PDF`

Creates a new PDF processor with specified options.

### `GetInfo(ctx context.Context, filePath string) (*Info, error)`

Extracts PDF metadata and page count.

### `Split(ctx context.Context, filePath string, config SplitConfig) ([]string, error)`

Splits PDF into multiple files based on page ranges.

### `Convert(ctx context.Context, filePath string, config ConvertConfig) ([]string, error)`

Converts PDF pages to images.

## Error Handling

The library returns descriptive errors for common issues:

- Tool not found or not available
- Invalid configuration
- File access problems
- Conversion failures

Handle tool availability errors:

```go
import "strings"

// Handle tool availability errors
images, err := p.Convert(ctx, "document.pdf", convertConfig)
if err != nil {
    if strings.Contains(err.Error(), "not available") {
        fmt.Println("Required conversion tool not available")
    }
    return err
}
```
