package pdf

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yaoapp/gou/process"
)

// ==== Test Helper Functions ====

// getProcessTestDataPath returns the path to test data files
func getProcessTestDataPath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "tests")
}

// getProcessTestPDFFile returns a test PDF file path
func getProcessTestPDFFile() string {
	return filepath.Join(getProcessTestDataPath(), "english_sample_1.pdf")
}

// ensureProcessTestData checks if test data exists
func ensureProcessTestData(t *testing.T) {
	t.Helper()
	testFile := getProcessTestPDFFile()
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test data not found: %s", testFile)
	}
}

// ==== Process Registration Tests ====

func TestProcessRegistration(t *testing.T) {
	// Verify that pdf process handlers are registered
	handlers := []string{"pdf.info", "pdf.split", "pdf.convert"}
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

// ==== pdf.Info Tests ====

func TestProcessInfo(t *testing.T) {
	ensureProcessTestData(t)

	testFiles := []string{
		filepath.Join(getProcessTestDataPath(), "english_sample_1.pdf"),
		filepath.Join(getProcessTestDataPath(), "english_sample_2.pdf"),
		filepath.Join(getProcessTestDataPath(), "chinese_sample_1.pdf"),
		filepath.Join(getProcessTestDataPath(), "chinese_sample_2.pdf"),
	}

	for _, testFile := range testFiles {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				t.Skipf("Test file not found: %s", testFile)
				return
			}

			p, err := process.Of("pdf.info", testFile)
			if err != nil {
				t.Fatalf("Failed to create process: %v", err)
			}

			result := p.Run()
			if result == nil {
				t.Fatal("pdf.Info returned nil")
			}

			info, ok := result.(*Info)
			if !ok {
				t.Fatalf("Expected *Info, got %T", result)
			}

			if info.PageCount <= 0 {
				t.Errorf("Expected PageCount > 0, got %d", info.PageCount)
			}

			if info.FileSize <= 0 {
				t.Errorf("Expected FileSize > 0, got %d", info.FileSize)
			}

			t.Logf("PDF: %s, Pages: %d, Size: %d bytes",
				filepath.Base(testFile), info.PageCount, info.FileSize)
		})
	}
}

