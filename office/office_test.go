package office

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// getOfficeTestDataDir returns the office test data directory
func getOfficeTestDataDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(currentFile)
	testDataDir := filepath.Join(currentDir, "tests")
	absPath, err := filepath.Abs(testDataDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to get absolute path for office test data dir: %v", err))
	}
	return absPath
}

// getDocxTestFiles returns all DOCX test files
func getDocxTestFiles() []string {
	testDataDir := getOfficeTestDataDir()
	docxDir := filepath.Join(testDataDir, "docx")

	files, err := os.ReadDir(docxDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to read DOCX test directory: %v", err))
	}

	var docxFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".docx") {
			docxFiles = append(docxFiles, filepath.Join(docxDir, file.Name()))
		}
	}
	return docxFiles
}

// getPptxTestFiles returns all PPTX test files
func getPptxTestFiles() []string {
	testDataDir := getOfficeTestDataDir()
	pptxDir := filepath.Join(testDataDir, "pptx")

	files, err := os.ReadDir(pptxDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to read PPTX test directory: %v", err))
	}

	var pptxFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".pptx") {
			pptxFiles = append(pptxFiles, filepath.Join(pptxDir, file.Name()))
		}
	}
	return pptxFiles
}

// Convenience functions for testing

// parseFile parses an Office document from a file path
func parseFile(filePath string) (*ParseResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	parser := NewParser()
	return parser.Parse(data)
}

// getMarkdownOnly extracts only the markdown content from a file
func getMarkdownOnly(filePath string) (string, error) {
	result, err := parseFile(filePath)
	if err != nil {
		return "", err
	}
	return result.Markdown, nil
}

// getMediaOnly extracts only the media files from a document
func getMediaOnly(filePath string) ([]Media, error) {
	result, err := parseFile(filePath)
	if err != nil {
		return nil, err
	}
	return result.Media, nil
}

// validateDocument validates if a document can be parsed successfully
func validateDocument(filePath string) error {
	_, err := parseFile(filePath)
	return err
}

// TestNewParser tests the creation of a new parser
func TestNewParser(t *testing.T) {
	parser := NewParser()
	if parser == nil {
		t.Fatal("NewParser() returned nil")
	}

	if parser.files == nil {
		t.Error("Parser files map is nil")
	}
}

// TestGetSupportedFormats tests the supported formats
func TestGetSupportedFormats(t *testing.T) {
	parser := NewParser()
	formats := parser.GetSupportedFormats()

	expectedFormats := []string{"docx", "pptx"}
	if len(formats) != len(expectedFormats) {
		t.Errorf("Expected %d formats, got %d", len(expectedFormats), len(formats))
	}

	for i, expected := range expectedFormats {
		if formats[i] != expected {
			t.Errorf("Expected format %s, got %s", expected, formats[i])
		}
	}
}

// TestParseDocxFiles tests parsing of all DOCX test files
func TestParseDocxFiles(t *testing.T) {
	parser := NewParser()
	docxFiles := getDocxTestFiles()

	if len(docxFiles) == 0 {
		t.Skip("No DOCX test files found")
	}

	for _, filepath := range docxFiles {
		t.Run(fmt.Sprintf("Parse_%s", filepath), func(t *testing.T) {
			data, err := os.ReadFile(filepath)
			if err != nil {
				t.Fatalf("Failed to read test file %s: %v", filepath, err)
			}

			result, err := parser.Parse(data)
			if err != nil {
				t.Fatalf("Failed to parse DOCX file %s: %v", filepath, err)
			}

			// Basic validation
			if result == nil {
				t.Fatal("Parse result is nil")
			}

			if result.Metadata == nil {
				t.Error("Metadata is nil")
			}

			if result.Media == nil {
				t.Error("Media array is nil")
			}

			// Check if markdown content exists
			if len(result.Markdown) == 0 {
				t.Log("Warning: No markdown content extracted")
			}

			// Log basic info
			t.Logf("File: %s", filepath)
			t.Logf("Markdown length: %d", len(result.Markdown))
			t.Logf("Media count: %d", len(result.Media))
			if result.Metadata != nil {
				t.Logf("Title: %s", result.Metadata.Title)
				t.Logf("Author: %s", result.Metadata.Author)
				t.Logf("Pages: %d", result.Metadata.Pages)
				t.Logf("Text ranges: %d", len(result.Metadata.TextRanges))
			}
		})
	}
}

