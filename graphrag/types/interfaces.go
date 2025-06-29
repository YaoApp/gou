package types

import (
	"context"
	"io"
)

// ===== Vector Database Interfaces =====

// VectorStore defines the interface for vector storage and retrieval
// Only handles vectors, text-to-vector conversion is done externally via EmbeddingFunction
type VectorStore interface {
	// Collection Management
	CreateCollection(ctx context.Context, config *VectorStoreConfig) error
	ListCollections(ctx context.Context) ([]string, error)
	DropCollection(ctx context.Context, collectionName string) error
	CollectionExists(ctx context.Context, collectionName string) (bool, error)
	DescribeCollection(ctx context.Context, collectionName string) (*VectorStoreStats, error)

	// Collection State Management (for databases like Milvus that support load/release)
	LoadCollection(ctx context.Context, collectionName string) error
	ReleaseCollection(ctx context.Context, collectionName string) error
	GetLoadState(ctx context.Context, collectionName string) (LoadState, error)

	// Document Operations (vector-focused)
	AddDocuments(ctx context.Context, opts *AddDocumentOptions) ([]string, error)
	GetDocuments(ctx context.Context, ids []string, opts *GetDocumentOptions) ([]*Document, error)
	DeleteDocuments(ctx context.Context, opts *DeleteDocumentOptions) error

	// Document Listing and Pagination
	ListDocuments(ctx context.Context, opts *ListDocumentsOptions) (*PaginatedDocumentsResult, error)
	ScrollDocuments(ctx context.Context, opts *ScrollOptions) (*ScrollResult, error)

	// Vector Search Operations (core functionality)
	SearchSimilar(ctx context.Context, opts *SearchOptions) (*SearchResult, error)
	SearchMMR(ctx context.Context, opts *MMRSearchOptions) (*SearchResult, error)
	SearchWithScoreThreshold(ctx context.Context, opts *ScoreThresholdOptions) (*SearchResult, error)
	SearchHybrid(ctx context.Context, opts *HybridSearchOptions) (*SearchResult, error)
	SearchBatch(ctx context.Context, opts []SearchOptionsInterface) ([]*SearchResult, error)

	// Maintenance and Stats
	GetStats(ctx context.Context, collectionName string) (*VectorStoreStats, error)
	GetSearchEngineStats(ctx context.Context, collectionName string) (*SearchEngineStats, error)
	Optimize(ctx context.Context, collectionName string) error

	// Backup and Restore
	Backup(ctx context.Context, writer io.Writer, opts *BackupOptions) error
	Restore(ctx context.Context, reader io.Reader, opts *RestoreOptions) error

	// Connection Management
	Connect(ctx context.Context, config VectorStoreConfig) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	Close() error
}

// ===== Chunking Interfaces =====

// Chunking represents a chunking function interface
// This handles text-to-chunk conversion, separate from chunk storage
type Chunking interface {
	Chunk(ctx context.Context, text string, options *ChunkingOptions, callback func(chunk *Chunk) error) error
	ChunkFile(ctx context.Context, file string, options *ChunkingOptions, callback func(chunk *Chunk) error) error
	ChunkStream(ctx context.Context, stream io.ReadSeeker, options *ChunkingOptions, callback func(chunk *Chunk) error) error
}

// ===== Embedding Interfaces =====

// Embedding represents an embedding function interface
// This handles text-to-vector conversion, separate from vector storage
type Embedding interface {
	// EmbedDocuments embeds a list of documents
	EmbedDocuments(ctx context.Context, texts []string, callback ...EmbeddingProgress) (*EmbeddingResults, error)

	// EmbedQuery embeds a single query
	EmbedQuery(ctx context.Context, text string, callback ...EmbeddingProgress) (*EmbeddingResult, error)

	// GetDimension returns the dimension of the embedding vectors
	GetDimension() int

	// GetModel returns the model of the embedding function
	GetModel() string
}

// EmbeddingProgress defines the callback function for progress reporting with flexible payload
type EmbeddingProgress func(status EmbeddingStatus, payload EmbeddingPayload)

// ===== Extraction Interfaces =====

// Extraction represents an extraction function interface
type Extraction interface {
	ExtractDocuments(ctx context.Context, texts []string, callback ...ExtractionProgress) ([]*ExtractionResult, error)
	ExtractQuery(ctx context.Context, text string, callback ...ExtractionProgress) (*ExtractionResult, error)
}

// ExtractionProgress defines the callback function for progress reporting with flexible payload
type ExtractionProgress func(status ExtractionStatus, payload ExtractionPayload)

// ===== Graph Database Interfaces =====

// GraphStore defines the interface for graph storage and retrieval
// Similar to VectorStore design - focused on core operations with flexible data structures
type GraphStore interface {
	// Connection Management
	Connect(ctx context.Context, config GraphStoreConfig) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	Close() error

	// Graph Management (similar to Collection Management in VectorStore)
	CreateGraph(ctx context.Context, graphName string, config *GraphConfig) error
	DropGraph(ctx context.Context, graphName string) error
	GraphExists(ctx context.Context, graphName string) (bool, error)
	ListGraphs(ctx context.Context) ([]string, error)
	DescribeGraph(ctx context.Context, graphName string) (*GraphStats, error)

	// Node Operations (core functionality)
	AddNodes(ctx context.Context, opts *AddNodesOptions) ([]string, error) // Upsert option handles updates
	GetNodes(ctx context.Context, opts *GetNodesOptions) ([]*GraphNode, error)
	DeleteNodes(ctx context.Context, opts *DeleteNodesOptions) error

	// Relationship Operations
	AddRelationships(ctx context.Context, opts *AddRelationshipsOptions) ([]string, error) // Upsert option handles updates
	GetRelationships(ctx context.Context, opts *GetRelationshipsOptions) ([]*GraphRelationship, error)
	DeleteRelationships(ctx context.Context, opts *DeleteRelationshipsOptions) error

	// Query Operations (flexible query interface)
	Query(ctx context.Context, opts *GraphQueryOptions) (*GraphResult, error)               // General-purpose graph query with Cypher, traversal, etc.
	Communities(ctx context.Context, opts *CommunityDetectionOptions) ([]*Community, error) // Community detection and analysis

	// Schema Operations (optional - for databases that support schema)
	GetSchema(ctx context.Context, graphName string) (*DynamicGraphSchema, error)
	CreateIndex(ctx context.Context, opts *CreateIndexOptions) error
	DropIndex(ctx context.Context, opts *DropIndexOptions) error

	// Statistics and Maintenance
	GetStats(ctx context.Context, graphName string) (*GraphStats, error)
	Optimize(ctx context.Context, graphName string) error

	// Backup and Restore
	Backup(ctx context.Context, writer io.Writer, opts *GraphBackupOptions) error
	Restore(ctx context.Context, reader io.Reader, opts *GraphRestoreOptions) error
}
