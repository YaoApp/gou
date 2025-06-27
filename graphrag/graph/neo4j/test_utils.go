package neo4j

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// GoroutineInfo represents information about a goroutine
type GoroutineInfo struct {
	ID       int    // 协程ID
	State    string // 协程状态 (running, select, IO wait, etc.)
	Function string // 协程执行的函数
	Stack    string // 完整堆栈跟踪
	IsSystem bool   // 是否为系统协程
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
		"runtime.",                         // Go 运行时
		"testing.",                         // 测试框架
		"os/signal.",                       // 信号处理
		"net/http.(*Server).",              // HTTP 服务器
		"net/http.(*conn).",                // HTTP 服务器连接
		"net.(*netFD).",                    // 网络文件描述符
		"internal/poll.",                   // 网络轮询
		"crypto/tls.",                      // TLS 操作
		"go.opencensus.io",                 // OpenCensus
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

	// Application goroutines that we need to monitor for leaks
	clientPatterns := []string{
		"net/http.(*persistConn).", // HTTP 客户端持久连接
		"net/http.(*Transport).",   // HTTP 客户端传输
	}

	for _, pattern := range clientPatterns {
		if strings.Contains(function, pattern) || strings.Contains(stack, pattern) {
			return false // These are application goroutines we care about
		}
	}

	return false // Unknown goroutines are considered application goroutines
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

// captureGoroutineState captures the current goroutine state for leak detection
func captureGoroutineState() []GoroutineInfo {
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // Allow any pending operations to complete
	stack := make([]byte, 1024*1024)
	stack = stack[:runtime.Stack(stack, true)]
	return parseGoroutines(stack)
}

// MemoryStats represents memory statistics for leak detection
type MemoryStats struct {
	Alloc      uint64
	HeapAlloc  uint64
	Sys        uint64
	NumGC      uint32
	TotalAlloc uint64
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

// MemoryGrowth calculates memory growth between two memory stats
type MemoryGrowth struct {
	AllocGrowth      int64
	HeapAllocGrowth  int64
	SysGrowth        int64
	NumGCDiff        uint32
	TotalAllocGrowth uint64
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

// StressTestConfig defines configuration for stress tests
type StressTestConfig struct {
	NumWorkers          int
	OperationsPerWorker int
	Timeout             time.Duration
	MinSuccessRate      float64
}

// DefaultStressConfig returns default stress test configuration
func DefaultStressConfig() StressTestConfig {
	return StressTestConfig{
		NumWorkers:          50,
		OperationsPerWorker: 100,
		Timeout:             5 * time.Second,
		MinSuccessRate:      95.0,
	}
}

// LightStressConfig returns a lighter stress test configuration for CI
func LightStressConfig() StressTestConfig {
	return StressTestConfig{
		NumWorkers:          10,
		OperationsPerWorker: 20,
		Timeout:             3 * time.Second,
		MinSuccessRate:      95.0,
	}
}

// StressTestResult represents the result of a stress test
type StressTestResult struct {
	TotalOperations int
	ErrorCount      int
	SuccessRate     float64
	Duration        time.Duration
}

// TestOperation represents a single test operation
type TestOperation func(ctx context.Context) error

// runStressTest runs a stress test with the given configuration and operation
func runStressTest(config StressTestConfig, operation TestOperation) StressTestResult {
	start := time.Now()

	errors := make(chan error, config.NumWorkers*config.OperationsPerWorker)
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
					errors <- err
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	for i := 0; i < config.NumWorkers; i++ {
		<-done
	}
	close(errors)

	// Count errors
	errorCount := 0
	for range errors {
		errorCount++
	}

	totalOperations := config.NumWorkers * config.OperationsPerWorker
	successRate := float64(totalOperations-errorCount) / float64(totalOperations) * 100

	return StressTestResult{
		TotalOperations: totalOperations,
		ErrorCount:      errorCount,
		SuccessRate:     successRate,
		Duration:        time.Since(start),
	}
}
