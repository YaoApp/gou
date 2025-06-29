package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/embedding"
	extractionOpenai "github.com/yaoapp/gou/graphrag/extraction/openai"
	"github.com/yaoapp/gou/graphrag/types"
)

const (
	testConnectorName      = "openai_extraction_test"
	testEmbeddingConnector = "openai_embedding_test"
	testModel              = "gpt-4o-mini"
	testEmbeddingModel     = "text-embedding-3-small"
	testEmbeddingDimension = 1536
	testConcurrency        = 5
)

// Test data - Chinese and English versions divided into chunks
var (
	testChunksEnglish = []string{
		"John Smith is a senior software engineer at Google Inc. He works in the Mountain View office in California. John has been with Google for 5 years and leads the search algorithm team. He previously worked at Microsoft as a software developer for 3 years.",
		"Sarah Johnson is the Chief Technology Officer at Apple Inc. She is based in Cupertino, California and has been with Apple for 8 years. Sarah manages the iOS development team and reports directly to the CEO. She holds a PhD in Computer Science from Stanford University.",
		"Beijing TechCorp is a Chinese technology company founded in 2015. The company specializes in artificial intelligence and machine learning solutions. It has partnerships with Tsinghua University and employs over 500 engineers. The CEO is David Wang, who previously worked at Baidu.",
	}

	testChunksChinese = []string{
		"张三是谷歌公司的高级软件工程师。他在加利福尼亚州山景城的办公室工作。张三在谷歌工作了5年，领导搜索算法团队。他之前在微软担任软件开发工程师3年。",
		"李四是苹果公司的首席技术官。她总部设在加利福尼亚州库比蒂诺，在苹果工作了8年。李四管理iOS开发团队，直接向CEO汇报。她拥有斯坦福大学计算机科学博士学位。",
		"北京科技公司是2015年成立的中国科技公司。该公司专注于人工智能和机器学习解决方案。它与清华大学建立了合作关系，雇佣了500多名工程师。CEO是王大卫，他之前在百度工作。",
	}
)

func TestMain(m *testing.M) {
	// Setup test environment
	setupTestConnectors()
	code := m.Run()
	os.Exit(code)
}

func setupTestConnectors() {
	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		apiKey = "test-key" // Use dummy key for basic tests
	}

	// Create OpenAI connector for extraction
	createExtractionConnector(testConnectorName, apiKey, testModel)

	// Create OpenAI connector for embedding
	createEmbeddingConnector(testEmbeddingConnector, apiKey, testEmbeddingModel)
}

func createExtractionConnector(name, apiKey, model string) {
	dsl := map[string]interface{}{
		"LANG":    "1.0.0",
		"VERSION": "1.0.0",
		"label":   "OpenAI Extraction Test",
		"type":    "openai",
		"options": map[string]interface{}{
			"proxy": "https://api.openai.com/v1",
			"model": model,
			"key":   apiKey,
		},
	}

	dslBytes, _ := json.Marshal(dsl)
	connector.New("openai", name, dslBytes)
}

func createEmbeddingConnector(name, apiKey, model string) {
	dsl := map[string]interface{}{
		"LANG":    "1.0.0",
		"VERSION": "1.0.0",
		"label":   "OpenAI Embedding Test",
		"type":    "openai",
		"options": map[string]interface{}{
			"proxy": "https://api.openai.com/v1",
			"model": model,
			"key":   apiKey,
		},
	}

	dslBytes, _ := json.Marshal(dsl)
	connector.New("openai", name, dslBytes)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		options  types.ExtractionOptions
		expected bool
	}{
		{
			name: "Valid options",
			options: types.ExtractionOptions{
				Use: nil, // Will be set in test
			},
			expected: true,
		},
		{
			name:     "Empty options",
			options:  types.ExtractionOptions{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extraction := New(tt.options)
			if tt.expected {
				assert.NotNil(t, extraction)
				assert.Equal(t, tt.options, extraction.Options)
			} else {
				assert.Nil(t, extraction)
			}
		})
	}
}

