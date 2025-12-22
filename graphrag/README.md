# GraphRAG

GraphRAG is a Go package that implements a comprehensive Graph-based Retrieval-Augmented Generation system. It provides a unified interface for managing collections, documents, segments, and performing vector and graph-based searches.

## Features

- **Multi-layer Storage**: Vector database, Graph database, and Key-Value store integration
- **Document Processing**: Support for files, text, URLs, and streams
- **Segment Management**: Fine-grained control over document segments
- **Search Capabilities**: Vector similarity search and graph traversal
- **Backup/Restore**: Complete collection backup and restore functionality
- **Concurrent Operations**: Thread-safe operations with comprehensive error handling

## Installation

```bash
go get github.com/yaoapp/gou/graphrag
```

## Quick Start

### Basic Configuration

```go
package main

import (
    "context"
    "log"

    "github.com/yaoapp/gou/graphrag"
    "github.com/yaoapp/gou/graphrag/types"
    "github.com/yaoapp/gou/graphrag/embedding"
    "github.com/yaoapp/gou/graphrag/extraction/openai"
    "github.com/yaoapp/gou/store"
    "github.com/yaoapp/kun/log"
)

func main() {
    // Configure GraphRAG
    config := &graphrag.Config{
        Vector: vectorStore,  // Your vector store implementation
        Graph:  graphStore,   // Your graph store implementation (optional)
        Store:  kvStore,      // Your key-value store implementation (optional)
        Logger: log.StandardLogger(),
        System: "system_collection",
    }

    // Create GraphRAG instance
    g, err := graphrag.New(config)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Create a collection
    collection := types.Collection{
        ID: "my_collection",
        Metadata: map[string]interface{}{
            "type": "documents",
            "description": "My document collection",
        },
        VectorConfig: &types.VectorStoreConfig{
            Dimension: 1536,
            Distance:  types.DistanceCosine,
            IndexType: types.IndexTypeHNSW,
        },
    }

    collectionID, err := g.CreateCollection(ctx, collection)
    if err != nil {
        log.Fatal(err)
    }

    // Create embedding function
    embeddingFunc, err := embedding.NewOpenai(embedding.OpenaiOptions{
        ConnectorName: "openai",
        Concurrent:    10,
        Dimension:     1536,
        Model:         "text-embedding-3-small",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create extraction function (optional, only if using graph storage)
    extractionFunc, err := openai.NewOpenai(openai.Options{
        ConnectorName: "openai",
        Concurrent:    5,
        Model:         "gpt-4o-mini",
        Temperature:   0.1,
        MaxTokens:     4000,
        RetryAttempts: 3,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Add a file (most common usage)
    options := &types.UpsertOptions{
        DocID:      "doc1",
        GraphName:  collectionID,
        Embedding:  embeddingFunc,
        Extraction: extractionFunc, // Only needed if using graph storage
        // Converter is optional - system will auto-detect based on file type
        Metadata: map[string]interface{}{
            "source": "example.pdf",
            "type":   "document",
        },
    }

    docID, err := g.AddFile(ctx, "/path/to/document.pdf", options)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Document added with ID: %s", docID)
}
```

## Core Interface

### Collection Management

```go
// Create a collection
collection := types.Collection{
    ID: "research_papers",
    Metadata: map[string]interface{}{
        "type": "research",
        "domain": "AI",
    },
    VectorConfig: &types.VectorStoreConfig{
        Dimension: 1536,
        Distance:  types.DistanceCosine,
        IndexType: types.IndexTypeHNSW,
    },
}

collectionID, err := g.CreateCollection(ctx, collection)

// Check if collection exists
exists, err := g.CollectionExists(ctx, collectionID)

// Get collections with filtering
collections, err := g.GetCollections(ctx, map[string]interface{}{
    "type": "research",
})

// Remove collection
removed, err := g.RemoveCollection(ctx, collectionID)
```

### Document Management

