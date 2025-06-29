package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/chunking"
	"github.com/yaoapp/gou/graphrag/embedding"
	"github.com/yaoapp/gou/graphrag/extraction"
	"github.com/yaoapp/gou/graphrag/extraction/openai"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/vector/qdrant"
)

func main() {
	var (
		filePath         = flag.String("file", "", "Path to the file to chunk (required for chunking)")
		dirPath          = flag.String("dir", "", "Path to the directory to embed (required for embedding) or extract (required for extraction)")
		size             = flag.Int("size", 300, "Chunk size")
		overlap          = flag.Int("overlap", 50, "Chunk overlap")
		maxDepth         = flag.Int("depth", 3, "Maximum chunk depth")
		maxConcurrent    = flag.Int("concurrent", 6, "Maximum concurrent operations for chunking (default 6), embedding (default 10), or extraction (default 5)")
		method           = flag.String("method", "structured", "Processing method: structured, semantic, both, embedding, extraction, or clear-collections")
		toolcall         = flag.Bool("toolcall", false, "Use toolcall for semantic chunking")
		connector        = flag.String("connector", "openai", "Connector type: openai, fastembed, custom")
		contextSize      = flag.Int("context", 1000, "Context size")
		embeddingModel   = flag.String("embedding-model", "text-embedding-3-small", "Embedding model for embedding method")
		extractionModel  = flag.String("extraction-model", "gpt-4o-mini", "Extraction model for extraction method")
		suffix           = flag.String("suffix", "", "Suffix for embedding/extraction files")
		dimension        = flag.Int("dimension", 1536, "Embedding dimension for embedding method")
		enableEmbedding  = flag.Bool("enable-embedding", false, "Enable embedding for extraction results (default false)")
		clearCollections = flag.Bool("clear-collections", false, "Clear all collections from Qdrant")
		help             = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Validate method parameter
	validMethods := map[string]bool{
		"structured":        true,
		"semantic":          true,
		"both":              true,
		"embedding":         true,
		"extraction":        true,
		"clear-collections": true,
	}
	if !validMethods[*method] {
		fmt.Fprintf(os.Stderr, "Error: Invalid method '%s'. Valid methods are: structured, semantic, both, embedding, extraction, clear-collections\n", *method)
		printHelp()
		os.Exit(1)
	}

	// Handle clear-collections method or flag
	if *method == "clear-collections" || *clearCollections {
		ctx := context.Background()
		if err := runClearCollections(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to clear collections: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n=== Collections cleared successfully ===")
		return
	}

	// Handle embedding method
	if *method == "embedding" {
		if *dirPath == "" {
			fmt.Fprintf(os.Stderr, "Error: -dir flag is required for embedding method\n")
			printHelp()
			os.Exit(1)
		}

		// Check if directory exists
		if _, err := os.Stat(*dirPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Directory %s does not exist\n", *dirPath)
			os.Exit(1)
		}

		// Set default concurrent for embedding if not specified
		if *maxConcurrent == 6 {
			*maxConcurrent = 10 // Default for embedding
		}

		fmt.Printf("Processing directory: %s\n", *dirPath)
		fmt.Printf("Embedding model: %s\n", *embeddingModel)
		fmt.Printf("Dimension: %d\n", *dimension)
		fmt.Printf("Concurrent: %d\n", *maxConcurrent)

		ctx := context.Background()
		if err := runEmbedding(ctx, *dirPath, *connector, *embeddingModel, *dimension, *maxConcurrent, *suffix); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Embedding failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n=== Embedding completed successfully ===")
		return
	}

	// Handle extraction method
	if *method == "extraction" {
		if *dirPath == "" {
			fmt.Fprintf(os.Stderr, "Error: -dir flag is required for extraction method\n")
			printHelp()
			os.Exit(1)
		}

		// Check if directory exists
		if _, err := os.Stat(*dirPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Directory %s does not exist\n", *dirPath)
			os.Exit(1)
		}

		// Set default concurrent for extraction if not specified
		if *maxConcurrent == 6 {
			*maxConcurrent = 5 // Default for extraction
		}

		fmt.Printf("Processing directory: %s\n", *dirPath)
		fmt.Printf("Extraction model: %s\n", *extractionModel)
		fmt.Printf("Concurrent: %d\n", *maxConcurrent)
		if *enableEmbedding {
			fmt.Printf("Embedding enabled: text-embedding-3-small (dimension: %d)\n", *dimension)
		}

		ctx := context.Background()
		if err := runExtraction(ctx, *dirPath, *connector, *extractionModel, *maxConcurrent, *suffix, *enableEmbedding, *embeddingModel, *dimension); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Extraction failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n=== Extraction completed successfully ===")
		return
	}

	// For chunking methods, require file parameter
	if *filePath == "" {
		fmt.Fprintf(os.Stderr, "Error: -file flag is required for chunking methods\n")
		printHelp()
		os.Exit(1)
	}

	// Check if file exists
	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File %s does not exist\n", *filePath)
		os.Exit(1)
	}

	// Get file info
	fileInfo, err := os.Stat(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot get file info: %v\n", err)
		os.Exit(1)
	}

	// Parse filename
	dir := filepath.Dir(*filePath)
	fullName := fileInfo.Name()
	ext := filepath.Ext(fullName)
	basename := strings.TrimSuffix(fullName, ext)

	// Create output directories based on method
	var semanticDir, structuredDir string
	if *method == "structured" || *method == "both" {
		structuredDir = filepath.Join(dir, "structured")
	}
	if *method == "semantic" || *method == "both" {
		semanticDir = filepath.Join(dir, "semantic")
	}

	if err := setupOutputDirectories(semanticDir, structuredDir, *method); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to setup output directories: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processing file: %s\n", *filePath)
	fmt.Printf("Basename: %s, Extension: %s\n", basename, ext)
	fmt.Printf("Chunking method: %s\n", *method)
	if structuredDir != "" {
		fmt.Printf("Structured output directory: %s\n", structuredDir)
	}
	if semanticDir != "" {
		fmt.Printf("Semantic output directory: %s\n", semanticDir)
	}

	// Create OpenAI connector for semantic chunking if needed
	if *method == "semantic" || *method == "both" {

		// Set toolcall to false if connector is not openai
		if *connector == "openai" {
			*toolcall = true
		}

		_, err = createAICConnector(*connector)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to create OpenAI connector: %v\n", err)
			os.Exit(1)
		}
	}

	ctx := context.Background()

	// Execute chunking based on method
	switch *method {
	case "structured":
		fmt.Println("\n=== Running Structured Chunking ===")
		if err := runStructuredChunking(ctx, *filePath, basename, ext, structuredDir, *size, *overlap, *maxDepth, *maxConcurrent); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Structured chunking failed: %v\n", err)
			os.Exit(1)
		}

	case "semantic":
		fmt.Println("\n=== Running Semantic Chunking ===")
		if err := runSemanticChunking(ctx, *filePath, basename, ext, semanticDir, "openai-chunking", *size, *overlap, *maxDepth, *maxConcurrent, *toolcall, *contextSize); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Semantic chunking failed: %v\n", err)
			os.Exit(1)
		}

	case "both":
		// Process structured chunking first
		fmt.Println("\n=== Running Structured Chunking ===")
		if err := runStructuredChunking(ctx, *filePath, basename, ext, structuredDir, *size, *overlap, *maxDepth, *maxConcurrent); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Structured chunking failed: %v\n", err)
			os.Exit(1)
		}

		// Process semantic chunking
		fmt.Println("\n=== Running Semantic Chunking ===")
		if err := runSemanticChunking(ctx, *filePath, basename, ext, semanticDir, "openai-chunking", *size, *overlap, *maxDepth, *maxConcurrent, *toolcall, *contextSize); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Semantic chunking failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("\n=== Chunking completed successfully ===")
}

