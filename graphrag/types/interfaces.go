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
	SearchSimilar(ctx context.Context, opts *SearchOptions) ([]*SearchResult, error)
	SearchMMR(ctx context.Context, opts *MMRSearchOptions) ([]*SearchResult, error)
	SearchWithScoreThreshold(ctx context.Context, opts *ScoreThresholdOptions) ([]*SearchResult, error)

	// Paginated Search Operations (for Search Engine scenarios)
	PaginatedSearchSimilar(ctx context.Context, opts *PaginatedSearchOptions) (*PaginatedSearchResult, error)
	PaginatedSearchMMR(ctx context.Context, opts *PaginatedMMRSearchOptions) (*PaginatedSearchResult, error)
	PaginatedSearchWithScoreThreshold(ctx context.Context, opts *PaginatedScoreThresholdSearchOptions) (*PaginatedSearchResult, error)
	HybridSearch(ctx context.Context, opts *HybridSearchOptions) (*PaginatedSearchResult, error)

	// Batch Operations
	BatchSearchSimilar(ctx context.Context, opts *BatchSearchOptions) ([][]*SearchResult, error)
	BatchSearchMMR(ctx context.Context, opts *BatchMMRSearchOptions) ([][]*SearchResult, error)
	BatchSearchWithScoreThreshold(ctx context.Context, opts *BatchScoreThresholdOptions) ([][]*SearchResult, error)

	// Maintenance and Stats
	GetStats(ctx context.Context, collectionName string) (*VectorStoreStats, error)
	GetSearchEngineStats(ctx context.Context, collectionName string) (*SearchEngineStats, error)
	Optimize(ctx context.Context, collectionName string) error

	// Backup and Restore
	Backup(ctx context.Context, opts *BackupOptions) error
	Restore(ctx context.Context, opts *RestoreOptions) error

	// Connection Management
	Connect(ctx context.Context, config VectorStoreConfig) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	Close() error
}

// EmbeddingFunction represents an embedding function interface
// This handles text-to-vector conversion, separate from vector storage
type EmbeddingFunction interface {
	// EmbedDocuments embeds a list of documents
	EmbedDocuments(ctx context.Context, texts []string) ([][]float64, error)

	// EmbedQuery embeds a single query
	EmbedQuery(ctx context.Context, text string) ([]float64, error)

	// GetDimension returns the dimension of the embedding vectors
	GetDimension() int
}

// ChunkingFunction represents a chunking function interface
// This handles text-to-chunk conversion, separate from chunk storage
type ChunkingFunction interface {
	Chunk(ctx context.Context, text string, options *ChunkingOptions, callback func(chunk *Chunk) error) error
	ChunkFile(ctx context.Context, file string, options *ChunkingOptions, callback func(chunk *Chunk) error) error
	ChunkStream(ctx context.Context, stream io.ReadSeeker, options *ChunkingOptions, callback func(chunk *Chunk) error) error
}

// VectorStoreFactory represents a factory for creating vector stores
type VectorStoreFactory interface {
	// Create vector stores with configuration
	CreateVectorStore(ctx context.Context, storeType string, config VectorStoreConfig) (VectorStore, error)

	// Create specific vector store types
	CreateQdrantStore(ctx context.Context, config VectorStoreConfig) (VectorStore, error)
	CreateMilvusStore(ctx context.Context, config VectorStoreConfig) (VectorStore, error)
	CreateWeaviateStore(ctx context.Context, config VectorStoreConfig) (VectorStore, error)
	CreateChromaStore(ctx context.Context, config VectorStoreConfig) (VectorStore, error)

	// Utility methods
	GetSupportedStores() []string
	ValidateConfig(storeType string, config VectorStoreConfig) error
}

// ===== High-Level Application Interfaces =====

// VectorStoreRetriever combines VectorStore and EmbeddingFunction for easy text-based operations
// This is the application layer that handles text-to-vector conversion + vector search
type VectorStoreRetriever interface {
	// Text-based search operations (internally converts text to vectors)
	SearchSimilarByText(ctx context.Context, collectionName, query string, k int, filter map[string]interface{}) ([]*SearchResult, error)
	SearchMMRByText(ctx context.Context, collectionName, query string, opts *MMRSearchOptions) ([]*SearchResult, error)

	// Text-based paginated search operations (for Search Engine scenarios)
	PaginatedSearchSimilarByText(ctx context.Context, collectionName, query string, page, pageSize int, filter map[string]interface{}) (*PaginatedSearchResult, error)
	PaginatedSearchMMRByText(ctx context.Context, collectionName, query string, opts *PaginatedMMRSearchOptions) (*PaginatedSearchResult, error)
	HybridSearchByText(ctx context.Context, collectionName, queryText string, opts *HybridSearchOptions) (*PaginatedSearchResult, error)

	// Document operations with automatic embedding
	AddTexts(ctx context.Context, collectionName string, texts []string, metadatas []map[string]interface{}) ([]string, error)
	AddDocumentsWithEmbedding(ctx context.Context, collectionName string, docs []*Document) ([]string, error)

	// Direct vector operations (bypass embedding)
	GetVectorStore() VectorStore
	GetEmbeddingFunction() EmbeddingFunction
}

// ===== Graph Database Interfaces =====

