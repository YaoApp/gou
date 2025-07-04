# Office Document Parser

A Go package for parsing Office documents (DOCX and PPTX) and converting them to Markdown with media extraction, without using any external dependencies.

## Features

- ✅ **DOCX Support**: Parse Word documents and extract text, formatting, and media
- ✅ **PPTX Support**: Parse PowerPoint presentations with slide-by-slide content extraction
- ✅ **Markdown Output**: Convert formatted text to Markdown with proper headings, bold, italic
- ✅ **Media Extraction**: Extract images, videos, and audio files with metadata
- ✅ **Text Range Tracking**: Track text positions and page/slide numbers for each content piece
- ✅ **No External Dependencies**: Pure Go implementation using only standard library
- ✅ **Command Line Tool**: Convert documents from command line
- ✅ **Comprehensive Testing**: Full test suite with real document samples

## Command Line Usage

### Install

```bash
go install github.com/yaoapp/gou/office/cmd
```

### Convert Documents

```bash
# Convert a DOCX file
cmd convert document.docx

# Convert a PPTX file
cmd convert presentation.pptx

# Convert multiple files
cmd convert *.docx *.pptx

# Or use go run directly
cd cmd
go run main.go convert document.docx
```

### Output Structure

For input file `document.docx`, the output will be:

```
document/
├── document.md          # Markdown content
├── metadata.json        # Document metadata and mapping
└── media/              # Extracted media files
    ├── image1.png
    ├── image2.jpg
    └── ...
```

The `metadata.json` file contains:

```json
{
  "title": "Document Title",
  "author": "Author Name",
  "subject": "Document Subject",
  "keywords": "document, keywords",
  "pages": 5,
  "text_ranges": [
    {
      "start_pos": 0,
      "end_pos": 100,
      "page": 1,
      "type": "heading"
    }
  ],
  "media_refs": {
    "rId8": "media/image1.png",
    "rId10": "media/image2.png"
  },
  "files": {
    "markdown": "document.md",
    "metadata": "metadata.json",
    "media_dir": "media/"
  }
}
```

## Library Usage

### Basic Usage

```go
package main

import (
    "fmt"
    "os"
    "github.com/yaoapp/gou/office"
)

func main() {
    // Create parser
    parser := office.NewParser()

    // Read document data
    data, err := os.ReadFile("document.docx")
    if err != nil {
        panic(err)
    }

    // Parse document
    result, err := parser.Parse(data)
    if err != nil {
        panic(err)
    }

    // Access the results
    fmt.Printf("Title: %s\n", result.Metadata.Title)
    fmt.Printf("Author: %s\n", result.Metadata.Author)
    fmt.Printf("Pages: %d\n", result.Metadata.Pages)
    fmt.Printf("Media files: %d\n", len(result.Media))

    // Print markdown content
    fmt.Println("Markdown Content:")
    fmt.Println(result.Markdown)

    // Access media files
    for _, media := range result.Media {
        fmt.Printf("Media: %s (%s, %d bytes)\n",
            media.Filename, media.Type, len(media.Content))
    }
}
```

### Supported Formats

```go
parser := office.NewParser()
formats := parser.GetSupportedFormats()
// Returns: ["docx", "pptx"]
```

## Output Structure

### ParseResult

```go
type ParseResult struct {
    Markdown string     `json:"markdown"`  // Converted markdown text
    Metadata *Metadata  `json:"metadata"`  // Document metadata
    Media    []Media    `json:"media"`     // Extracted media files
}
```

### Metadata

```go
type Metadata struct {
    Title      string            `json:"title"`       // Document title
    Author     string            `json:"author"`      // Document author
    Subject    string            `json:"subject"`     // Document subject
    Keywords   string            `json:"keywords"`    // Document keywords
    Pages      int               `json:"pages"`       // Number of pages/slides
    TextRanges []TextRange       `json:"text_ranges"` // Text position tracking
    MediaRefs  map[string]string `json:"media_refs"`  // Media reference mapping
}
```

### TextRange

```go
type TextRange struct {
    StartPos int    `json:"start_pos"` // Start position in markdown
    EndPos   int    `json:"end_pos"`   // End position in markdown
    Page     int    `json:"page"`      // Page/slide number
    Type     string `json:"type"`      // Content type (text, heading, list, slide)
}
```

### Media

```go
type Media struct {
    ID       string `json:"id"`       // Unique media ID
    Type     string `json:"type"`     // Media type (image, video, audio)
    Format   string `json:"format"`   // File format (png, jpg, mp4, etc.)
    Content  []byte `json:"content"`  // File content bytes
    Filename string `json:"filename"` // Original filename
    RefID    string `json:"ref_id"`   // Reference ID for linking
}
```

## DOCX Features

- **Text Extraction**: Plain text and formatted content
- **Heading Detection**: Automatic heading level detection (H1-H6)
- **Formatting**: Bold, italic, and combined formatting
- **Media Support**: Images, videos, audio files
- **Metadata**: Title, author, subject, keywords
- **Text Ranges**: Precise position tracking

## PPTX Features

- **Slide Extraction**: Individual slide processing
- **Shape Processing**: Text boxes, titles, content shapes
- **Slide Separators**: Clear slide boundaries in markdown
- **Title Detection**: Automatic title shape recognition
- **Media Support**: All embedded media types
- **Slide Numbering**: Proper slide sequence tracking

## Performance

Benchmarks on Apple M2 Max:

- **DOCX Parsing**: ~8.1ms per document
- **PPTX Parsing**: ~2.4ms per document

## Testing

Run the test suite:

```bash
go test -v
```

Run benchmarks:

```bash
go test -bench=.
```

The package includes comprehensive tests with real document samples covering:

- Document type detection
- Content extraction
- Media file handling
- Text range tracking
- Error handling
- Performance benchmarks

## Architecture

The package is structured with separate parsers for each document type:

- `office.go` - Main parser interface and common functionality
- `docx.go` - DOCX-specific parsing logic
- `pptx.go` - PPTX-specific parsing logic
- `cmd.go` - Command line tool implementation
- `cmd/` - Command line application
- `office_test.go` - Comprehensive test suite
- `cmd_test.go` - Command line tool tests

## Implementation Details

### Document Detection

The parser automatically detects document type by examining the internal file structure:

- DOCX: Looks for `word/document.xml`
- PPTX: Looks for `ppt/presentation.xml`

### XML Parsing

Office documents are ZIP archives containing XML files. The parser:

1. Extracts the ZIP archive
2. Parses relevant XML files using Go's `encoding/xml`
3. Handles namespaces and relationships
4. Extracts text content and formatting

### Media Extraction

Media files are extracted from:

- DOCX: `word/media/` directory
- PPTX: `ppt/media/` directory

Supported media types:

- **Images**: PNG, JPG, JPEG, GIF, BMP, TIFF, WEBP
- **Videos**: MP4, AVI, MOV, WMV, FLV
- **Audio**: MP3, WAV, AAC, FLAC

## License

This package is part of the Yao framework and follows the same licensing terms.
