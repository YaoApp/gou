package qdrant

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// =============================================================================
// Test Data Structures
// =============================================================================

// TestDocument represents a test document loaded from JSON files
type TestDocument struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Vector     []float64              `json:"vector"`
	Metadata   map[string]interface{} `json:"metadata"`
	ChunkIndex int                    `json:"chunk_index"`
	ChunkFile  string                 `json:"chunk_file"`
	Language   string                 `json:"language"`
	Category   string                 `json:"category"`
}

// TestDataSet represents a collection of test documents
type TestDataSet struct {
	Documents      []*TestDocument
	CollectionName string
	Language       string
	VectorDim      int
	Loaded         bool
	mu             sync.RWMutex
}

// SearchTestEnvironment holds the search test environment
type SearchTestEnvironment struct {
	Store       *Store
	Config      types.VectorStoreConfig
	DataSets    map[string]*TestDataSet
	initialized bool
	mu          sync.RWMutex
}

// Global test environment - singleton to avoid re-creating data
var globalSearchTestEnv *SearchTestEnvironment
var searchTestEnvOnce sync.Once

// =============================================================================
// Test Data Preparation Functions
// =============================================================================

// getTestDataPath returns the absolute path to test data directory
func getTestDataPath() string {
	_, filename, _, _ := runtime.Caller(0)
	sourceDir := filepath.Dir(filename)
	// Navigate to the test data directory: gou/graphrag/tests/
	testDataPath := filepath.Join(sourceDir, "..", "..", "tests")
	absPath, _ := filepath.Abs(testDataPath)
	return absPath
}

// loadMappingData loads the mapping JSON file for metadata
// MappingEntry represents a single mapping entry
type MappingEntry struct {
	ID           string                   `json:"id"`
	Index        int                      `json:"index"`
	Depth        int                      `json:"depth"`
	ParentID     string                   `json:"parent_id"`
	Filename     string                   `json:"filename"`
	TextSize     int                      `json:"text_size"`
	IsLeaf       bool                     `json:"is_leaf"`
	IsRoot       bool                     `json:"is_root"`
	TextPosition map[string]interface{}   `json:"text_position"`
	Parents      []map[string]interface{} `json:"parents"`
}

func loadMappingData(mappingFile string) (map[string]*MappingEntry, error) {
	data, err := os.ReadFile(mappingFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file %s: %w", mappingFile, err)
	}

	var mappingArray []MappingEntry
	if err := json.Unmarshal(data, &mappingArray); err != nil {
		return nil, fmt.Errorf("failed to parse mapping file %s: %w", mappingFile, err)
	}

	// Convert array to map for efficient lookup by filename
	mappingMap := make(map[string]*MappingEntry)
	for i := range mappingArray {
		entry := &mappingArray[i]
		mappingMap[entry.Filename] = entry
	}

	return mappingMap, nil
}

// loadTestDocumentsFromDir loads test documents from a directory of JSON files
func loadTestDocumentsFromDir(dirPath, language string) ([]*TestDocument, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	var documents []*TestDocument
	var mu sync.Mutex
	var wg sync.WaitGroup
	errors := make(chan error, len(files))

	// Load mapping data if available
	var mappingData map[string]*MappingEntry
	mappingFiles := []string{
		filepath.Join(dirPath, fmt.Sprintf("semantic-%s.mapping.json", language)),
		filepath.Join(dirPath, fmt.Sprintf("structured-%s.mapping.json", language)),
		filepath.Join(dirPath, "semantic-zh.mapping.json"),
		filepath.Join(dirPath, "structured-zh.mapping.json"),
		filepath.Join(dirPath, "semantic-en.mapping.json"),
		filepath.Join(dirPath, "structured-en.mapping.json"),
	}

	for _, mappingFile := range mappingFiles {
		if _, err := os.Stat(mappingFile); err == nil {
			if data, err := loadMappingData(mappingFile); err == nil {
				mappingData = data
				break
			}
		}
	}

	// Process JSON files concurrently
	semaphore := make(chan struct{}, 10) // Limit concurrent goroutines

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" &&
			!stringContains(file.Name(), "mapping") {
			wg.Add(1)
			go func(filename string) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire semaphore
				defer func() { <-semaphore }() // Release semaphore

				filePath := filepath.Join(dirPath, filename)
				doc, err := loadTestDocumentFromFile(filePath, filename, language, mappingData)
				if err != nil {
					errors <- fmt.Errorf("failed to load %s: %w", filename, err)
					return
				}

				if doc != nil {
					mu.Lock()
					documents = append(documents, doc)
					mu.Unlock()
				}
			}(file.Name())
		}
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		return nil, err
	}

	return documents, nil
}