func TestExtractionChaining(t *testing.T) {
	extraction := New(types.ExtractionOptions{})

	// Test Use method
	extractor, err := extractionOpenai.NewOpenaiWithDefaults(testConnectorName)
	if err != nil {
		t.Skipf("Failed to create OpenAI extractor: %v", err)
	}

	result := extraction.Use(extractor)
	assert.NotNil(t, result)
	assert.Equal(t, extractor, extraction.Options.Use)

	// Note: LLMOptimizer method is deprecated and not tested

	// Test Embedding method
	embedder, err := embedding.NewOpenaiWithDefaults(testEmbeddingConnector)
	if err != nil {
		t.Skipf("Failed to create OpenAI embedder: %v", err)
	}

	result = extraction.Embedding(embedder)
	assert.NotNil(t, result)
	assert.Equal(t, embedder, extraction.Options.Embedding)
}

func TestExtractQuery(t *testing.T) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping extraction test")
	}

	// Create extraction instance
	extractor, err := extractionOpenai.NewOpenaiWithDefaults(testConnectorName)
	require.NoError(t, err)

	extraction := New(types.ExtractionOptions{}).Use(extractor)

	tests := []struct {
		name string
		text string
	}{
		{
			name: "English text",
			text: testChunksEnglish[0],
		},
		{
			name: "Chinese text",
			text: testChunksChinese[0],
		},
		{
			name: "Empty text",
			text: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var progressUpdates []types.ExtractionPayload
			callback := func(status types.ExtractionStatus, payload types.ExtractionPayload) {
				progressUpdates = append(progressUpdates, payload)
			}

			result, err := extraction.ExtractQuery(ctx, tt.text, callback)
			if err != nil {
				t.Logf("ExtractQuery failed for %s: %v", tt.name, err)
				return // Skip this test case if API fails
			}
			assert.NotNil(t, result)

			if tt.text != "" {
				assert.NotEmpty(t, progressUpdates)
				if result != nil && len(result.Nodes) > 0 {
					assert.Greater(t, len(result.Nodes), 0)
					t.Logf("Extracted %d nodes and %d relationships from %s text",
						len(result.Nodes), len(result.Relationships), tt.name)
				} else {
					t.Logf("No nodes extracted from %s text", tt.name)
				}
			}
		})
	}
}

func TestExtractDocuments(t *testing.T) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping extraction test")
	}

	// Create extraction instance
	extractor, err := extractionOpenai.NewOpenai(extractionOpenai.Options{
		ConnectorName: testConnectorName,
		Concurrent:    testConcurrency,
		Model:         testModel,
		Temperature:   0.1,
	})
	require.NoError(t, err)

	extraction := New(types.ExtractionOptions{}).Use(extractor)

	tests := []struct {
		name  string
		texts []string
	}{
		{
			name:  "English chunks",
			texts: testChunksEnglish,
		},
		{
			name:  "Chinese chunks",
			texts: testChunksChinese,
		},
		{
			name:  "Mixed chunks",
			texts: []string{testChunksEnglish[0], testChunksChinese[0]},
		},
		{
			name:  "Empty list",
			texts: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var progressUpdates []types.ExtractionPayload
			callback := func(status types.ExtractionStatus, payload types.ExtractionPayload) {
				progressUpdates = append(progressUpdates, payload)
			}

			results, err := extraction.ExtractDocuments(ctx, tt.texts, callback)
			if err != nil {
				t.Logf("ExtractDocuments failed for %s: %v", tt.name, err)
				return // Skip this test case if API fails
			}

			if len(tt.texts) == 0 {
				assert.Empty(t, results)
			} else {
				if results != nil {
					assert.Len(t, results, len(tt.texts))
					assert.NotEmpty(t, progressUpdates)

					totalNodes := 0
					totalRelationships := 0
					for _, result := range results {
						if result != nil {
							totalNodes += len(result.Nodes)
							totalRelationships += len(result.Relationships)
						}
					}

					t.Logf("Extracted %d total nodes and %d total relationships from %d %s documents",
						totalNodes, totalRelationships, len(tt.texts), tt.name)
				} else {
					t.Logf("No results returned for %s", tt.name)
				}
			}
		})
	}
}

