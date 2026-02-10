package office

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yaoapp/gou/process"
)

// ==== Test Helper Functions ====

// getProcessTestDataDir returns the office test data directory
func getProcessTestDataDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(currentFile)
	testDataDir := filepath.Join(currentDir, "tests")
	absPath, err := filepath.Abs(testDataDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to get absolute path for office test data dir: %v", err))
	}
	return absPath
}

// ensureProcessTestDataExists checks if test data exists
func ensureProcessTestDataExists(t *testing.T) {
	t.Helper()
	testDir := getProcessTestDataDir()
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skipf("Office test data directory does not exist: %s", testDir)
	}
}

// getProcessTestDocxFiles returns all DOCX test files
func getProcessTestDocxFiles() []string {
	testDataDir := getProcessTestDataDir()
	docxDir := filepath.Join(testDataDir, "docx")

	files, err := os.ReadDir(docxDir)
	if err != nil {
		return nil
	}

	var docxFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".docx") {
			docxFiles = append(docxFiles, filepath.Join(docxDir, file.Name()))
		}
	}
	return docxFiles
}

// getProcessTestPptxFiles returns all PPTX test files
func getProcessTestPptxFiles() []string {
	testDataDir := getProcessTestDataDir()
	pptxDir := filepath.Join(testDataDir, "pptx")

	files, err := os.ReadDir(pptxDir)
	if err != nil {
		return nil
	}

	var pptxFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".pptx") {
			pptxFiles = append(pptxFiles, filepath.Join(pptxDir, file.Name()))
		}
	}
	return pptxFiles
}

// ==== Process Registration Tests ====

func TestProcessRegistration(t *testing.T) {
	// Verify that office process handlers are registered
	handlers := []string{"office.parse", "office.parsebytes"}
	for _, name := range handlers {
		p, err := process.Of(name, nil)
		if err != nil {
			t.Errorf("Process %s should be registered, got error: %v", name, err)
			continue
		}
		if p == nil {
			t.Errorf("Process %s returned nil", name)
		}
	}
}

// ==== office.Parse Tests ====

func TestProcessParse_DOCX(t *testing.T) {
	ensureProcessTestDataExists(t)

	docxFiles := getProcessTestDocxFiles()
	if len(docxFiles) == 0 {
		t.Skip("No DOCX test files found")
	}

	for _, testFile := range docxFiles {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			outputDir := filepath.Join(os.TempDir(), "gou_office_test_parse_docx_"+filepath.Base(testFile))
			defer os.RemoveAll(outputDir)

			config := map[string]interface{}{
				"output_dir": outputDir,
			}

			p, err := process.Of("office.parse", testFile, config)
			if err != nil {
				t.Fatalf("Failed to create process: %v", err)
			}

			result := p.Run()
			if result == nil {
				t.Fatal("office.Parse returned nil")
			}

			parseResult, ok := result.(*ProcessParseResult)
			if !ok {
				t.Fatalf("Expected *ProcessParseResult, got %T", result)
			}

			// Validate markdown content
			if parseResult.Markdown == "" {
				t.Error("Expected non-empty markdown")
			}

			// Validate metadata
			if parseResult.Metadata == nil {
				t.Error("Expected non-nil metadata")
			}

			// Log results
			t.Logf("DOCX %s: markdown=%d chars, media=%d items",
				filepath.Base(testFile), len(parseResult.Markdown), len(parseResult.Media))

			// If there are media items, verify files exist on disk
			for _, media := range parseResult.Media {
				if media.Path == "" {
					t.Errorf("Media %s has empty path", media.ID)
					continue
				}

				// Verify file exists on disk
				fi, err := os.Stat(media.Path)
				if err != nil {
					t.Errorf("Media file does not exist: %s: %v", media.Path, err)
					continue
				}

				if fi.Size() <= 0 {
					t.Errorf("Media file is empty: %s", media.Path)
				}

				if media.Type == "" {
					t.Errorf("Media %s has empty type", media.ID)
				}
				if media.Format == "" {
					t.Errorf("Media %s has empty format", media.ID)
				}

				t.Logf("  Media: id=%s, type=%s, format=%s, path=%s (%d bytes)",
					media.ID, media.Type, media.Format, filepath.Base(media.Path), fi.Size())
			}
		})
	}
}