// loadTestDocumentFromFile loads a single test document from JSON file
func loadTestDocumentFromFile(filePath, filename, language string, mappingData map[string]*MappingEntry) (*TestDocument, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse the JSON structure
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}

	// Extract vector data - try both "embedding" and "vector" fields
	var vectorData interface{}
	var ok bool

	if vectorData, ok = jsonData["embedding"]; !ok || vectorData == nil {
		if vectorData, ok = jsonData["vector"]; !ok || vectorData == nil {
			return nil, fmt.Errorf("no vector/embedding data found in %s", filename)
		}
	}

	var vector []float64
	switch v := vectorData.(type) {
	case []interface{}:
		vector = make([]float64, len(v))
		for i, val := range v {
			if fVal, ok := val.(float64); ok {
				vector[i] = fVal
			} else {
				return nil, fmt.Errorf("invalid vector value in %s", filename)
			}
		}
	case []float64:
		vector = v
	default:
		return nil, fmt.Errorf("unsupported vector format in %s", filename)
	}

	// Extract content - first try direct fields, then try reading from corresponding txt file
	content := ""
	if contentData, ok := jsonData["content"]; ok {
		if contentStr, ok := contentData.(string); ok {
			content = contentStr
		}
	}
	if content == "" {
		if textData, ok := jsonData["text"]; ok {
			if textStr, ok := textData.(string); ok {
				content = textStr
			}
		}
	}

	// If no content in JSON, try to read from corresponding txt file
	if content == "" {
		// Try to find the corresponding txt file
		var txtFilePath string
		if fileField, ok := jsonData["file"]; ok {
			if fileStr, ok := fileField.(string); ok {
				// Use the directory of JSON file and the filename from JSON
				txtFilePath = filepath.Join(filepath.Dir(filePath), fileStr)
			}
		}

		// If no explicit file field, try to derive from JSON filename
		if txtFilePath == "" {
			// Convert "semantic-en.3.chunk-0.txt.json" to "semantic-en.3.chunk-0.txt"
			txtFilename := strings.TrimSuffix(filename, ".json")
			txtFilePath = filepath.Join(filepath.Dir(filePath), txtFilename)
		}

		// Try to read the txt file
		if txtFilePath != "" {
			if txtData, err := os.ReadFile(txtFilePath); err == nil {
				content = string(txtData)
			}
		}

		// If still no content, use a placeholder
		if content == "" {
			content = fmt.Sprintf("Content for %s", filename)
		}
	}

	// Generate document ID from filename - ensure uniqueness
	baseFilename := filepath.Base(filename)
	// Remove .json extension if present
	baseFilename = strings.TrimSuffix(baseFilename, ".json")
	docID := fmt.Sprintf("%s_%s", language, baseFilename)

	// Add a unique suffix based on file path to avoid collisions
	if strings.Contains(filePath, "/") {
		// Use a hash of the full path for uniqueness
		pathHash := fmt.Sprintf("_%x", len(filePath))
		docID = docID + pathHash
	}

	// Prepare metadata
	metadata := make(map[string]interface{})
	metadata["filename"] = filename
	metadata["language"] = language
	metadata["vector_dim"] = len(vector)
	metadata["file_path"] = filePath

	// Add dimension info if available
	if dimData, ok := jsonData["dimension"]; ok {
		metadata["dimension"] = dimData
	}

	// Add model info if available
	if modelData, ok := jsonData["model"]; ok {
		metadata["model"] = modelData
	}

	// Add text length if available
	if textLenData, ok := jsonData["text_length"]; ok {
		metadata["text_length"] = textLenData
	}

	// Add chunk information
	if chunkInfo, ok := jsonData["chunk_index"]; ok {
		metadata["chunk_index"] = chunkInfo
	}

	// Add mapping data if available
	if mappingData != nil {
		// Try to find mapping data by filename - handle different filename formats
		var mappingEntry *MappingEntry

		// First try direct filename match
		if entry, ok := mappingData[filename]; ok {
			mappingEntry = entry
		} else {
			// Try without .json extension
			filenameWithoutJSON := strings.TrimSuffix(filename, ".json")
			if entry, ok := mappingData[filenameWithoutJSON]; ok {
				mappingEntry = entry
			} else {
				// Try without .txt.json extension
				filenameWithoutTxtJSON := strings.TrimSuffix(filenameWithoutJSON, ".txt")
				if entry, ok := mappingData[filenameWithoutTxtJSON]; ok {
					mappingEntry = entry
				}
			}
		}

		// If mapping entry found, add its data to metadata
		if mappingEntry != nil {
			metadata["mapping_id"] = mappingEntry.ID
			metadata["index"] = mappingEntry.Index
			metadata["depth"] = mappingEntry.Depth
			metadata["parent_id"] = mappingEntry.ParentID
			metadata["text_size"] = mappingEntry.TextSize
			metadata["is_leaf"] = mappingEntry.IsLeaf
			metadata["is_root"] = mappingEntry.IsRoot

			if mappingEntry.TextPosition != nil {
				metadata["text_position"] = mappingEntry.TextPosition
			}

			if mappingEntry.Parents != nil {
				metadata["parents"] = mappingEntry.Parents
			}

			// Use the mapping ID as the document ID if available
			if mappingEntry.ID != "" {
				docID = mappingEntry.ID
			}
		}
	}

	// Add any additional metadata from the JSON
	if metaData, ok := jsonData["metadata"]; ok {
		if metaMap, ok := metaData.(map[string]interface{}); ok {
			for key, value := range metaMap {
				metadata[key] = value
			}
		}
	}

	// Determine category based on content and metadata
	category := "general"
	if categoryData, ok := metadata["category"]; ok {
		if categoryStr, ok := categoryData.(string); ok {
			category = categoryStr
		}
	}

	return &TestDocument{
		ID:         docID,
		Content:    content,
		Vector:     vector,
		Metadata:   metadata,
		ChunkIndex: extractChunkIndex(filename),
		ChunkFile:  filename,
		Language:   language,
		Category:   category,
	}, nil
}

