package types

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ===== Vector Index Type Enums =====

// IndexType represents the vector index algorithm type
type IndexType string

const (
	// IndexTypeHNSW Hierarchical Navigable Small World - high accuracy, suitable for large-scale data
	IndexTypeHNSW IndexType = "hnsw"

	// IndexTypeIVF Inverted File Index - balanced accuracy and performance
	IndexTypeIVF IndexType = "ivf"

	// IndexTypeFlat Brute force search - highest accuracy, suitable for small datasets
	IndexTypeFlat IndexType = "flat"

	// IndexTypeLSH Locality Sensitive Hashing - fast approximate search
	IndexTypeLSH IndexType = "lsh"
)

// String returns the string representation of IndexType
func (it IndexType) String() string {
	return string(it)
}

// IsValid validates if the index type is supported
func (it IndexType) IsValid() bool {
	switch it {
	case IndexTypeHNSW, IndexTypeIVF, IndexTypeFlat, IndexTypeLSH:
		return true
	default:
		return false
	}
}

// GetSupportedIndexTypes returns all supported index types
func GetSupportedIndexTypes() []IndexType {
	return []IndexType{
		IndexTypeHNSW,
		IndexTypeIVF,
		IndexTypeFlat,
		IndexTypeLSH,
	}
}

// ===== Distance Metric Enums =====

// DistanceMetric represents the distance measurement method
type DistanceMetric string

const (
	// DistanceCosine Cosine distance - most commonly used, suitable for text vectors
	DistanceCosine DistanceMetric = "cosine"

	// DistanceEuclidean Euclidean distance - suitable for image vectors
	DistanceEuclidean DistanceMetric = "euclidean"

	// DistanceDot Dot product distance - suitable for normalized vectors
	DistanceDot DistanceMetric = "dot"

	// DistanceManhattan Manhattan distance - suitable for sparse vectors
	DistanceManhattan DistanceMetric = "manhattan"
)

// String returns the string representation of DistanceMetric
func (dm DistanceMetric) String() string {
	return string(dm)
}

// IsValid validates if the distance metric is supported
func (dm DistanceMetric) IsValid() bool {
	switch dm {
	case DistanceCosine, DistanceEuclidean, DistanceDot, DistanceManhattan:
		return true
	default:
		return false
	}
}

// GetSupportedDistanceMetrics returns all supported distance metrics
func GetSupportedDistanceMetrics() []DistanceMetric {
	return []DistanceMetric{
		DistanceCosine,
		DistanceEuclidean,
		DistanceDot,
		DistanceManhattan,
	}
}

// ==== Chunking Enums =====

// ChunkingStatus represents the status of a chunk
type ChunkingStatus string

const (
	// ChunkingStatusPending is the status of a chunk that is pending
	ChunkingStatusPending ChunkingStatus = "pending"

	// ChunkingStatusProcessing is the status of a chunk that is processing
	ChunkingStatusProcessing ChunkingStatus = "processing"

	// ChunkingStatusCompleted is the status of a chunk that is completed
	ChunkingStatusCompleted ChunkingStatus = "completed"

	// ChunkingStatusFailed is the status of a chunk that is failed
	ChunkingStatusFailed ChunkingStatus = "failed"
)

// ===== Chunking Types =====

// TextPosition represents position information for text-based content
type TextPosition struct {
	StartIndex int `json:"start_index"` // Character offset from beginning of text
	EndIndex   int `json:"end_index"`   // Character offset end position
	StartLine  int `json:"start_line"`  // Line number where chunk starts
	EndLine    int `json:"end_line"`    // Line number where chunk ends
}

// MediaPosition represents position information for media content
type MediaPosition struct {
	StartTime int `json:"start_time"` // Start time in seconds (for video/audio)
	EndTime   int `json:"end_time"`   // End time in seconds (for video/audio)
	Page      int `json:"page"`       // Page number (for PDF, Word, etc.)
}

// Chunk represents a chunk of content with position information
type Chunk struct {
	ID       string       `json:"id,omitempty"`
	Text     string       `json:"text"`
	Type     ChunkingType `json:"type"` // Chunking type from ChunkingType enum
	ParentID string       `json:"parent_id,omitempty"`
	Depth    int          `json:"depth"`
	Leaf     bool         `json:"leaf"` // Whether the chunk is a leaf node
	Root     bool         `json:"root"` // Whether the chunk is a root node

	// Status and index
	Index  int            `json:"index"`  // Index of the chunk in the parent chunk
	Status ChunkingStatus `json:"status"` // Status of the chunk, for example, "pending", "processing", "completed", "failed"

	// Parents of the chunk
	Parents []Chunk `json:"parents"` // Parents of the chunk

	// Position information (only one should be populated based on content type)
	TextPos  *TextPosition  `json:"text_position,omitempty"`  // For text, code, etc.
	MediaPos *MediaPosition `json:"media_position,omitempty"` // For PDF, video, audio, etc.
}

