package office

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CommandLineConverter handles command line operations
type CommandLineConverter struct {
	parser *Parser
}

// NewCommandLineConverter creates a new command line converter
func NewCommandLineConverter() *CommandLineConverter {
	return &CommandLineConverter{
		parser: NewParser(),
	}
}

// ConvertFile converts a single Office document to markdown and saves to output directory
func (c *CommandLineConverter) ConvertFile(inputPath string) error {
	// Read input file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}

	// Parse document
	result, err := c.parser.Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse document: %v", err)
	}

	// Create output directory structure
	outputDir, err := c.createOutputDir(inputPath)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Save markdown file
	if err := c.saveMarkdown(outputDir, inputPath, result.Markdown); err != nil {
		return fmt.Errorf("failed to save markdown: %v", err)
	}

	// Save metadata JSON
	if err := c.saveMetadata(outputDir, result.Metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %v", err)
	}

	// Save media files
	if err := c.saveMediaFiles(outputDir, result.Media); err != nil {
		return fmt.Errorf("failed to save media files: %v", err)
	}

	fmt.Printf("✅ Converted: %s -> %s\n", inputPath, outputDir)
	return nil
}

// createOutputDir creates the output directory structure
func (c *CommandLineConverter) createOutputDir(inputPath string) (string, error) {
	// Get directory and filename
	dir := filepath.Dir(inputPath)
	filename := filepath.Base(inputPath)

	// Remove extension to get base name
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Create output directory path
	outputDir := filepath.Join(dir, baseName)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	// Create media subdirectory
	mediaDir := filepath.Join(outputDir, "media")
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		return "", err
	}

	return outputDir, nil
}

// saveMarkdown saves the markdown content to file
func (c *CommandLineConverter) saveMarkdown(outputDir, inputPath, markdown string) error {
	filename := filepath.Base(inputPath)
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	markdownPath := filepath.Join(outputDir, baseName+".md")

	return os.WriteFile(markdownPath, []byte(markdown), 0644)
}

// saveMetadata saves metadata as JSON file
func (c *CommandLineConverter) saveMetadata(outputDir string, metadata *Metadata) error {
	metadataPath := filepath.Join(outputDir, "metadata.json")

	// Create enhanced metadata with file mapping
	enhancedMetadata := map[string]interface{}{
		"title":       metadata.Title,
		"author":      metadata.Author,
		"subject":     metadata.Subject,
		"keywords":    metadata.Keywords,
		"pages":       metadata.Pages,
		"text_ranges": metadata.TextRanges,
		"media_refs":  metadata.MediaRefs,
		"files": map[string]string{
			"markdown":  filepath.Base(outputDir) + ".md",
			"metadata":  "metadata.json",
			"media_dir": "media/",
		},
	}

	jsonData, err := json.MarshalIndent(enhancedMetadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metadataPath, jsonData, 0644)
}

// saveMediaFiles saves all media files to the media directory
func (c *CommandLineConverter) saveMediaFiles(outputDir string, media []Media) error {
	mediaDir := filepath.Join(outputDir, "media")

	for _, mediaFile := range media {
		mediaPath := filepath.Join(mediaDir, mediaFile.Filename)

		if err := os.WriteFile(mediaPath, mediaFile.Content, 0644); err != nil {
			return fmt.Errorf("failed to save media file %s: %v", mediaFile.Filename, err)
		}
	}

	return nil
}

// RunCommand runs the command line interface
func RunCommand() {
	var (
		showHelp    = flag.Bool("h", false, "Show help message")
		showVersion = flag.Bool("v", false, "Show version")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <command> <files...>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  convert <files...>    Convert Office documents to Markdown\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s convert document.docx\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s convert presentation.pptx\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s convert *.docx *.pptx\n", os.Args[0])
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		return
	}

	if *showVersion {
		fmt.Println("Office Document Parser v1.0.0")
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := args[0]

	switch command {
	case "convert":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: convert command requires at least one file argument\n")
			flag.Usage()
			os.Exit(1)
		}

		converter := NewCommandLineConverter()
		files := args[1:]

		var errors []string
		successCount := 0

		for _, file := range files {
			if err := converter.ConvertFile(file); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", file, err))
			} else {
				successCount++
			}
		}

		// Print summary
		fmt.Printf("\n📊 Conversion Summary:\n")
		fmt.Printf("   ✅ Successfully converted: %d files\n", successCount)
		if len(errors) > 0 {
			fmt.Printf("   ❌ Failed: %d files\n", len(errors))
			for _, err := range errors {
				fmt.Printf("      - %s\n", err)
			}
		}

		if len(errors) > 0 {
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n", command)
		flag.Usage()
		os.Exit(1)
	}
}