func printHelp() {
	fmt.Println("GraphRAG Chunking, Embedding & Extraction Tool")
	fmt.Println("Usage: go run tools.go [options]")
	fmt.Println()
	fmt.Println("For chunking methods:")
	fmt.Println("  go run tools.go -file <path> [options]")
	fmt.Println()
	fmt.Println("For embedding method:")
	fmt.Println("  go run tools.go -method embedding -dir <path> [options]")
	fmt.Println()
	fmt.Println("For extraction method:")
	fmt.Println("  go run tools.go -method extraction -dir <path> [options]")
	fmt.Println()
	fmt.Println("For clearing collections:")
	fmt.Println("  go run tools.go -method clear-collections")
	fmt.Println("  go run tools.go -clear-collections")
	fmt.Println()
	fmt.Println("Required flags:")
	fmt.Println("  -file string    Path to the file to chunk (required for chunking methods)")
	fmt.Println("  -dir string     Path to the directory to embed/extract (required for embedding/extraction methods)")
	fmt.Println()
	fmt.Println("Optional flags:")
	fmt.Println("  -method string        Processing method: structured, semantic, both, embedding, extraction, or clear-collections (default structured)")
	fmt.Println("  -size int             Chunk size (default 300)")
	fmt.Println("  -overlap int          Chunk overlap (default 50)")
	fmt.Println("  -depth int            Maximum chunk depth (default 3)")
	fmt.Println("  -concurrent int       Maximum concurrent operations - chunking (default 6), embedding (default 10), extraction (default 5)")
	fmt.Println("  -toolcall             Use toolcall for semantic chunking (default false)")
	fmt.Println("  -connector string     Connector type: openai (with toolcall), custom (without toolcall) (default openai)")
	fmt.Println("  -context int          Context size (default 1000)")
	fmt.Println("  -embedding-model string Embedding model for embedding method (default text-embedding-3-small)")
	fmt.Println("  -extraction-model string Extraction model for extraction method (default gpt-4o-mini)")
	fmt.Println("  -dimension int        Embedding dimension for embedding method (default 1536)")
	fmt.Println("  -enable-embedding     Enable embedding for extraction results using OpenAI text-embedding-3-small (default false)")
	fmt.Println("  -suffix string        Suffix for embedding/extraction files (default json)")
	fmt.Println("  -clear-collections    Clear all collections from Qdrant")
	fmt.Println("  -help                 Show this help message")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  OPENAI_TEST_KEY       OpenAI API key for semantic chunking, embedding, and extraction")
	fmt.Println("  RAG_LLM_TEST_KEY      Custom LLM API key")
	fmt.Println("  RAG_LLM_TEST_URL      Custom LLM API URL")
	fmt.Println("  RAG_LLM_TEST_SMODEL   Custom LLM model name")
	fmt.Println("  QDRANT_TEST_HOST      Qdrant server host (default 127.0.0.1)")
	fmt.Println("  QDRANT_TEST_PORT      Qdrant server port (default 6334)")
	fmt.Println()
	fmt.Println("Output:")
	fmt.Println("  Chunking files: basename.chunk-index.ext")
	fmt.Println("  Structured chunks: <dir>/structured/")
	fmt.Println("  Semantic chunks: <dir>/semantic/")
	fmt.Println("  Position mapping: basename.mapping.json")
	fmt.Println("  Embedding files: original-filename.json")
	fmt.Println("  Extraction files: original-filename.json")
}