// extractChunkIndex extracts chunk index from filename
func extractChunkIndex(filename string) int {
	// Try to extract number from filename like "semantic-en.3.chunk-97.txt.json"
	var chunkIndex int
	fmt.Sscanf(filename, "semantic-%*[^.].%*d.chunk-%d.txt.json", &chunkIndex)
	return chunkIndex
}

// prepareTestDataSet prepares a test data set for a specific language
func prepareTestDataSet(language string) (*TestDataSet, error) {
	testDataPath := getTestDataPath()
	var dirName string

	switch language {
	case "en":
		dirName = "semantic-en"
	case "zh":
		dirName = "structured-zh"
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	dirPath := filepath.Join(testDataPath, dirName)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("test data directory not found: %s", dirPath)
	}

	documents, err := loadTestDocumentsFromDir(dirPath, language)
	if err != nil {
		return nil, fmt.Errorf("failed to load test documents: %w", err)
	}

	if len(documents) == 0 {
		return nil, fmt.Errorf("no test documents loaded from %s", dirPath)
	}

	// Determine vector dimension from first document
	vectorDim := 0
	if len(documents) > 0 {
		vectorDim = len(documents[0].Vector)
	}

	collectionName := fmt.Sprintf("test_search_%s_%d", language, time.Now().UnixNano())

	return &TestDataSet{
		Documents:      documents,
		CollectionName: collectionName,
		Language:       language,
		VectorDim:      vectorDim,
		Loaded:         true,
	}, nil
}

// getOrCreateSearchTestEnvironment returns the global search test environment
func getOrCreateSearchTestEnvironment(t *testing.T) *SearchTestEnvironment {
	searchTestEnvOnce.Do(func() {
		globalSearchTestEnv = &SearchTestEnvironment{
			DataSets: make(map[string]*TestDataSet),
		}
	})

	globalSearchTestEnv.mu.Lock()
	defer globalSearchTestEnv.mu.Unlock()

	if !globalSearchTestEnv.initialized {
		// Initialize store using getTestConfig
		store := NewStore()
		testConfig := getTestConfig()

		// Create VectorStoreConfig from TestConfig
		config := types.VectorStoreConfig{
			Dimension: 128, // Default dimension, will be updated per collection
			Distance:  types.DistanceCosine,
			IndexType: types.IndexTypeHNSW,
			ExtraParams: map[string]interface{}{
				"host":     testConfig.Host,
				"port":     testConfig.Port,
				"api_key":  testConfig.APIKey,
				"username": testConfig.Username,
				"password": testConfig.Password,
			},
		}

		// Connect to Qdrant
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := store.Connect(ctx, config); err != nil {
			t.Fatalf("Failed to connect to Qdrant: %v", err)
		}

		globalSearchTestEnv.Store = store
		globalSearchTestEnv.Config = config
		globalSearchTestEnv.initialized = true
	}

	return globalSearchTestEnv
}

