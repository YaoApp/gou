# RAG (Retrieval-Augmented Generation) Package

A powerful and flexible Retrieval-Augmented Generation (RAG) system that supports vector search and document management with multiple backend drivers.

## Features

- Multiple vector store support (currently supports Qdrant)
- Flexible vectorization with OpenAI embeddings
- Document management with chunking and metadata
- Asynchronous batch operations
- File upload and processing capabilities
- Vector similarity search

## Installation

```go
import "github.com/yaoapp/gou/rag"
```

## Quick Start

### 1. Initialize the Vectorizer

First, create a vectorizer (e.g., using OpenAI):

```go
vectorizerConfig := driver.VectorizeConfig{
    Provider: "openai",
    Model:    "text-embedding-ada-002",
    Options: map[string]string{
        "api_key": "your-openai-api-key",
    },
}

vectorizer, err := rag.NewVectorizer(rag.DriverOpenAI, vectorizerConfig)
if err != nil {
    log.Fatal(err)
}
defer vectorizer.Close()
```

### 2. Create a Vector Store Engine

Initialize a vector store engine (e.g., Qdrant):

```go
indexConfig := driver.IndexConfig{
    Name:   "my-index",
    Driver: "qdrant",
    Options: map[string]string{
        "host":    "localhost",
        "port":    "6333",
        "api_key": "your-qdrant-api-key", // Optional
    },
}

engine, err := rag.NewEngine(rag.DriverQdrant, indexConfig, vectorizer)
if err != nil {
    log.Fatal(err)
}
defer engine.Close()
```

### 3. Upload Documents

Create a file upload handler:

```go
fileUploader, err := rag.NewFileUpload(rag.DriverQdrant, engine, vectorizer)
if err != nil {
    log.Fatal(err)
}

// Upload a file
opts := driver.FileUploadOptions{
    Async:        true,
    ChunkSize:    1000,
    ChunkOverlap: 100,
    IndexName:    "my-index",
}

result, err := fileUploader.UploadFile(context.Background(), "path/to/your/file", opts)
if err != nil {
    log.Fatal(err)
}
```

### 4. Search Documents

Perform vector similarity search:

```go
searchOpts := driver.VectorSearchOptions{
    TopK:      5,
    MinScore:  0.7,
    QueryText: "your search query",
}

// The engine will automatically vectorize the query text using the configured vectorizer
results, err := engine.Search(context.Background(), "my-index", nil, searchOpts)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("Document ID: %s, Score: %f\n", result.DocID, result.Score)
    fmt.Printf("Content: %s\n", result.Content)
}
```

## Configuration

### Vectorizer Configuration

The OpenAI vectorizer supports the following options:

```go
VectorizeConfig{
    Provider: "openai",
    Model:    "text-embedding-ada-002", // OpenAI embedding model
    Options: map[string]string{
        "api_key": "your-openai-api-key",
    },
}
```

### Qdrant Engine Configuration

The Qdrant vector store supports the following options:

```go
IndexConfig{
    Name:   "index-name",
    Driver: "qdrant",
    Options: map[string]string{
        "host":    "localhost",    // Qdrant server host
        "port":    "6333",         // Qdrant server port
        "api_key": "your-api-key", // Optional API key
    },
}
```

## Document Processing

Documents can be processed with customizable chunking:

```go
doc := &driver.Document{
    DocID:        "unique-id",
    Content:      "Your document content",
    Metadata:     map[string]interface{}{"key": "value"},
    ChunkSize:    1000,    // Characters per chunk
    ChunkOverlap: 100,     // Overlap between chunks
}

err := engine.IndexDoc(context.Background(), "my-index", doc)
```

## Batch Operations

For large-scale operations, use batch processing:

```go
docs := []*driver.Document{
    // ... multiple documents
}

taskID, err := engine.IndexBatch(context.Background(), "my-index", docs)
if err != nil {
    log.Fatal(err)
}

// Check task status
taskInfo, err := engine.GetTaskInfo(context.Background(), taskID)
if err != nil {
    log.Fatal(err)
}
```

## Error Handling

The package provides detailed error information for all operations. Always check returned errors and task statuses for batch operations.

## License

[Add your license information here]