```go
// Import required packages:
// "github.com/yaoapp/gou/graphrag/embedding"
// "github.com/yaoapp/gou/graphrag/extraction/openai"
// "github.com/yaoapp/gou/graphrag/converter"
// "github.com/yaoapp/gou/graphrag/chunking"

// Create embedding and extraction functions (required for all document operations)
embeddingFunc, err := embedding.NewOpenai(embedding.OpenaiOptions{
    ConnectorName: "openai",
    Concurrent:    10,
    Dimension:     1536,
    Model:         "text-embedding-3-small",
})
if err != nil {
    log.Fatal(err)
}

extractionFunc, err := openai.NewOpenai(openai.Options{
    ConnectorName: "openai",
    Concurrent:    5,
    Model:         "gpt-4o-mini",
    Temperature:   0.1,
    MaxTokens:     4000,
    RetryAttempts: 3,
})
if err != nil {
    log.Fatal(err)
}

// Create converter (optional - system will auto-detect if not provided)
// For most cases, you can omit this and let the system auto-detect
var converterFunc types.Converter

// Example: Vision converter for images
visionConverter, err := converter.NewVision(converter.VisionOption{
    ConnectorName: "openai",
    Model:         "gpt-4o-mini",
    CompressSize:  1024,
    Language:      "Auto",
    Options:       map[string]interface{}{"max_tokens": 1000, "temperature": 0.1},
})
if err != nil {
    log.Fatal(err)
}
converterFunc = visionConverter

// Example: Office converter for Word/PowerPoint files
// officeConverter, err := converter.NewOffice(converter.OfficeOption{
//     VisionConverter: visionConverter, // Required for multimedia content
//     MaxConcurrency:  4,
//     CleanupTemp:     true,
// })

// Example: Video converter for video files
// videoConverter, err := converter.NewVideo(converter.VideoOption{
//     AudioConverter:     whisperConverter, // Required for audio processing
//     VisionConverter:    visionConverter,  // Required for frame analysis
//     KeyframeInterval:   10.0,             // Extract keyframes every 10 seconds
//     MaxKeyframes:       20,               // Max 20 keyframes per video
//     MaxConcurrency:     4,
//     CleanupTemp:        true,
// })

// Create semantic chunking instance (recommended for intelligent segmentation)
semanticChunker := chunking.NewSemanticChunker(func(chunkID, progress, step string, data interface{}) error {
    log.Printf("Chunking progress - %s: %s", step, progress)
    return nil
})

// Configure semantic chunking options
chunkingOptions := &types.ChunkingOptions{
    Type:          types.ChunkingTypeText,
    Size:          500,
    Overlap:       50,
    MaxDepth:      3,
    MaxConcurrent: 4,
    SemanticOptions: &types.SemanticOptions{
        Connector:     "openai",
        ContextSize:   1500,
        MaxRetry:      3,
        MaxConcurrent: 2,
        Toolcall:      true,
        Options:       `{"temperature": 0.1}`,
        Prompt:        "Split this text into meaningful semantic segments",
    },
}

// Add different types of documents
options := &types.UpsertOptions{
    DocID:           "doc1",
    GraphName:       collectionID,
    Embedding:       embeddingFunc,
    Extraction:      extractionFunc, // Only needed if using graph storage
    Converter:       converterFunc,  // Optional - system will auto-detect if not provided
    Chunking:        semanticChunker, // Recommended: semantic chunking for intelligent segmentation
    ChunkingOptions: chunkingOptions, // Semantic chunking configuration
    Metadata: map[string]interface{}{
        "source": "research",
        "author": "John Doe",
    },
}

// Add file (most common usage)
docID, err := g.AddFile(ctx, "/path/to/document.pdf", options)

// Add URL content
docID, err := g.AddURL(ctx, "https://example.com/article", options)

// Add text
docID, err := g.AddText(ctx, "Research paper content...", options)

// Add from stream
file, _ := os.Open("document.txt")
docID, err := g.AddStream(ctx, file, options)

// Example: Using different chunking strategies
// 1. Semantic chunking for intelligent text segmentation (recommended)
semanticChunker := chunking.NewSemanticChunker(func(chunkID, progress, step string, data interface{}) error {
    log.Printf("Semantic chunk %s: %s - %s", chunkID, step, progress)
    return nil
})
semanticOptions := &types.UpsertOptions{
    DocID:           "academic_paper",
    GraphName:       collectionID,
    Embedding:       embeddingFunc,
    Extraction:      extractionFunc,
    Chunking:        semanticChunker,
    ChunkingOptions: &types.ChunkingOptions{
        Type:          types.ChunkingTypeText,
        Size:          800,
        Overlap:       100,
        MaxDepth:      3,
        MaxConcurrent: 4,
        SemanticOptions: &types.SemanticOptions{
            Connector:     "openai",
            ContextSize:   2400,
            MaxRetry:      3,
            MaxConcurrent: 2,
            Toolcall:      true,
            Options:       `{"temperature": 0.1}`,
            Prompt:        "Intelligently segment this text into coherent sections",
        },
    },
    Metadata: map[string]interface{}{
        "type":   "academic",
        "domain": "AI research",
    },
}
docID, err := g.AddFile(ctx, "/path/to/paper.pdf", semanticOptions)

// 2. Structured chunking for basic text splitting (when semantic analysis is not needed)
structuredChunker := chunking.NewStructuredChunker()
structuredOptions := &types.UpsertOptions{
    DocID:           "simple_doc",
    GraphName:       collectionID,
    Embedding:       embeddingFunc,
    Extraction:      extractionFunc,
    Chunking:        structuredChunker,
    ChunkingOptions: chunking.NewStructuredOptions(types.ChunkingTypeText), // Basic configuration
    Metadata: map[string]interface{}{
        "type": "simple",
        "note": "using basic chunking",
    },
}
docID, err := g.AddFile(ctx, "/path/to/simple.txt", structuredOptions)

// Remove documents
removedCount, err := g.RemoveDocs(ctx, []string{"doc1", "doc2"})

// Note: The system automatically detects the appropriate converter based on file type:
// - PDF files: Uses OCR converter for text extraction
// - Images (jpg, png, gif, webp): Uses Vision converter
// - Audio files (mp3, wav, flac): Uses Whisper converter
// - Video files (mp4, avi, mov): Uses Video converter with audio + vision
// - Text files (txt, md, json, code): Uses UTF8 converter
// - Office files (docx, pptx): Uses Office converter
```