// getOrCreateTestDataSet loads or creates a test data set
func getOrCreateTestDataSet(t *testing.T, language string) *TestDataSet {
	env := getOrCreateSearchTestEnvironment(t)

	env.mu.Lock()
	defer env.mu.Unlock()

	key := fmt.Sprintf("dataset_%s", language)
	if dataSet, exists := env.DataSets[key]; exists {
		return dataSet
	}

	// Load test data set
	dataSet, err := prepareTestDataSet(language)
	if err != nil {
		if testing.Short() {
			t.Skipf("Skipping test data preparation in short mode: %v", err)
		}
		t.Fatalf("Failed to prepare test data set for %s: %v", language, err)
	}

	// Create collection and add documents
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Update config dimension for this collection
	config := env.Config
	config.Dimension = dataSet.VectorDim
	config.CollectionName = dataSet.CollectionName

	if err := env.Store.CreateCollection(ctx, &config); err != nil {
		t.Fatalf("Failed to create collection %s: %v", dataSet.CollectionName, err)
	}

	// Convert test documents to vector store documents
	var documents []*types.Document
	for _, testDoc := range dataSet.Documents {
		doc := &types.Document{
			ID:          testDoc.ID,
			PageContent: testDoc.Content,
			Vector:      testDoc.Vector,
			Metadata:    testDoc.Metadata,
		}
		documents = append(documents, doc)
	}

	// Add documents to collection in batches
	batchSize := 50
	for i := 0; i < len(documents); i += batchSize {
		end := i + batchSize
		if end > len(documents) {
			end = len(documents)
		}

		batch := documents[i:end]
		addOpts := &types.AddDocumentOptions{
			CollectionName: dataSet.CollectionName,
			Documents:      batch,
			BatchSize:      batchSize,
		}

		if _, err := env.Store.AddDocuments(ctx, addOpts); err != nil {
			t.Fatalf("Failed to add document batch %d-%d for %s: %v", i, end-1, language, err)
		}
	}

	env.DataSets[key] = dataSet
	return dataSet
}

// stringContains checks if a string contains a substring
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr) && func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	code := m.Run()

	// Cleanup
	if globalSearchTestEnv != nil && globalSearchTestEnv.Store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Note: Collection cleanup would need to be implemented based on available methods
		// For now, we'll just disconnect
		globalSearchTestEnv.Store.Disconnect(ctx)
	}

	os.Exit(code)
}

// =============================================================================
// SearchSimilar Tests
// =============================================================================

// TestSearchSimilar_BasicFunctionality tests basic search functionality
func TestSearchSimilar_BasicFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	ctx := context.Background()
	queryVector := testDataSet.Documents[0].Vector // Use first document as query

	t.Run("BasicSimilaritySearch", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              5,
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar failed: %v", err)
		}

		if len(result.Documents) == 0 {
			t.Fatal("No documents returned")
		}

		if len(result.Documents) > 5 {
			t.Errorf("Expected at most 5 documents, got %d", len(result.Documents))
		}

		// Verify that scores are in descending order
		for i := 1; i < len(result.Documents); i++ {
			if result.Documents[i-1].Score < result.Documents[i].Score {
				t.Errorf("Documents not ordered by score: doc[%d].Score=%.6f < doc[%d].Score=%.6f",
					i-1, result.Documents[i-1].Score, i, result.Documents[i].Score)
			}
		}

		// Verify MaxScore and MinScore
		if len(result.Documents) > 0 {
			firstScore := result.Documents[0].Score
			lastScore := result.Documents[len(result.Documents)-1].Score

			if result.MaxScore != firstScore {
				t.Errorf("MaxScore mismatch: expected %.6f, got %.6f", firstScore, result.MaxScore)
			}
			if result.MinScore != lastScore {
				t.Errorf("MinScore mismatch: expected %.6f, got %.6f", lastScore, result.MinScore)
			}
		}
	})

	t.Run("SearchWithDifferentK", func(t *testing.T) {
		for _, k := range []int{1, 3, 10, 20} {
			t.Run(fmt.Sprintf("K=%d", k), func(t *testing.T) {
				opts := &types.SearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              k,
				}

				result, err := env.Store.SearchSimilar(ctx, opts)
				if err != nil {
					t.Fatalf("SearchSimilar with K=%d failed: %v", k, err)
				}

				maxExpected := min(k, len(testDataSet.Documents))
				if len(result.Documents) > maxExpected {
					t.Errorf("With K=%d, expected at most %d documents, got %d",
						k, maxExpected, len(result.Documents))
				}
			})
		}
	})

	t.Run("PaginationTest", func(t *testing.T) {
		pageSize := 3
		totalPages := 2

		var allResults []*types.SearchResultItem

		for page := 1; page <= totalPages; page++ {
			opts := &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              pageSize * totalPages,
				Page:           page,
				PageSize:       pageSize,
				IncludeTotal:   true,
			}

			result, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				t.Fatalf("Paginated search (page %d) failed: %v", page, err)
			}

			// Verify pagination metadata
			if result.Page != page {
				t.Errorf("Page mismatch: expected %d, got %d", page, result.Page)
			}
			if result.PageSize != pageSize {
				t.Errorf("PageSize mismatch: expected %d, got %d", pageSize, result.PageSize)
			}

			if page == 1 {
				if result.HasPrevious {
					t.Error("First page should not have previous page")
				}
			} else {
				if !result.HasPrevious {
					t.Error("Non-first page should have previous page")
				}
				if result.PreviousPage != page-1 {
					t.Errorf("PreviousPage mismatch: expected %d, got %d", page-1, result.PreviousPage)
				}
			}

			allResults = append(allResults, result.Documents...)
		}

		// Verify no duplicates across pages
		seenIDs := make(map[string]bool)
		for i, doc := range allResults {
			if seenIDs[doc.Document.ID] {
				t.Errorf("Duplicate document ID across pages: %s (found at result index %d)", doc.Document.ID, i)
			}
			seenIDs[doc.Document.ID] = true
		}

		t.Logf("Pagination test completed: total results across pages: %d, unique IDs: %d",
			len(allResults), len(seenIDs))
	})
}