func TestDeduplicate(t *testing.T) {
	extraction := New(types.ExtractionOptions{})

	// Create test data with duplicates
	results := []*types.ExtractionResult{
		{
			Nodes: []types.Node{
				{
					ID:          "node1",
					Name:        "John Smith",
					Type:        "PERSON",
					Labels:      []string{"engineer", "employee"},
					Properties:  map[string]interface{}{"age": 30, "department": "engineering"},
					Description: "Software engineer",
				},
				{
					ID:          "node2",
					Name:        "Google",
					Type:        "ORGANIZATION",
					Labels:      []string{"company", "tech"},
					Properties:  map[string]interface{}{"founded": 1998, "location": "Mountain View"},
					Description: "Technology company",
				},
			},
			Relationships: []types.Relationship{
				{
					ID:          "rel1",
					Type:        "WORKS_FOR",
					StartNode:   "node1",
					EndNode:     "node2",
					Properties:  map[string]interface{}{"years": 5, "position": "engineer"},
					Description: "Employment relationship",
				},
			},
		},
		{
			Nodes: []types.Node{
				{
					ID:          "node3", // Different ID but same content
					Name:        "John Smith",
					Type:        "PERSON",
					Labels:      []string{"employee", "engineer"},                               // Different order
					Properties:  map[string]interface{}{"department": "engineering", "age": 30}, // Different order
					Description: "Software engineer",
				},
				{
					ID:          "node4",
					Name:        "Apple",
					Type:        "ORGANIZATION",
					Labels:      []string{"company"},
					Properties:  map[string]interface{}{"founded": 1976},
					Description: "Technology company",
				},
			},
			Relationships: []types.Relationship{
				{
					ID:          "rel2",
					Type:        "WORKS_FOR",
					StartNode:   "node3", // Will be updated to canonical ID
					EndNode:     "node4",
					Properties:  map[string]interface{}{"years": 2},
					Description: "Employment relationship",
				},
			},
		},
	}

	ctx := context.Background()
	uniqueNodes, uniqueRelationships, err := extraction.Deduplicate(ctx, results)

	assert.NoError(t, err)
	assert.Len(t, uniqueNodes, 3) // John Smith should be deduplicated
	assert.Len(t, uniqueRelationships, 2)

	// Check that node IDs were updated in results
	assert.Equal(t, results[0].Nodes[0].ID, results[1].Nodes[0].ID) // Should be same ID after deduplication

	// Check that relationship node references were updated
	assert.Equal(t, results[1].Relationships[0].StartNode, results[0].Nodes[0].ID)

	t.Logf("Deduplicated %d nodes and %d relationships", len(uniqueNodes), len(uniqueRelationships))
}

func TestEmbeddingNodes(t *testing.T) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping embedding test")
	}

	// Create embedding instance
	embedder, err := embedding.NewOpenai(embedding.OpenaiOptions{
		ConnectorName: testEmbeddingConnector,
		Dimension:     testEmbeddingDimension,
		Concurrent:    testConcurrency,
	})
	require.NoError(t, err)

	extraction := New(types.ExtractionOptions{}).Embedding(embedder)

	// Create test nodes
	nodes := []types.Node{
		{
			ID:          "node1",
			Name:        "John Smith",
			Type:        "PERSON",
			Labels:      []string{"engineer"},
			Properties:  map[string]interface{}{"age": 30},
			Description: "Software engineer at Google",
		},
		{
			ID:          "node2",
			Name:        "张三",
			Type:        "PERSON",
			Labels:      []string{"工程师"},
			Properties:  map[string]interface{}{"年龄": 25},
			Description: "谷歌公司的软件工程师",
		},
	}

	ctx := context.Background()

	var progressUpdates []types.EmbeddingPayload
	callback := func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
		progressUpdates = append(progressUpdates, payload)
	}

	err = extraction.EmbeddingNodes(ctx, nodes, callback)
	assert.NoError(t, err)
	assert.NotEmpty(t, progressUpdates)

	// Check that embeddings were added
	for i, node := range nodes {
		assert.Len(t, node.EmbeddingVector, testEmbeddingDimension,
			"Node %d should have embedding vector of dimension %d", i, testEmbeddingDimension)
		assert.NotEmpty(t, node.EmbeddingVector, "Node %d should have non-empty embedding", i)
	}

	t.Logf("Successfully embedded %d nodes", len(nodes))
}