### Segment Management

```go
// Create embedding and extraction functions
embeddingFunc, err := embedding.NewOpenai(embedding.OpenaiOptions{
    ConnectorName: "openai",
    Concurrent:    10,
    Dimension:     1536,
    Model:         "text-embedding-3-small",
})
if err != nil {
    log.Fatal(err)
}

extractionFunc, err := openai.NewOpenai(openai.Options{
    ConnectorName: "openai",
    Concurrent:    5,
    Model:         "gpt-4o-mini",
    Temperature:   0.1,
    MaxTokens:     4000,
    RetryAttempts: 3,
})
if err != nil {
    log.Fatal(err)
}

// Add segments manually
segmentTexts := []types.SegmentText{
    {
        ID:   "seg1",
        Text: "First segment about machine learning fundamentals.",
    },
    {
        ID:   "seg2",
        Text: "Second segment about neural networks.",
    },
}

// Example: Using semantic chunking for intelligent segmentation
semanticChunker := chunking.NewSemanticChunker(func(chunkID, progress, step string, data interface{}) error {
    // Progress callback for monitoring chunking progress
    log.Printf("Chunk %s: %s - %s", chunkID, step, progress)
    return nil
})

semanticChunkingOptions := &types.ChunkingOptions{
    Type:          types.ChunkingTypeText,
    Size:          500,
    Overlap:       50,
    MaxDepth:      2,
    MaxConcurrent: 3,
    SemanticOptions: &types.SemanticOptions{
        Connector:     "openai",
        ContextSize:   1500,
        MaxRetry:      3,
        MaxConcurrent: 2,
        Toolcall:      true,
        Options:       `{"temperature": 0.1}`,
        Prompt:        "Split this text into meaningful semantic segments",
    },
}

options := &types.UpsertOptions{
    DocID:           "doc1",
    GraphName:       collectionID,
    Embedding:       embeddingFunc,
    Extraction:      extractionFunc, // Only needed if using graph storage
    Converter:       converterFunc,  // Optional - system will auto-detect if not provided
    Chunking:        semanticChunker, // Using semantic chunking for intelligent segmentation
    ChunkingOptions: semanticChunkingOptions, // Semantic chunking configuration
    Metadata: map[string]interface{}{
        "source": "manual",
        "type":   "segment",
    },
}

segmentIDs, err := g.AddSegments(ctx, "doc1", segmentTexts, options)

// Update segments
updatedSegments := []types.SegmentText{
    {
        ID:   "seg1",
        Text: "Updated first segment with more details.",
    },
}

updatedCount, err := g.UpdateSegments(ctx, updatedSegments, options)

// Get segments
segments, err := g.GetSegments(ctx, []string{"seg1", "seg2"})

// List segments with pagination
listOptions := &types.ListSegmentsOptions{
    Limit:  10,
    Offset: 0,
}
result, err := g.ListSegments(ctx, "doc1", listOptions)

// Remove segments
removedCount, err := g.RemoveSegments(ctx, []string{"seg1", "seg2"})
```

