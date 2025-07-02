package converter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// ==== Test Data Utils ====

// getTestDataDir returns the test data directory based on runtime path
func getTestDataDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	// Get directory of current test file
	currentDir := filepath.Dir(currentFile)
	// Navigate to test data directory: ../tests/converter/utf8
	testDataDir := filepath.Join(currentDir, "..", "tests", "converter", "utf8")
	absPath, err := filepath.Abs(testDataDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to get absolute path for test data dir: %v", err))
	}
	return absPath
}

// getTestFilePath returns the full path to a test file
func getTestFilePath(filename string) string {
	return filepath.Join(getTestDataDir(), filename)
}

// ensureTestDataExists checks if test data directory and files exist
func ensureTestDataExists(t *testing.T) {
	t.Helper()

	testDir := getTestDataDir()
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatalf("Test data directory does not exist: %s", testDir)
	}

	// Check for required test files
	requiredFiles := []string{
		"test.utf8.txt", "test.utf8.txt.gz",
		"test.gbk.txt", "test.gbk.txt.gz",
		"test.latin1.txt", "test.latin1.txt.gz",
		"test.big5.txt", "test.big5.txt.gz",
		"test.shiftjis.txt", "test.shiftjis.txt.gz",
		"test.utf8bom.txt", "test.utf8bom.txt.gz",
		"test.empty.txt", "test.empty.txt.gz",
		"test.large.txt", "test.large.txt.gz",
		"test.binary.dat", "test.binary.dat.gz",
		"test.image.png", "test.image.png.gz",
	}

	for _, filename := range requiredFiles {
		filePath := getTestFilePath(filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatalf("Required test file does not exist: %s", filePath)
		}
	}
}

// TestFileInfo contains information about a test file
type TestFileInfo struct {
	Name         string
	Path         string
	ExpectedUTF8 string
	ShouldFail   bool
	Encoding     string
	Description  string
}

// getTextTestFiles returns all text test files that should convert successfully
func getTextTestFiles() []TestFileInfo {
	return []TestFileInfo{
		{
			Name:        "test.utf8.txt",
			Path:        getTestFilePath("test.utf8.txt"),
			Encoding:    "UTF-8",
			Description: "Standard UTF-8 file (fast path)",
		},
		{
			Name:        "test.gbk.txt",
			Path:        getTestFilePath("test.gbk.txt"),
			Encoding:    "GBK",
			Description: "GBK encoded Chinese text",
		},
		{
			Name:        "test.latin1.txt",
			Path:        getTestFilePath("test.latin1.txt"),
			Encoding:    "ISO-8859-1",
			Description: "Latin1 encoded text",
		},
		{
			Name:        "test.big5.txt",
			Path:        getTestFilePath("test.big5.txt"),
			Encoding:    "Big5",
			Description: "Big5 encoded Traditional Chinese",
		},
		{
			Name:        "test.shiftjis.txt",
			Path:        getTestFilePath("test.shiftjis.txt"),
			ShouldFail:  true, // Shift-JIS contains many control chars that trigger binary detection
			Encoding:    "Shift-JIS",
			Description: "Shift-JIS encoded Japanese (fails binary detection)",
		},
		{
			Name:        "test.utf8bom.txt",
			Path:        getTestFilePath("test.utf8bom.txt"),
			Encoding:    "UTF-8-BOM",
			Description: "UTF-8 with BOM",
		},
		{
			Name:        "test.empty.txt",
			Path:        getTestFilePath("test.empty.txt"),
			Encoding:    "Empty",
			Description: "Empty file",
		},
		{
			Name:        "test.large.txt",
			Path:        getTestFilePath("test.large.txt"),
			Encoding:    "UTF-8",
			Description: "Large UTF-8 file for performance testing",
		},
		{
			Name:        "test.binary.dat",
			Path:        getTestFilePath("test.binary.dat"),
			Encoding:    "Binary",
			Description: "Random binary data (may pass as text)",
		},
	}
}