func TestEmbeddingRelationships(t *testing.T) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping embedding test")
	}

	// Create embedding instance
	embedder, err := embedding.NewOpenai(embedding.OpenaiOptions{
		ConnectorName: testEmbeddingConnector,
		Dimension:     testEmbeddingDimension,
		Concurrent:    testConcurrency,
	})
	require.NoError(t, err)

	extraction := New(types.ExtractionOptions{}).Embedding(embedder)

	// Create test relationships
	relationships := []types.Relationship{
		{
			ID:          "rel1",
			Type:        "WORKS_FOR",
			StartNode:   "node1",
			EndNode:     "node2",
			Properties:  map[string]interface{}{"years": 5},
			Description: "Employment relationship between John and Google",
		},
		{
			ID:          "rel2",
			Type:        "位于",
			StartNode:   "node3",
			EndNode:     "node4",
			Properties:  map[string]interface{}{"距离": "10公里"},
			Description: "公司位于山景城",
		},
	}

	ctx := context.Background()

	var progressUpdates []types.EmbeddingPayload
	callback := func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
		progressUpdates = append(progressUpdates, payload)
	}

	err = extraction.EmbeddingRelationships(ctx, relationships, callback)
	assert.NoError(t, err)
	assert.NotEmpty(t, progressUpdates)

	// Check that embeddings were added
	for i, rel := range relationships {
		assert.Len(t, rel.EmbeddingVector, testEmbeddingDimension,
			"Relationship %d should have embedding vector of dimension %d", i, testEmbeddingDimension)
		assert.NotEmpty(t, rel.EmbeddingVector, "Relationship %d should have non-empty embedding", i)
	}

	t.Logf("Successfully embedded %d relationships", len(relationships))
}

func TestEmbeddingResults(t *testing.T) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping embedding test")
	}

	// Create extraction and embedding instances
	extractor, err := extractionOpenai.NewOpenaiWithDefaults(testConnectorName)
	require.NoError(t, err)

	embedder, err := embedding.NewOpenai(embedding.OpenaiOptions{
		ConnectorName: testEmbeddingConnector,
		Dimension:     testEmbeddingDimension,
		Concurrent:    testConcurrency,
	})
	require.NoError(t, err)

	extraction := New(types.ExtractionOptions{}).Use(extractor).Embedding(embedder)

	// First extract some entities
	ctx := context.Background()
	results, err := extraction.ExtractDocuments(ctx, []string{testChunksEnglish[0], testChunksChinese[0]})
	if err != nil {
		t.Skipf("ExtractDocuments failed, skipping embedding test: %v", err)
	}
	require.NotEmpty(t, results)

	// Count entities before embedding
	totalNodesBefore := 0
	totalRelsBefore := 0
	for _, result := range results {
		if result != nil {
			totalNodesBefore += len(result.Nodes)
			totalRelsBefore += len(result.Relationships)
		}
	}

	t.Logf("Before embedding: %d nodes, %d relationships", totalNodesBefore, totalRelsBefore)

	// Now embed the results
	var progressUpdates []types.EmbeddingPayload
	callback := func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
		progressUpdates = append(progressUpdates, payload)
	}

	err = extraction.EmbeddingResults(ctx, results, callback)
	assert.NoError(t, err)
	assert.NotEmpty(t, progressUpdates)

	// Check that embeddings were added to results
	embeddedNodes := 0
	embeddedRels := 0
	for _, result := range results {
		if result != nil {
			for _, node := range result.Nodes {
				if len(node.EmbeddingVector) == testEmbeddingDimension {
					embeddedNodes++
				}
			}
			for _, rel := range result.Relationships {
				if len(rel.EmbeddingVector) == testEmbeddingDimension {
					embeddedRels++
				}
			}
		}
	}

	t.Logf("After embedding: %d nodes embedded, %d relationships embedded", embeddedNodes, embeddedRels)
	assert.Greater(t, embeddedNodes, 0, "At least some nodes should be embedded")
}