### Segment Scoring and Weighting

```go
// Update segment votes
votes := []types.SegmentVote{
    {
        SegmentID: "seg1",
        Vote:      5,
    },
}
updatedCount, err := g.UpdateVote(ctx, votes)

// Update segment scores
scores := []types.SegmentScore{
    {
        SegmentID: "seg1",
        Score:     0.95,
    },
}
updatedCount, err := g.UpdateScore(ctx, scores)

// Update segment weights
weights := []types.SegmentWeight{
    {
        SegmentID: "seg1",
        Weight:    0.8,
    },
}
updatedCount, err := g.UpdateWeight(ctx, weights)
```

### Search

```go
// Search for segments
queryOptions := &types.QueryOptions{
    Query:     "machine learning",
    TopK:      10,
    GraphName: collectionID,
}

segments, err := g.Search(ctx, queryOptions)

// Multi-search
multiOptions := []types.QueryOptions{
    {
        Query:     "neural networks",
        TopK:      5,
        GraphName: collectionID,
    },
    {
        Query:     "deep learning",
        TopK:      5,
        GraphName: collectionID,
    },
}

results, err := g.MultiSearch(ctx, multiOptions)
```

### Backup and Restore

```go
// Backup collection
var backupBuffer bytes.Buffer
err := g.Backup(ctx, &backupBuffer, collectionID)

// Restore collection
err = g.Restore(ctx, &backupBuffer, "restored_collection_id")
```

## Configuration Options

**Important Note**: Collection names for vector stores and graph stores are automatically generated based on the Collection ID using the system's naming convention. You don't need to specify `CollectionName` or `GraphName` manually.

### Vector Store Configuration

```go
vectorConfig := types.VectorStoreConfig{
    Dimension: 1536,
    Distance:  types.DistanceCosine,
    IndexType: types.IndexTypeHNSW,
    M:         16,
    EfConstruction: 100,
    // Note: CollectionName is automatically generated based on Collection ID
}
```

### Graph Store Configuration

```go
graphConfig := types.GraphStoreConfig{
    StoreType:   "neo4j",
    DatabaseURL: "neo4j://localhost:7687",
    DriverConfig: map[string]interface{}{
        "username": "neo4j",
        "password": "password",
    },
    // Note: GraphName is automatically generated based on Collection ID
}
```

### Upsert Options

```go
options := &types.UpsertOptions{
    DocID:           "unique_doc_id",
    GraphName:       "collection_name",
    Embedding:       embeddingFunction,
    Extraction:      extractionFunction,
    Converter:       converterFunction,       // Optional - system will auto-detect if not provided
    Chunking:        chunkingInstance,        // Optional - system will use default if not provided
    ChunkingOptions: chunkingOptionsConfig,   // Optional - chunking configuration
    Fetcher:         fetcherInstance,         // Optional - for URL processing
    Progress:        progressCallback,        // Optional - progress monitoring
    Metadata: map[string]interface{}{
        "source":     "web",
        "created_at": time.Now(),
        "tags":       []string{"ai", "ml"},
    },
}
```

## Implemented Components

### Converters

