package office

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCommandLineConverter tests the command line converter
func TestCommandLineConverter(t *testing.T) {
	// Get test files
	docxFiles := getDocxTestFiles()
	if len(docxFiles) == 0 {
		t.Skip("No DOCX test files found")
	}

	// Create a temporary directory for output
	tempDir := t.TempDir()

	// Copy a test file to temp directory
	testFile := filepath.Join(tempDir, "test_document.docx")
	sourceData, err := os.ReadFile(docxFiles[0])
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}

	if err := os.WriteFile(testFile, sourceData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create converter and convert file
	converter := NewCommandLineConverter()
	err = converter.ConvertFile(testFile)
	if err != nil {
		t.Fatalf("ConvertFile failed: %v", err)
	}

	// Verify output directory exists
	outputDir := filepath.Join(tempDir, "test_document")
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Fatalf("Output directory was not created: %s", outputDir)
	}

	// Verify markdown file exists
	markdownFile := filepath.Join(outputDir, "test_document.md")
	if _, err := os.Stat(markdownFile); os.IsNotExist(err) {
		t.Errorf("Markdown file was not created: %s", markdownFile)
	}

	// Verify metadata file exists
	metadataFile := filepath.Join(outputDir, "metadata.json")
	if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
		t.Errorf("Metadata file was not created: %s", metadataFile)
	}

	// Verify media directory exists
	mediaDir := filepath.Join(outputDir, "media")
	if _, err := os.Stat(mediaDir); os.IsNotExist(err) {
		t.Errorf("Media directory was not created: %s", mediaDir)
	}

	// Check file contents
	markdownContent, err := os.ReadFile(markdownFile)
	if err != nil {
		t.Errorf("Failed to read markdown file: %v", err)
	} else if len(markdownContent) == 0 {
		t.Error("Markdown file is empty")
	}

	metadataContent, err := os.ReadFile(metadataFile)
	if err != nil {
		t.Errorf("Failed to read metadata file: %v", err)
	} else if len(metadataContent) == 0 {
		t.Error("Metadata file is empty")
	}

	t.Logf("✅ Successfully converted file to: %s", outputDir)
	t.Logf("   📄 Markdown: %d bytes", len(markdownContent))
	t.Logf("   📊 Metadata: %d bytes", len(metadataContent))
}

// TestCreateOutputDir tests the output directory creation
func TestCreateOutputDir(t *testing.T) {
	converter := NewCommandLineConverter()
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "sample.docx")

	outputDir, err := converter.createOutputDir(testFile)
	if err != nil {
		t.Fatalf("createOutputDir failed: %v", err)
	}

	expectedDir := filepath.Join(tempDir, "sample")
	if outputDir != expectedDir {
		t.Errorf("Expected output dir %s, got %s", expectedDir, outputDir)
	}

	// Verify directory was created
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Errorf("Output directory was not created: %s", outputDir)
	}

	// Verify media subdirectory was created
	mediaDir := filepath.Join(outputDir, "media")
	if _, err := os.Stat(mediaDir); os.IsNotExist(err) {
		t.Errorf("Media directory was not created: %s", mediaDir)
	}
}
