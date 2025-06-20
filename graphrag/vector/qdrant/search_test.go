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

	// Update config dimension for this collection and enable sparse vectors for hybrid search
	config := env.Config
	config.Dimension = dataSet.VectorDim
	config.CollectionName = dataSet.CollectionName
	config.EnableSparseVectors = true  // Enable sparse vectors for hybrid search support
	config.DenseVectorName = "dense"   // Named vector for dense embeddings
	config.SparseVectorName = "sparse" // Named vector for sparse vectors (BM25, TF-IDF, etc.)

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

// =============================================================================
// Utility Functions
// =============================================================================

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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