- **UTF8 Converter**: Plain text and UTF-8 file conversion with streaming support

  ```go
  converter := converter.NewUTF8()
  ```

- **Office Converter**: Microsoft Office documents (DOCX, PPTX) with multimedia processing

  ```go
  officeConverter, err := converter.NewOffice(converter.OfficeOption{
      VisionConverter:  visionConverter,  // Required for image processing
      VideoConverter:   videoConverter,   // Optional for video processing
      WhisperConverter: whisperConverter, // Optional for audio processing
      MaxConcurrency:   4,
      TempDir:          "",   // Use system temp dir
      CleanupTemp:      true,
  })
  ```

- **Vision Converter**: Image-to-text conversion using AI vision models

  ```go
  visionConverter, err := converter.NewVision(converter.VisionOption{
      ConnectorName: "openai",
      Model:         "gpt-4o-mini",
      CompressSize:  1024,
      Language:      "Auto",
      Options:       map[string]interface{}{"max_tokens": 1000, "temperature": 0.1},
  })
  ```

- **Video Converter**: Video file processing and transcription

  ```go
  videoConverter, err := converter.NewVideo(converter.VideoOption{
      AudioConverter:     whisperConverter, // Required for audio processing
      VisionConverter:    visionConverter,  // Required for frame analysis
      KeyframeInterval:   10.0,             // Extract keyframes every 10 seconds
      MaxKeyframes:       20,               // Max 20 keyframes per video
      MaxConcurrency:     4,
      CleanupTemp:        true,
  })
  ```

- **Whisper Converter**: Audio-to-text conversion using Whisper

  ```go
  whisperConverter, err := converter.NewWhisper(converter.WhisperOption{
      ConnectorName:          "openai",
      Model:                  "whisper-1",
      Language:               "auto",
      ChunkDuration:          30.0,
      MappingDuration:        5.0,
      SilenceThreshold:       -40.0,
      SilenceMinLength:       1.0,
      EnableSilenceDetection: true,
      MaxConcurrency:         4,
      CleanupTemp:            true,
  })
  ```

- **OCR Converter**: Optical Character Recognition for document images

  ```go
  ocrConverter, err := converter.NewOCR(converter.OCROption{
      Vision:         visionConverter, // Required vision converter
      Mode:           converter.OCRModeConcurrent,
      MaxConcurrency: 4,
      CompressSize:   1024,
      ForceImageMode: true, // Force image mode for PDFs
  })
  ```

- **MCP Converter**: Model Context Protocol converter
  ```go
  mcpConverter, err := converter.NewMCP(&converter.MCPOptions{
      ID:   "mcp_client",
      Tool: "convert_tool",
      ArgumentsMapping: map[string]string{
          "data": "{{data_uri}}",
      },
      ResultMapping: map[string]string{
          "text": "{{content.0.text}}",
      },
  })
  ```

### Embedding

- **OpenAI Embedding**: Uses OpenAI's embedding API for text vectorization

  ```go
  embedding, err := embedding.NewOpenai(embedding.OpenaiOptions{
      ConnectorName: "openai",
      Concurrent:    10,
      Dimension:     1536,
      Model:         "text-embedding-3-small",
  })
  ```

- **FastEmbed**: Local embedding service with high performance
  ```go
  embedding, err := embedding.NewFastEmbed(embedding.FastEmbedOptions{
      ConnectorName: "fastembed",
      Concurrent:    10,
      Dimension:     384,
      Model:         "BAAI/bge-small-en-v1.5",
      Host:          "http://localhost:8000",
  })
  ```

### Fetcher

- **HTTP Fetcher**: Fetch content from HTTP/HTTPS URLs

  ```go
  httpFetcher := fetcher.NewHTTPFetcher() // Uses default configuration

  // Or with custom options:
  // httpFetcher := fetcher.NewHTTPFetcher(&fetcher.HTTPOptions{
  //     UserAgent: "Custom-Agent/1.0",
  //     Timeout:   300 * time.Second,
  //     Headers: map[string]string{
  //         "Accept": "text/html,application/json",
  //     },
  // })
  ```