// getGzipTestFiles returns all gzip test files that should convert successfully
func getGzipTestFiles() []TestFileInfo {
	return []TestFileInfo{
		{
			Name:        "test.utf8.txt.gz",
			Path:        getTestFilePath("test.utf8.txt.gz"),
			Encoding:    "UTF-8-gzip",
			Description: "Gzipped UTF-8 file",
		},
		{
			Name:        "test.gbk.txt.gz",
			Path:        getTestFilePath("test.gbk.txt.gz"),
			Encoding:    "GBK-gzip",
			Description: "Gzipped GBK file",
		},
		{
			Name:        "test.latin1.txt.gz",
			Path:        getTestFilePath("test.latin1.txt.gz"),
			Encoding:    "ISO-8859-1-gzip",
			Description: "Gzipped Latin1 file",
		},
		{
			Name:        "test.big5.txt.gz",
			Path:        getTestFilePath("test.big5.txt.gz"),
			Encoding:    "Big5-gzip",
			Description: "Gzipped Big5 file",
		},
		{
			Name:        "test.shiftjis.txt.gz",
			Path:        getTestFilePath("test.shiftjis.txt.gz"),
			ShouldFail:  true, // Shift-JIS contains many control chars that trigger binary detection
			Encoding:    "Shift-JIS-gzip",
			Description: "Gzipped Shift-JIS file (fails binary detection)",
		},
		{
			Name:        "test.utf8bom.txt.gz",
			Path:        getTestFilePath("test.utf8bom.txt.gz"),
			Encoding:    "UTF-8-BOM-gzip",
			Description: "Gzipped UTF-8 with BOM",
		},
		{
			Name:        "test.empty.txt.gz",
			Path:        getTestFilePath("test.empty.txt.gz"),
			Encoding:    "Empty-gzip",
			Description: "Gzipped empty file",
		},
		{
			Name:        "test.large.txt.gz",
			Path:        getTestFilePath("test.large.txt.gz"),
			Encoding:    "UTF-8-gzip",
			Description: "Gzipped large file",
		},
		{
			Name:        "test.binary.dat.gz",
			Path:        getTestFilePath("test.binary.dat.gz"),
			Encoding:    "Binary-gzip",
			Description: "Gzipped binary data (may pass as text)",
		},
	}
}

// getBinaryTestFiles returns all binary test files that should fail
func getBinaryTestFiles() []TestFileInfo {
	return []TestFileInfo{
		{
			Name:        "test.image.png",
			Path:        getTestFilePath("test.image.png"),
			ShouldFail:  true,
			Encoding:    "PNG",
			Description: "PNG image file",
		},
		{
			Name:        "test.image.png.gz",
			Path:        getTestFilePath("test.image.png.gz"),
			ShouldFail:  true,
			Encoding:    "PNG-gzip",
			Description: "Gzipped PNG image",
		},
	}
}

// getAllTestFiles returns all test files
func getAllTestFiles() []TestFileInfo {
	var all []TestFileInfo
	all = append(all, getTextTestFiles()...)
	all = append(all, getGzipTestFiles()...)
	all = append(all, getBinaryTestFiles()...)
	return all
}

// ==== Concurrency and Leak Detection Utils ====

// GoroutineInfo represents information about a goroutine
type GoroutineInfo struct {
	ID       int    // Goroutine ID
	State    string // Goroutine state (running, select, IO wait, etc.)
	Function string // Goroutine function
	Stack    string // Full stack trace
	IsSystem bool   // Whether it's a system goroutine
}

// MemoryStats represents memory statistics for leak detection
type MemoryStats struct {
	Alloc      uint64
	HeapAlloc  uint64
	Sys        uint64
	NumGC      uint32
	TotalAlloc uint64
}

// MemoryGrowth calculates memory growth between two memory stats
type MemoryGrowth struct {
	AllocGrowth      int64
	HeapAllocGrowth  int64
	SysGrowth        int64
	NumGCDiff        uint32
	TotalAllocGrowth uint64
}

// StressTestConfig defines configuration for stress tests
type StressTestConfig struct {
	NumWorkers          int
	OperationsPerWorker int
	Timeout             time.Duration
	MinSuccessRate      float64
}

// StressTestResult represents the result of a stress test
type StressTestResult struct {
	TotalOperations int
	ErrorCount      int
	SuccessRate     float64
	Duration        time.Duration
	Errors          []error
}

// TestOperation represents a single test operation
type TestOperation func(ctx context.Context) error

// LeakDetectionResult represents the result of leak detection
type LeakDetectionResult struct {
	MemoryGrowth     MemoryGrowth
	GoroutineLeaks   []GoroutineInfo
	HasMemoryLeak    bool
	HasGoroutineLeak bool
}

