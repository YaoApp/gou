package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
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

// SeparateDBStressConfig returns stress test configuration for separate database mode
// Uses very conservative settings to avoid hitting database limits
func SeparateDBStressConfig() StressTestConfig {
	return StressTestConfig{
		NumWorkers:          1,                // Single worker to avoid database limit
		OperationsPerWorker: 2,                // Even fewer operations to stay within database limit
		Timeout:             30 * time.Second, // Longer timeout for database operations
		MinSuccessRate:      50.0,             // Lower success rate tolerance due to database limits
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

// TestGraphData represents test graph data structure
type TestGraphData struct {
	Entities           []TestEntity       `json:"entities"`
	Relationships      []TestRelationship `json:"relationships"`
	File               string             `json:"file"`
	FullPath           string             `json:"full_path"`
	GeneratedAt        string             `json:"generated_at"`
	Model              string             `json:"model"`
	TextLength         int                `json:"text_length"`
	TotalEntities      int                `json:"total_entities"`
	TotalRelationships int                `json:"total_relationships"`
	Usage              TestUsage          `json:"usage"`
}

// TestEntity represents an entity in test data
type TestEntity struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	Labels           []string               `json:"labels"`
	Properties       map[string]interface{} `json:"properties"`
	Description      string                 `json:"description"`
	Confidence       float64                `json:"confidence"`
	ExtractionMethod string                 `json:"extraction_method"`
	CreatedAt        int64                  `json:"created_at"`
	Version          int                    `json:"version"`
	Status           string                 `json:"status"`
}

// TestRelationship represents a relationship in test data
type TestRelationship struct {
	Type             string                 `json:"type"`
	StartNode        string                 `json:"start_node"`
	EndNode          string                 `json:"end_node"`
	Properties       map[string]interface{} `json:"properties"`
	Description      string                 `json:"description"`
	Confidence       float64                `json:"confidence"`
	Weight           float64                `json:"weight"`
	ExtractionMethod string                 `json:"extraction_method"`
	CreatedAt        int64                  `json:"created_at"`
	Version          int                    `json:"version"`
	Status           string                 `json:"status"`
}

// TestUsage represents usage statistics in test data
type TestUsage struct {
	TotalTokens  int `json:"total_tokens"`
	PromptTokens int `json:"prompt_tokens"`
	TotalTexts   int `json:"total_texts"`
}

// LoadTestGraphDataFromDir loads all graph test data files from a directory
func LoadTestGraphDataFromDir(dirPath string) ([]*TestGraphData, error) {
	var allData []*TestGraphData

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Only process .graph.json files
		if !strings.HasSuffix(d.Name(), ".graph.json") {
			return nil
		}

		data, err := LoadTestGraphDataFromFile(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		allData = append(allData, data)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
	}

	return allData, nil
}

// LoadTestGraphDataFromFile loads graph test data from a single file
func LoadTestGraphDataFromFile(filePath string) (*TestGraphData, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var data TestGraphData
	err = json.Unmarshal(content, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", filePath, err)
	}

	return &data, nil
}

// ConvertTestEntitiesToGraphNodes converts test entities to GraphNode format
func ConvertTestEntitiesToGraphNodes(entities []TestEntity) []*types.GraphNode {
	nodes := make([]*types.GraphNode, len(entities))

	for i, entity := range entities {
		node := &types.GraphNode{
			ID:          entity.ID,
			Labels:      entity.Labels,
			Properties:  entity.Properties,
			EntityType:  entity.Type,
			Description: entity.Description,
			Confidence:  entity.Confidence,
			CreatedAt:   time.Unix(entity.CreatedAt, 0),
			Version:     entity.Version,
		}

		if node.Properties == nil {
			node.Properties = make(map[string]interface{})
		}

		// Add entity name as a property if not already present
		if _, exists := node.Properties["name"]; !exists && entity.Name != "" {
			node.Properties["name"] = entity.Name
		}

		nodes[i] = node
	}

	return nodes
}

// ConvertTestRelationshipsToGraphRelationships converts test relationships to GraphRelationship format
func ConvertTestRelationshipsToGraphRelationships(relationships []TestRelationship) []*types.GraphRelationship {
	rels := make([]*types.GraphRelationship, len(relationships))

	for i, rel := range relationships {
		graphRel := &types.GraphRelationship{
			ID:          fmt.Sprintf("%s_%s_%s", rel.StartNode, rel.Type, rel.EndNode),
			Type:        rel.Type,
			StartNode:   rel.StartNode,
			EndNode:     rel.EndNode,
			Properties:  rel.Properties,
			Description: rel.Description,
			Confidence:  rel.Confidence,
			Weight:      rel.Weight,
			CreatedAt:   time.Unix(rel.CreatedAt, 0),
			Version:     rel.Version,
		}

		if graphRel.Properties == nil {
			graphRel.Properties = make(map[string]interface{})
		}

		rels[i] = graphRel
	}

	return rels
}

// LoadTestDataset loads test dataset for the specified language and returns nodes and relationships
func LoadTestDataset(language string) ([]*types.GraphNode, []*types.GraphRelationship, error) {
	var dirPath string
	if language == "zh" {
		dirPath = "/Users/max/Yao/gou/graphrag/tests/semantic-zh"
	} else if language == "en" {
		dirPath = "/Users/max/Yao/gou/graphrag/tests/semantic-en"
	} else {
		return nil, nil, fmt.Errorf("unsupported language: %s (supported: zh, en)", language)
	}

	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("test data directory does not exist: %s", dirPath)
	}

	// Load all test data files
	allData, err := LoadTestGraphDataFromDir(dirPath)
	if err != nil {
		return nil, nil, err
	}

	if len(allData) == 0 {
		return nil, nil, fmt.Errorf("no test data files found in %s", dirPath)
	}

	// Aggregate all entities and relationships
	var allEntities []TestEntity
	var allRelationships []TestRelationship

	for _, data := range allData {
		allEntities = append(allEntities, data.Entities...)
		allRelationships = append(allRelationships, data.Relationships...)
	}

	// Convert to GraphNode and GraphRelationship format
	nodes := ConvertTestEntitiesToGraphNodes(allEntities)
	relationships := ConvertTestRelationshipsToGraphRelationships(allRelationships)

	return nodes, relationships, nil
}