func setupOutputDirectories(semanticDir, structuredDir, method string) error {
	var dirs []string

	// Only add directories that need to be created based on method
	if method == "structured" || method == "both" {
		if structuredDir != "" {
			dirs = append(dirs, structuredDir)
		}
	}
	if method == "semantic" || method == "both" {
		if semanticDir != "" {
			dirs = append(dirs, semanticDir)
		}
	}

	// Remove existing directories if they exist
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err == nil {
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("failed to remove existing directory %s: %w", dir, err)
			}
			fmt.Printf("Cleared existing directory: %s\n", dir)
		}
	}

	// Create directories
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("Created directory: %s\n", dir)
	}

	return nil
}

func createAICConnector(name string) (connector.Connector, error) {
	switch name {
	case "openai":
		return createOpenaiConnector()
	case "fastembed":
		return createFastEmbedConnector()
	case "custom":
		return createCustomConnector()
	}
	return nil, fmt.Errorf("invalid connector type: %s", name)
}

func createCustomConnector() (connector.Connector, error) {
	apiKey := os.Getenv("RAG_LLM_TEST_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("RAG_LLM_TEST_KEY environment variable is not set")
	}
	url := os.Getenv("RAG_LLM_TEST_URL")
	if url == "" {
		return nil, fmt.Errorf("RAG_LLM_TEST_URL environment variable is not set")
	}

	model := os.Getenv("RAG_LLM_TEST_SMODEL")
	if model == "" {
		return nil, fmt.Errorf("RAG_LLM_TEST_SMODEL environment variable is not set")
	}

	dsl := map[string]interface{}{
		"name":    "openai-chunking",
		"type":    "openai",
		"options": map[string]interface{}{"key": apiKey, "proxy": url, "model": model},
	}

	dslBytes, err := json.Marshal(dsl)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connector DSL: %w", err)
	}

	conn, err := connector.New("openai", "openai-chunking", dslBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI connector: %w", err)
	}

	return conn, nil
}

func createOpenaiConnector() (connector.Connector, error) {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_TEST_KEY environment variable is not set")
	}

	// Create connector DSL
	dsl := map[string]interface{}{
		"name":    "openai-chunking",
		"type":    "openai",
		"options": map[string]interface{}{"key": apiKey, "model": "gpt-4o-mini"},
	}

	dslBytes, err := json.Marshal(dsl)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connector DSL: %w", err)
	}

	// Create new connector
	conn, err := connector.New("openai", "openai-chunking", dslBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI connector: %w", err)
	}

	return conn, nil
}

func createFastEmbedConnector() (connector.Connector, error) {
	apiKey := os.Getenv("FASTEMBED_TEST_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("FASTEMBED_TEST_KEY environment variable is not set")
	}

	host := os.Getenv("FASTEMBED_TEST_HOST")
	if host == "" {
		return nil, fmt.Errorf("FASTEMBED_TEST_HOST environment variable is not set")
	}

	dsl := map[string]interface{}{
		"name":    "fastembed",
		"type":    "fastembed",
		"options": map[string]interface{}{"key": apiKey, "host": host, "model": "BAAI/bge-small-en-v1.5"},
	}

	dslBytes, err := json.Marshal(dsl)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connector DSL: %w", err)
	}

	conn, err := connector.New("fastembed", "fastembed", dslBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create FastEmbed connector: %w", err)
	}

	return conn, nil
}

func createEmbeddingConnector() (connector.Connector, error) {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_TEST_KEY environment variable is not set")
	}

	// Create connector DSL specifically for embedding
	dsl := map[string]interface{}{
		"name": "openai-embedding",
		"type": "openai",
		"options": map[string]interface{}{
			"key":   apiKey,
			"model": "text-embedding-3-small",
		},
	}

	dslBytes, err := json.Marshal(dsl)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding connector DSL: %w", err)
	}

	// Create new connector
	conn, err := connector.New("openai", "openai-embedding", dslBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI embedding connector: %w", err)
	}

	return conn, nil
}

func runStructuredChunking(ctx context.Context, filePath, basename, ext, outputDir string, size, overlap, maxDepth, maxConcurrent int) error {

	start := time.Now()
	chunker := chunking.NewStructuredChunker()

	options := &types.ChunkingOptions{
		Size:          size,
		Overlap:       overlap,
		MaxDepth:      maxDepth,
		MaxConcurrent: maxConcurrent,
	}

	var chunks []*types.Chunk
	var mu sync.Mutex

	// Create mapping file for position information
	mappingFile := filepath.Join(outputDir, fmt.Sprintf("%s.mapping.json", basename))

	callback := func(chunk *types.Chunk) error {
		mu.Lock()
		defer mu.Unlock()

		chunks = append(chunks, chunk)

		// Generate filename: basename.chunk-index.ext
		filename := fmt.Sprintf("%s.%d.chunk-%d%s", basename, chunk.Depth, chunk.Index, ext)
		filepath := filepath.Join(outputDir, filename)

		// Write chunk to file
		if err := os.WriteFile(filepath, []byte(chunk.Text), 0644); err != nil {
			return fmt.Errorf("failed to write chunk file %s: %w", filepath, err)
		}

		fmt.Printf("  Structured chunk %d: %s (depth: %d, size: %d)\n", chunk.Index, filename, chunk.Depth, len(chunk.Text))

		return nil
	}

	if err := chunker.ChunkFile(ctx, filePath, options, callback); err != nil {
		return fmt.Errorf("structured chunking failed: %w", err)
	}

	// Write position mapping file
	if err := writePositionMapping(mappingFile, chunks); err != nil {
		return fmt.Errorf("failed to write position mapping: %w", err)
	}

	cost := time.Since(start)
	fmt.Printf("\n--------------------------------\n")
	fmt.Printf("Structured chunking completed: %d chunks generated in %s\n", len(chunks), cost.Round(time.Microsecond))
	fmt.Printf("Position mapping saved to: %s\n", mappingFile)
	fmt.Printf("--------------------------------\n")
	fmt.Printf("Chunks Count: %d\n", len(chunks))
	fmt.Printf("Size: %d\n", size)
	fmt.Printf("Overlap: %d\n", overlap)
	fmt.Printf("Depth: %d\n", maxDepth)
	fmt.Printf("Concurrent: %d\n", maxConcurrent)
	fmt.Printf("Time Cost: %s\n", cost)
	fmt.Printf("--------------------------------\n")
	return nil
}