- **MCP Fetcher**: Model Context Protocol content fetcher
  ```go
  mcpFetcher, err := fetcher.NewMCP(&fetcher.MCPOptions{
      ID:   "mcp_client",
      Tool: "fetch_tool",
      ArgumentsMapping: map[string]string{
          "url": "{{url}}",
      },
      ResultMapping: map[string]string{
          "content": "{{result.content}}",
          "title":   "{{result.title}}",
      },
  })
  ```

### Extraction

- **OpenAI Extraction**: Uses OpenAI's API for entity and relationship extraction

  ```go
  extraction, err := openai.NewOpenai(openai.Options{
      ConnectorName: "openai",
      Concurrent:    5,
      Model:         "gpt-4o-mini",
      Temperature:   0.1,
      MaxTokens:     4000,
      RetryAttempts: 3,
  })
  ```

- **Extraction Framework**: Generic extraction with deduplication and embedding
  ```go
  extractor := extraction.New(types.ExtractionOptions{
      Use:       openaiExtraction, // Specific extraction implementation
      Embedding: embeddingFunction, // Optional embedding for results
  })
  ```

### Chunking

- **Semantic Chunking**: ⭐ **Recommended** - LLM-powered semantic chunking for meaningful content segmentation

  ```go
  import "github.com/yaoapp/gou/graphrag/chunking"

  // Create semantic chunker with progress callback
  chunker := chunking.NewSemanticChunker(func(chunkID, progress, step string, data interface{}) error {
      // Progress callback
      log.Printf("Chunk %s: %s - %s", chunkID, step, progress)
      return nil
  })

  // Or create without progress callback
  // chunker := chunking.NewSemanticChunker(nil)

  // Configure semantic options
  options := &types.ChunkingOptions{
      Type:          types.ChunkingTypeText,
      Size:          500,
      Overlap:       50,
      MaxDepth:      2,
      MaxConcurrent: 3,
      SemanticOptions: &types.SemanticOptions{
          Connector:     "openai",
          ContextSize:   1500,
          MaxRetry:      3,
          MaxConcurrent: 2,
          Toolcall:      true,
          Options:       `{"temperature": 0.1}`,
          Prompt:        "Split this text into meaningful semantic chunks",
      },
  }

  // Chunk with semantic analysis
  err := chunker.ChunkStream(ctx, stream, options, func(chunk *types.Chunk) error {
      // Process semantic chunk
      log.Printf("Semantic chunk %d (depth %d): %s", chunk.Index, chunk.Depth, chunk.Text)
      return nil
  })
  ```

- **Chunking Type Support**: Automatic configuration for different content types

  ```go
  // Semantic chunking automatically adapts to content type
  semanticChunker := chunking.NewSemanticChunker(progressCallback)
  semanticOptions := &types.ChunkingOptions{
      Type:          types.ChunkingTypeText, // Auto-adapts to content
      SemanticOptions: &types.SemanticOptions{
          Connector: "openai",
          // ... other semantic options
      },
  }

  // For basic structured chunking (when semantic analysis is not available)
  codeOptions := chunking.NewStructuredOptions(types.ChunkingTypeCode)
  // Returns: Size: 800, Overlap: 100, MaxDepth: 3, MaxConcurrent: 10

  textOptions := chunking.NewStructuredOptions(types.ChunkingTypeText)
  // Returns: Size: 300, Overlap: 20, MaxDepth: 1, MaxConcurrent: 10

  // Supported types: Text, Code, JSON, Video, Audio, Image
  ```

- **Multi-Input Support**: Process files, streams, or text directly

  ```go
  // File chunking
  err := chunker.ChunkFile(ctx, "document.txt", options, callback)

  // Stream chunking
  err := chunker.ChunkStream(ctx, reader, options, callback)

     // Text chunking
   err := chunker.Chunk(ctx, textContent, options, callback)
  ```

