package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/yaoapp/gou/graphrag/chunking"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

func main() {
	fmt.Println("=== Streaming Semantic Analysis Demo ===")

	// Sample text
	text := `
The development of artificial intelligence can be divided into several important stages. The first stage is symbolic AI, mainly based on logical reasoning and knowledge representation.
Representative work in this stage includes the construction of expert systems and knowledge graphs.

The second stage is connectionist AI, with neural networks as the core technology. The rise of deep learning marks an important breakthrough in this stage.
Convolutional neural networks have achieved great success in image recognition, and recurrent neural networks have performed excellently in natural language processing.

The third stage is the current era of large models, with large language models based on Transformer architecture showing powerful capabilities.
GPT series, BERT series and other models have achieved breakthrough progress in various NLP tasks.

The future development of AI will move towards more general and more intelligent directions, with multimodal AI and AGI being important development directions.
`

	// Create progress callback function
	progressCallback := func(chunkID, progress, step string, data interface{}) error {
		fmt.Printf("📊 Progress Update: [%s] %s - %s", chunkID, progress, step)

		// Show streaming data details
		if dataMap, ok := data.(map[string]interface{}); ok {
			if step == "llm_response" {
				if posCount, exists := dataMap["positions_count"]; exists {
					fmt.Printf(" (Parsed positions: %v)", posCount)
				}
				if contentLen, exists := dataMap["content_length"]; exists {
					fmt.Printf(" (Content length: %v)", contentLen)
				}
				if finished, exists := dataMap["finished"]; exists && finished.(bool) {
					fmt.Printf(" ✅ Completed")
				}
			}
		}
		fmt.Println()
		return nil
	}

	// Create semantic chunker
	chunker := chunking.NewSemanticChunker(progressCallback)

	// Configuration options
	options := &types.ChunkingOptions{
		Type:          types.ChunkingTypeText,
		Size:          200, // Smaller chunk size for demo
		Overlap:       20,
		MaxDepth:      2,
		MaxConcurrent: 2,
		SemanticOptions: &types.SemanticOptions{
			Connector:     "local-llm", // Use local LLM
			MaxRetry:      2,
			MaxConcurrent: 2,
			ContextSize:   800,
			Toolcall:      false, // Use regular response mode
			Prompt:        "",    // Use default prompt
		},
	}

	fmt.Printf("📝 Original text length: %d characters\n", len(strings.TrimSpace(text)))
	fmt.Printf("⚙️  Configuration: chunk_size=%d, overlap=%d, max_depth=%d\n",
		options.Size, options.Overlap, options.MaxDepth)
	fmt.Printf("🤖 Using connector: %s\n", options.SemanticOptions.Connector)
	fmt.Println()

	// Collect generated chunks
	var chunks []*types.Chunk
	chunkCallback := func(chunk *types.Chunk) error {
		chunks = append(chunks, chunk)
		fmt.Printf("📦 Generated chunk #%d: ID=%s, depth=%d, size=%d chars\n",
			len(chunks), chunk.ID[:8], chunk.Depth, len(chunk.Text))

		// Show chunk content preview
		preview := strings.ReplaceAll(strings.TrimSpace(chunk.Text), "\n", " ")
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		fmt.Printf("   Content preview: %s\n", preview)
		fmt.Println()
		return nil
	}

	// Execute semantic chunking
	fmt.Println("🚀 Starting streaming semantic analysis...")
	fmt.Println()

	ctx := context.Background()
	err := chunker.Chunk(ctx, text, options, chunkCallback)

	if err != nil {
		log.Fatalf("❌ Semantic chunking failed: %v", err)
	}

	// Show result statistics
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("✅ Semantic analysis completed!\n")
	fmt.Printf("📊 Total generated chunks: %d\n", len(chunks))

	// Statistics by depth
	depthCount := make(map[int]int)
	for _, chunk := range chunks {
		depthCount[chunk.Depth]++
	}

	fmt.Println("📈 Depth distribution:")
	for depth := 1; depth <= options.MaxDepth; depth++ {
		if count, exists := depthCount[depth]; exists {
			fmt.Printf("   Depth %d: %d chunks\n", depth, count)
		}
	}

	// Demonstrate stream parser functionality
	fmt.Println()
	fmt.Println("🔧 Stream Parser Demo:")
	demonstrateStreamParser()
}

func demonstrateStreamParser() {
	// Create stream parser
	parser := utils.NewStreamParser(false) // Regular response mode

	// Simulate streaming data
	streamChunks := []string{
		`{"choices":[{"delta":{"content":"Here are the semantic segments:\n["}}]}`,
		`{"choices":[{"delta":{"content":"{\"start_pos\": 0, \"end_pos\": 120},"}}]}`,
		`{"choices":[{"delta":{"content":"{\"start_pos\": 120, \"end_pos\": 250},"}}]}`,
		`{"choices":[{"delta":{"content":"{\"start_pos\": 250, \"end_pos\": 380}"}}]}`,
		`{"choices":[{"delta":{"content":"]\n\nThese segments are based on semantic relevance..."}}]}`,
	}

	fmt.Println("Simulating streaming LLM response parsing:")

	for i, chunk := range streamChunks {
		fmt.Printf("  📡 Received chunk %d: %s\n", i+1, chunk)

		data, err := parser.ParseStreamChunk([]byte(chunk))
		if err != nil {
			fmt.Printf("     ❌ Parse error: %v\n", err)
			continue
		}

		fmt.Printf("     📄 Accumulated content length: %d\n", len(data.Content))
		fmt.Printf("     🎯 Parsed positions count: %d\n", len(data.Positions))

		if len(data.Positions) > 0 {
			fmt.Printf("     📍 Position details: ")
			for j, pos := range data.Positions {
				if j > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("[%d-%d]", pos.StartPos, pos.EndPos)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}