func TestProcessParse_PPTX(t *testing.T) {
	ensureProcessTestDataExists(t)

	pptxFiles := getProcessTestPptxFiles()
	if len(pptxFiles) == 0 {
		t.Skip("No PPTX test files found")
	}

	for _, testFile := range pptxFiles {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			outputDir := filepath.Join(os.TempDir(), "gou_office_test_parse_pptx_"+filepath.Base(testFile))
			defer os.RemoveAll(outputDir)

			config := map[string]interface{}{
				"output_dir": outputDir,
			}

			p, err := process.Of("office.parse", testFile, config)
			if err != nil {
				t.Fatalf("Failed to create process: %v", err)
			}

			result := p.Run()
			if result == nil {
				t.Fatal("office.Parse returned nil")
			}

			parseResult, ok := result.(*ProcessParseResult)
			if !ok {
				t.Fatalf("Expected *ProcessParseResult, got %T", result)
			}

			// Validate markdown content
			if parseResult.Markdown == "" {
				t.Error("Expected non-empty markdown")
			}

			// Validate metadata
			if parseResult.Metadata == nil {
				t.Error("Expected non-nil metadata")
			}

			t.Logf("PPTX %s: markdown=%d chars, media=%d items, pages=%d",
				filepath.Base(testFile), len(parseResult.Markdown),
				len(parseResult.Media), parseResult.Metadata.Pages)
		})
	}
}

func TestProcessParse_AutoTempDir(t *testing.T) {
	ensureProcessTestDataExists(t)

	docxFiles := getProcessTestDocxFiles()
	if len(docxFiles) == 0 {
		t.Skip("No DOCX test files found")
	}

	// Pick a file that has media
	var testFile string
	for _, f := range docxFiles {
		testFile = f
		break
	}

	// No output_dir — should auto-create temp dir
	p, err := process.Of("office.parse", testFile)
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	result := p.Run()
	if result == nil {
		t.Fatal("office.Parse returned nil")
	}

	parseResult, ok := result.(*ProcessParseResult)
	if !ok {
		t.Fatalf("Expected *ProcessParseResult, got %T", result)
	}

	// Clean up auto temp dir if media was extracted
	if len(parseResult.Media) > 0 {
		dir := filepath.Dir(parseResult.Media[0].Path)
		defer os.RemoveAll(dir)

		for _, media := range parseResult.Media {
			if _, err := os.Stat(media.Path); os.IsNotExist(err) {
				t.Errorf("Media file does not exist: %s", media.Path)
			}
		}
	}

	t.Logf("Auto temp dir: markdown=%d chars, media=%d items",
		len(parseResult.Markdown), len(parseResult.Media))
}

func TestProcessParse_NonExistentFile(t *testing.T) {
	p, err := process.Of("office.parse", "/non/existent/document.docx")
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic for non-existent file, but got none")
		}
		t.Logf("Correctly panicked with: %v", r)
	}()

	p.Run()
}

func TestProcessParse_UnsupportedFormat(t *testing.T) {
	// Create a temporary non-office file
	tempFile := filepath.Join(os.TempDir(), "test_unsupported.txt")
	err := os.WriteFile(tempFile, []byte("This is not an office file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)

	p, pErr := process.Of("office.parse", tempFile)
	if pErr != nil {
		t.Fatalf("Failed to create process: %v", pErr)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic for unsupported format, but got none")
		}
		t.Logf("Correctly panicked with: %v", r)
	}()

	p.Run()
}

// ==== office.ParseBytes Tests ====

func TestProcessParseBytes_DOCX(t *testing.T) {
	ensureProcessTestDataExists(t)

	docxFiles := getProcessTestDocxFiles()
	if len(docxFiles) == 0 {
		t.Skip("No DOCX test files found")
	}

	// Test with the first DOCX file
	testFile := docxFiles[0]

	outputDir := filepath.Join(os.TempDir(), "gou_office_test_parsebytes")
	defer os.RemoveAll(outputDir)

	// Read file and encode to base64
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	base64Data := base64.StdEncoding.EncodeToString(data)
	config := map[string]interface{}{
		"output_dir": outputDir,
	}

	p, pErr := process.Of("office.parsebytes", base64Data, config)
	if pErr != nil {
		t.Fatalf("Failed to create process: %v", pErr)
	}

	result := p.Run()
	if result == nil {
		t.Fatal("office.ParseBytes returned nil")
	}

	parseResult, ok := result.(*ProcessParseResult)
	if !ok {
		t.Fatalf("Expected *ProcessParseResult, got %T", result)
	}

	// Validate markdown content
	if parseResult.Markdown == "" {
		t.Error("Expected non-empty markdown")
	}

	// Verify media files exist
	for _, media := range parseResult.Media {
		if _, err := os.Stat(media.Path); os.IsNotExist(err) {
			t.Errorf("Media file does not exist: %s", media.Path)
		}
	}

	t.Logf("ParseBytes DOCX: markdown=%d chars, media=%d items",
		len(parseResult.Markdown), len(parseResult.Media))
}

func TestProcessParseBytes_ConsistencyWithParse(t *testing.T) {
	ensureProcessTestDataExists(t)

	docxFiles := getProcessTestDocxFiles()
	if len(docxFiles) == 0 {
		t.Skip("No DOCX test files found")
	}

	testFile := docxFiles[0]

	// Parse via file path
	outputDir1 := filepath.Join(os.TempDir(), "gou_office_test_consistency_parse")
	defer os.RemoveAll(outputDir1)

	p1, err := process.Of("office.parse", testFile, map[string]interface{}{
		"output_dir": outputDir1,
	})
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	result1 := p1.Run()
	parseResult1, ok := result1.(*ProcessParseResult)
	if !ok {
		t.Fatalf("Expected *ProcessParseResult, got %T", result1)
	}

	// Parse via base64 bytes
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	outputDir2 := filepath.Join(os.TempDir(), "gou_office_test_consistency_parsebytes")
	defer os.RemoveAll(outputDir2)

	base64Data := base64.StdEncoding.EncodeToString(data)

	p2, err := process.Of("office.parsebytes", base64Data, map[string]interface{}{
		"output_dir": outputDir2,
	})
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	result2 := p2.Run()
	parseResult2, ok := result2.(*ProcessParseResult)
	if !ok {
		t.Fatalf("Expected *ProcessParseResult, got %T", result2)
	}

	// Compare results - markdown should be identical
	if parseResult1.Markdown != parseResult2.Markdown {
		t.Errorf("Markdown content should be identical between Parse and ParseBytes")
		t.Logf("Parse: %d chars", len(parseResult1.Markdown))
		t.Logf("ParseBytes: %d chars", len(parseResult2.Markdown))
	}

	if len(parseResult1.Media) != len(parseResult2.Media) {
		t.Errorf("Media count should be identical: Parse=%d, ParseBytes=%d",
			len(parseResult1.Media), len(parseResult2.Media))
	}

	// Verify media files have matching sizes (match by ID, not index)
	media2ByID := make(map[string]ProcessMediaEntry)
	for _, m := range parseResult2.Media {
		media2ByID[m.ID] = m
	}
	for _, m1 := range parseResult1.Media {
		m2, ok := media2ByID[m1.ID]
		if !ok {
			t.Errorf("Media %s missing in ParseBytes result", m1.ID)
			continue
		}
		fi1, _ := os.Stat(m1.Path)
		fi2, _ := os.Stat(m2.Path)
		if fi1 != nil && fi2 != nil && fi1.Size() != fi2.Size() {
			t.Errorf("Media %s size mismatch: Parse=%d, ParseBytes=%d",
				m1.ID, fi1.Size(), fi2.Size())
		}
	}

	t.Logf("Consistency check passed: both methods produce identical results")
}

func TestProcessParseBytes_InvalidBase64(t *testing.T) {
	p, err := process.Of("office.parsebytes", "!!!invalid-base64!!!")
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic for invalid base64, but got none")
		}
		t.Logf("Correctly panicked with: %v", r)
	}()

	p.Run()
}