// ChunkingOptions represents options for chunking
type ChunkingOptions struct {
	Type            ChunkingType     `json:"type,omitempty"`  // Content type, auto-detected if not provided
	Size            int              `json:"size"`            // For text, PDF, Word, only, default is QA 300, Code 800,
	Overlap         int              `json:"overlap"`         // For text, PDF, Word, only, default is QA 20, Code 100,
	MaxDepth        int              `json:"max_depth"`       // For text, PDF, Word, only, default is 5
	SizeMultiplier  int              `json:"size_multiplier"` // Base multiplier for chunk size calculation, default is 3
	MaxConcurrent   int              `json:"max_concurrent"`
	SemanticOptions *SemanticOptions `json:"semantic_options"` // For Semantic recognition, etc.
	VideoConnector  string           `json:"video_connector"`  // For Video recognition, etc.
	AudioConnector  string           `json:"audio_connector"`  // For Audio recognition, etc.
	ImageConnector  string           `json:"image_connector"`  // For Image recognition, etc.
	FFmpegPath      string           `json:"ffmpeg_path"`      // ffmpeg path, for video, audio, etc.
	FFprobePath     string           `json:"ffprobe_path"`     // ffprobe path, for video, audio, etc.
	FFmpegOptions   string           `json:"ffmpeg_options"`   // ffmpeg options, for video, audio, etc.
	FFprobeOptions  string           `json:"ffprobe_options"`  // ffprobe options, for video, audio, etc.
}

// SemanticOptions represents options for semantic recognition
type SemanticOptions struct {
	Connector     string `json:"connector"`      // For Semantic recognition, etc.
	ContextSize   int    `json:"context_size"`   // Context size for Semantic recognition. Default L1 Size (ChunkSize * 6)
	Options       string `json:"options"`        // Model options, for example, temperature, top_p, etc.
	Prompt        string `json:"prompt"`         // System prompt for Semantic recognition.
	Toolcall      bool   `json:"toolcall"`       // Whether to use toolcall for Semantic recognition.
	MaxRetry      int    `json:"max_retry"`      // Max retry times for Semantic recognition.
	MaxConcurrent int    `json:"max_concurrent"` // Max concurrent requests for Semantic recognition.
}

// ChunkingType for chunking type
type ChunkingType string

const (
	// ChunkingTypeText is for text
	ChunkingTypeText ChunkingType = "text"

	// ChunkingTypeCode is for Code
	ChunkingTypeCode ChunkingType = "code"

	// ChunkingTypePDF is for PDF
	ChunkingTypePDF ChunkingType = "pdf"

	// ChunkingTypeWord is for Word
	ChunkingTypeWord ChunkingType = "word"

	// ChunkingTypeCSV is for CSV
	ChunkingTypeCSV ChunkingType = "csv"

	// ChunkingTypeExcel is for Excel
	ChunkingTypeExcel ChunkingType = "excel"

	// ChunkingTypeJSON is for JSON
	ChunkingTypeJSON ChunkingType = "json"

	// ChunkingTypeImage is for Image
	ChunkingTypeImage ChunkingType = "image"

	// ChunkingTypeVideo is for Video
	ChunkingTypeVideo ChunkingType = "video"

	// ChunkingTypeAudio is for Audio
	ChunkingTypeAudio ChunkingType = "audio"
)