// TestSearchSimilar_ErrorScenarios tests error conditions
func TestSearchSimilar_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	ctx := context.Background()

	tests := []struct {
		name     string
		opts     *types.SearchOptions
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:    "NilOptions",
			opts:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "search options cannot be nil")
			},
		},
		{
			name: "EmptyCollectionName",
			opts: &types.SearchOptions{
				CollectionName: "",
				QueryVector:    testDataSet.Documents[0].Vector,
				K:              5,
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "collection name is required")
			},
		},
		{
			name: "EmptyQueryVector",
			opts: &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    []float64{},
				K:              5,
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "query vector is required")
			},
		},
		{
			name: "NilQueryVector",
			opts: &types.SearchOptions{
				CollectionName: testDataSet.CollectionName,
				QueryVector:    nil,
				K:              5,
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "query vector is required")
			},
		},
		{
			name: "NonexistentCollection",
			opts: &types.SearchOptions{
				CollectionName: "nonexistent_collection_12345",
				QueryVector:    testDataSet.Documents[0].Vector,
				K:              5,
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return stringContains(err.Error(), "failed to perform similarity search")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.Store.SearchSimilar(ctx, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SearchSimilar() expected error, got nil")
				} else if tt.errCheck != nil && !tt.errCheck(err) {
					t.Errorf("SearchSimilar() error = %v, error check failed", err)
				}
			} else {
				if err != nil {
					t.Errorf("SearchSimilar() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("SearchSimilar() returned nil result")
				}
			}
		})
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestSearchSimilar_EdgeCases tests edge cases
func TestSearchSimilar_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataSet := getOrCreateTestDataSet(t, "en")
	env := getOrCreateSearchTestEnvironment(t)

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available")
	}

	ctx := context.Background()
	queryVector := testDataSet.Documents[0].Vector

	t.Run("ZeroK", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              0,
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with K=0 failed: %v", err)
		}

		// Should return some default number of results
		if len(result.Documents) == 0 {
			t.Log("K=0 returned no documents (acceptable)")
		}
	})

	t.Run("VeryHighK", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10000,
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with very high K failed: %v", err)
		}

		// Should not return more documents than available
		maxPossible := len(testDataSet.Documents)
		if len(result.Documents) > maxPossible {
			t.Errorf("Expected at most %d documents, got %d", maxPossible, len(result.Documents))
		}
	})

	t.Run("VeryHighMinScore", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			MinScore:       0.99999,
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with very high MinScore failed: %v", err)
		}

		// Might return no documents if none meet the threshold
		for _, doc := range result.Documents {
			if doc.Score < 0.99999 {
				t.Errorf("Document score %.6f should be >= 0.99999", doc.Score)
			}
		}
	})

	t.Run("ZeroVector", func(t *testing.T) {
		zeroVector := make([]float64, len(queryVector))

		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    zeroVector,
			K:              5,
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with zero vector failed: %v", err)
		}

		// Should still return results, just with different scores
		if len(result.Documents) == 0 {
			t.Log("Zero vector returned no documents (acceptable)")
		}
	})

	t.Run("PaginationBeyondData", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			Page:           1000,
			PageSize:       10,
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with pagination beyond data failed: %v", err)
		}

		// Should return empty results
		if len(result.Documents) > 0 {
			t.Logf("Pagination beyond data returned %d documents", len(result.Documents))
		}
	})

	t.Run("VeryShortTimeout", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              5,
			Timeout:        1, // 1 millisecond
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		// This might timeout, which is acceptable
		if err != nil {
			if stringContains(err.Error(), "timeout") || stringContains(err.Error(), "context deadline exceeded") {
				t.Log("Very short timeout caused timeout error (acceptable)")
				return
			}
			t.Fatalf("SearchSimilar with very short timeout failed with unexpected error: %v", err)
		}

		// If it didn't timeout, it should still return valid results
		if result != nil && len(result.Documents) > 0 {
			t.Log("Very short timeout completed successfully")
		}
	})

	t.Run("FilterNoMatches", func(t *testing.T) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			Filter: map[string]interface{}{
				"nonexistent_field": "nonexistent_value",
			},
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar with no-match filter failed: %v", err)
		}

		// Should return empty results or very few results
		if len(result.Documents) > 0 {
			t.Logf("Filter with no matches returned %d documents", len(result.Documents))
		}
	})
}