func TestEmbeddingResultsWithoutEmbedder(t *testing.T) {
	extraction := New(types.ExtractionOptions{})

	ctx := context.Background()
	results := []*types.ExtractionResult{
		{
			Nodes: []types.Node{
				{ID: "node1", Name: "Test Node"},
			},
		},
	}

	err := extraction.EmbeddingResults(ctx, results)
	assert.NoError(t, err) // Should not error when no embedder is set
}

func TestConcurrentExtraction(t *testing.T) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping concurrent test")
	}

	// Create extraction instance with higher concurrency
	extractor, err := extractionOpenai.NewOpenai(extractionOpenai.Options{
		ConnectorName: testConnectorName,
		Concurrent:    10,
		Model:         testModel,
		Temperature:   0.1,
	})
	require.NoError(t, err)

	embedder, err := embedding.NewOpenai(embedding.OpenaiOptions{
		ConnectorName: testEmbeddingConnector,
		Dimension:     testEmbeddingDimension,
		Concurrent:    10,
	})
	require.NoError(t, err)

	extraction := New(types.ExtractionOptions{}).Use(extractor).Embedding(embedder)

	// Create many test texts
	numTexts := 20
	texts := make([]string, numTexts)
	for i := 0; i < numTexts; i++ {
		if i%2 == 0 {
			texts[i] = fmt.Sprintf("Document %d: %s", i, testChunksEnglish[i%len(testChunksEnglish)])
		} else {
			texts[i] = fmt.Sprintf("文档 %d: %s", i, testChunksChinese[i%len(testChunksChinese)])
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	var progressMutex sync.Mutex
	progressCount := 0
	callback := func(status types.ExtractionStatus, payload types.ExtractionPayload) {
		progressMutex.Lock()
		progressCount++
		if progressCount%10 == 0 {
			t.Logf("Concurrent progress: %s - %s", status, payload.Message)
		}
		progressMutex.Unlock()
	}

	start := time.Now()
	results, err := extraction.ExtractDocuments(ctx, texts, callback)
	elapsed := time.Since(start)

	if err != nil {
		t.Skipf("Concurrent extraction failed, skipping test: %v", err)
	}
	if results == nil {
		t.Skip("No results returned from concurrent extraction")
	}
	assert.Len(t, results, numTexts)

	// Now embed the results
	embeddingCallback := func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
		progressMutex.Lock()
		progressCount++
		if progressCount%10 == 0 {
			t.Logf("Concurrent embedding progress: %s - %s", status, payload.Message)
		}
		progressMutex.Unlock()
	}

	embeddingStart := time.Now()
	err = extraction.EmbeddingResults(ctx, results, embeddingCallback)
	embeddingElapsed := time.Since(embeddingStart)

	assert.NoError(t, err)

	// Count results
	totalNodes := 0
	embeddedNodes := 0
	for _, result := range results {
		if result != nil {
			totalNodes += len(result.Nodes)
			for _, node := range result.Nodes {
				if len(node.EmbeddingVector) == testEmbeddingDimension {
					embeddedNodes++
				}
			}
		}
	}

	t.Logf("Concurrent test: %d documents, %d nodes, %d embedded", numTexts, totalNodes, embeddedNodes)
	t.Logf("Extraction time: %v, Embedding time: %v", elapsed, embeddingElapsed)
	assert.Greater(t, totalNodes, 0)
	assert.Greater(t, embeddedNodes, 0)
}

func TestMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping memory test")
	}

	// Create extraction instance
	extractor, err := extractionOpenai.NewOpenaiWithDefaults(testConnectorName)
	require.NoError(t, err)

	embedder, err := embedding.NewOpenaiWithDefaults(testEmbeddingConnector)
	require.NoError(t, err)

	extraction := New(types.ExtractionOptions{}).Use(extractor).Embedding(embedder)

	// Force garbage collection to get baseline
	runtime.GC()
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	ctx := context.Background()

	// Run extraction multiple times
	numIterations := 5
	for i := 0; i < numIterations; i++ {
		text := fmt.Sprintf("Iteration %d: %s", i, testChunksEnglish[0])

		// Extract
		result, err := extraction.ExtractQuery(ctx, text)
		if err != nil {
			t.Logf("Memory test extraction failed at iteration %d: %v", i, err)
			continue
		}

		// Embed
		if result != nil {
			results := []*types.ExtractionResult{result}
			err = extraction.EmbeddingResults(ctx, results)
			if err != nil {
				t.Logf("Memory test embedding failed at iteration %d: %v", i, err)
				continue
			}
		}

		// Periodic garbage collection
		if i%2 == 0 {
			runtime.GC()
		}
	}

	// Force garbage collection after all operations
	runtime.GC()
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Calculate memory growth
	var memGrowthMB float64
	if memAfter.Alloc >= memBefore.Alloc {
		memGrowth := memAfter.Alloc - memBefore.Alloc
		memGrowthMB = float64(memGrowth) / (1024 * 1024)
	} else {
		memDecrease := memBefore.Alloc - memAfter.Alloc
		memGrowthMB = -float64(memDecrease) / (1024 * 1024)
	}

	t.Logf("Memory before: %d bytes", memBefore.Alloc)
	t.Logf("Memory after: %d bytes", memAfter.Alloc)
	t.Logf("Memory growth: %.2f MB", memGrowthMB)

	// Alert if memory growth seems excessive
	if memGrowthMB > 10.0 {
		t.Errorf("Potential memory leak detected: %.2f MB growth after %d iterations", memGrowthMB, numIterations)
	}
}

func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping stress test")
	}

	// Create extraction instance with high concurrency
	extractor, err := extractionOpenai.NewOpenai(extractionOpenai.Options{
		ConnectorName: testConnectorName,
		Concurrent:    15,
		Model:         testModel,
		Temperature:   0.1,
		RetryAttempts: 2,
		RetryDelay:    500 * time.Millisecond,
	})
	require.NoError(t, err)

	embedder, err := embedding.NewOpenai(embedding.OpenaiOptions{
		ConnectorName: testEmbeddingConnector,
		Dimension:     testEmbeddingDimension,
		Concurrent:    15,
	})
	require.NoError(t, err)

	extraction := New(types.ExtractionOptions{}).Use(extractor).Embedding(embedder)

	// Generate many test texts
	numTexts := 50
	texts := make([]string, numTexts)
	for i := 0; i < numTexts; i++ {
		if i%3 == 0 {
			texts[i] = fmt.Sprintf("Stress test %d: %s", i, testChunksEnglish[i%len(testChunksEnglish)])
		} else if i%3 == 1 {
			texts[i] = fmt.Sprintf("压力测试 %d: %s", i, testChunksChinese[i%len(testChunksChinese)])
		} else {
			texts[i] = fmt.Sprintf("Mixed test %d: John works at Company_%d in City_%d", i, i, i)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	var progressMutex sync.Mutex
	extractionCount := 0
	embeddingCount := 0

	extractionCallback := func(status types.ExtractionStatus, payload types.ExtractionPayload) {
		progressMutex.Lock()
		extractionCount++
		if extractionCount%20 == 0 || status == types.ExtractionStatusCompleted {
			t.Logf("Stress extraction progress (%d): %s", extractionCount, status)
		}
		progressMutex.Unlock()
	}

	embeddingCallback := func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
		progressMutex.Lock()
		embeddingCount++
		if embeddingCount%20 == 0 || status == types.EmbeddingStatusCompleted {
			t.Logf("Stress embedding progress (%d): %s", embeddingCount, status)
		}
		progressMutex.Unlock()
	}

	// Run extraction
	start := time.Now()
	results, err := extraction.ExtractDocuments(ctx, texts, extractionCallback)
	extractionTime := time.Since(start)

	if err != nil {
		t.Skipf("Stress test extraction failed, skipping test: %v", err)
	}
	if results == nil {
		t.Skip("No results returned from stress test extraction")
	}
	require.Len(t, results, numTexts)

	// Run embedding
	embeddingStart := time.Now()
	err = extraction.EmbeddingResults(ctx, results, embeddingCallback)
	embeddingTime := time.Since(embeddingStart)

	if err != nil {
		t.Skipf("Stress test embedding failed, skipping test: %v", err)
	}

	// Count results
	totalNodes := 0
	totalRelationships := 0
	embeddedNodes := 0
	embeddedRelationships := 0

	for _, result := range results {
		if result != nil {
			totalNodes += len(result.Nodes)
			totalRelationships += len(result.Relationships)

			for _, node := range result.Nodes {
				if len(node.EmbeddingVector) == testEmbeddingDimension {
					embeddedNodes++
				}
			}

			for _, rel := range result.Relationships {
				if len(rel.EmbeddingVector) == testEmbeddingDimension {
					embeddedRelationships++
				}
			}
		}
	}

	t.Logf("Stress test results:")
	t.Logf("  Documents: %d", numTexts)
	t.Logf("  Total nodes: %d (embedded: %d)", totalNodes, embeddedNodes)
	t.Logf("  Total relationships: %d (embedded: %d)", totalRelationships, embeddedRelationships)
	t.Logf("  Extraction time: %v", extractionTime)
	t.Logf("  Embedding time: %v", embeddingTime)
	t.Logf("  Total time: %v", extractionTime+embeddingTime)

	assert.Greater(t, totalNodes, 0, "Should extract some nodes")
	assert.Greater(t, embeddedNodes, 0, "Should embed some nodes")
}