// TestParsePptxFiles tests parsing of all PPTX test files
func TestParsePptxFiles(t *testing.T) {
	parser := NewParser()
	pptxFiles := getPptxTestFiles()

	if len(pptxFiles) == 0 {
		t.Skip("No PPTX test files found")
	}

	for _, filepath := range pptxFiles {
		t.Run(fmt.Sprintf("Parse_%s", filepath), func(t *testing.T) {
			data, err := os.ReadFile(filepath)
			if err != nil {
				t.Fatalf("Failed to read test file %s: %v", filepath, err)
			}

			result, err := parser.Parse(data)
			if err != nil {
				t.Fatalf("Failed to parse PPTX file %s: %v", filepath, err)
			}

			// Basic validation
			if result == nil {
				t.Fatal("Parse result is nil")
			}

			if result.Metadata == nil {
				t.Error("Metadata is nil")
			}

			if result.Media == nil {
				t.Error("Media array is nil")
			}

			// Check if markdown content exists
			if len(result.Markdown) == 0 {
				t.Log("Warning: No markdown content extracted")
			}

			// Log basic info
			t.Logf("File: %s", filepath)
			t.Logf("Markdown length: %d", len(result.Markdown))
			t.Logf("Media count: %d", len(result.Media))
			if result.Metadata != nil {
				t.Logf("Title: %s", result.Metadata.Title)
				t.Logf("Author: %s", result.Metadata.Author)
				t.Logf("Pages: %d", result.Metadata.Pages)
				t.Logf("Text ranges: %d", len(result.Metadata.TextRanges))
			}
		})
	}
}

// TestDocumentTypeDetection tests the document type detection
func TestDocumentTypeDetection(t *testing.T) {
	parser := NewParser()

	// Test with DOCX files
	docxFiles := getDocxTestFiles()
	if len(docxFiles) > 0 {
		data, err := os.ReadFile(docxFiles[0])
		if err != nil {
			t.Fatalf("Failed to read DOCX test file: %v", err)
		}

		result, err := parser.Parse(data)
		if err != nil {
			t.Fatalf("Failed to parse DOCX file: %v", err)
		}

		// Should not return PPTX-specific content
		if strings.Contains(result.Markdown, "## Slide") {
			t.Error("DOCX file incorrectly detected as PPTX")
		}
	}

	// Test with PPTX files
	pptxFiles := getPptxTestFiles()
	if len(pptxFiles) > 0 {
		data, err := os.ReadFile(pptxFiles[0])
		if err != nil {
			t.Fatalf("Failed to read PPTX test file: %v", err)
		}

		result, err := parser.Parse(data)
		if err != nil {
			t.Fatalf("Failed to parse PPTX file: %v", err)
		}

		// Should contain slide separators
		if !strings.Contains(result.Markdown, "## Slide") {
			t.Error("PPTX file should contain slide separators")
		}
	}
}

// TestParseInvalidFile tests parsing of invalid files
func TestParseInvalidFile(t *testing.T) {
	parser := NewParser()

	// Test with invalid data
	invalidData := []byte("This is not a valid Office document")
	result, err := parser.Parse(invalidData)

	if err == nil {
		t.Error("Expected error for invalid file data")
	}

	if result != nil {
		t.Error("Expected nil result for invalid file data")
	}
}