// ChunkMappingInfo represents position mapping information for a chunk
type ChunkMappingInfo struct {
	ID            string               `json:"id"`
	Index         int                  `json:"index"`
	Depth         int                  `json:"depth"`
	ParentID      string               `json:"parent_id,omitempty"`
	Filename      string               `json:"filename"`
	TextSize      int                  `json:"text_size"`
	IsLeaf        bool                 `json:"is_leaf"`
	IsRoot        bool                 `json:"is_root"`
	TextPosition  *types.TextPosition  `json:"text_position,omitempty"`
	MediaPosition *types.MediaPosition `json:"media_position,omitempty"`
	Parents       []ChunkParentInfo    `json:"parents,omitempty"`
}

// ChunkParentInfo represents parent chunk information
type ChunkParentInfo struct {
	ID    string `json:"id"`
	Depth int    `json:"depth"`
	Index int    `json:"index"`
}

// writePositionMapping writes the position mapping information to a JSON file
func writePositionMapping(mappingFile string, chunks []*types.Chunk) error {
	var mappingInfos []ChunkMappingInfo

	for _, chunk := range chunks {
		// Get parent info
		var parents []ChunkParentInfo
		for _, parent := range chunk.Parents {
			parents = append(parents, ChunkParentInfo{
				ID:    parent.ID,
				Depth: parent.Depth,
				Index: parent.Index,
			})
		}

		// Generate filename for this chunk
		filename := fmt.Sprintf("%s.%d.chunk-%d",
			strings.TrimSuffix(filepath.Base(mappingFile), ".mapping.json"),
			chunk.Depth,
			chunk.Index)

		mappingInfo := ChunkMappingInfo{
			ID:            chunk.ID,
			Index:         chunk.Index,
			Depth:         chunk.Depth,
			ParentID:      chunk.ParentID,
			Filename:      filename,
			TextSize:      len(chunk.Text),
			IsLeaf:        chunk.Leaf,
			IsRoot:        chunk.Root,
			TextPosition:  chunk.TextPos,
			MediaPosition: chunk.MediaPos,
			Parents:       parents,
		}

		mappingInfos = append(mappingInfos, mappingInfo)
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(mappingInfos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mapping data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(mappingFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write mapping file: %w", err)
	}

	return nil
}

func runSemanticChunking(ctx context.Context, filePath, basename, ext, outputDir string, conn string, size, overlap, maxDepth, maxConcurrent int, toolcall bool, contextSize int) error {

	// Progress callback for semantic chunking
	progressCallback := func(chunkID, progress, step string, data interface{}) error {
		fmt.Printf("  Semantic progress [%s]: %s - %s\n", chunkID, progress, step)
		fmt.Println("--------------------------------")
		raw, _ := json.MarshalIndent(data, "", "  ")
		fmt.Printf("Positions %s\n", string(raw))
		fmt.Println("--------------------------------")
		return nil
	}

	chunker := chunking.NewSemanticChunker(progressCallback)

	options := &types.ChunkingOptions{
		Size:          size,
		Overlap:       overlap,
		MaxDepth:      maxDepth,
		MaxConcurrent: maxConcurrent,
		SemanticOptions: &types.SemanticOptions{
			Connector:     conn,
			ContextSize:   contextSize,
			Options:       `{"temperature": 0.1}`,
			Prompt:        "", // Use default prompt
			Toolcall:      toolcall,
			MaxRetry:      3,
			MaxConcurrent: maxConcurrent,
		},
	}

	var chunks []*types.Chunk
	var mu sync.Mutex

	// Create mapping file for position information
	mappingFile := filepath.Join(outputDir, fmt.Sprintf("%s.mapping.json", basename))

	callback := func(chunk *types.Chunk) error {
		mu.Lock()
		defer mu.Unlock()

		chunks = append(chunks, chunk)

		// Generate filename: basename.chunk-index.ext
		filename := fmt.Sprintf("%s.%d.chunk-%d%s", basename, chunk.Depth, chunk.Index, ext)
		filepath := filepath.Join(outputDir, filename)

		// Write chunk to file
		if err := os.WriteFile(filepath, []byte(chunk.Text), 0644); err != nil {
			return fmt.Errorf("failed to write chunk file %s: %w", filepath, err)
		}

		color.Green("  Semantic chunk %d: %s (depth: %d, size: %d)\n", chunk.Index, filename, chunk.Depth, len(chunk.Text))
		return nil
	}

	if err := chunker.ChunkFile(ctx, filePath, options, callback); err != nil {
		return fmt.Errorf("semantic chunking failed: %w", err)
	}

	// Write position mapping file
	if err := writePositionMapping(mappingFile, chunks); err != nil {
		return fmt.Errorf("failed to write position mapping: %w", err)
	}

	fmt.Printf("\n--------------------------------\n")
	fmt.Printf("Semantic chunking completed: %d chunks generated\n", len(chunks))
	fmt.Printf("Position mapping saved to: %s\n", mappingFile)
	fmt.Printf("--------------------------------\n")
	fmt.Printf("Chunks Count: %d\n", len(chunks))
	fmt.Printf("Size: %d\n", size)
	fmt.Printf("Overlap: %d\n", overlap)
	fmt.Printf("Depth: %d\n", maxDepth)
	fmt.Printf("Concurrent: %d\n", maxConcurrent)
	fmt.Printf("Toolcall: %t\n", toolcall)
	fmt.Printf("Context Size: %d\n", options.SemanticOptions.ContextSize)
	fmt.Printf("--------------------------------\n")
	return nil
}

// runEmbedding processes all files in a directory and generates embeddings using batch processing
func runEmbedding(ctx context.Context, dirPath, connectorType, model string, dimension, concurrent int, suffix string) error {
	start := time.Now()

	// Create connector
	_, err := createAICConnector(connectorType)
	if err != nil {
		return fmt.Errorf("failed to create connector: %w", err)
	}

	var embedder types.Embedding

	// Create embedding instance
	switch connectorType {
	case "openai":
		embedder, err = embedding.NewOpenai(embedding.OpenaiOptions{
			ConnectorName: getConnectorName(connectorType),
			Concurrent:    concurrent,
			Dimension:     dimension,
			Model:         model,
		})

	case "fastembed":
		c, err := createFastEmbedConnector()
		if err != nil {
			return fmt.Errorf("failed to create connector: %w", err)
		}

		setting := c.Setting()
		if model == "" {
			model = setting["model"].(string)
		}

		embedder, err = embedding.NewFastEmbed(embedding.FastEmbedOptions{
			ConnectorName: getConnectorName(connectorType),
			Concurrent:    concurrent,
			Dimension:     dimension,
			Model:         model,
			Host:          setting["host"].(string),
			Key:           setting["key"].(string),
		})
	}

	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}

	// Find all files in directory
	files, err := findFilesInDirectory(dirPath)
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No files found in directory")
		return nil
	}

	fmt.Printf("Found %d files to process\n", len(files))

	// Read all file contents
	fmt.Println("Reading file contents...")
	fileContents := make([]string, 0, len(files))
	validFiles := make([]string, 0, len(files))
	errorCount := 0

	for _, filePath := range files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			color.Red("  Error reading %s: %v\n", filepath.Base(filePath), err)
			errorCount++
			continue
		}

		text := string(content)
		if text == "" {
			color.Yellow("  Skipping empty file: %s\n", filepath.Base(filePath))
			continue
		}

		fileContents = append(fileContents, text)
		validFiles = append(validFiles, filePath)
		fmt.Printf("  Read %s (%d chars)\n", filepath.Base(filePath), len(text))
	}

	if len(fileContents) == 0 {
		fmt.Println("No valid files to process")
		return nil
	}

	fmt.Printf("Processing %d valid files...\n", len(fileContents))

	// Create progress callback for batch processing
	progressCallback := func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
		switch status {
		case types.EmbeddingStatusStarting:
			color.Cyan("Starting batch embedding (%s): %s\n", connectorType, payload.Message)
		case types.EmbeddingStatusProcessing:
			if payload.DocumentIndex != nil {
				color.Yellow("Processing document %d/%d: %s\n",
					payload.Current, payload.Total, filepath.Base(validFiles[*payload.DocumentIndex]))
			} else {
				color.Yellow("Processing: %s (%d/%d)\n", payload.Message, payload.Current, payload.Total)
			}
		case types.EmbeddingStatusCompleted:
			color.Green("Batch embedding completed: %s\n", payload.Message)
		case types.EmbeddingStatusError:
			if payload.DocumentIndex != nil {
				color.Red("Error processing document %d: %s\n", *payload.DocumentIndex, payload.Message)
			} else {
				color.Red("Error: %s\n", payload.Message)
			}
		}
	}

	// Batch embed all documents
	embeddingResults, err := embedder.EmbedDocuments(ctx, fileContents, progressCallback)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	if embeddingResults.Count() != len(validFiles) {
		return fmt.Errorf("embedding count mismatch: got %d embeddings for %d files", embeddingResults.Count(), len(validFiles))
	}

	// Save embeddings to files
	fmt.Println("Saving embedding files...")
	for i, filePath := range validFiles {
		if err := saveEmbeddingFile(filePath, fileContents[i], embeddingResults, i, embedder, suffix); err != nil {
			color.Red("  Error saving %s: %v\n", filepath.Base(filePath), err)
			errorCount++
		} else {
			color.Green("  Saved %s -> %s.json\n", filepath.Base(filePath), filepath.Base(filePath))
		}
	}

	cost := time.Since(start)
	fmt.Printf("\n--------------------------------\n")
	fmt.Printf("Embedding completed: %d files processed, %d errors in %s\n", len(validFiles), errorCount, cost.Round(time.Millisecond))
	fmt.Printf("--------------------------------\n")
	fmt.Printf("Directory: %s\n", dirPath)
	fmt.Printf("Model: %s\n", embeddingResults.Model)
	fmt.Printf("Embedding Type: %s\n", embeddingResults.Type)
	fmt.Printf("Dimension: %d\n", dimension)
	fmt.Printf("Concurrent: %d (used by EmbedDocuments internally)\n", concurrent)
	fmt.Printf("Total Tokens: %d\n", embeddingResults.Usage.TotalTokens)
	fmt.Printf("Total Texts: %d\n", embeddingResults.Usage.TotalTexts)
	fmt.Printf("Time Cost: %s\n", cost)
	fmt.Printf("--------------------------------\n")

	return nil
}