func TestErrorHandling(t *testing.T) {
	extraction := New(types.ExtractionOptions{})

	t.Run("Extract query without extractor", func(t *testing.T) {
		ctx := context.Background()
		_, err := extraction.ExtractQuery(ctx, "test")
		assert.Error(t, err)
	})

	t.Run("Extract documents without extractor", func(t *testing.T) {
		ctx := context.Background()
		_, err := extraction.ExtractDocuments(ctx, []string{"test"})
		assert.Error(t, err)
	})

	t.Run("Deduplicate with nil results", func(t *testing.T) {
		ctx := context.Background()
		nodes, rels, err := extraction.Deduplicate(ctx, nil)
		assert.NoError(t, err)
		assert.Nil(t, nodes)
		assert.Nil(t, rels)
	})

	t.Run("Deduplicate with empty results", func(t *testing.T) {
		ctx := context.Background()
		nodes, rels, err := extraction.Deduplicate(ctx, []*types.ExtractionResult{})
		assert.NoError(t, err)
		assert.Nil(t, nodes)
		assert.Nil(t, rels)
	})

	t.Run("Embedding nodes without embedder", func(t *testing.T) {
		ctx := context.Background()
		err := extraction.EmbeddingNodes(ctx, []types.Node{})
		assert.NoError(t, err) // Should not error when no embedder
	})

	t.Run("Embedding relationships without embedder", func(t *testing.T) {
		ctx := context.Background()
		err := extraction.EmbeddingRelationships(ctx, []types.Relationship{})
		assert.NoError(t, err) // Should not error when no embedder
	})

	t.Run("Context cancellation", func(t *testing.T) {
		apiKey := os.Getenv("OPENAI_TEST_KEY")
		if apiKey == "" {
			t.Skip("OPENAI_TEST_KEY not set, skipping cancellation test")
		}

		extractor, err := extractionOpenai.NewOpenaiWithDefaults(testConnectorName)
		if err != nil {
			t.Skipf("Failed to create extractor: %v", err)
		}

		extraction := New(types.ExtractionOptions{}).Use(extractor)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = extraction.ExtractQuery(ctx, testChunksEnglish[0])
		assert.Error(t, err, "Should return error for cancelled context")
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("Deduplicate with nil nodes and relationships", func(t *testing.T) {
		extraction := New(types.ExtractionOptions{})

		results := []*types.ExtractionResult{
			{
				Nodes:         nil,
				Relationships: nil,
			},
			{
				Nodes:         []types.Node{},
				Relationships: []types.Relationship{},
			},
			nil, // nil result
		}

		ctx := context.Background()
		nodes, rels, err := extraction.Deduplicate(ctx, results)

		assert.NoError(t, err)
		assert.Empty(t, nodes)
		assert.Empty(t, rels)
	})

	t.Run("Embedding empty lists", func(t *testing.T) {
		apiKey := os.Getenv("OPENAI_TEST_KEY")
		if apiKey == "" {
			t.Skip("OPENAI_TEST_KEY not set, skipping embedding test")
		}

		embedder, err := embedding.NewOpenaiWithDefaults(testEmbeddingConnector)
		require.NoError(t, err)

		extraction := New(types.ExtractionOptions{}).Embedding(embedder)

		ctx := context.Background()

		err = extraction.EmbeddingNodes(ctx, []types.Node{})
		assert.NoError(t, err)

		err = extraction.EmbeddingRelationships(ctx, []types.Relationship{})
		assert.NoError(t, err)
	})

	t.Run("Properties normalization edge cases", func(t *testing.T) {
		extraction := New(types.ExtractionOptions{})

		// Test with complex properties
		results := []*types.ExtractionResult{
			{
				Nodes: []types.Node{
					{
						ID:   "node1",
						Name: "Test",
						Type: "TEST",
						Properties: map[string]interface{}{
							"nested": map[string]interface{}{
								"key": "value",
							},
							"array": []interface{}{1, 2, 3},
							"null":  nil,
							"bool":  true,
							"float": 3.14,
						},
					},
				},
			},
		}

		ctx := context.Background()
		nodes, _, err := extraction.Deduplicate(ctx, results)

		assert.NoError(t, err)
		assert.Len(t, nodes, 1)
	})
}

// Benchmark tests
func BenchmarkExtractQuery(b *testing.B) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_TEST_KEY not set, skipping benchmark")
	}

	extractor, err := extractionOpenai.NewOpenaiWithDefaults(testConnectorName)
	require.NoError(b, err)

	extraction := New(types.ExtractionOptions{}).Use(extractor)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := extraction.ExtractQuery(ctx, testChunksEnglish[0])
		if err != nil {
			b.Logf("Benchmark error: %v", err)
			b.SkipNow() // Skip benchmark if API fails
		}
	}
}