// CreateTestNodes creates a set of test nodes for unit testing
func CreateTestNodes(count int) []*types.GraphNode {
	nodes := make([]*types.GraphNode, count)

	for i := 0; i < count; i++ {
		node := &types.GraphNode{
			ID:     fmt.Sprintf("test_node_%d", i),
			Labels: []string{"TestNode", "Entity"},
			Properties: map[string]interface{}{
				"name":    fmt.Sprintf("Test Node %d", i),
				"type":    "test",
				"index":   i,
				"created": "test",
			},
			EntityType:  "test",
			Description: fmt.Sprintf("Test node number %d", i),
			Confidence:  0.9,
			Importance:  float64(i) / float64(count),
			CreatedAt:   time.Now(),
			Version:     1,
		}
		nodes[i] = node
	}

	return nodes
}

// CreateTestRelationships creates a set of test relationships for unit testing
func CreateTestRelationships(count int) []*types.GraphRelationship {
	relationships := make([]*types.GraphRelationship, count)

	for i := 0; i < count; i++ {
		rel := &types.GraphRelationship{
			ID:        fmt.Sprintf("test_rel_%d", i),
			Type:      "RELATED_TO",
			StartNode: fmt.Sprintf("test_node_%d", i),
			EndNode:   fmt.Sprintf("test_node_%d", (i+1)%count), // Create circular relationships
			Properties: map[string]interface{}{
				"name":     fmt.Sprintf("Test Relationship %d", i),
				"type":     "test",
				"index":    i,
				"source":   "test",
				"strength": float64(i) / float64(count),
			},
			Description: fmt.Sprintf("Test relationship number %d", i),
			Confidence:  0.8,
			Weight:      float64(i) / float64(count),
			CreatedAt:   time.Now(),
			Version:     1,
		}
		relationships[i] = rel
	}

	return relationships
}

// CreateTestRelationshipsWithNodes creates test relationships with specific start and end nodes
func CreateTestRelationshipsWithNodes(startNodes, endNodes []string, relType string) []*types.GraphRelationship {
	if len(startNodes) == 0 || len(endNodes) == 0 {
		return []*types.GraphRelationship{}
	}

	var relationships []*types.GraphRelationship
	for i, startNode := range startNodes {
		for j, endNode := range endNodes {
			if startNode != endNode { // Avoid self-loops
				rel := &types.GraphRelationship{
					ID:        fmt.Sprintf("%s_%s_%s_%d_%d", startNode, relType, endNode, i, j),
					Type:      relType,
					StartNode: startNode,
					EndNode:   endNode,
					Properties: map[string]interface{}{
						"name":      fmt.Sprintf("Relationship from %s to %s", startNode, endNode),
						"start_idx": i,
						"end_idx":   j,
						"created":   "test",
					},
					Description: fmt.Sprintf("Test relationship from %s to %s", startNode, endNode),
					Confidence:  0.9,
					Weight:      0.5,
					CreatedAt:   time.Now(),
					Version:     1,
				}
				relationships = append(relationships, rel)
			}
		}
	}

	return relationships
}

// CleanupAllTestDatabases removes all test databases to prevent limit issues
func CleanupAllTestDatabases(t *testing.T, store *Store) {
	if t != nil {
		t.Helper()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Only clean up if we're in separate database mode
	if !store.useSeparateDatabase {
		return
	}

	// List all databases
	databases, err := store.listSeparateDatabaseGraphs(ctx)
	if err != nil {
		if t != nil {
			t.Logf("Failed to list databases for cleanup: %v", err)
		}
		return
	}

	// Drop test databases (keep system databases)
	for _, dbName := range databases {
		if strings.Contains(dbName, "test") && dbName != "neo4j" && dbName != "system" {
			err := store.dropSeparateDatabaseGraph(ctx, dbName)
			if err != nil && t != nil {
				t.Logf("Failed to cleanup test database %s: %v", dbName, err)
			}
		}
	}
}