func TestProcessInfo_NonExistentFile(t *testing.T) {
	p, err := process.Of("pdf.info", "/non/existent/file.pdf")
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

// ==== pdf.Convert Tests ====

func TestProcessConvert(t *testing.T) {
	ensureProcessTestData(t)

	testFile := getProcessTestPDFFile()
	outputDir := filepath.Join(os.TempDir(), "gou_pdf_process_test_convert")
	defer os.RemoveAll(outputDir)

	t.Run("Convert to PNG", func(t *testing.T) {
		config := map[string]interface{}{
			"format":     "png",
			"dpi":        150,
			"pages":      "1-2",
			"output_dir": outputDir,
		}

		p, err := process.Of("pdf.convert", testFile, config)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		if result == nil {
			t.Fatal("pdf.Convert returned nil")
		}

		files, ok := result.([]string)
		if !ok {
			t.Fatalf("Expected []string, got %T", result)
		}

		if len(files) == 0 {
			t.Error("Expected at least 1 output file")
		}

		// Verify files exist
		for _, f := range files {
			if _, err := os.Stat(f); os.IsNotExist(err) {
				t.Errorf("Output file does not exist: %s", f)
			}
		}

		t.Logf("Converted %d pages to PNG", len(files))
	})

	t.Run("Convert with default config", func(t *testing.T) {
		outputDir2 := filepath.Join(os.TempDir(), "gou_pdf_process_test_convert_default")
		defer os.RemoveAll(outputDir2)

		config := map[string]interface{}{
			"output_dir": outputDir2,
		}

		p, err := process.Of("pdf.convert", testFile, config)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		if result == nil {
			t.Fatal("pdf.Convert returned nil")
		}

		files, ok := result.([]string)
		if !ok {
			t.Fatalf("Expected []string, got %T", result)
		}

		if len(files) == 0 {
			t.Error("Expected at least 1 output file")
		}

		t.Logf("Converted %d pages with defaults", len(files))
	})
}

func TestProcessConvert_AutoTempDir(t *testing.T) {
	ensureProcessTestData(t)

	testFile := getProcessTestPDFFile()

	// No output_dir specified - should auto-create temp dir
	config := map[string]interface{}{
		"format": "png",
		"pages":  "1",
	}

	p, err := process.Of("pdf.convert", testFile, config)
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	result := p.Run()
	if result == nil {
		t.Fatal("pdf.Convert returned nil")
	}

	files, ok := result.([]string)
	if !ok {
		t.Fatalf("Expected []string, got %T", result)
	}

	if len(files) == 0 {
		t.Error("Expected at least 1 output file")
	}

	// Verify files exist and clean up
	for _, f := range files {
		fi, err := os.Stat(f)
		if err != nil {
			t.Errorf("Output file does not exist: %s", f)
			continue
		}
		t.Logf("Auto temp: %s (%d bytes)", f, fi.Size())
	}

	// Clean up the auto-created temp dir
	if len(files) > 0 {
		os.RemoveAll(filepath.Dir(files[0]))
	}

	t.Logf("Auto temp dir convert: %d pages", len(files))
}

func TestProcessConvert_NonExistentFile(t *testing.T) {
	config := map[string]interface{}{
		"format": "png",
	}

	p, err := process.Of("pdf.convert", "/non/existent/file.pdf", config)
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

// ==== pdf.Split Tests ====

func TestProcessSplit(t *testing.T) {
	ensureProcessTestData(t)

	testFile := getProcessTestPDFFile()
	outputDir := filepath.Join(os.TempDir(), "gou_pdf_process_test_split")
	defer os.RemoveAll(outputDir)

	t.Run("Split by page ranges", func(t *testing.T) {
		config := map[string]interface{}{
			"pages":      "1-2",
			"output_dir": outputDir,
		}

		p, err := process.Of("pdf.split", testFile, config)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		if result == nil {
			t.Fatal("pdf.Split returned nil")
		}

		files, ok := result.([]string)
		if !ok {
			t.Fatalf("Expected []string, got %T", result)
		}

		if len(files) == 0 {
			t.Error("Expected at least 1 output file")
		}

		// Verify files exist
		for _, f := range files {
			if _, err := os.Stat(f); os.IsNotExist(err) {
				t.Errorf("Output file does not exist: %s", f)
			}
		}

		t.Logf("Split into %d files", len(files))
	})
}

func TestProcessSplit_AutoTempDir(t *testing.T) {
	ensureProcessTestData(t)

	testFile := getProcessTestPDFFile()

	// No output_dir specified - should auto-create temp dir
	config := map[string]interface{}{
		"pages": "1",
	}

	p, err := process.Of("pdf.split", testFile, config)
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	result := p.Run()
	if result == nil {
		t.Fatal("pdf.Split returned nil")
	}

	files, ok := result.([]string)
	if !ok {
		t.Fatalf("Expected []string, got %T", result)
	}

	if len(files) == 0 {
		t.Error("Expected at least 1 output file")
	}

	// Verify files exist and clean up
	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("Output file does not exist: %s", f)
		}
	}

	// Clean up the auto-created temp dir
	if len(files) > 0 {
		os.RemoveAll(filepath.Dir(files[0]))
	}

	t.Logf("Auto temp dir split: %d files", len(files))
}

func TestProcessSplit_NonExistentFile(t *testing.T) {
	config := map[string]interface{}{
		"pages": "1-2",
	}

	p, err := process.Of("pdf.split", "/non/existent/file.pdf", config)
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

// ==== Comprehensive Test ====

func TestProcessPDF_Comprehensive(t *testing.T) {
	ensureProcessTestData(t)

	testFile := getProcessTestPDFFile()

	// Step 1: Get info
	t.Run("Step 1: Get Info", func(t *testing.T) {
		p, err := process.Of("pdf.info", testFile)
		if err != nil {
			t.Fatalf("Failed to create process: %v", err)
		}

		result := p.Run()
		info, ok := result.(*Info)
		if !ok {
			t.Fatalf("Expected *Info, got %T", result)
		}

		t.Logf("PDF has %d pages, %d bytes", info.PageCount, info.FileSize)

		// Step 2: Convert first 2 pages
		t.Run("Step 2: Convert Pages", func(t *testing.T) {
			outputDir := filepath.Join(os.TempDir(), "gou_pdf_process_test_comprehensive")
			defer os.RemoveAll(outputDir)

			pages := "1-2"
			if info.PageCount < 2 {
				pages = "1"
			}

			config := map[string]interface{}{
				"format":     "png",
				"dpi":        100,
				"pages":      pages,
				"output_dir": outputDir,
			}

			p, err := process.Of("pdf.convert", testFile, config)
			if err != nil {
				t.Fatalf("Failed to create process: %v", err)
			}

			result := p.Run()
			files, ok := result.([]string)
			if !ok {
				t.Fatalf("Expected []string, got %T", result)
			}

			t.Logf("Converted %d pages to PNG", len(files))

			// Verify each file
			for _, f := range files {
				fi, err := os.Stat(f)
				if err != nil {
					t.Errorf("Output file not found: %s", f)
					continue
				}
				t.Logf("  %s (%d bytes)", filepath.Base(f), fi.Size())
			}
		})
	})
}