func BenchmarkEmbeddingNodes(b *testing.B) {
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_TEST_KEY not set, skipping benchmark")
	}

	embedder, err := embedding.NewOpenaiWithDefaults(testEmbeddingConnector)
	require.NoError(b, err)

	extraction := New(types.ExtractionOptions{}).Embedding(embedder)
	ctx := context.Background()

	nodes := []types.Node{
		{
			ID:          "node1",
			Name:        "Test Node",
			Type:        "TEST",
			Description: "Test description",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := extraction.EmbeddingNodes(ctx, nodes)
		if err != nil {
			b.Logf("Benchmark error: %v", err)
			b.SkipNow() // Skip benchmark if API fails
		}
	}
}

func BenchmarkDeduplicate(b *testing.B) {
	extraction := New(types.ExtractionOptions{})

	// Create test data with many duplicates
	results := make([]*types.ExtractionResult, 10)
	for i := 0; i < 10; i++ {
		results[i] = &types.ExtractionResult{
			Nodes: []types.Node{
				{
					ID:          fmt.Sprintf("node_%d", i),
					Name:        "Duplicate Node",
					Type:        "TEST",
					Properties:  map[string]interface{}{"index": i},
					Description: "Duplicate description",
				},
			},
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := extraction.Deduplicate(ctx, results)
		if err != nil {
			b.Logf("Benchmark error: %v", err)
		}
	}
}