- **Structured Chunking**: Basic rule-based hierarchical chunking (when semantic analysis is not needed)

  ```go
  import "github.com/yaoapp/gou/graphrag/chunking"

  // Create structured chunker
  chunker := chunking.NewStructuredChunker()

  // Configure chunking options
  options := &types.ChunkingOptions{
      Type:            types.ChunkingTypeText,
      Size:            500,
      Overlap:         50,
      MaxDepth:        3,
      SizeMultiplier:  3,
      MaxConcurrent:   5,
  }

  // Chunk text
  err := chunker.Chunk(ctx, text, options, func(chunk *types.Chunk) error {
      // Process each chunk
      log.Printf("Chunk %d: %s", chunk.Index, chunk.Text)
      return nil
  })
  ```

### Auto-Detection

- **DetectConverter**: Automatically selects converters based on file type
- **DetectChunking**: Automatically selects chunking methods and configurations
- **DetectFetcher**: Automatically selects fetchers for URL content
- **DetectExtractor**: Automatically selects extraction methods
- **DetectEmbedding**: Automatically selects embedding functions

```go
// Auto-detection examples
// Recommended: Use semantic chunking for intelligent content segmentation
semanticChunker := chunking.NewSemanticChunker(progressCallback)

// Basic: Auto-configured structured chunking for specific content types
options := chunking.NewStructuredOptions(types.ChunkingTypeCode) // Auto-configured for code files

// Other auto-detection utilities
converter := converter.DetectConverter(filePath)                  // Auto-detects based on file extension
fetcher := fetcher.DetectFetcher(url)                            // Auto-detects based on URL pattern
```

## Supported Data Storage

### Vector Store - Qdrant

High-performance vector database for embedding storage and similarity search:

- Collection management with custom configurations
- Multiple search algorithms (similarity, MMR, hybrid)
- Batch operations and pagination
- Backup and restore capabilities
- Score threshold filtering

### Graph Store - Neo4j

Graph database for storing entities and relationships:

- Node and relationship management
- Graph queries and traversals
- Community detection algorithms
- Dynamic schema management
- Backup and restore capabilities

### Store - Key-Value Store

Generic key-value store for metadata and auxiliary data:

- Collection metadata storage
- Segment voting, scoring, and weighting data
- Document metadata and tracking
- Multiple backend implementations (Redis, MongoDB, Xun, LRU Cache)

#### Store Configuration Examples

**Recommended Stores**: MongoDB (production) and Xun (database-backed with cache)

**MongoDB Store (Document Database) - ⭐ Recommended for Production**

```go
import "github.com/yaoapp/gou/store"
import "github.com/yaoapp/gou/connector"

// Create MongoDB connector
mongoConnector, err := connector.New("mongo", "mongo_store", []byte(`{
    "label": "MongoDB Store",
    "type": "mongodb",
    "options": {
        "host": "localhost",
        "port": "27017",
        "user": "admin",
        "pass": "password",
        "database": "graphrag"
    }
}`))

// Create MongoDB store
kvStore, err := store.New(mongoConnector, nil)
```

**Xun Store (Database-backed with LRU Cache) - ⭐ Recommended for Applications with Database**

```go
// Create Xun store using existing database connection
kvStore, err := store.New(nil, store.Option{
    "type":             "xun",
    "table":            "__graphrag_store",  // Table name for storage
    "connector":        "default",           // Database connector name
    "cache_size":       10240,               // LRU cache size (default: 10240)
    "persist_interval": 60,                  // Async persistence interval in seconds (default: 60)
    "cleanup_interval": 5,                   // Expired data cleanup interval in minutes (default: 5)
})
```

**Xun Store Features:**

- LRU cache layer for fast reads
- Asynchronous batch persistence to reduce database load
- Lazy loading - data loaded from database on first access
- Automatic table creation and schema management
- TTL support with background cleanup
- Supports MySQL, PostgreSQL, SQLite via Xun connectors
- **Note**: Up to `persist_interval` seconds of data may be lost on crash

**Redis Store (Distributed Cache) - ⚠️ Requires Persistence Configuration**

```go
import "github.com/yaoapp/gou/store"
import "github.com/yaoapp/gou/connector"

// Create Redis connector
redisConnector, err := connector.New("redis", "redis_store", []byte(`{
    "label": "Redis Store",
    "type": "redis",
    "options": {
        "host": "localhost",
        "port": "6379",
        "pass": "password",
        "db": "0"
    }
}`))

// Create Redis store
kvStore, err := store.New(redisConnector, nil)
```