// findFilesInDirectory recursively finds all files in a directory
func findFilesInDirectory(dirPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Skip JSON files (to avoid processing already generated embedding files)
		if strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// saveEmbeddingFile saves embedding data to a JSON file (supports both dense and sparse embeddings)
func saveEmbeddingFile(filePath, text string, embeddingResults *types.EmbeddingResults, index int, embedder types.Embedding, suffix string) error {
	// Prepare base embedding data
	embeddingData := map[string]interface{}{
		"file":         filepath.Base(filePath),
		"full_path":    filePath,
		"model":        embeddingResults.Model,
		"dimension":    embedder.GetDimension(),
		"text_length":  len(text),
		"type":         string(embeddingResults.Type),
		"generated_at": time.Now().Format(time.RFC3339),
		"usage":        embeddingResults.Usage,
	}

	// Add embedding data based on type
	if embeddingResults.Type == types.EmbeddingTypeDense {
		denseEmbeddings := embeddingResults.GetDenseEmbeddings()
		if index < len(denseEmbeddings) {
			embeddingData["embedding"] = denseEmbeddings[index]
		} else {
			return fmt.Errorf("dense embedding index %d out of range (have %d)", index, len(denseEmbeddings))
		}
	} else if embeddingResults.Type == types.EmbeddingTypeSparse {
		sparseEmbeddings := embeddingResults.GetSparseEmbeddings()
		if index < len(sparseEmbeddings) {
			sparse := sparseEmbeddings[index]
			embeddingData["sparse_embedding"] = map[string]interface{}{
				"indices": sparse.Indices,
				"values":  sparse.Values,
			}
		} else {
			return fmt.Errorf("sparse embedding index %d out of range (have %d)", index, len(sparseEmbeddings))
		}
	} else {
		return fmt.Errorf("unsupported embedding type: %s", embeddingResults.Type)
	}

	if suffix == "" {
		suffix = "json"
	}

	// Generate output filename
	dir := filepath.Dir(filePath)
	filename := filepath.Base(filePath)
	outputFile := filepath.Join(dir, filename+"."+suffix)

	// Write embedding to JSON file
	jsonData, err := json.MarshalIndent(embeddingData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal embedding data: %w", err)
	}

	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write embedding file: %w", err)
	}

	return nil
}

