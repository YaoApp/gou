package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// getTestDataPath returns the path to test data files using runtime
func getTestDataPath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "tests")
}

// getTestPDFFiles returns all test PDF files
func getTestPDFFiles() []string {
	testDataPath := getTestDataPath()
	return []string{
		filepath.Join(testDataPath, "english_sample_1.pdf"),
		filepath.Join(testDataPath, "english_sample_2.pdf"),
		filepath.Join(testDataPath, "chinese_sample_1.pdf"),
		filepath.Join(testDataPath, "chinese_sample_2.pdf"),
	}
}

// cleanupTestOutput removes test output directories
func cleanupTestOutput(t *testing.T, paths ...string) {
	for _, path := range paths {
		if path != "" && path != "/" {
			if err := os.RemoveAll(path); err != nil {
				t.Logf("Warning: failed to cleanup %s: %v", path, err)
			}
		}
	}
}

func TestGetInfo(t *testing.T) {
	pdf := New(Options{})
	ctx := context.Background()
	testFiles := getTestPDFFiles()

	for _, testFile := range testFiles {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			// Check if test file exists
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				t.Skipf("Test file %s does not exist", testFile)
				return
			}

			// Get PDF info
			info, err := pdf.GetInfo(ctx, testFile)
			if err != nil {
				t.Fatalf("GetInfo failed: %v", err)
			}

			// Validate returned info
			if info.FilePath != testFile {
				t.Errorf("Expected FilePath %s, got %s", testFile, info.FilePath)
			}

			if info.FileSize <= 0 {
				t.Errorf("Expected FileSize > 0, got %d", info.FileSize)
			}

			if info.PageCount <= 0 {
				t.Errorf("Expected PageCount > 0, got %d", info.PageCount)
			}

			if info.Metadata == nil {
				t.Error("Expected Metadata to be initialized")
			}

			t.Logf("PDF: %s, Pages: %d, Size: %d bytes",
				filepath.Base(testFile), info.PageCount, info.FileSize)

			// Log metadata if available
			if len(info.Metadata) > 0 {
				t.Logf("Metadata: %+v", info.Metadata)
			}
		})
	}
}

func TestGetInfo_NonExistentFile(t *testing.T) {
	pdf := New(Options{})
	ctx := context.Background()

	_, err := pdf.GetInfo(ctx, "nonexistent.pdf")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestSplit tests the Split method with test PDF files
func TestSplit(t *testing.T) {
	processor := New(Options{})
	ctx := context.Background()
	testFiles := getTestPDFFiles()

	for _, testFile := range testFiles {
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Skipf("Test file %s does not exist, skipping", testFile)
			continue
		}

		t.Run(filepath.Base(testFile), func(t *testing.T) {
			// Create temporary output directory
			outputDir := filepath.Join(os.TempDir(), "pdf_test_split_"+fmt.Sprintf("%d", time.Now().UnixNano()))
			defer cleanupTestOutput(t, outputDir)

			// Test split configuration
			config := SplitConfig{
				OutputDir:    outputDir,
				OutputPrefix: "test_split",
				// Default: split each page as separate file
			}

			// Split the PDF
			outputFiles, err := processor.Split(ctx, testFile, config)
			if err != nil {
				t.Fatalf("Split failed: %v", err)
			}

			// Verify output files exist
			if len(outputFiles) == 0 {
				t.Error("No output files generated")
				return
			}

			t.Logf("Split generated %d files:", len(outputFiles))
			for _, file := range outputFiles {
				if _, err := os.Stat(file); os.IsNotExist(err) {
					t.Errorf("Output file %s does not exist", file)
				} else {
					t.Logf("  - %s", file)
				}
			}
		})
	}
}

func TestSplit_InvalidConfig(t *testing.T) {
	testFiles := getTestPDFFiles()
	if len(testFiles) == 0 {
		t.Skip("No test files available")
	}

	testFile := testFiles[0]
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test file does not exist")
	}

	pdf := New(Options{})
	ctx := context.Background()

	// Test with empty output directory
	config := SplitConfig{
		OutputDir:    "",
		OutputPrefix: "test",
		PageRanges:   []string{"1-1"},
	}

	_, err := pdf.Split(ctx, testFile, config)
	if err == nil {
		t.Error("Expected error for empty output directory")
	}
}