// MimeToChunkingType maps MIME types to ChunkingType
var MimeToChunkingType = map[string]ChunkingType{
	// Text types
	"text/plain":      ChunkingTypeText,
	"text/markdown":   ChunkingTypeText,
	"text/html":       ChunkingTypeText,
	"text/xml":        ChunkingTypeText,
	"text/rtf":        ChunkingTypeText,
	"application/rtf": ChunkingTypeText,

	// Code types
	"text/x-go":              ChunkingTypeCode,
	"text/x-python":          ChunkingTypeCode,
	"text/x-java":            ChunkingTypeCode,
	"text/x-c":               ChunkingTypeCode,
	"text/x-c++":             ChunkingTypeCode,
	"text/x-csharp":          ChunkingTypeCode,
	"text/javascript":        ChunkingTypeCode,
	"application/javascript": ChunkingTypeCode,
	"text/typescript":        ChunkingTypeCode,
	"application/typescript": ChunkingTypeCode,
	"text/x-php":             ChunkingTypeCode,
	"text/x-ruby":            ChunkingTypeCode,
	"text/x-shell":           ChunkingTypeCode,
	"application/x-sh":       ChunkingTypeCode,

	// JSON types
	"application/json": ChunkingTypeJSON,
	"text/json":        ChunkingTypeJSON,

	// PDF types
	"application/pdf": ChunkingTypePDF,

	// Word types
	"application/msword": ChunkingTypeWord,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ChunkingTypeWord,

	// Excel types
	"application/vnd.ms-excel": ChunkingTypeExcel,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": ChunkingTypeExcel,

	// CSV types
	"text/csv":        ChunkingTypeCSV,
	"application/csv": ChunkingTypeCSV,

	// Image types
	"image/jpeg":    ChunkingTypeImage,
	"image/jpg":     ChunkingTypeImage,
	"image/png":     ChunkingTypeImage,
	"image/gif":     ChunkingTypeImage,
	"image/bmp":     ChunkingTypeImage,
	"image/webp":    ChunkingTypeImage,
	"image/tiff":    ChunkingTypeImage,
	"image/svg+xml": ChunkingTypeImage,

	// Video types
	"video/mp4":       ChunkingTypeVideo,
	"video/avi":       ChunkingTypeVideo,
	"video/mov":       ChunkingTypeVideo,
	"video/wmv":       ChunkingTypeVideo,
	"video/flv":       ChunkingTypeVideo,
	"video/webm":      ChunkingTypeVideo,
	"video/mkv":       ChunkingTypeVideo,
	"video/quicktime": ChunkingTypeVideo,

	// Audio types
	"audio/mp3":  ChunkingTypeAudio,
	"audio/mpeg": ChunkingTypeAudio,
	"audio/wav":  ChunkingTypeAudio,
	"audio/flac": ChunkingTypeAudio,
	"audio/aac":  ChunkingTypeAudio,
	"audio/ogg":  ChunkingTypeAudio,
	"audio/wma":  ChunkingTypeAudio,
	"audio/m4a":  ChunkingTypeAudio,
}

// GetChunkingTypeFromMime returns the ChunkingType for a given MIME type
func GetChunkingTypeFromMime(mimeType string) ChunkingType {
	if chunkingType, exists := MimeToChunkingType[mimeType]; exists {
		return chunkingType
	}
	// Default to text for unknown types
	return ChunkingTypeText
}

// GetChunkingTypeFromFilename returns the ChunkingType based on file extension
func GetChunkingTypeFromFilename(filename string) ChunkingType {
	// Simple extension-based detection as fallback
	extension := filepath.Ext(strings.ToLower(filename))

	switch extension {
	case ".go", ".py", ".java", ".c", ".cpp", ".cs", ".js", ".ts", ".php", ".rb", ".sh":
		return ChunkingTypeCode
	case ".json":
		return ChunkingTypeJSON
	case ".pdf":
		return ChunkingTypePDF
	case ".doc", ".docx":
		return ChunkingTypeWord
	case ".xls", ".xlsx":
		return ChunkingTypeExcel
	case ".csv":
		return ChunkingTypeCSV
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tiff", ".svg":
		return ChunkingTypeImage
	case ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".mkv":
		return ChunkingTypeVideo
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma", ".m4a":
		return ChunkingTypeAudio
	case ".md", ".txt", ".html", ".xml", ".rtf":
		return ChunkingTypeText
	default:
		return ChunkingTypeText
	}
}

// ===== Vector Store Types =====

// Document represents a document with content and metadata
type Document struct {
	ID          string                 `json:"id,omitempty"`
	PageContent string                 `json:"page_content"`
	Vector      []float64              `json:"vector,omitempty"` // Document embedding vector
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SearchResult represents a search result with document and score
type SearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
}

// VectorStoreConfig represents configuration for vector store
type VectorStoreConfig struct {
	// Vector Configuration
	Dimension int            `json:"dimension"`  // Vector dimension (e.g., 1536 for OpenAI embeddings)
	Distance  DistanceMetric `json:"distance"`   // Distance metric
	IndexType IndexType      `json:"index_type"` // Index type

	// Index Parameters (for HNSW)
	M              int `json:"m,omitempty"`               // Number of bidirectional links for each node (HNSW)
	EfConstruction int `json:"ef_construction,omitempty"` // Size of dynamic candidate list (HNSW)
	EfSearch       int `json:"ef_search,omitempty"`       // Size of dynamic candidate list for search (HNSW)

	// Index Parameters (for IVF)
	NumLists  int `json:"num_lists,omitempty"`  // Number of clusters (IVF)
	NumProbes int `json:"num_probes,omitempty"` // Number of clusters to search (IVF)

	// Storage Configuration
	CollectionName string `json:"collection_name"`        // Collection/Table name
	PersistPath    string `json:"persist_path,omitempty"` // Path for persistent storage

	// Database-specific settings
	DatabaseURL    string                 `json:"database_url,omitempty"`    // Database connection URL
	ConnectionPool int                    `json:"connection_pool,omitempty"` // Connection pool size
	Timeout        int                    `json:"timeout,omitempty"`         // Operation timeout in seconds
	ExtraParams    map[string]interface{} `json:"extra_params,omitempty"`    // Database-specific parameters
}