// getConnectorName returns the appropriate connector name based on type
func getConnectorName(connectorType string) string {
	switch connectorType {
	case "openai":
		return "openai-chunking"
	case "fastembed":
		return "fastembed"
	case "custom":
		return "openai-chunking"
	default:
		return "openai-chunking"
	}
}

// runClearCollections connects to Qdrant and clears all collections
func runClearCollections(ctx context.Context) error {
	start := time.Now()

	// Get Qdrant connection parameters from environment variables
	host := os.Getenv("QDRANT_TEST_HOST")
	if host == "" {
		host = "127.0.0.1" // Default host
	}

	portStr := os.Getenv("QDRANT_TEST_PORT")
	if portStr == "" {
		portStr = "6334" // Default port
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid QDRANT_TEST_PORT value '%s': %w", portStr, err)
	}

	fmt.Printf("Connecting to Qdrant at %s:%d\n", host, port)

	// Create Qdrant store
	store := qdrant.NewStore()

	// Create config with connection parameters in ExtraParams
	config := types.VectorStoreConfig{
		CollectionName: "dummy", // Required field, but not used for connection
		Dimension:      1,       // Required field, but not used for connection
		ExtraParams: map[string]interface{}{
			"host": host,
			"port": port,
		},
	}

	// Connect to Qdrant
	if err := store.Connect(ctx, config); err != nil {
		return fmt.Errorf("failed to connect to Qdrant: %w", err)
	}
	defer store.Close()

	// List all collections
	fmt.Println("Listing all collections...")
	collections, err := store.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	if len(collections) == 0 {
		color.Yellow("No collections found to clear.\n")
		return nil
	}

	fmt.Printf("Found %d collections to clear:\n", len(collections))
	for i, collection := range collections {
		fmt.Printf("  %d. %s\n", i+1, collection)
	}

	// Drop all collections
	fmt.Println("\nClearing collections...")
	errorCount := 0
	for i, collection := range collections {
		fmt.Printf("  Dropping collection (%d/%d): %s", i+1, len(collections), collection)

		if err := store.DropCollection(ctx, collection); err != nil {
			color.Red(" - FAILED: %v\n", err)
			errorCount++
		} else {
			color.Green(" - OK\n")
		}
	}

	cost := time.Since(start)
	fmt.Printf("\n--------------------------------\n")

	if errorCount > 0 {
		color.Red("Clear collections completed with %d errors in %s\n", errorCount, cost.Round(time.Millisecond))
		fmt.Printf("Successfully cleared: %d/%d collections\n", len(collections)-errorCount, len(collections))
	} else {
		color.Green("All collections cleared successfully in %s\n", cost.Round(time.Millisecond))
		fmt.Printf("Total collections cleared: %d\n", len(collections))
	}

	fmt.Printf("Qdrant Server: %s:%d\n", host, port)
	fmt.Printf("Time Cost: %s\n", cost)
	fmt.Printf("--------------------------------\n")

	if errorCount > 0 {
		return fmt.Errorf("failed to clear %d collections", errorCount)
	}

	return nil
}

