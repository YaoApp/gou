package driver

import (
	"context"
	"io"
)

// Document represents a single document in the RAG system
type Document struct {
	DocID        string                 // Document ID
	Content      string                 // Raw content
	Metadata     map[string]interface{} // Additional metadata
	Embeddings   []float32              // Vector embeddings if available
	ChunkSize    int                    // Size of each chunk if document is split
	ChunkOverlap int                    // Overlap between chunks
}

// VectorSearchOptions contains options for vector search
type VectorSearchOptions struct {
	TopK      int     // Maximum number of results to return
	MinScore  float64 // Minimum similarity score threshold
	QueryText string  // Optional text query to be vectorized
}

// SearchOptions contains parameters for search operations
type SearchOptions struct {
	Limit      int                  // Maximum number of results
	VectorOpts *VectorSearchOptions // Vector search specific options, nil for text-only search
}

// SearchResult represents a single search result
type SearchResult struct {
	DocID      string                 // Document ID
	Score      float64                // Relevance score
	Content    string                 // Matched content
	Metadata   map[string]interface{} // Document metadata
	Embeddings []float32              // Vector embeddings if available
}

// IndexConfig represents configuration for the index
type IndexConfig struct {
	Name        string            // Name of the index
	Driver      string            // Index driver (e.g., "bleve", "elastic")
	StoragePath string            // Path to store index data
	Options     map[string]string // Driver-specific options
}

// VectorizeConfig represents configuration for vectorization
type VectorizeConfig struct {
	Provider   string            // Vectorization provider (e.g., "openai", "local")
	Model      string            // Model to use for vectorization
	Dimensions int               // Vector dimensions
	Options    map[string]string // Provider-specific options
}

// TaskStatus represents the status of a batch operation
type TaskStatus string

// Task status constants
const (
	// StatusPending indicates the task is waiting to be processed
	StatusPending TaskStatus = "pending"
	// StatusRunning indicates the task is currently being processed
	StatusRunning TaskStatus = "running"
	// StatusComplete indicates the task has been successfully completed
	StatusComplete TaskStatus = "complete"
	// StatusFailed indicates the task has failed
	StatusFailed TaskStatus = "failed"
)

// TaskInfo represents information about a batch operation
type TaskInfo struct {
	TaskID    string     // Unique identifier for the task
	Status    TaskStatus // Current status of the task
	Total     int        // Total number of items to process
	Processed int        // Number of items processed
	Failed    int        // Number of items that failed
	Error     string     // Error message if task failed
	Created   int64      // Creation timestamp
	Updated   int64      // Last update timestamp
}

// Engine is the main interface for RAG operations
type Engine interface {
	// Index operations
	CreateIndex(ctx context.Context, config IndexConfig) error
	DeleteIndex(ctx context.Context, name string) error
	ListIndexes(ctx context.Context) ([]string, error)
	HasIndex(ctx context.Context, name string) (bool, error)

	// Document operations
	IndexDoc(ctx context.Context, indexName string, doc *Document) error
	IndexBatch(ctx context.Context, indexName string, docs []*Document) (string, error) // Returns TaskID
	DeleteDoc(ctx context.Context, indexName string, DocID string) error
	DeleteBatch(ctx context.Context, indexName string, DocIDs []string) (string, error) // Returns TaskID
	HasDocument(ctx context.Context, indexName string, DocID string) (bool, error)
	GetMetadata(ctx context.Context, indexName string, DocID string) (map[string]interface{}, error) // Get document metadata only

	// Task operations
	GetTaskInfo(ctx context.Context, taskID string) (*TaskInfo, error)
	ListTasks(ctx context.Context, indexName string) ([]*TaskInfo, error)
	CancelTask(ctx context.Context, taskID string) error

	// Search operations
	Search(ctx context.Context, indexName string, vector []float32, opts VectorSearchOptions) ([]SearchResult, error)

	// Utility operations
	GetDocument(ctx context.Context, indexName string, DocID string) (*Document, error)
	Close() error
}

// Vectorizer handles document vectorization
type Vectorizer interface {
	Vectorize(ctx context.Context, text string) ([]float32, error)
	VectorizeBatch(ctx context.Context, texts []string) ([][]float32, error)
	Close() error
}

// FileUploadResult represents the result of a file upload operation
type FileUploadResult struct {
	Documents []*Document // Processed documents
	TaskID    string      // Task ID for async operations
}

// FileUploadOptions contains options for file upload
type FileUploadOptions struct {
	Async        bool   // Whether to process asynchronously
	ChunkSize    int    // Size of each chunk if document is split
	ChunkOverlap int    // Overlap between chunks
	IndexName    string // Target index name
}

// FileUpload handles document file uploading and preprocessing
type FileUpload interface {
	// Upload processes content from a reader
	Upload(ctx context.Context, reader io.Reader, opts FileUploadOptions) (*FileUploadResult, error)
	// UploadFile processes content from a file path
	UploadFile(ctx context.Context, filepath string, opts FileUploadOptions) (*FileUploadResult, error)
}