// GraphStore is an interface for graph database operations, supporting Kuzu and Neo4j
type GraphStore interface {
	// Connection and Transaction Management
	Connect(ctx context.Context, config map[string]interface{}) error
	Disconnect(ctx context.Context) error
	BeginTx(ctx context.Context) (GraphTransaction, error)
	IsConnected() bool

	// Schema Operations
	GetSchema(ctx context.Context) (*GraphSchema, error)
	CreateIndex(ctx context.Context, label string, properties []string, indexType string) error
	DropIndex(ctx context.Context, label string, properties []string) error
	CreateConstraint(ctx context.Context, constraint SchemaConstraint) error
	DropConstraint(ctx context.Context, constraint SchemaConstraint) error

	// Node Operations
	CreateNode(ctx context.Context, node Node) (*Node, error)
	CreateNodes(ctx context.Context, nodes []Node) ([]Node, error)
	GetNode(ctx context.Context, id string) (*Node, error)
	GetNodesByLabel(ctx context.Context, label string, properties map[string]interface{}) ([]Node, error)
	UpdateNode(ctx context.Context, id string, properties map[string]interface{}) error
	DeleteNode(ctx context.Context, id string) error

	// Relationship Operations
	CreateRelationship(ctx context.Context, rel Relationship) (*Relationship, error)
	CreateRelationships(ctx context.Context, rels []Relationship) ([]Relationship, error)
	GetRelationship(ctx context.Context, id string) (*Relationship, error)
	GetRelationships(ctx context.Context, nodeID string, direction string, relTypes []string) ([]Relationship, error)
	UpdateRelationship(ctx context.Context, id string, properties map[string]interface{}) error
	DeleteRelationship(ctx context.Context, id string) error

	// Query Operations
	ExecuteQuery(ctx context.Context, query string, parameters map[string]interface{}) (*GraphResult, error)
	ExecuteReadQuery(ctx context.Context, query string, parameters map[string]interface{}) (*GraphResult, error)
	ExecuteWriteQuery(ctx context.Context, query string, parameters map[string]interface{}) (*GraphResult, error)

	// Graph Traversal
	Traverse(ctx context.Context, startNodeIDs []string, opts GraphTraversalOptions) (*GraphResult, error)
	FindPaths(ctx context.Context, startNodeID, endNodeID string, opts GraphTraversalOptions) ([]Path, error)
	FindShortestPath(ctx context.Context, startNodeID, endNodeID string, maxDepth int) (*Path, error)

	// Graph Analytics
	RunCommunityDetection(ctx context.Context, opts CommunityDetectionOptions) ([]Community, error)
	ComputeNodeMetrics(ctx context.Context, nodeIDs []string, opts GraphAnalyticsOptions) ([]NodeMetrics, error)
	GetNeighborhood(ctx context.Context, nodeID string, depth int) (*GraphResult, error)

	// Knowledge Graph Operations for GraphRAG
	ExtractEntities(ctx context.Context, text string, entityTypes []string) ([]Node, error)
	ExtractRelationships(ctx context.Context, text string, entities []Node) ([]Relationship, error)
	AddKnowledgeTriples(ctx context.Context, subject, predicate, object string, properties map[string]interface{}) error
	QueryKnowledge(ctx context.Context, query string, opts *GraphQueryOptions) (*GraphResult, error)

	// Vector Integration (for hybrid GraphRAG)
	AddNodeEmbedding(ctx context.Context, nodeID string, embedding []float64) error
	FindSimilarNodes(ctx context.Context, embedding []float64, k int, threshold float64) ([]Node, error)
	HybridSearch(ctx context.Context, textQuery string, embedding []float64, opts *GraphQueryOptions) (*GraphResult, error)

	// Batch Operations
	ExecuteBatch(ctx context.Context, operations []GraphOperation) error

	// Utility Operations
	GetStats(ctx context.Context) (map[string]interface{}, error)
	ExportGraph(ctx context.Context, format string) ([]byte, error)
	ImportGraph(ctx context.Context, data []byte, format string) error
}

// GraphTransaction represents a graph database transaction
type GraphTransaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	ExecuteQuery(ctx context.Context, query string, parameters map[string]interface{}) (*GraphResult, error)
	CreateNode(ctx context.Context, node Node) (*Node, error)
	CreateRelationship(ctx context.Context, rel Relationship) (*Relationship, error)
}

// GraphRetriever represents a graph-based retriever for RAG
type GraphRetriever interface {
	// Vector-based graph retrieval
	VectorGraphSearch(ctx context.Context, query string, k int) (*GraphResult, error)

	// Text-to-Cypher/GQL retrieval
	Text2GraphQuery(ctx context.Context, naturalLanguageQuery string) (string, error)
	ExecuteGeneratedQuery(ctx context.Context, query string) (*GraphResult, error)

	// Community-based retrieval (Global search)
	CommunitySearch(ctx context.Context, query string, level int) ([]Community, error)

	// Local graph exploration
	LocalGraphSearch(ctx context.Context, startEntities []string, query string, depth int) (*GraphResult, error)

	// Hybrid retrieval combining vector and graph
	HybridGraphRetrieval(ctx context.Context, query string, vectorK int, graphDepth int) (*GraphResult, error)
}

// GraphStoreFactory represents a factory for creating graph stores
type GraphStoreFactory interface {
	// CreateKuzuStore creates a Kuzu graph store
	CreateKuzuStore(ctx context.Context, dbPath string, config map[string]interface{}) (GraphStore, error)

	// CreateNeo4jStore creates a Neo4j graph store
	CreateNeo4jStore(ctx context.Context, uri, username, password string, config map[string]interface{}) (GraphStore, error)

	// CreateFromConfig creates a graph store from configuration
	CreateFromConfig(ctx context.Context, config map[string]interface{}) (GraphStore, error)
}