// Validate validates the vector store configuration
func (c *VectorStoreConfig) Validate() error {
	if !c.IndexType.IsValid() {
		return fmt.Errorf("invalid index type: %s, supported types: %v", c.IndexType, GetSupportedIndexTypes())
	}
	if !c.Distance.IsValid() {
		return fmt.Errorf("invalid distance metric: %s, supported metrics: %v", c.Distance, GetSupportedDistanceMetrics())
	}
	if c.Dimension <= 0 {
		return fmt.Errorf("dimension must be positive, got: %d", c.Dimension)
	}
	if c.CollectionName == "" {
		return fmt.Errorf("collection name cannot be empty")
	}
	return nil
}

// AddDocumentOptions represents options for adding documents
type AddDocumentOptions struct {
	CollectionName string      `json:"collection_name"`
	Documents      []*Document `json:"documents"`            // Documents to add (ID in Document will be used if provided, otherwise auto-generated)
	BatchSize      int         `json:"batch_size,omitempty"` // Batch size for bulk insert
	Timeout        int         `json:"timeout,omitempty"`    // Operation timeout in seconds
	Upsert         bool        `json:"upsert,omitempty"`     // If true, update existing documents with same ID
}

// SearchOptions represents options for similarity search
type SearchOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVector    []float64              `json:"query_vector"`     // Query vector for similarity search
	K              int                    `json:"k,omitempty"`      // Number of documents to return
	Filter         map[string]interface{} `json:"filter,omitempty"` // Metadata filter

	// Search-specific parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter (HNSW)
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes (IVF)
	Rescore     bool `json:"rescore,omitempty"`     // Whether to rescore results
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search
	Timeout     int  `json:"timeout,omitempty"`     // Search timeout in milliseconds
}

// MMRSearchOptions represents options for maximal marginal relevance search
type MMRSearchOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVector    []float64              `json:"query_vector"`          // Query vector for similarity search
	K              int                    `json:"k,omitempty"`           // Number of documents to return
	FetchK         int                    `json:"fetch_k,omitempty"`     // Number of documents to fetch for MMR algorithm
	LambdaMult     float64                `json:"lambda_mult,omitempty"` // Diversity parameter (0-1, 0=max diversity, 1=max similarity)
	Filter         map[string]interface{} `json:"filter,omitempty"`      // Metadata filter

	// Search parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search
}

// ScoreThresholdOptions represents options for similarity search with score threshold
type ScoreThresholdOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVector    []float64              `json:"query_vector"`     // Query vector for similarity search
	ScoreThreshold float64                `json:"score_threshold"`  // Minimum relevance score threshold
	K              int                    `json:"k,omitempty"`      // Number of documents to return
	Filter         map[string]interface{} `json:"filter,omitempty"` // Metadata filter

	// Search parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search
}

// ===== Batch Search Options =====

// BatchSearchOptions represents options for batch similarity search
type BatchSearchOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVectors   [][]float64            `json:"query_vectors"`    // Multiple query vectors for batch search
	K              int                    `json:"k,omitempty"`      // Number of documents to return per query
	Filter         map[string]interface{} `json:"filter,omitempty"` // Metadata filter

	// Search-specific parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter (HNSW)
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes (IVF)
	Rescore     bool `json:"rescore,omitempty"`     // Whether to rescore results
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search
	Timeout     int  `json:"timeout,omitempty"`     // Search timeout in milliseconds
}

// BatchMMRSearchOptions represents options for batch maximal marginal relevance search
type BatchMMRSearchOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVectors   [][]float64            `json:"query_vectors"`         // Multiple query vectors for batch search
	K              int                    `json:"k,omitempty"`           // Number of documents to return per query
	FetchK         int                    `json:"fetch_k,omitempty"`     // Number of documents to fetch for MMR algorithm
	LambdaMult     float64                `json:"lambda_mult,omitempty"` // Diversity parameter (0-1, 0=max diversity, 1=max similarity)
	Filter         map[string]interface{} `json:"filter,omitempty"`      // Metadata filter

	// Search parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search
}

// BatchScoreThresholdOptions represents options for batch similarity search with score threshold
type BatchScoreThresholdOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVectors   [][]float64            `json:"query_vectors"`    // Multiple query vectors for batch search
	ScoreThreshold float64                `json:"score_threshold"`  // Minimum relevance score threshold
	K              int                    `json:"k,omitempty"`      // Number of documents to return per query
	Filter         map[string]interface{} `json:"filter,omitempty"` // Metadata filter

	// Search parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search
}