// TestSearchSimilar_NotConnectedStore tests error when store is not connected
func TestSearchSimilar_NotConnectedStore(t *testing.T) {
	store := NewStore()

	opts := &types.SearchOptions{
		CollectionName: "test_collection",
		QueryVector:    []float64{1.0, 2.0, 3.0},
		K:              5,
	}

	result, err := store.SearchSimilar(context.Background(), opts)

	if err == nil {
		t.Error("SearchSimilar() should fail when store is not connected")
	}

	if !stringContains(err.Error(), "not connected") {
		t.Errorf("Expected 'not connected' error, got: %v", err)
	}

	if result != nil {
		t.Error("SearchSimilar() should return nil result when not connected")
	}
}

// TestSearchSimilar_MultiLanguageData tests cross-language search
func TestSearchSimilar_MultiLanguageData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load both English and Chinese datasets
	enDataSet := getOrCreateTestDataSet(t, "en")
	zhDataSet := getOrCreateTestDataSet(t, "zh")
	env := getOrCreateSearchTestEnvironment(t)

	if len(enDataSet.Documents) == 0 || len(zhDataSet.Documents) == 0 {
		t.Skip("Not enough test documents available for multi-language test")
	}

	ctx := context.Background()

	t.Run("SearchEnglishData", func(t *testing.T) {
		queryVector := enDataSet.Documents[0].Vector

		opts := &types.SearchOptions{
			CollectionName:  enDataSet.CollectionName,
			QueryVector:     queryVector,
			K:               5,
			IncludeMetadata: true,
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar on English data failed: %v", err)
		}

		if len(result.Documents) == 0 {
			t.Fatal("No documents returned for English search")
		}

		// Verify that returned documents are from English dataset
		for i, doc := range result.Documents {
			if lang, ok := doc.Document.Metadata["language"]; ok {
				if lang != "en" {
					t.Errorf("Document %d should be English, got language: %v", i, lang)
				}
			}
		}
	})

	t.Run("SearchChineseData", func(t *testing.T) {
		queryVector := zhDataSet.Documents[0].Vector

		opts := &types.SearchOptions{
			CollectionName:  zhDataSet.CollectionName,
			QueryVector:     queryVector,
			K:               5,
			IncludeMetadata: true,
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("SearchSimilar on Chinese data failed: %v", err)
		}

		if len(result.Documents) == 0 {
			t.Fatal("No documents returned for Chinese search")
		}

		// Verify that returned documents are from Chinese dataset
		for i, doc := range result.Documents {
			if lang, ok := doc.Document.Metadata["language"]; ok {
				if lang != "zh" {
					t.Errorf("Document %d should be Chinese, got language: %v", i, lang)
				}
			}
		}
	})

	t.Run("CrossLanguageQuery", func(t *testing.T) {
		// Use English vector to search Chinese collection
		enVector := enDataSet.Documents[0].Vector

		opts := &types.SearchOptions{
			CollectionName:  zhDataSet.CollectionName,
			QueryVector:     enVector,
			K:               3,
			IncludeMetadata: true,
		}

		result, err := env.Store.SearchSimilar(ctx, opts)
		if err != nil {
			t.Fatalf("Cross-language search failed: %v", err)
		}

		// Results should still be from Chinese collection
		for i, doc := range result.Documents {
			if lang, ok := doc.Document.Metadata["language"]; ok {
				if lang != "zh" {
					t.Errorf("Cross-language search result %d should be Chinese, got: %v", i, lang)
				}
			}
		}
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

// BenchmarkSearchSimilar benchmarks the SearchSimilar method
func BenchmarkSearchSimilar(b *testing.B) {
	// Setup test environment
	env := getOrCreateSearchTestEnvironment(&testing.T{})
	testDataSet := getOrCreateTestDataSet(&testing.T{}, "en")

	if len(testDataSet.Documents) == 0 {
		b.Skip("No test documents available for benchmarking")
	}

	ctx := context.Background()
	queryVector := testDataSet.Documents[0].Vector

	b.Run("BasicSearch", func(b *testing.B) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				b.Fatalf("SearchSimilar failed: %v", err)
			}
		}
	})

	b.Run("SearchWithVectors", func(b *testing.B) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			IncludeVector:  true,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				b.Fatalf("SearchSimilar with vectors failed: %v", err)
			}
		}
	})

	b.Run("SearchWithFilter", func(b *testing.B) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              10,
			Filter: map[string]interface{}{
				"language": "en",
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				b.Fatalf("SearchSimilar with filter failed: %v", err)
			}
		}
	})

	b.Run("SearchWithPagination", func(b *testing.B) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              20,
			Page:           1,
			PageSize:       5,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				b.Fatalf("SearchSimilar with pagination failed: %v", err)
			}
		}
	})

	b.Run("HighK", func(b *testing.B) {
		opts := &types.SearchOptions{
			CollectionName: testDataSet.CollectionName,
			QueryVector:    queryVector,
			K:              100,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				b.Fatalf("SearchSimilar with high K failed: %v", err)
			}
		}
	})
}