// ==== Comprehensive Integration Test ====

func TestProcessOffice_Comprehensive(t *testing.T) {
	ensureProcessTestDataExists(t)

	t.Run("All DOCX files", func(t *testing.T) {
		docxFiles := getProcessTestDocxFiles()
		for _, testFile := range docxFiles {
			t.Run(filepath.Base(testFile), func(t *testing.T) {
				outputDir := filepath.Join(os.TempDir(), "gou_office_comprehensive_"+filepath.Base(testFile))
				defer os.RemoveAll(outputDir)

				p, err := process.Of("office.parse", testFile, map[string]interface{}{
					"output_dir": outputDir,
				})
				if err != nil {
					t.Fatalf("Failed to create process: %v", err)
				}

				result := p.Run()
				parseResult, ok := result.(*ProcessParseResult)
				if !ok {
					t.Fatalf("Expected *ProcessParseResult, got %T", result)
				}

				if parseResult.Markdown == "" {
					t.Error("Expected non-empty markdown")
				}
				if parseResult.Metadata == nil {
					t.Error("Expected non-nil metadata")
				}

				t.Logf("%s: %d chars, %d media",
					filepath.Base(testFile), len(parseResult.Markdown), len(parseResult.Media))
			})
		}
	})

	t.Run("All PPTX files", func(t *testing.T) {
		pptxFiles := getProcessTestPptxFiles()
		for _, testFile := range pptxFiles {
			t.Run(filepath.Base(testFile), func(t *testing.T) {
				outputDir := filepath.Join(os.TempDir(), "gou_office_comprehensive_"+filepath.Base(testFile))
				defer os.RemoveAll(outputDir)

				p, err := process.Of("office.parse", testFile, map[string]interface{}{
					"output_dir": outputDir,
				})
				if err != nil {
					t.Fatalf("Failed to create process: %v", err)
				}

				result := p.Run()
				parseResult, ok := result.(*ProcessParseResult)
				if !ok {
					t.Fatalf("Expected *ProcessParseResult, got %T", result)
				}

				if parseResult.Markdown == "" {
					t.Error("Expected non-empty markdown")
				}

				t.Logf("%s: %d chars, %d media, pages=%d",
					filepath.Base(testFile), len(parseResult.Markdown),
					len(parseResult.Media), parseResult.Metadata.Pages)
			})
		}
	})
}