// captureGoroutineState captures the current goroutine state for leak detection
func captureGoroutineState() []GoroutineInfo {
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // Allow any pending operations to complete
	stack := make([]byte, 1024*1024)
	stack = stack[:runtime.Stack(stack, true)]
	return parseGoroutines(stack)
}

// parseGoroutines parses the output of runtime.Stack and returns goroutine information
func parseGoroutines(stackTrace []byte) []GoroutineInfo {
	lines := strings.Split(string(stackTrace), "\n")
	var goroutines []GoroutineInfo
	var current *GoroutineInfo

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a goroutine header line
		if strings.HasPrefix(line, "goroutine ") {
			// Save previous goroutine if exists
			if current != nil {
				goroutines = append(goroutines, *current)
			}

			// Parse goroutine header: "goroutine 1 [running]:"
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				current = &GoroutineInfo{}
				if id, err := strconv.Atoi(parts[1]); err == nil {
					current.ID = id
				}
				// Extract state from [state]:
				state := strings.Trim(parts[2], "[]:")
				current.State = state
			}
		} else if current != nil && current.Function == "" {
			// This should be the function line
			current.Function = line
			// Build stack trace
			stackLines := []string{line}
			// Look ahead for more stack lines
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" || strings.HasPrefix(nextLine, "goroutine ") {
					break
				}
				stackLines = append(stackLines, nextLine)
			}
			current.Stack = strings.Join(stackLines, "\n")

			// Determine if this is a system goroutine
			current.IsSystem = isSystemGoroutine(current.Function, current.Stack)
		}
	}

	// Don't forget the last goroutine
	if current != nil {
		goroutines = append(goroutines, *current)
	}

	return goroutines
}

// isSystemGoroutine determines if a goroutine is a system goroutine based on its function and stack
func isSystemGoroutine(function, stack string) bool {
	systemPatterns := []string{
		"runtime.",            // Go runtime
		"testing.",            // Test framework
		"os/signal.",          // Signal handling
		"net/http.(*Server).", // HTTP server
		"net/http.(*conn).",   // HTTP server connection
		"net.(*netFD).",       // Network file descriptor
		"internal/poll.",      // Network polling
		"crypto/tls.",         // TLS operations
	}

	// Check function name
	for _, pattern := range systemPatterns {
		if strings.Contains(function, pattern) {
			return true
		}
	}

	// Check stack trace
	for _, pattern := range systemPatterns {
		if strings.Contains(stack, pattern) {
			return true
		}
	}

	return false
}

// captureMemoryStats captures current memory statistics
func captureMemoryStats() MemoryStats {
	runtime.GC()
	runtime.GC() // Double GC to ensure clean state

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return MemoryStats{
		Alloc:      m.Alloc,
		HeapAlloc:  m.HeapAlloc,
		Sys:        m.Sys,
		NumGC:      m.NumGC,
		TotalAlloc: m.TotalAlloc,
	}
}

// calculateMemoryGrowth calculates memory growth between before and after stats
func calculateMemoryGrowth(before, after MemoryStats) MemoryGrowth {
	return MemoryGrowth{
		AllocGrowth:      int64(after.Alloc) - int64(before.Alloc),
		HeapAllocGrowth:  int64(after.HeapAlloc) - int64(before.HeapAlloc),
		SysGrowth:        int64(after.Sys) - int64(before.Sys),
		NumGCDiff:        after.NumGC - before.NumGC,
		TotalAllocGrowth: after.TotalAlloc - before.TotalAlloc,
	}
}

// analyzeGoroutineChanges compares before and after goroutine states
func analyzeGoroutineChanges(before, after []GoroutineInfo) (leaked, cleaned []GoroutineInfo) {
	beforeMap := make(map[int]GoroutineInfo)
	for _, g := range before {
		beforeMap[g.ID] = g
	}

	afterMap := make(map[int]GoroutineInfo)
	for _, g := range after {
		afterMap[g.ID] = g
	}

	// Find leaked goroutines (new in after)
	for id, g := range afterMap {
		if _, exists := beforeMap[id]; !exists {
			leaked = append(leaked, g)
		}
	}

	// Find cleaned goroutines (removed from before)
	for id, g := range beforeMap {
		if _, exists := afterMap[id]; !exists {
			cleaned = append(cleaned, g)
		}
	}

	return leaked, cleaned
}