// =============================================================================
// Memory and Stress Tests
// =============================================================================

// TestSearchSimilar_MemoryLeakDetection tests for memory leaks during searches
func TestSearchSimilar_MemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	env := getOrCreateSearchTestEnvironment(t)
	testDataSet := getOrCreateTestDataSet(t, "en")

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available for memory leak test")
	}

	ctx := context.Background()
	queryVector := testDataSet.Documents[0].Vector

	// Get initial memory stats
	var initialStats, finalStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialStats)

	// Perform many searches
	iterations := 500
	searchesPerIteration := 5

	for i := 0; i < iterations; i++ {
		for j := 0; j < searchesPerIteration; j++ {
			opts := &types.SearchOptions{
				CollectionName:  testDataSet.CollectionName,
				QueryVector:     queryVector,
				K:               10,
				IncludeVector:   j%2 == 0,
				IncludeMetadata: j%3 == 0,
				IncludeContent:  j%4 == 0,
			}

			result, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				t.Fatalf("SearchSimilar failed at iteration %d, search %d: %v", i, j, err)
			}

			if len(result.Documents) == 0 {
				t.Fatalf("No documents returned at iteration %d, search %d", i, j)
			}

			// Force result to go out of scope
			result = nil
		}

		// Periodic cleanup and progress reporting
		if i%100 == 0 {
			runtime.GC()
			var currentStats runtime.MemStats
			runtime.ReadMemStats(&currentStats)
			t.Logf("Iteration %d/%d: HeapAlloc=%d KB, NumGC=%d",
				i, iterations, currentStats.HeapAlloc/1024, currentStats.NumGC)
		}
	}

	// Final memory check
	runtime.GC()
	runtime.ReadMemStats(&finalStats)

	// Calculate memory growth
	heapGrowth := int64(finalStats.HeapAlloc) - int64(initialStats.HeapAlloc)
	totalAllocGrowth := int64(finalStats.TotalAlloc) - int64(initialStats.TotalAlloc)

	t.Logf("Memory leak test completed:")
	t.Logf("  Operations: %d searches", iterations*searchesPerIteration)
	t.Logf("  Initial HeapAlloc: %d KB", initialStats.HeapAlloc/1024)
	t.Logf("  Final HeapAlloc: %d KB", finalStats.HeapAlloc/1024)
	t.Logf("  Heap Growth: %d KB", heapGrowth/1024)
	t.Logf("  Total Alloc Growth: %d KB", totalAllocGrowth/1024)
	t.Logf("  GC Runs: %d", finalStats.NumGC-initialStats.NumGC)

	// Check for excessive memory growth
	// Allow up to 100MB heap growth and 1GB total allocation growth
	if heapGrowth > 100*1024*1024 {
		t.Errorf("Excessive heap growth: %d MB", heapGrowth/(1024*1024))
	}
	if totalAllocGrowth > 1024*1024*1024 {
		t.Errorf("Excessive total allocation growth: %d MB", totalAllocGrowth/(1024*1024))
	}
}