// runExtraction processes all chunk files in a directory and generates knowledge graph extractions
func runExtraction(ctx context.Context, dirPath, connectorType, model string, concurrent int, suffix string, enableEmbedding bool, embeddingModel string, dimension int) error {
	start := time.Now()

	// Create connector
	_, err := createAICConnector(connectorType)
	if err != nil {
		return fmt.Errorf("failed to create connector: %w", err)
	}

	var extractor types.Extraction

	// Determine model name for later use
	var actualModel string
	if connectorType == "openai" {
		actualModel = model
	} else if connectorType == "custom" {
		actualModel = os.Getenv("RAG_LLM_TEST_SMODEL")
		if actualModel == "" {
			return fmt.Errorf("RAG_LLM_TEST_SMODEL environment variable is not set for custom connector")
		}
	}

	// Create extraction instance based on connector type
	switch connectorType {
	case "openai", "custom":
		connectorName := getConnectorName(connectorType)

		// Determine toolcall support based on connector type
		var toolcall *bool
		if connectorType == "openai" {
			// OpenAI supports toolcall, use default (true)
			toolcall = nil
		} else if connectorType == "custom" {
			// Custom/local LLM typically doesn't support toolcall
			toolcallDisabled := false
			toolcall = &toolcallDisabled
		}

		extractor, err = openai.NewOpenai(openai.Options{
			ConnectorName: connectorName,
			Concurrent:    concurrent,
			Model:         actualModel,
			Temperature:   0.1, // Low temperature for consistent extraction
			MaxTokens:     4000,
			Toolcall:      toolcall,
			RetryAttempts: 3,
			RetryDelay:    time.Second,
		})
		if err != nil {
			return fmt.Errorf("failed to create OpenAI extractor: %w", err)
		}
	default:
		return fmt.Errorf("unsupported connector type for extraction: %s", connectorType)
	}

	// Find all chunk files in directory (only *.1.chunk-*.txt files)
	files, err := findChunkFilesInDirectory(dirPath)
	if err != nil {
		return fmt.Errorf("failed to find chunk files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No chunk files found in directory")
		return nil
	}

	fmt.Printf("Found %d chunk files to process\n", len(files))

	// Read all file contents
	fmt.Println("Reading chunk file contents...")
	fileContents := make([]string, 0, len(files))
	validFiles := make([]string, 0, len(files))
	errorCount := 0

	for _, filePath := range files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			color.Red("  Error reading %s: %v\n", filepath.Base(filePath), err)
			errorCount++
			continue
		}

		text := string(content)
		if text == "" {
			color.Yellow("  Skipping empty file: %s\n", filepath.Base(filePath))
			continue
		}

		fileContents = append(fileContents, text)
		validFiles = append(validFiles, filePath)
		fmt.Printf("  Read %s (%d chars)\n", filepath.Base(filePath), len(text))
	}

	if len(fileContents) == 0 {
		fmt.Println("No valid chunk files to process")
		return nil
	}

	fmt.Printf("Processing %d valid chunk files...\n", len(fileContents))

	// Show toolcall status
	if connectorType == "openai" {
		fmt.Printf("Using toolcall mode: enabled (OpenAI API)\n")
	} else if connectorType == "custom" {
		fmt.Printf("Using toolcall mode: disabled (Custom LLM)\n")
	}

	// Create progress callback for batch processing
	progressCallback := func(status types.ExtractionStatus, payload types.ExtractionPayload) {
		switch status {
		case types.ExtractionStatusStarting:
			color.Cyan("Starting batch extraction (%s): %s\n", connectorType, payload.Message)
		case types.ExtractionStatusProcessing:
			if payload.DocumentIndex != nil {
				color.Yellow("Processing chunk %d/%d: %s\n",
					payload.Current, payload.Total, filepath.Base(validFiles[*payload.DocumentIndex]))
			} else {
				color.Yellow("Processing: %s (%d/%d)\n", payload.Message, payload.Current, payload.Total)
			}
		case types.ExtractionStatusCompleted:
			color.Green("Batch extraction completed: %s\n", payload.Message)
		case types.ExtractionStatusError:
			if payload.DocumentIndex != nil {
				color.Red("Error processing chunk %d: %s\n", *payload.DocumentIndex, payload.Message)
			} else {
				color.Red("Error: %s\n", payload.Message)
			}
		}
	}

	// Extract entities and relationships from all documents
	extractionResults, err := extractor.ExtractDocuments(ctx, fileContents, progressCallback)
	if err != nil {
		return fmt.Errorf("failed to extract entities and relationships: %w", err)
	}

	// Perform embedding if enabled
	if enableEmbedding {
		fmt.Printf("Embedding enabled: text-embedding-3-small (dimension: %d)\n", dimension)

		// Create dedicated embedding connector
		embeddingConnector, err := createEmbeddingConnector()
		if err != nil {
			return fmt.Errorf("failed to create embedding connector: %w", err)
		}

		// Create embedding instance with dedicated connector
		embedder, err := embedding.NewOpenai(embedding.OpenaiOptions{
			ConnectorName: "openai-embedding",
			Model:         "text-embedding-3-small",
			Dimension:     dimension,
		})
		if err != nil {
			return fmt.Errorf("failed to create embedding instance: %w", err)
		}
		_ = embeddingConnector // Use the connector

		// Create extraction instance with embedding
		extractionInstance := &extraction.Extraction{
			Options: types.ExtractionOptions{
				Use:       extractor,
				Embedding: embedder,
			},
		}

		// Create embedding progress callback
		embeddingProgressCallback := func(status types.EmbeddingStatus, payload types.EmbeddingPayload) {
			switch status {
			case types.EmbeddingStatusStarting:
				color.Cyan("Starting embedding: %s\n", payload.Message)
			case types.EmbeddingStatusProcessing:
				color.Yellow("Embedding progress: %s (%d/%d)\n", payload.Message, payload.Current, payload.Total)
			case types.EmbeddingStatusCompleted:
				color.Green("Embedding completed: %s\n", payload.Message)
			case types.EmbeddingStatusError:
				color.Red("Embedding error: %s\n", payload.Message)
			}
		}

		// Perform embedding on results
		fmt.Println("Performing embedding on extraction results...")
		err = extractionInstance.EmbeddingResults(ctx, extractionResults, embeddingProgressCallback)
		if err != nil {
			color.Red("Warning: Failed to embed extraction results: %v\n", err)
			// Continue processing without failing the entire operation
		} else {
			color.Green("Embedding completed successfully\n")
		}
	}

	// Save extraction results for each file (each result corresponds to its source file by index)
	fmt.Println("Saving extraction result files...")
	successCount := 0

	// Process each file individually - extractionResults[i] corresponds to validFiles[i] and fileContents[i]
	for i, filePath := range validFiles {
		var fileResult *types.ExtractionResult

		// Get the extraction result for this file by index
		if extractionResults != nil && i < len(extractionResults) && extractionResults[i] != nil {
			// Use the result at index i for file at index i
			fileResult = extractionResults[i]
			successCount++
		} else {
			// Create empty result for failed extraction at index i
			fileResult = &types.ExtractionResult{
				Usage: types.ExtractionUsage{
					TotalTokens:  0,
					PromptTokens: 0,
					TotalTexts:   1,
				},
				Model:         actualModel,
				Nodes:         []types.Node{},
				Relationships: []types.Relationship{},
			}
		}

		// Save the result for file at index i using its corresponding content and result
		if err := saveExtractionFile(filePath, fileContents[i], fileResult, i, extractor, suffix); err != nil {
			color.Red("  Error saving %s: %v\n", filepath.Base(filePath), err)
			errorCount++
		} else {
			if len(fileResult.Nodes) > 0 || len(fileResult.Relationships) > 0 {
				color.Green("  Saved %s -> %s.json (%d entities, %d relationships)\n",
					filepath.Base(filePath), filepath.Base(filePath),
					len(fileResult.Nodes), len(fileResult.Relationships))
			} else {
				color.Yellow("  Saved %s -> %s.json (empty - extraction failed)\n",
					filepath.Base(filePath), filepath.Base(filePath))
			}
		}
	}

	// Calculate totals from all results
	totalEntities := 0
	totalRelationships := 0
	totalTokens := 0
	embeddedEntities := 0
	embeddedRelationships := 0
	modelName := actualModel

	for _, result := range extractionResults {
		if result != nil {
			totalEntities += len(result.Nodes)
			totalRelationships += len(result.Relationships)
			totalTokens += result.Usage.TotalTokens
			if result.Model != "" {
				modelName = result.Model
			}

			// Count embedded entities and relationships
			for _, node := range result.Nodes {
				if len(node.EmbeddingVector) > 0 {
					embeddedEntities++
				}
			}
			for _, rel := range result.Relationships {
				if len(rel.EmbeddingVector) > 0 {
					embeddedRelationships++
				}
			}
		}
	}

	cost := time.Since(start)
	fmt.Printf("\n--------------------------------\n")
	fmt.Printf("Extraction completed: %d files processed, %d errors in %s\n", len(validFiles), errorCount, cost.Round(time.Millisecond))
	fmt.Printf("--------------------------------\n")
	fmt.Printf("Directory: %s\n", dirPath)
	fmt.Printf("Model: %s\n", modelName)
	fmt.Printf("Concurrent: %d\n", concurrent)
	fmt.Printf("Total Entities: %d\n", totalEntities)
	fmt.Printf("Total Relationships: %d\n", totalRelationships)
	fmt.Printf("Total Tokens: %d\n", totalTokens)
	fmt.Printf("Total Texts: %d\n", len(extractionResults))

	// Display embedding statistics if enabled
	if enableEmbedding {
		fmt.Printf("Embedding Model: text-embedding-3-small\n")
		fmt.Printf("Embedding Dimension: %d\n", dimension)

		if totalEntities > 0 {
			fmt.Printf("Embedded Entities: %d/%d (%.1f%%)\n", embeddedEntities, totalEntities,
				float64(embeddedEntities)/float64(totalEntities)*100)
		} else {
			fmt.Printf("Embedded Entities: %d/%d (0.0%%)\n", embeddedEntities, totalEntities)
		}

		if totalRelationships > 0 {
			fmt.Printf("Embedded Relationships: %d/%d (%.1f%%)\n", embeddedRelationships, totalRelationships,
				float64(embeddedRelationships)/float64(totalRelationships)*100)
		} else {
			fmt.Printf("Embedded Relationships: %d/%d (0.0%%)\n", embeddedRelationships, totalRelationships)
		}
	}

	fmt.Printf("Time Cost: %s\n", cost)
	fmt.Printf("--------------------------------\n")

	return nil
}

