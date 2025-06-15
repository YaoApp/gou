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
		// toolcall      = flag.Bool("toolcall", true, "Use toolcall for semantic chunking")
		help = flag.Bool("help", false, "Show help message")
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

	// Create output directories
	semanticDir := filepath.Join(dir, "semantic")
	structuredDir := filepath.Join(dir, "structured")

	if err := setupOutputDirectories(semanticDir, structuredDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to setup output directories: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processing file: %s\n", *filePath)
	fmt.Printf("Basename: %s, Extension: %s\n", basename, ext)
	fmt.Printf("Output directories: %s, %s\n", semanticDir, structuredDir)

	// Create OpenAI connector for semantic chunking
	// openaiConnector, err := createOpenAIConnector()
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error: Failed to create OpenAI connector: %v\n", err)
	// 	os.Exit(1)
	// }

	ctx := context.Background()

	// Process structured chunking
	fmt.Println("\n=== Running Structured Chunking ===")
	if err := runStructuredChunking(ctx, *filePath, basename, ext, structuredDir, *size, *overlap, *maxDepth, *maxConcurrent); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Structured chunking failed: %v\n", err)
		os.Exit(1)
	}

	// Process semantic chunking
	// fmt.Println("\n=== Running Semantic Chunking ===")
	// if err := runSemanticChunking(ctx, *filePath, basename, ext, semanticDir, openaiConnector, *size, *overlap, *maxDepth, *maxConcurrent, *toolcall); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error: Semantic chunking failed: %v\n", err)
	// 	os.Exit(1)
	// }

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
	fmt.Println("  -depth int      Maximum chunk depth (default 2)")
	fmt.Println("  -concurrent int Maximum concurrent operations (default 4)")
	fmt.Println("  -toolcall       Use toolcall for semantic chunking (default true)")
	fmt.Println("  -help          Show this help message")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  OPENAI_TEST_KEY  OpenAI API key for semantic chunking")
	fmt.Println()
	fmt.Println("Output:")
	fmt.Println("  Files will be saved as: basename.chunk-index.ext")
	fmt.Println("  Structured chunks: <dir>/structured/")
	fmt.Println("  Semantic chunks: <dir>/semantic/")
}

func setupOutputDirectories(semanticDir, structuredDir string) error {
	// Remove existing directories if they exist
	for _, dir := range []string{semanticDir, structuredDir} {
		if _, err := os.Stat(dir); err == nil {
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("failed to remove existing directory %s: %w", dir, err)
			}
			fmt.Printf("Cleared existing directory: %s\n", dir)
		}
	}

	// Create directories
	for _, dir := range []string{semanticDir, structuredDir} {
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

	cost := time.Since(start)
	fmt.Printf("\n--------------------------------\n")
	fmt.Printf("Structured chunking completed: %d chunks generated in %s\n", len(chunks), cost.Round(time.Microsecond))
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

	var chunks []*types.Chunk
	var mu sync.Mutex
	chunkIndex := 0

	callback := func(chunk *types.Chunk) error {
		mu.Lock()
		defer mu.Unlock()

		chunks = append(chunks, chunk)

		// Generate filename: basename.chunk-index.ext
		filename := fmt.Sprintf("%s.%d.chunk-%d-%s%s", basename, chunk.Depth, chunk.Index, chunk.ParentID, ext)
		filepath := filepath.Join(outputDir, filename)

		// Write chunk to file
		if err := os.WriteFile(filepath, []byte(chunk.Text), 0644); err != nil {
			return fmt.Errorf("failed to write chunk file %s: %w", filepath, err)
		}

		fmt.Printf("  Semantic chunk %d: %s (depth: %d, size: %d)\n", chunkIndex, filename, chunk.Depth, len(chunk.Text))
		chunkIndex++

		return nil
	}

	if err := chunker.ChunkFile(ctx, filePath, options, callback); err != nil {
		return fmt.Errorf("semantic chunking failed: %w", err)
	}

	fmt.Printf("Semantic chunking completed: %d chunks generated\n", len(chunks))
	return nil
}