func TestConvert(t *testing.T) {
	testFiles := getTestPDFFiles()
	ctx := context.Background()

	// Test with available tools
	tools := []ConvertTool{ToolPdftoppm, ToolMutool, ToolImageMagick}

	for _, tool := range tools {
		t.Run(string(tool), func(t *testing.T) {
			pdf := New(Options{ConvertTool: tool})

			// Check if tool is available
			if !pdf.cmd.IsAvailable(tool) {
				t.Skipf("Tool %s is not available", tool)
				return
			}

			// Test with first available PDF file
			var testFile string
			for _, file := range testFiles {
				if _, err := os.Stat(file); err == nil {
					testFile = file
					break
				}
			}

			if testFile == "" {
				t.Skip("No test files available")
				return
			}

			// Create output directory
			outputDir := filepath.Join(os.TempDir(), fmt.Sprintf("pdf_convert_test_%d", time.Now().UnixNano()))
			defer cleanupTestOutput(t, outputDir)

			// Test PNG conversion
			t.Run("ConvertToPNG", func(t *testing.T) {
				config := ConvertConfig{
					OutputDir:    outputDir,
					OutputPrefix: "page",
					Format:       "png",
					DPI:          150,
					PageRange:    "1-1", // Convert only first page
				}

				files, err := pdf.Convert(ctx, testFile, config)
				if err != nil {
					t.Fatalf("Convert failed: %v", err)
				}

				if len(files) != 1 {
					t.Errorf("Expected 1 file, got %d", len(files))
				}

				// Check if file exists
				for _, file := range files {
					if _, err := os.Stat(file); os.IsNotExist(err) {
						t.Errorf("Convert file %s does not exist", file)
					}
				}

				t.Logf("Converted %s to %d PNG files using %s",
					filepath.Base(testFile), len(files), tool)
			})

			// Test JPEG conversion
			t.Run("ConvertToJPEG", func(t *testing.T) {
				config := ConvertConfig{
					OutputDir:    outputDir,
					OutputPrefix: "page_jpg",
					Format:       "jpg",
					DPI:          150,
					Quality:      85,
					PageRange:    "1-1", // Convert only first page
				}

				files, err := pdf.Convert(ctx, testFile, config)
				if err != nil {
					t.Fatalf("Convert failed: %v", err)
				}

				if len(files) != 1 {
					t.Errorf("Expected 1 file, got %d", len(files))
				}

				// Check if file exists
				for _, file := range files {
					if _, err := os.Stat(file); os.IsNotExist(err) {
						t.Errorf("Convert file %s does not exist", file)
					}
				}

				t.Logf("Converted %s to %d JPEG files using %s",
					filepath.Base(testFile), len(files), tool)
			})
		})
	}
}

func TestConvert_InvalidConfig(t *testing.T) {
	testFiles := getTestPDFFiles()
	if len(testFiles) == 0 {
		t.Skip("No test files available")
	}

	testFile := testFiles[0]
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test file does not exist")
	}

	pdf := New(Options{})
	ctx := context.Background()

	// Test with empty output directory
	config := ConvertConfig{
		OutputDir:    "",
		OutputPrefix: "test",
		Format:       "png",
		PageRange:    "1-1",
	}

	_, err := pdf.Convert(ctx, testFile, config)
	if err == nil {
		t.Error("Expected error for empty output directory")
	}
}

func TestConvert_UnsupportedTool(t *testing.T) {
	testFiles := getTestPDFFiles()
	if len(testFiles) == 0 {
		t.Skip("No test files available")
	}

	testFile := testFiles[0]
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test file does not exist")
	}

	// Create PDF with unsupported tool
	pdf := New(Options{ConvertTool: "unsupported"})
	ctx := context.Background()

	outputDir := filepath.Join(os.TempDir(), fmt.Sprintf("pdf_convert_unsupported_%d", time.Now().UnixNano()))
	defer cleanupTestOutput(t, outputDir)

	config := ConvertConfig{
		OutputDir:    outputDir,
		OutputPrefix: "test",
		Format:       "png",
		PageRange:    "1-1",
	}

	_, err := pdf.Convert(ctx, testFile, config)
	if err == nil {
		t.Error("Expected error for unsupported tool")
	}

	// Check error message contains expected text
	if !contains(err.Error(), "unsupported") {
		t.Errorf("Expected error message to contain 'unsupported', got: %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkGetInfo(b *testing.B) {
	testFiles := getTestPDFFiles()
	if len(testFiles) == 0 {
		b.Skip("No test files available")
	}

	testFile := testFiles[0]
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("Test file does not exist")
	}

	pdf := New(Options{})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pdf.GetInfo(ctx, testFile)
		if err != nil {
			b.Fatalf("GetInfo failed: %v", err)
		}
	}
}

func BenchmarkSplit(b *testing.B) {
	testFiles := getTestPDFFiles()
	if len(testFiles) == 0 {
		b.Skip("No test files available")
	}

	testFile := testFiles[0]
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("Test file does not exist")
	}

	pdf := New(Options{})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputDir := filepath.Join(os.TempDir(), fmt.Sprintf("pdf_benchmark_split_%d", time.Now().UnixNano()))

		config := SplitConfig{
			OutputDir:    outputDir,
			OutputPrefix: "benchmark",
			PageRanges:   []string{"1-1"}, // Only split first page for benchmark
		}

		_, err := pdf.Split(ctx, testFile, config)
		if err != nil {
			b.Fatalf("Split failed: %v", err)
		}

		// Clean up
		os.RemoveAll(outputDir)
	}
}