**⚠️ Important Redis Notes:**

- Redis requires proper persistence configuration (RDB/AOF) to avoid data loss
- Configure Redis with `appendonly yes` and appropriate save intervals
- Consider Redis clustering for high availability in production
- Monitor Redis memory usage and configure appropriate eviction policies

**LRU Cache Store (In-Memory) - For Testing Only**

```go
// Create LRU cache store (no connector needed)
kvStore, err := store.New(nil, store.Option{
    "size": 10000, // Max number of items in cache
})
```

**Using Store in GraphRAG Config**

```go
config := &graphrag.Config{
    Vector: vectorStore,
    Graph:  graphStore,
    Store:  kvStore,        // Any of the above store implementations
    Logger: logger,
    System: "my_system",
}
```

## Storage Architecture

The system uses a three-layer storage architecture:

1. **Vector Layer**: Stores embeddings and enables similarity search
2. **Graph Layer**: Stores entities and relationships for graph traversal
3. **Metadata Layer**: Stores collection metadata and auxiliary data

Collections are identified consistently across all storage layers using generated collection IDs.

## Advanced Examples

### Complete Document Processing Pipeline

```go
func processDocument(g *graphrag.GraphRag, filePath string) error {
    ctx := context.Background()

    // Create collection
    collection := types.Collection{
        ID: "document_processing",
        Metadata: map[string]interface{}{
            "type": "documents",
            "pipeline": "complete",
        },
        VectorConfig: &types.VectorStoreConfig{
            Dimension: 1536,
            Distance:  types.DistanceCosine,
            IndexType: types.IndexTypeHNSW,
        },
        GraphStoreConfig: &types.GraphStoreConfig{
            StoreType:   "neo4j",
            DatabaseURL: "neo4j://localhost:7687",
            DriverConfig: map[string]interface{}{
                "username": "neo4j",
                "password": "password",
            },
        },
    }

    collectionID, err := g.CreateCollection(ctx, collection)
    if err != nil {
        return err
    }

    // Configure processing options
    options := &types.UpsertOptions{
        DocID:     filepath.Base(filePath),
        GraphName: collectionID,
        Metadata: map[string]interface{}{
            "source": "file",
            "path":   filePath,
        },
    }

    // Process document
    docID, err := g.AddFile(ctx, filePath, options)
    if err != nil {
        return err
    }

    log.Printf("Document processed: %s", docID)
    return nil
}
```

### Concurrent Collection Operations

```go
func concurrentOperations(g *graphrag.GraphRag) error {
    ctx := context.Background()
    var wg sync.WaitGroup

    // Create multiple collections concurrently
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(index int) {
            defer wg.Done()

            collection := types.Collection{
                ID: fmt.Sprintf("collection_%d", index),
                Metadata: map[string]interface{}{
                    "type":  "concurrent",
                    "index": index,
                },
                VectorConfig: &types.VectorStoreConfig{
                    Dimension: 1536,
                    Distance:  types.DistanceCosine,
                    IndexType: types.IndexTypeHNSW,
                },
            }

            _, err := g.CreateCollection(ctx, collection)
            if err != nil {
                log.Printf("Error creating collection %d: %v", index, err)
            }
        }(i)
    }

    wg.Wait()
    return nil
}
```

## Error Handling

The system provides comprehensive error handling:

```go
// Graceful error handling
docID, err := g.AddFile(ctx, filePath, options)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "connection refused"):
        log.Println("Database connection issue")
    case strings.Contains(err.Error(), "file not found"):
        log.Println("File does not exist")
    default:
        log.Printf("Unexpected error: %v", err)
    }
    return err
}
```

## Testing

The package includes extensive test coverage:

```bash
# Run all tests
go test ./...

# Run with coverage
go test -v -cover ./...

# Run stress tests
go test -v -run TestStress ./...
```

## Performance Considerations

- Use batch operations for multiple documents
- Configure appropriate vector dimensions for your use case
- Monitor memory usage during large document processing
- Use pagination for large result sets
- Consider using Store layer for frequently accessed metadata

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