// DefaultStressConfig returns default stress test configuration
func DefaultStressConfig() StressTestConfig {
	return StressTestConfig{
		NumWorkers:          10,
		OperationsPerWorker: 20,
		Timeout:             3 * time.Second,
		MinSuccessRate:      95.0,
	}
}

// LightStressConfig returns a lighter stress test configuration for CI
func LightStressConfig() StressTestConfig {
	return StressTestConfig{
		NumWorkers:          5,
		OperationsPerWorker: 10,
		Timeout:             2 * time.Second,
		MinSuccessRate:      90.0,
	}
}

// runStressTest runs a stress test with the given configuration and operation
func runStressTest(config StressTestConfig, operation TestOperation) StressTestResult {
	start := time.Now()

	errorsChan := make(chan error, config.NumWorkers*config.OperationsPerWorker)
	done := make(chan bool, config.NumWorkers)

	// Start workers
	for i := 0; i < config.NumWorkers; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			for j := 0; j < config.OperationsPerWorker; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
				err := operation(ctx)
				cancel()

				if err != nil {
					errorsChan <- err
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	for i := 0; i < config.NumWorkers; i++ {
		<-done
	}
	close(errorsChan)

	// Count errors
	var errors []error
	for err := range errorsChan {
		errors = append(errors, err)
	}

	totalOperations := config.NumWorkers * config.OperationsPerWorker
	errorCount := len(errors)
	successRate := float64(totalOperations-errorCount) / float64(totalOperations) * 100

	return StressTestResult{
		TotalOperations: totalOperations,
		ErrorCount:      errorCount,
		SuccessRate:     successRate,
		Duration:        time.Since(start),
		Errors:          errors,
	}
}

// runWithLeakDetection runs a test operation with memory and goroutine leak detection
func runWithLeakDetection(t *testing.T, operation func() error) LeakDetectionResult {
	t.Helper()

	// Capture initial state
	beforeMem := captureMemoryStats()
	beforeGoroutines := captureGoroutineState()

	// Run the operation
	err := operation()
	if err != nil {
		t.Errorf("Operation failed: %v", err)
	}

	// Allow some time for cleanup
	time.Sleep(200 * time.Millisecond)

	// Capture final state
	afterMem := captureMemoryStats()
	afterGoroutines := captureGoroutineState()

	// Analyze results
	memoryGrowth := calculateMemoryGrowth(beforeMem, afterMem)
	leakedGoroutines, _ := analyzeGoroutineChanges(beforeGoroutines, afterGoroutines)

	// Filter out system goroutines from leaks
	var realLeaks []GoroutineInfo
	for _, g := range leakedGoroutines {
		if !g.IsSystem {
			realLeaks = append(realLeaks, g)
		}
	}

	// Determine if there are actual leaks
	hasMemoryLeak := memoryGrowth.AllocGrowth > 1024*1024 // 1MB threshold
	hasGoroutineLeak := len(realLeaks) > 0

	return LeakDetectionResult{
		MemoryGrowth:     memoryGrowth,
		GoroutineLeaks:   realLeaks,
		HasMemoryLeak:    hasMemoryLeak,
		HasGoroutineLeak: hasGoroutineLeak,
	}
}

// runConcurrentStressWithLeakDetection runs concurrent stress test with leak detection
func runConcurrentStressWithLeakDetection(t *testing.T, config StressTestConfig, operation TestOperation) (StressTestResult, LeakDetectionResult) {
	t.Helper()

	// Capture initial state
	beforeMem := captureMemoryStats()
	beforeGoroutines := captureGoroutineState()

	// Run stress test
	result := runStressTest(config, operation)

	// Allow some time for cleanup
	time.Sleep(500 * time.Millisecond)

	// Capture final state
	afterMem := captureMemoryStats()
	afterGoroutines := captureGoroutineState()

	// Analyze results
	memoryGrowth := calculateMemoryGrowth(beforeMem, afterMem)
	leakedGoroutines, _ := analyzeGoroutineChanges(beforeGoroutines, afterGoroutines)

	// Filter out system goroutines from leaks
	var realLeaks []GoroutineInfo
	for _, g := range leakedGoroutines {
		if !g.IsSystem {
			realLeaks = append(realLeaks, g)
		}
	}

	// Determine if there are actual leaks
	hasMemoryLeak := memoryGrowth.AllocGrowth > 2*1024*1024 // 2MB threshold for stress tests
	hasGoroutineLeak := len(realLeaks) > 0

	leakResult := LeakDetectionResult{
		MemoryGrowth:     memoryGrowth,
		GoroutineLeaks:   realLeaks,
		HasMemoryLeak:    hasMemoryLeak,
		HasGoroutineLeak: hasGoroutineLeak,
	}

	return result, leakResult
}

// assertStressTestResult asserts that a stress test result meets expectations
func assertStressTestResult(t *testing.T, result StressTestResult, config StressTestConfig, testName string) {
	t.Helper()

	if result.SuccessRate < config.MinSuccessRate {
		t.Errorf("%s: Success rate %.2f%% is below minimum %.2f%%",
			testName, result.SuccessRate, config.MinSuccessRate)

		// Log first few errors for debugging
		if len(result.Errors) > 0 {
			t.Logf("First few errors from %s:", testName)
			for i, err := range result.Errors {
				if i >= 5 { // Limit to first 5 errors
					break
				}
				t.Logf("  Error %d: %v", i+1, err)
			}
		}
	}

	t.Logf("%s: %d operations, %.2f%% success rate, %d errors, duration: %v",
		testName, result.TotalOperations, result.SuccessRate, result.ErrorCount, result.Duration)
}

// assertNoLeaks asserts that no significant leaks were detected
func assertNoLeaks(t *testing.T, result LeakDetectionResult, testName string) {
	t.Helper()

	if result.HasMemoryLeak {
		t.Errorf("%s: Memory leak detected - Alloc growth: %d bytes, Heap growth: %d bytes",
			testName, result.MemoryGrowth.AllocGrowth, result.MemoryGrowth.HeapAllocGrowth)
	}

	if result.HasGoroutineLeak {
		t.Errorf("%s: Goroutine leak detected - %d leaked goroutines",
			testName, len(result.GoroutineLeaks))

		// Log details of leaked goroutines
		for i, g := range result.GoroutineLeaks {
			if i >= 3 { // Limit to first 3 goroutines
				break
			}
			t.Logf("  Leaked goroutine %d: ID=%d, State=%s, Function=%s",
				i+1, g.ID, g.State, g.Function)
		}
	}

	if !result.HasMemoryLeak && !result.HasGoroutineLeak {
		t.Logf("%s: No significant leaks detected - Alloc growth: %d bytes, Goroutines: %d",
			testName, result.MemoryGrowth.AllocGrowth, len(result.GoroutineLeaks))
	}
}

// ==== Progress Callback Utils ====

// TestProgressCallback is a helper for testing progress callbacks
type TestProgressCallback struct {
	Calls    []types.ConverterPayload
	LastCall *types.ConverterPayload
}

// NewTestProgressCallback creates a new test progress callback
func NewTestProgressCallback() *TestProgressCallback {
	return &TestProgressCallback{
		Calls: make([]types.ConverterPayload, 0),
	}
}

// Callback implements the ConverterProgress interface
func (tpc *TestProgressCallback) Callback(status types.ConverterStatus, payload types.ConverterPayload) {
	tpc.Calls = append(tpc.Calls, payload)
	tpc.LastCall = &payload
}

// GetCallCount returns the number of callback calls
func (tpc *TestProgressCallback) GetCallCount() int {
	return len(tpc.Calls)
}

// GetLastStatus returns the last callback status
func (tpc *TestProgressCallback) GetLastStatus() types.ConverterStatus {
	if tpc.LastCall != nil {
		return tpc.LastCall.Status
	}
	return types.ConverterStatusPending
}

// GetLastProgress returns the last progress value
func (tpc *TestProgressCallback) GetLastProgress() float64 {
	if tpc.LastCall != nil {
		return tpc.LastCall.Progress
	}
	return 0.0
}

// Reset clears all callback data
func (tpc *TestProgressCallback) Reset() {
	tpc.Calls = tpc.Calls[:0]
	tpc.LastCall = nil
}

// ==== Vision-Specific Test Utils ====

// VisionTestImageInfo represents test image metadata
type VisionTestImageInfo struct {
	Width        int
	Height       int
	Format       string
	FileSize     int64
	ExpectedMIME string
	HasAnimation bool
	ColorDepth   int
}

// VisionTestStats collects statistics during vision testing
type VisionTestStats struct {
	TotalFiles      int
	SuccessfulFiles int
	FailedFiles     int
	TotalSize       int64
	ProcessedSize   int64
	AverageProgress float64
	ProcessingTime  time.Duration
}

// NewVisionTestStats creates a new vision test statistics collector
func NewVisionTestStats() *VisionTestStats {
	return &VisionTestStats{}
}

// RecordFileResult records the result of processing a file
func (vts *VisionTestStats) RecordFileResult(fileSize int64, success bool, progressCalls int, duration time.Duration) {
	vts.TotalFiles++
	vts.TotalSize += fileSize
	vts.ProcessingTime += duration

	if success {
		vts.SuccessfulFiles++
		vts.ProcessedSize += fileSize
	} else {
		vts.FailedFiles++
	}

	// Update average progress (simplified calculation)
	vts.AverageProgress = float64(vts.SuccessfulFiles) / float64(vts.TotalFiles)
}

// GetSummary returns a summary string of the test statistics
func (vts *VisionTestStats) GetSummary() string {
	successRate := float64(vts.SuccessfulFiles) / float64(vts.TotalFiles) * 100
	return fmt.Sprintf("Vision Test Summary: %d files, %.1f%% success, %d bytes processed, %v total time",
		vts.TotalFiles, successRate, vts.ProcessedSize, vts.ProcessingTime)
}

// validateVisionResult validates that a vision conversion result is reasonable
func validateVisionResult(result string, minLength int, expectedKeywords []string) error {
	if len(result) < minLength {
		return fmt.Errorf("result too short: %d characters (minimum %d)", len(result), minLength)
	}

	resultLower := strings.ToLower(result)

	// Check for basic description keywords
	basicKeywords := []string{"image", "photo", "picture", "shows", "contains", "depicts"}
	hasBasicKeyword := false
	for _, keyword := range basicKeywords {
		if strings.Contains(resultLower, keyword) {
			hasBasicKeyword = true
			break
		}
	}

	if !hasBasicKeyword {
		return fmt.Errorf("result lacks basic image description keywords")
	}

	// Check for expected keywords if provided
	if len(expectedKeywords) > 0 {
		foundKeywords := 0
		for _, keyword := range expectedKeywords {
			if strings.Contains(resultLower, strings.ToLower(keyword)) {
				foundKeywords++
			}
		}

		if foundKeywords == 0 {
			return fmt.Errorf("result lacks any expected keywords: %v", expectedKeywords)
		}
	}

	return nil
}

// runVisionStressTest runs a specialized stress test for vision operations
func runVisionStressTest(t *testing.T, config StressTestConfig, imageFiles []string, visionOperation func(string) error) *VisionTestStats {
	t.Helper()

	stats := NewVisionTestStats()
	start := time.Now()

	errorsChan := make(chan error, config.NumWorkers*config.OperationsPerWorker)
	done := make(chan bool, config.NumWorkers)

	// Start workers
	for i := 0; i < config.NumWorkers; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			for j := 0; j < config.OperationsPerWorker; j++ {
				// Pick a file for this operation
				fileIndex := (workerID*config.OperationsPerWorker + j) % len(imageFiles)
				imageFile := imageFiles[fileIndex]

				operationStart := time.Now()
				err := visionOperation(imageFile)
				operationDuration := time.Since(operationStart)

				// Get file size
				var fileSize int64
				if fileInfo, statErr := os.Stat(imageFile); statErr == nil {
					fileSize = fileInfo.Size()
				}

				// Record result
				stats.RecordFileResult(fileSize, err == nil, 1, operationDuration)

				if err != nil {
					errorsChan <- err
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	for i := 0; i < config.NumWorkers; i++ {
		<-done
	}
	close(errorsChan)

	// Collect errors
	var errors []error
	for err := range errorsChan {
		errors = append(errors, err)
	}

	totalTime := time.Since(start)

	t.Logf("Vision stress test completed: %s, %d errors in %v",
		stats.GetSummary(), len(errors), totalTime)

	// Log some errors if any
	if len(errors) > 0 && len(errors) <= 5 {
		for i, err := range errors {
			t.Logf("  Error %d: %v", i+1, err)
		}
	} else if len(errors) > 5 {
		t.Logf("  First 3 errors:")
		for i := 0; i < 3; i++ {
			t.Logf("    Error %d: %v", i+1, errors[i])
		}
		t.Logf("  ... and %d more errors", len(errors)-3)
	}

	return stats
}