// TestConvenienceFunctions tests the convenience functions
func TestConvenienceFunctions(t *testing.T) {
	docxFiles := getDocxTestFiles()
	if len(docxFiles) == 0 {
		t.Skip("No DOCX test files found")
	}

	testFile := docxFiles[0]

	// Test parseFile
	result, err := parseFile(testFile)
	if err != nil {
		t.Fatalf("parseFile failed: %v", err)
	}
	if result == nil {
		t.Fatal("parseFile returned nil result")
	}

	// Test getMarkdownOnly
	markdown, err := getMarkdownOnly(testFile)
	if err != nil {
		t.Fatalf("getMarkdownOnly failed: %v", err)
	}
	if markdown != result.Markdown {
		t.Error("getMarkdownOnly returned different markdown")
	}

	// Test getMediaOnly
	media, err := getMediaOnly(testFile)
	if err != nil {
		t.Fatalf("getMediaOnly failed: %v", err)
	}
	if len(media) != len(result.Media) {
		t.Error("getMediaOnly returned different media count")
	}

	// Test validateDocument
	err = validateDocument(testFile)
	if err != nil {
		t.Fatalf("validateDocument failed: %v", err)
	}

	// Test with non-existent file
	err = validateDocument("non_existent_file.docx")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestMediaExtraction tests media file extraction
func TestMediaExtraction(t *testing.T) {
	parser := NewParser()

	// Test with files that might contain media
	allFiles := append(getDocxTestFiles(), getPptxTestFiles()...)

	for _, filepath := range allFiles {
		t.Run(fmt.Sprintf("Media_%s", filepath), func(t *testing.T) {
			data, err := os.ReadFile(filepath)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			result, err := parser.Parse(data)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			// Log media information
			t.Logf("File: %s", filepath)
			t.Logf("Media count: %d", len(result.Media))

			for i, media := range result.Media {
				t.Logf("Media %d: ID=%s, Type=%s, Format=%s, Size=%d",
					i+1, media.ID, media.Type, media.Format, len(media.Content))

				// Validate media structure
				if media.ID == "" {
					t.Error("Media ID is empty")
				}
				if media.Type == "" {
					t.Error("Media Type is empty")
				}
				if media.Format == "" {
					t.Error("Media Format is empty")
				}
				if len(media.Content) == 0 {
					t.Error("Media Content is empty")
				}
			}
		})
	}
}

// TestTextRangeTracking tests text range tracking
func TestTextRangeTracking(t *testing.T) {
	parser := NewParser()
	docxFiles := getDocxTestFiles()

	if len(docxFiles) == 0 {
		t.Skip("No DOCX test files found")
	}

	for _, filepath := range docxFiles {
		t.Run(fmt.Sprintf("TextRange_%s", filepath), func(t *testing.T) {
			data, err := os.ReadFile(filepath)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			result, err := parser.Parse(data)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			if result.Metadata == nil {
				t.Fatal("Metadata is nil")
			}

			t.Logf("File: %s", filepath)
			t.Logf("Text ranges: %d", len(result.Metadata.TextRanges))

			for i, tr := range result.Metadata.TextRanges {
				t.Logf("Range %d: Type=%s, Page=%d, Pos=%d-%d",
					i+1, tr.Type, tr.Page, tr.StartPos, tr.EndPos)

				// Validate text range structure
				if tr.StartPos < 0 {
					t.Error("StartPos should not be negative")
				}
				if tr.EndPos < tr.StartPos {
					t.Error("EndPos should be >= StartPos")
				}
				if tr.Page < 1 {
					t.Error("Page should be >= 1")
				}
				if tr.Type == "" {
					t.Error("Type should not be empty")
				}
			}
		})
	}
}

// BenchmarkParseDocx benchmarks DOCX parsing
func BenchmarkParseDocx(b *testing.B) {
	docxFiles := getDocxTestFiles()
	if len(docxFiles) == 0 {
		b.Skip("No DOCX test files found")
	}

	data, err := os.ReadFile(docxFiles[0])
	if err != nil {
		b.Fatalf("Failed to read test file: %v", err)
	}

	parser := NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

// BenchmarkParsePptx benchmarks PPTX parsing
func BenchmarkParsePptx(b *testing.B) {
	pptxFiles := getPptxTestFiles()
	if len(pptxFiles) == 0 {
		b.Skip("No PPTX test files found")
	}

	data, err := os.ReadFile(pptxFiles[0])
	if err != nil {
		b.Fatalf("Failed to read test file: %v", err)
	}

	parser := NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}
