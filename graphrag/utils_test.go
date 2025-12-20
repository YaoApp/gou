package graphrag

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/graph/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/vector/qdrant"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/log"
)

// ==== Test Utils ====

func GetTestConfigs() map[string]*Config {
	vectorStore := qdrant.NewStore()

	// Create graph store with proper configuration and connect
	graphStoreConfig := getGraphStore("test")
	graphStore := neo4j.NewStoreWithConfig(graphStoreConfig)

	// Auto-connect graph store
	ctx := context.Background()
	if err := graphStore.Connect(ctx); err != nil {
		// Log warning but don't fail - tests will skip if not connected
		logger := log.StandardLogger()
		logger.Warnf("Failed to connect graph store: %v", err)
	}

	logger := log.StandardLogger()
	testStore := getRedisStore("store_redis", 6)

	return map[string]*Config{
		// Basic configurations
		"vector":        {Vector: vectorStore},
		"vector+graph":  {Graph: graphStore, Vector: vectorStore},
		"vector+store":  {Vector: vectorStore, Store: testStore},
		"vector+system": {Vector: vectorStore, System: "test_system"},
		"vector+logger": {Vector: vectorStore, Logger: logger},

		// Two component combinations
		"vector+graph+store":   {Vector: vectorStore, Graph: graphStore, Store: testStore},
		"vector+graph+logger":  {Vector: vectorStore, Graph: graphStore, Logger: logger},
		"vector+graph+system":  {Vector: vectorStore, Graph: graphStore, System: "test_system"},
		"vector+store+logger":  {Vector: vectorStore, Store: testStore, Logger: logger},
		"vector+store+system":  {Vector: vectorStore, Store: testStore, System: "test_system"},
		"vector+logger+system": {Vector: vectorStore, Logger: logger, System: "test_system"},

		// Three component combinations
		"vector+graph+store+logger":  {Vector: vectorStore, Graph: graphStore, Store: testStore, Logger: logger},
		"vector+graph+store+system":  {Vector: vectorStore, Graph: graphStore, Store: testStore, System: "test_system"},
		"vector+graph+logger+system": {Vector: vectorStore, Graph: graphStore, Logger: logger, System: "test_system"},
		"vector+store+logger+system": {Vector: vectorStore, Store: testStore, Logger: logger, System: "test_system"},

		// Complete configuration
		"complete": {Vector: vectorStore, Graph: graphStore, Store: testStore, Logger: logger, System: "custom_system"},

		// Edge cases
		"store+system": {Store: testStore, System: "test_system"},
		"graph+store":  {Graph: graphStore, Store: testStore},
		"invalid":      {},
	}
}

// getVectorStore returns the vector store for the given name
func getVectorStore(name string, dimension int) types.VectorStoreConfig {
	return types.VectorStoreConfig{
		ExtraParams: map[string]interface{}{
			"host": getEnvOrDefault("QDRANT_TEST_HOST", "localhost"),
			"port": getEnvOrDefault("QDRANT_TEST_PORT", "6334"),
		},
	}
}

func getGraphStore(name string) types.GraphStoreConfig {
	return types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: getEnvOrDefault("NEO4J_TEST_URL", "neo4j://localhost:7687"),
		DriverConfig: map[string]interface{}{
			"username": getEnvOrDefault("NEO4J_TEST_USER", "neo4j"),
			"password": getEnvOrDefault("NEO4J_TEST_PASS", "Yao2026Neo4j"),
		},
	}
}

func getRedisStore(name string, db int) store.Store {
	conn, err := getRedisStoreConnector(name, db)
	if err != nil {
		panic(err)
	}

	s, err := store.New(conn, nil)
	if err != nil {
		panic(err)
	}
	return s
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getOpenAIConnector(name, model string) (connector.Connector, error) {
	return connector.New("openai", fmt.Sprintf("test_%s", name), []byte(
		`{
			"label": "`+name+`",
			"options": {
				"key": "`+getEnvOrDefault("OPENAI_TEST_KEY", "")+`",
				"model": "`+model+`",
			}
		}`,
	))
}

func getDeepSeekConnector(name string) (connector.Connector, error) {
	model := getEnvOrDefault("RAG_LLM_TEST_SMODEL", "")
	return connector.New("openai", fmt.Sprintf("test_%s", name), []byte(
		`{
			"label": "`+name+`",
			"options": {
				"key": "`+getEnvOrDefault("RAG_LLM_TEST_KEY", "")+`",
				"model": "`+model+`",
				"proxy": "`+getEnvOrDefault("RAG_LLM_TEST_URL", "")+`"
			}
		}`,
	))
}

func getRedisStoreConnector(name string, db int) (connector.Connector, error) {
	return connector.New("redis", fmt.Sprintf("test_%s", name), []byte(
		`{
			"label": "`+name+fmt.Sprintf("_%d", db)+`", 
			"type": "redis",
			"options": {
				"host": "`+getEnvOrDefault("REDIS_TEST_HOST", "localhost")+`",
				"port": "`+getEnvOrDefault("REDIS_TEST_PORT", "6379")+`",
				"pass": "`+getEnvOrDefault("REDIS_TEST_PASS", "")+`",
				"db": "`+getEnvOrDefault("REDIS_TEST_RAG_DB", fmt.Sprintf("%d", db))+`"
			}
		}`,
	))
}

// ==== Concurrent and Leak Detection Utils ====

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
		"runtime.",                         // Go runtime
		"testing.",                         // Test framework
		"os/signal.",                       // Signal handling
		"net/http.(*Server).",              // HTTP server
		"net/http.(*conn).",                // HTTP server connection
		"net.(*netFD).",                    // Network file descriptor
		"internal/poll.",                   // Network polling
		"crypto/tls.",                      // TLS operations
		"go.opencensus.io",                 // OpenCensus
		"google.golang.org/grpc",           // gRPC
		"github.com/qdrant/go-client",      // Qdrant client internals
		"github.com/neo4j/neo4j-go-driver", // Neo4j driver internals
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
		OperationsPerWorker: 50,
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

// HeavyStressConfig returns a heavy stress test configuration
func HeavyStressConfig() StressTestConfig {
	return StressTestConfig{
		NumWorkers:          50,
		OperationsPerWorker: 100,
		Timeout:             5 * time.Second,
		MinSuccessRate:      85.0,
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
	hasMemoryLeak := memoryGrowth.AllocGrowth > 5*1024*1024 // 5MB threshold for stress tests
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
