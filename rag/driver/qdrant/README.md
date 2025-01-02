# Qdrant RAG Engine

This package implements the RAG (Retrieval-Augmented Generation) Engine interface using Qdrant as the vector store backend.

## Features

- Vector similarity search with cosine distance
- Document chunking with configurable size and overlap
- Async and sync document indexing
- Batch operations support
- Automatic vector generation using configurable vectorizer
- Task management for async operations

## Requirements

- Qdrant server (v1.0.0 or later)
- OpenAI API key (for default vectorizer)

## Installation

```bash
go get github.com/yaoapp/gou/rag/qdrant
```

## Usage

### Basic Setup

```go
import (
    "github.com/yaoapp/gou/rag"
    "github.com/yaoapp/gou/rag/qdrant"
    "github.com/yaoapp/gou/rag/openai"
)

// Create a vectorizer (using OpenAI as example)
vectorizer, err := openai.New(openai.Config{
    APIKey: "your-openai-api-key",
    Model:  "text-embedding-ada-002",
})
if err != nil {
    log.Fatal(err)
}

// Create Qdrant engine
engine, err := qdrant.NewEngine(qdrant.Config{
    Host:       "localhost",
    Port:       6334,
    Vectorizer: vectorizer,
})
if err != nil {
    log.Fatal(err)
}
defer engine.Close()

// Create an index
err = engine.CreateIndex(context.Background(), rag.IndexConfig{
    Name:   "my_index",
    Driver: "qdrant",
})
```

### Document Operations

```go
// Index a single document
doc := &rag.Document{
    DocID:    "00000000-0000-0000-0000-000000000001",
    Content:  "This is a test document",
    Metadata: map[string]interface{}{"type": "test"},
}
err = engine.IndexDoc(context.Background(), "my_index", doc)

// Batch indexing
docs := []*rag.Document{
    {
        DocID:    "00000000-0000-0000-0000-000000000002",
        Content:  "Document 2",
    },
    {
        DocID:    "00000000-0000-0000-0000-000000000003",
        Content:  "Document 3",
    },
}
taskID, err := engine.IndexBatch(context.Background(), "my_index", docs)

// Check task status
taskInfo, err := engine.GetTaskInfo(context.Background(), taskID)
```

### Search Operations

```go
// Search by text
results, err := engine.Search(context.Background(), "my_index", nil, rag.VectorSearchOptions{
    QueryText: "test document",
    TopK:      5,
    MinScore:  0.7,
})

// Search by vector
vector := []float32{0.1, 0.2, ...} // Your vector here
results, err = engine.Search(context.Background(), "my_index", vector, rag.VectorSearchOptions{
    TopK:     5,
    MinScore: 0.7,
})
```

### File Upload

```go
// Create file uploader
uploader := qdrant.NewFileUploader(engine)

// Upload from reader
reader := strings.NewReader("Document content")
result, err := uploader.Upload(context.Background(), reader, rag.FileUploadOptions{
    IndexName:    "my_index",
    ChunkSize:    1000,
    ChunkOverlap: 200,
    Async:        true,
})

// Upload from file
result, err = uploader.UploadFile(context.Background(), "path/to/file.txt", rag.FileUploadOptions{
    IndexName:    "my_index",
    ChunkSize:    1000,
    ChunkOverlap: 200,
})
```

## Configuration

### Engine Configuration

- `Host`: Qdrant server host address (default: "localhost")
- `Port`: Qdrant server gRPC port (default: 6334)
- `APIKey`: Optional API key for authentication
- `Vectorizer`: Implementation of the `rag.Vectorizer` interface

### File Upload Options

- `IndexName`: Target index name
- `ChunkSize`: Size of each document chunk (default: 1000)
- `ChunkOverlap`: Overlap between chunks
- `Async`: Whether to process documents asynchronously

## Error Handling

The package provides detailed error messages for common scenarios:

- Collection/index not found
- Document not found
- Invalid vector dimensions
- Connection issues
- Task status errors

## Best Practices

1. Always use proper UUID format for document IDs
2. Close the engine when done to release resources
3. Use batch operations for large document sets
4. Set appropriate chunk sizes based on your use case
5. Handle async operations properly by checking task status

## License

MIT License