// findChunkFilesInDirectory finds all *.1.chunk-*.txt files in a directory
func findChunkFilesInDirectory(dirPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Only process *.1.chunk-*.txt files
		filename := info.Name()
		if strings.Contains(filename, ".1.chunk-") && strings.HasSuffix(filename, ".txt") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// saveExtractionFile saves extraction data to a JSON file with entities and relationships
func saveExtractionFile(filePath, text string, extractionResults *types.ExtractionResult, index int, extractor types.Extraction, suffix string) error {
	// Prepare extraction data
	extractionData := map[string]interface{}{
		"file":                filepath.Base(filePath),
		"full_path":           filePath,
		"model":               extractionResults.Model,
		"text_length":         len(text),
		"generated_at":        time.Now().Format(time.RFC3339),
		"usage":               extractionResults.Usage,
		"total_entities":      len(extractionResults.Nodes),
		"total_relationships": len(extractionResults.Relationships),
		"entities":            extractionResults.Nodes,
		"relationships":       extractionResults.Relationships,
	}

	if suffix == "" {
		suffix = "json"
	}

	// Generate output filename
	dir := filepath.Dir(filePath)
	filename := filepath.Base(filePath)
	outputFile := filepath.Join(dir, filename+"."+suffix)

	// Write extraction to JSON file
	jsonData, err := json.MarshalIndent(extractionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal extraction data: %w", err)
	}

	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write extraction file: %w", err)
	}

	return nil
}