// RetrieverOptions represents options for creating a retriever
type RetrieverOptions struct {
	SearchType   string                 `json:"search_type,omitempty"`   // "similarity", "mmr", or "similarity_score_threshold"
	SearchKwargs map[string]interface{} `json:"search_kwargs,omitempty"` // Search-specific parameters
}

// VectorStoreStats represents statistics about the vector store
type VectorStoreStats struct {
	TotalVectors   int64                  `json:"total_vectors"`
	Dimension      int                    `json:"dimension"`
	IndexType      IndexType              `json:"index_type"`
	DistanceMetric DistanceMetric         `json:"distance_metric"`
	IndexSize      int64                  `json:"index_size_bytes,omitempty"`
	MemoryUsage    int64                  `json:"memory_usage_bytes,omitempty"`
	ExtraStats     map[string]interface{} `json:"extra_stats,omitempty"`
}

// ===== Graph Database Types =====

// Node represents a graph node
type Node struct {
	ID         string                 `json:"id"`
	Labels     []string               `json:"labels,omitempty"`     // Node labels/types
	Properties map[string]interface{} `json:"properties,omitempty"` // Node properties
}

// Relationship represents a graph relationship/edge
type Relationship struct {
	ID         string                 `json:"id,omitempty"`
	Type       string                 `json:"type"`
	StartNode  string                 `json:"start_node"`           // Start node ID
	EndNode    string                 `json:"end_node"`             // End node ID
	Properties map[string]interface{} `json:"properties,omitempty"` // Relationship properties
}

// Path represents a graph path
type Path struct {
	Nodes         []Node         `json:"nodes"`
	Relationships []Relationship `json:"relationships"`
	Length        int            `json:"length"`
}

// GraphResult represents a graph query result
type GraphResult struct {
	Nodes         []Node         `json:"nodes,omitempty"`
	Relationships []Relationship `json:"relationships,omitempty"`
	Paths         []Path         `json:"paths,omitempty"`
	Records       []interface{}  `json:"records,omitempty"` // Raw query results
}

// GraphSchema represents the graph database schema
type GraphSchema struct {
	NodeLabels        []string            `json:"node_labels"`
	RelationshipTypes []string            `json:"relationship_types"`
	NodeProperties    map[string][]string `json:"node_properties"` // label -> property names
	RelProperties     map[string][]string `json:"rel_properties"`  // type -> property names
	Constraints       []SchemaConstraint  `json:"constraints"`
	Indexes           []SchemaIndex       `json:"indexes"`
}

// SchemaConstraint represents a schema constraint
type SchemaConstraint struct {
	Type       string   `json:"type"`       // "UNIQUE", "NOT_NULL", etc.
	Label      string   `json:"label"`      // Node label or relationship type
	Properties []string `json:"properties"` // Property names
}

// SchemaIndex represents a schema index
type SchemaIndex struct {
	Type       string   `json:"type"`       // "BTREE", "FULLTEXT", "VECTOR", etc.
	Label      string   `json:"label"`      // Node label or relationship type
	Properties []string `json:"properties"` // Property names
}

// GraphQueryOptions represents options for graph queries
type GraphQueryOptions struct {
	Limit      int                    `json:"limit,omitempty"`
	Skip       int                    `json:"skip,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"` // Query parameters
	Timeout    int                    `json:"timeout,omitempty"`    // Query timeout in seconds
}

// GraphTraversalOptions represents options for graph traversal
type GraphTraversalOptions struct {
	MaxDepth    int                    `json:"max_depth,omitempty"`    // Maximum traversal depth
	MinDepth    int                    `json:"min_depth,omitempty"`    // Minimum traversal depth
	NodeFilter  map[string]interface{} `json:"node_filter,omitempty"`  // Node filtering criteria
	RelFilter   map[string]interface{} `json:"rel_filter,omitempty"`   // Relationship filtering criteria
	Direction   string                 `json:"direction,omitempty"`    // "INCOMING", "OUTGOING", "BOTH"
	ReturnPaths bool                   `json:"return_paths,omitempty"` // Whether to return full paths
	UniquePaths bool                   `json:"unique_paths,omitempty"` // Whether to ensure unique paths
	Limit       int                    `json:"limit,omitempty"`        // Maximum number of results
}

// CommunityDetectionOptions represents options for community detection
type CommunityDetectionOptions struct {
	Algorithm  string                 `json:"algorithm"`            // "leiden", "louvain", "label_propagation"
	MaxLevels  int                    `json:"max_levels,omitempty"` // Maximum hierarchy levels
	Resolution float64                `json:"resolution,omitempty"` // Resolution parameter
	Randomness float64                `json:"randomness,omitempty"` // Randomness parameter
	Parameters map[string]interface{} `json:"parameters,omitempty"` // Algorithm-specific parameters
}