// TestSearchSimilar_ConcurrentStress tests concurrent search operations
func TestSearchSimilar_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent stress test in short mode")
	}

	env := getOrCreateSearchTestEnvironment(t)
	testDataSet := getOrCreateTestDataSet(t, "en")

	if len(testDataSet.Documents) == 0 {
		t.Skip("No test documents available for concurrent stress test")
	}

	ctx := context.Background()

	// Test parameters
	numGoroutines := 20
	operationsPerGoroutine := 50

	// Different test scenarios
	testScenarios := []struct {
		name string
		opts func(int) *types.SearchOptions
	}{
		{
			name: "basic",
			opts: func(i int) *types.SearchOptions {
				queryVector := testDataSet.Documents[i%len(testDataSet.Documents)].Vector
				return &types.SearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              10,
				}
			},
		},
		{
			name: "paginated",
			opts: func(i int) *types.SearchOptions {
				queryVector := testDataSet.Documents[i%len(testDataSet.Documents)].Vector
				return &types.SearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              20,
					Page:           (i % 3) + 1,
					PageSize:       5,
				}
			},
		},
		{
			name: "filtered",
			opts: func(i int) *types.SearchOptions {
				queryVector := testDataSet.Documents[i%len(testDataSet.Documents)].Vector
				return &types.SearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              10,
					Filter: map[string]interface{}{
						"language": "en",
					},
				}
			},
		},
		{
			name: "high_k",
			opts: func(i int) *types.SearchOptions {
				queryVector := testDataSet.Documents[i%len(testDataSet.Documents)].Vector
				return &types.SearchOptions{
					CollectionName: testDataSet.CollectionName,
					QueryVector:    queryVector,
					K:              50,
				}
			},
		},
	}

	for _, scenario := range testScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			var wg sync.WaitGroup
			errors := make(chan error, numGoroutines*operationsPerGoroutine)
			results := make(chan *types.SearchResult, numGoroutines*operationsPerGoroutine)

			startTime := time.Now()

			// Launch concurrent goroutines
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()

					for j := 0; j < operationsPerGoroutine; j++ {
						opID := goroutineID*operationsPerGoroutine + j
						opts := scenario.opts(opID)

						result, err := env.Store.SearchSimilar(ctx, opts)
						if err != nil {
							errors <- fmt.Errorf("goroutine %d, operation %d: %w", goroutineID, j, err)
							continue
						}

						if result == nil {
							errors <- fmt.Errorf("goroutine %d, operation %d: nil result", goroutineID, j)
							continue
						}

						results <- result
					}
				}(i)
			}

			wg.Wait()
			close(errors)
			close(results)

			duration := time.Since(startTime)

			// Collect results and errors
			var errorList []error
			var resultList []*types.SearchResult

			for err := range errors {
				errorList = append(errorList, err)
			}

			for result := range results {
				resultList = append(resultList, result)
			}

			// Calculate statistics
			totalOperations := numGoroutines * operationsPerGoroutine
			successfulOperations := len(resultList)
			errorRate := float64(len(errorList)) / float64(totalOperations) * 100
			opsPerSecond := float64(successfulOperations) / duration.Seconds()

			t.Logf("Concurrent stress test (%s) completed:", scenario.name)
			t.Logf("  Total operations: %d", totalOperations)
			t.Logf("  Successful operations: %d", successfulOperations)
			t.Logf("  Errors: %d", len(errorList))
			t.Logf("  Error rate: %.2f%%", errorRate)
			t.Logf("  Duration: %v", duration)
			t.Logf("  Operations per second: %.2f", opsPerSecond)

			// Verify that error rate is reasonable (allow up to 5% error rate)
			if errorRate > 5.0 {
				t.Errorf("Error rate too high: %.2f%% (expected <= 5%%)", errorRate)

				// Log first few errors for debugging
				for i, err := range errorList {
					if i >= 5 {
						t.Logf("... and %d more errors", len(errorList)-5)
						break
					}
					t.Logf("Error %d: %v", i+1, err)
				}
			}

			// Verify that results are reasonable
			for i, result := range resultList {
				if i >= 10 { // Only check first 10 results
					break
				}

				if len(result.Documents) == 0 {
					t.Errorf("Result %d has no documents", i)
				}

				// Verify score ordering
				for j := 1; j < len(result.Documents); j++ {
					if result.Documents[j-1].Score < result.Documents[j].Score {
						t.Errorf("Result %d: documents not ordered by score at positions %d, %d", i, j-1, j)
						break
					}
				}
			}
		})
	}
}

// TestSearchSimilar_ConcurrentWithDifferentCollections tests concurrent access to different collections
func TestSearchSimilar_ConcurrentWithDifferentCollections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent collections test in short mode")
	}

	env := getOrCreateSearchTestEnvironment(t)
	enDataSet := getOrCreateTestDataSet(t, "en")
	zhDataSet := getOrCreateTestDataSet(t, "zh")

	if len(enDataSet.Documents) == 0 || len(zhDataSet.Documents) == 0 {
		t.Skip("Not enough test documents for concurrent collections test")
	}

	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent searches on English collection
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			queryVector := enDataSet.Documents[idx%len(enDataSet.Documents)].Vector
			opts := &types.SearchOptions{
				CollectionName: enDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
			}

			result, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				errors <- fmt.Errorf("EN search %d: %w", idx, err)
				return
			}

			if len(result.Documents) == 0 {
				errors <- fmt.Errorf("EN search %d: no results", idx)
			}
		}(i)
	}

	// Concurrent searches on Chinese collection
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			queryVector := zhDataSet.Documents[idx%len(zhDataSet.Documents)].Vector
			opts := &types.SearchOptions{
				CollectionName: zhDataSet.CollectionName,
				QueryVector:    queryVector,
				K:              5,
			}

			result, err := env.Store.SearchSimilar(ctx, opts)
			if err != nil {
				errors <- fmt.Errorf("ZH search %d: %w", idx, err)
				return
			}

			if len(result.Documents) == 0 {
				errors <- fmt.Errorf("ZH search %d: no results", idx)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		t.Errorf("Concurrent collections test had %d errors:", len(errorList))
		for i, err := range errorList {
			if i >= 5 {
				t.Logf("... and %d more errors", len(errorList)-5)
				break
			}
			t.Logf("Error %d: %v", i+1, err)
		}
	}
}
