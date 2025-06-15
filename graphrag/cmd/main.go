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

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/chunking"
	"github.com/yaoapp/gou/graphrag/types"
)

func main() {
	var (
		filePath      = flag.String("file", "", "Path to the file to chunk (required)")
		size          = flag.Int("size", 300, "Chunk size")
		overlap       = flag.Int("overlap", 50, "Chunk overlap")
		maxDepth      = flag.Int("depth", 3, "Maximum chunk depth")
		maxConcurrent = flag.Int("concurrent", 6, "Maximum concurrent operations")
		method        = flag.String("method", "structured", "Chunking method: structured, semantic, or both")
		toolcall      = flag.Bool("toolcall", false, "Use toolcall for semantic chunking")
		help          = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	if *filePath == "" {
		fmt.Fprintf(os.Stderr, "Error: -file flag is required\n")
		printHelp()
		os.Exit(1)
	}

	// Validate method parameter
	validMethods := map[string]bool{
		"structured": true,
		"semantic":   true,
		"both":       true,
	}
	if !validMethods[*method] {
		fmt.Fprintf(os.Stderr, "Error: Invalid method '%s'. Valid methods are: structured, semantic, both\n", *method)
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
	var openaiConnector connector.Connector
	if *method == "semantic" || *method == "both" {
		openaiConnector, err = createOpenAIConnector()
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
		if err := runSemanticChunking(ctx, *filePath, basename, ext, semanticDir, openaiConnector, *size, *overlap, *maxDepth, *maxConcurrent, *toolcall); err != nil {
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
		if err := runSemanticChunking(ctx, *filePath, basename, ext, semanticDir, openaiConnector, *size, *overlap, *maxDepth, *maxConcurrent, *toolcall); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Semantic chunking failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("\n=== Chunking completed successfully ===")
}

func printHelp() {
	fmt.Println("GraphRAG Chunking Tool")
	fmt.Println("Usage: go run tools.go -file <path> [options]")
	fmt.Println()
	fmt.Println("Required flags:")
	fmt.Println("  -file string    Path to the file to chunk")
	fmt.Println()
	fmt.Println("Optional flags:")
	fmt.Println("  -size int       Chunk size (default 300)")
	fmt.Println("  -overlap int    Chunk overlap (default 50)")
	fmt.Println("  -depth int      Maximum chunk depth (default 3)")
	fmt.Println("  -concurrent int Maximum concurrent operations (default 6)")
	fmt.Println("  -method string  Chunking method: structured, semantic, or both (default structured)")
	fmt.Println("  -toolcall       Use toolcall for semantic chunking (default false)")
	fmt.Println("  -help          Show this help message")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  OPENAI_TEST_KEY  OpenAI API key for semantic chunking")
	fmt.Println()
	fmt.Println("Output:")
	fmt.Println("  Files will be saved as: basename.chunk-index.ext")
	fmt.Println("  Structured chunks: <dir>/structured/")
	fmt.Println("  Semantic chunks: <dir>/semantic/")
	fmt.Println("  Position mapping: basename.mapping.json")
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

func createOpenAIConnector() (connector.Connector, error) {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_TEST_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_TEST_KEY environment variable is not set")
	}

	// Create connector DSL
	dsl := map[string]interface{}{
		"name": "openai-chunking",
		"type": "openai",
		"options": map[string]interface{}{
			"key":   apiKey,
			"model": "gpt-4o-mini",
		},
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

func runSemanticChunking(ctx context.Context, filePath, basename, ext, outputDir string, conn connector.Connector, size, overlap, maxDepth, maxConcurrent int, toolcall bool) error {

	// Progress callback for semantic chunking
	progressCallback := func(chunkID, progress, step string, data interface{}) error {
		fmt.Printf("  Semantic progress [%s]: %s - %s\n", chunkID, progress, step)
		return nil
	}

	chunker := chunking.NewSemanticChunker(progressCallback)

	options := &types.ChunkingOptions{
		Size:          size,
		Overlap:       overlap,
		MaxDepth:      maxDepth,
		MaxConcurrent: maxConcurrent,
		SemanticOptions: &types.SemanticOptions{
			Connector:     "openai-chunking",
			ContextSize:   size * maxDepth * 3,
			Options:       `{"temperature": 0.1}`,
			Prompt:        "", // Use default prompt
			Toolcall:      toolcall,
			MaxRetry:      3,
			MaxConcurrent: maxConcurrent,
		},
	}

	fmt.Printf("--------------------------------\n")
	fmt.Printf("Size: %d\n", size)
	fmt.Printf("Overlap: %d\n", overlap)
	fmt.Printf("Depth: %d\n", maxDepth)
	fmt.Printf("Concurrent: %d\n", maxConcurrent)
	fmt.Printf("Toolcall: %t\n", toolcall)
	fmt.Printf("Context Size: %d\n", options.SemanticOptions.ContextSize)
	fmt.Printf("--------------------------------\n")

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

		fmt.Printf("  Semantic chunk %d: %s (depth: %d, size: %d)\n", chunk.Index, filename, chunk.Depth, len(chunk.Text))

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