// Community represents a detected community
type Community struct {
	ID         string                 `json:"id"`
	Level      int                    `json:"level"`                // Hierarchy level (0 = leaf level)
	ParentID   string                 `json:"parent_id,omitempty"`  // Parent community ID
	Members    []string               `json:"members"`              // Node IDs in this community
	Size       int                    `json:"size"`                 // Number of members
	Title      string                 `json:"title,omitempty"`      // Community title/summary
	Summary    string                 `json:"summary,omitempty"`    // Community description
	Properties map[string]interface{} `json:"properties,omitempty"` // Additional properties
}

// GraphAnalyticsOptions represents options for graph analytics
type GraphAnalyticsOptions struct {
	Algorithm     string                 `json:"algorithm"` // "pagerank", "betweenness", "closeness", etc.
	Iterations    int                    `json:"iterations,omitempty"`
	DampingFactor float64                `json:"damping_factor,omitempty"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
}

// NodeMetrics represents analytical metrics for a node
type NodeMetrics struct {
	NodeID                string  `json:"node_id"`
	PageRank              float64 `json:"pagerank,omitempty"`
	BetweennessCentrality float64 `json:"betweenness_centrality,omitempty"`
	ClosenessCentrality   float64 `json:"closeness_centrality,omitempty"`
	DegreeCentrality      float64 `json:"degree_centrality,omitempty"`
	ClusteringCoefficient float64 `json:"clustering_coefficient,omitempty"`
}

// GraphOperation represents a batch operation
type GraphOperation struct {
	Type       string                 `json:"type"` // "CREATE_NODE", "CREATE_REL", "UPDATE_NODE", etc.
	Data       interface{}            `json:"data"` // Operation data (Node, Relationship, etc.)
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ===== GraphRAG Configuration =====

// Config represents configuration for GraphRAG
type Config struct {
	GraphStore        GraphStore        `json:"graph_store"`
	VectorStore       VectorStore       `json:"vector_store"`
	EmbeddingFunction EmbeddingFunction `json:"embedding_function"`

	// Community detection settings
	CommunityDetection CommunityDetectionOptions `json:"community_detection"`

	// Retrieval settings
	MaxSearchDepth int     `json:"max_search_depth"`
	VectorTopK     int     `json:"vector_top_k"`
	HybridWeight   float64 `json:"hybrid_weight"` // Balance between vector and graph search

	// LLM settings for text-to-query
	LLMModel       string `json:"llm_model"`
	PromptTemplate string `json:"prompt_template"`
}

// VoteOption is a struct that contains the options for the vote.
type VoteOption struct {
	Model  string
	Reason string
	Vote   string
}

// RestoreOptions represents options for restoring data
type RestoreOptions struct {
	BackupPath     string                 `json:"backup_path"`
	CollectionName string                 `json:"collection_name"`
	Force          bool                   `json:"force"`
	ExtraParams    map[string]interface{} `json:"extra_params,omitempty"`
}

// ===== Document Operation Options =====

// GetDocumentOptions represents options for retrieving documents
type GetDocumentOptions struct {
	CollectionName string   `json:"collection_name"`
	Fields         []string `json:"fields,omitempty"` // Specific fields to retrieve
	IncludeVector  bool     `json:"include_vector"`   // Whether to include vector data
	IncludePayload bool     `json:"include_payload"`  // Whether to include payload/metadata
}

// DeleteDocumentOptions represents options for deleting documents
type DeleteDocumentOptions struct {
	CollectionName string                 `json:"collection_name"`
	IDs            []string               `json:"ids,omitempty"`    // Specific document IDs to delete
	Filter         map[string]interface{} `json:"filter,omitempty"` // Filter conditions for bulk delete
	DryRun         bool                   `json:"dry_run"`          // Preview what would be deleted
}

// ===== Collection Load States =====

// LoadState represents the loading state of a collection
type LoadState int

const (
	// LoadStateNotExist indicates that the collection does not exist
	LoadStateNotExist LoadState = iota

	// LoadStateNotLoad indicates that the collection exists but is not loaded
	LoadStateNotLoad

	// LoadStateLoading indicates that the collection is currently being loaded
	LoadStateLoading

	// LoadStateLoaded indicates that the collection is fully loaded and ready
	LoadStateLoaded
)

// String returns the string representation of LoadState
func (ls LoadState) String() string {
	switch ls {
	case LoadStateNotExist:
		return "NotExist"
	case LoadStateNotLoad:
		return "NotLoad"
	case LoadStateLoading:
		return "Loading"
	case LoadStateLoaded:
		return "Loaded"
	default:
		return "Unknown"
	}
}

// ===== Backup and Restore Options =====

// BackupOptions represents options for creating backups
type BackupOptions struct {
	CollectionName string                 `json:"collection_name"`
	BackupPath     string                 `json:"backup_path"`
	Compress       bool                   `json:"compress"`
	ExtraParams    map[string]interface{} `json:"extra_params,omitempty"`
}

// ===== Pagination and Listing Options =====

// ListDocumentsOptions represents options for listing documents with pagination
type ListDocumentsOptions struct {
	CollectionName string                 `json:"collection_name"`
	Filter         map[string]interface{} `json:"filter,omitempty"`   // Metadata filter
	Limit          int                    `json:"limit,omitempty"`    // Number of documents per page (default: 100)
	Offset         int                    `json:"offset,omitempty"`   // Offset for pagination
	OrderBy        []string               `json:"order_by,omitempty"` // Fields to order by
	IncludeVector  bool                   `json:"include_vector"`     // Whether to include vector data
	IncludePayload bool                   `json:"include_payload"`    // Whether to include payload/metadata
	Fields         []string               `json:"fields,omitempty"`   // Specific fields to retrieve
}

// PaginatedDocumentsResult represents paginated query results
type PaginatedDocumentsResult struct {
	Documents  []*Document `json:"documents"`
	Total      int64       `json:"total"`       // Total number of matching documents (if supported)
	HasMore    bool        `json:"has_more"`    // Whether there are more pages
	NextOffset int         `json:"next_offset"` // Offset for next page
}

// ScrollOptions represents options for scrolling through documents (iterator-style)
type ScrollOptions struct {
	CollectionName string                 `json:"collection_name"`
	Filter         map[string]interface{} `json:"filter,omitempty"`     // Metadata filter
	BatchSize      int                    `json:"batch_size,omitempty"` // Number of documents per batch (default: 100)
	ScrollID       string                 `json:"scroll_id,omitempty"`  // Scroll ID for continuing pagination
	IncludeVector  bool                   `json:"include_vector"`       // Whether to include vector data
	IncludePayload bool                   `json:"include_payload"`      // Whether to include payload/metadata
	Fields         []string               `json:"fields,omitempty"`     // Specific fields to retrieve
}

// ScrollResult represents scroll-based query results
type ScrollResult struct {
	Documents []*Document `json:"documents"`
	ScrollID  string      `json:"scroll_id,omitempty"` // ID for next scroll request
	HasMore   bool        `json:"has_more"`            // Whether there are more results
}

// ===== Paginated Search Types (for Search Engine scenarios) =====

// PaginatedSearchOptions represents options for paginated similarity search
type PaginatedSearchOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVector    []float64              `json:"query_vector"`     // Query vector for similarity search
	Page           int                    `json:"page"`             // Page number (1-based)
	PageSize       int                    `json:"page_size"`        // Number of results per page (default: 10)
	Filter         map[string]interface{} `json:"filter,omitempty"` // Metadata filter

	// Search-specific parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter (HNSW)
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes (IVF)
	Rescore     bool `json:"rescore,omitempty"`     // Whether to rescore results
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search
	Timeout     int  `json:"timeout,omitempty"`     // Search timeout in milliseconds

	// Search engine specific options
	MinScore        float64  `json:"min_score,omitempty"`        // Minimum similarity score to include
	MaxResults      int      `json:"max_results,omitempty"`      // Maximum total results to consider (default: 1000)
	SortBy          []string `json:"sort_by,omitempty"`          // Secondary sorting criteria
	IncludeTotal    bool     `json:"include_total"`              // Whether to calculate total count (expensive)
	FacetFields     []string `json:"facet_fields,omitempty"`     // Fields for faceted search
	HighlightFields []string `json:"highlight_fields,omitempty"` // Fields to highlight in results
}

// PaginatedMMRSearchOptions represents options for paginated MMR search
type PaginatedMMRSearchOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVector    []float64              `json:"query_vector"`          // Query vector for similarity search
	Page           int                    `json:"page"`                  // Page number (1-based)
	PageSize       int                    `json:"page_size"`             // Number of results per page (default: 10)
	FetchK         int                    `json:"fetch_k,omitempty"`     // Number of documents to fetch for MMR algorithm
	LambdaMult     float64                `json:"lambda_mult,omitempty"` // Diversity parameter (0-1)
	Filter         map[string]interface{} `json:"filter,omitempty"`      // Metadata filter

	// Search parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search

	// Search engine specific options
	MinScore     float64  `json:"min_score,omitempty"`    // Minimum similarity score to include
	MaxResults   int      `json:"max_results,omitempty"`  // Maximum total results to consider
	IncludeTotal bool     `json:"include_total"`          // Whether to calculate total count
	FacetFields  []string `json:"facet_fields,omitempty"` // Fields for faceted search
}

// PaginatedScoreThresholdSearchOptions represents options for paginated score threshold search
type PaginatedScoreThresholdSearchOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVector    []float64              `json:"query_vector"`     // Query vector for similarity search
	ScoreThreshold float64                `json:"score_threshold"`  // Minimum relevance score threshold
	Page           int                    `json:"page"`             // Page number (1-based)
	PageSize       int                    `json:"page_size"`        // Number of results per page (default: 10)
	Filter         map[string]interface{} `json:"filter,omitempty"` // Metadata filter

	// Search parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search

	// Search engine specific options
	MaxResults   int      `json:"max_results,omitempty"`  // Maximum total results to consider
	SortBy       []string `json:"sort_by,omitempty"`      // Secondary sorting criteria
	IncludeTotal bool     `json:"include_total"`          // Whether to calculate total count
	FacetFields  []string `json:"facet_fields,omitempty"` // Fields for faceted search
}

// SearchFacet represents a facet for faceted search
type SearchFacet struct {
	Field  string           `json:"field"`  // Metadata field name
	Values map[string]int64 `json:"values"` // Value -> count mapping
}

// PaginatedSearchResult represents paginated search results
type PaginatedSearchResult struct {
	Documents    []*SearchResult `json:"documents"`               // Search results for current page
	Page         int             `json:"page"`                    // Current page number
	PageSize     int             `json:"page_size"`               // Number of results per page
	Total        int64           `json:"total,omitempty"`         // Total number of matching documents (if IncludeTotal=true)
	TotalPages   int             `json:"total_pages,omitempty"`   // Total number of pages (if IncludeTotal=true)
	HasNext      bool            `json:"has_next"`                // Whether there are more pages
	HasPrevious  bool            `json:"has_previous"`            // Whether there are previous pages
	NextPage     int             `json:"next_page,omitempty"`     // Next page number (if HasNext=true)
	PreviousPage int             `json:"previous_page,omitempty"` // Previous page number (if HasPrevious=true)

	// Search engine features
	QueryTime   int64                   `json:"query_time_ms"`         // Query execution time in milliseconds
	Facets      map[string]*SearchFacet `json:"facets,omitempty"`      // Faceted search results
	Suggestions []string                `json:"suggestions,omitempty"` // Query suggestions for typos/alternatives
	MaxScore    float64                 `json:"max_score,omitempty"`   // Highest score in results
	MinScore    float64                 `json:"min_score,omitempty"`   // Lowest score in results
}

// HybridSearchOptions represents options for hybrid (vector + keyword) search with pagination
type HybridSearchOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVector    []float64              `json:"query_vector"`     // Vector query
	QueryText      string                 `json:"query_text"`       // Text query for keyword search
	Page           int                    `json:"page"`             // Page number (1-based)
	PageSize       int                    `json:"page_size"`        // Number of results per page
	Filter         map[string]interface{} `json:"filter,omitempty"` // Metadata filter

	// Hybrid search weights
	VectorWeight  float64 `json:"vector_weight"`  // Weight for vector similarity (0-1)
	KeywordWeight float64 `json:"keyword_weight"` // Weight for keyword relevance (0-1)

	// Search parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search

	// Keyword search parameters
	KeywordFields []string           `json:"keyword_fields,omitempty"` // Fields to search for keywords
	FuzzyMatch    bool               `json:"fuzzy_match,omitempty"`    // Enable fuzzy keyword matching
	BoostFields   map[string]float64 `json:"boost_fields,omitempty"`   // Field -> boost factor mapping

	// Search engine specific options
	MinScore        float64  `json:"min_score,omitempty"`        // Minimum combined score
	MaxResults      int      `json:"max_results,omitempty"`      // Maximum total results to consider
	IncludeTotal    bool     `json:"include_total"`              // Whether to calculate total count
	FacetFields     []string `json:"facet_fields,omitempty"`     // Fields for faceted search
	HighlightFields []string `json:"highlight_fields,omitempty"` // Fields to highlight in results
}

// SearchEngineStats represents search engine performance statistics
type SearchEngineStats struct {
	TotalQueries     int64    `json:"total_queries"`     // Total number of queries processed
	AverageQueryTime float64  `json:"avg_query_time_ms"` // Average query time in milliseconds
	CacheHitRate     float64  `json:"cache_hit_rate"`    // Cache hit rate (0-1)
	PopularQueries   []string `json:"popular_queries"`   // Most popular queries
	SlowQueries      []string `json:"slow_queries"`      // Slowest queries
	ErrorRate        float64  `json:"error_rate"`        // Error rate (0-1)
	IndexSize        int64    `json:"index_size_bytes"`  // Total index size in bytes
	DocumentCount    int64    `json:"document_count"`    // Total number of documents
}
