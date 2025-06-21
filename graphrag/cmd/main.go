package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/chunking"
	"github.com/yaoapp/gou/graphrag/embedding"
	"github.com/yaoapp/gou/graphrag/types"
)

func main() {
	var (
		filePath       = flag.String("file", "", "Path to the file to chunk (required for chunking)")
		dirPath        = flag.String("dir", "", "Path to the directory to embed (required for embedding)")
		size           = flag.Int("size", 300, "Chunk size")
		overlap        = flag.Int("overlap", 50, "Chunk overlap")
		maxDepth       = flag.Int("depth", 3, "Maximum chunk depth")
		maxConcurrent  = flag.Int("concurrent", 6, "Maximum concurrent operations for chunking (default 6), or for embedding (default 10)")
		method         = flag.String("method", "structured", "Processing method: structured, semantic, both, or embedding")
		toolcall       = flag.Bool("toolcall", false, "Use toolcall for semantic chunking")
		connector      = flag.String("connector", "openai", "Connector type: openai, fastembed, custom")
		contextSize    = flag.Int("context", 1000, "Context size")
		embeddingModel = flag.String("embedding-model", "text-embedding-3-small", "Embedding model for embedding method")
		suffix         = flag.String("suffix", "", "Suffix for embedding files")
		dimension      = flag.Int("dimension", 1536, "Embedding dimension for embedding method")
		help           = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Validate method parameter
	validMethods := map[string]bool{
		"structured": true,
		"semantic":   true,
		"both":       true,
		"embedding":  true,
	}
	if !validMethods[*method] {
		fmt.Fprintf(os.Stderr, "Error: Invalid method '%s'. Valid methods are: structured, semantic, both, embedding\n", *method)
		printHelp()
		os.Exit(1)
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
	fmt.Println("GraphRAG Chunking & Embedding Tool")
	fmt.Println("Usage: go run tools.go [options]")
	fmt.Println()
	fmt.Println("For chunking methods:")
	fmt.Println("  go run tools.go -file <path> [options]")
	fmt.Println()
	fmt.Println("For embedding method:")
	fmt.Println("  go run tools.go -method embedding -dir <path> [options]")
	fmt.Println()
	fmt.Println("Required flags:")
	fmt.Println("  -file string    Path to the file to chunk (required for chunking methods)")
	fmt.Println("  -dir string     Path to the directory to embed (required for embedding method)")
	fmt.Println()
	fmt.Println("Optional flags:")
	fmt.Println("  -method string        Processing method: structured, semantic, both, or embedding (default structured)")
	fmt.Println("  -size int             Chunk size (default 300)")
	fmt.Println("  -overlap int          Chunk overlap (default 50)")
	fmt.Println("  -depth int            Maximum chunk depth (default 3)")
	fmt.Println("  -concurrent int       Maximum concurrent operations - chunking (default 6), embedding (default 10)")
	fmt.Println("  -toolcall             Use toolcall for semantic chunking (default false)")
	fmt.Println("  -connector string     Connector type: openai, custom (default openai)")
	fmt.Println("  -context int          Context size (default 1000)")
	fmt.Println("  -embedding-model string Embedding model for embedding method (default text-embedding-3-small)")
	fmt.Println("  -dimension int        Embedding dimension for embedding method (default 1536)")
	fmt.Println("  -help                 Show this help message")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  OPENAI_TEST_KEY       OpenAI API key for semantic chunking and embedding")
	fmt.Println("  RAG_LLM_TEST_KEY      Custom LLM API key")
	fmt.Println("  RAG_LLM_TEST_URL      Custom LLM API URL")
	fmt.Println("  RAG_LLM_TEST_SMODEL   Custom LLM model name")
	fmt.Println()
	fmt.Println("Output:")
	fmt.Println("  Chunking files: basename.chunk-index.ext")
	fmt.Println("  Structured chunks: <dir>/structured/")
	fmt.Println("  Semantic chunks: <dir>/semantic/")
	fmt.Println("  Position mapping: basename.mapping.json")
	fmt.Println("  Embedding files: original-filename.json")
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
			color.Cyan("Starting batch embedding: %s\n", payload.Message)
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
	embeddings, err := embedder.EmbedDocuments(ctx, fileContents, progressCallback)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	if len(embeddings) != len(validFiles) {
		return fmt.Errorf("embedding count mismatch: got %d embeddings for %d files", len(embeddings), len(validFiles))
	}

	// Save embeddings to files
	fmt.Println("Saving embedding files...")
	for i, filePath := range validFiles {
		if err := saveEmbeddingFile(filePath, fileContents[i], embeddings[i], embedder, suffix); err != nil {
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
	fmt.Printf("Model: %s\n", model)
	fmt.Printf("Dimension: %d\n", dimension)
	fmt.Printf("Concurrent: %d (used by EmbedDocuments internally)\n", concurrent)
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

// saveEmbeddingFile saves embedding data to a JSON file
func saveEmbeddingFile(filePath, text string, embedding []float64, embedder types.Embedding, suffix string) error {
	// Prepare embedding data
	embeddingData := map[string]interface{}{
		"file":         filepath.Base(filePath),
		"full_path":    filePath,
		"model":        embedder.GetModel(),
		"dimension":    embedder.GetDimension(),
		"text_length":  len(text),
		"embedding":    embedding,
		"generated_at": time.Now().Format(time.RFC3339),
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
