package types

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
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

// == Vote Enums ==

// VoteType represents the type of vote
type VoteType string

const (

	// VotePositive is the type of vote that is positive
	VotePositive VoteType = "positive"

	// VoteNegative is the type of vote that is negative
	VoteNegative VoteType = "negative"
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

// Position represents a position in the text
type Position struct {
	StartPos int `json:"s"`
	EndPos   int `json:"e"`
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

	// Extracted text
	Extracted *ExtractionResult `json:"extracted,omitempty"` // Extracted text from the chunk

	// Metadata of the chunk
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Metadata of the chunk
}

// Embeddings is a slice of EmbeddingResult
type Embeddings []*EmbeddingResult

// ChunkingOptions represents options for chunking
type ChunkingOptions struct {
	Type            ChunkingType     `json:"type,omitempty"`   // Content type, auto-detected if not provided
	Size            int              `json:"size"`             // For text, PDF, Word, only, default is QA 300, Code 800,
	Overlap         int              `json:"overlap"`          // For text, PDF, Word, only, default is QA 20, Code 100,
	MaxDepth        int              `json:"max_depth"`        // For text, PDF, Word, only, default is 5
	SizeMultiplier  int              `json:"size_multiplier"`  // Base multiplier for chunk size calculation, default is 3
	MaxConcurrent   int              `json:"max_concurrent"`   // Maximum concurrent operations
	Separator       string           `json:"separator"`        // Custom separator pattern (regex supported)
	EnableDebug     bool             `json:"enable_debug"`     // Enable debug mode for detailed splitting information
	SemanticOptions *SemanticOptions `json:"semantic_options"` // For Semantic recognition, etc.
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

// ===== Pagination Types =====

// PaginationStrategy represents different pagination strategies supported by vector databases
type PaginationStrategy string

const (
	// PaginationStrategyOffset represents traditional offset/limit pagination (page/pagesize)
	// Pros: Simple, supports random access to pages, familiar API
	// Cons: Performance degrades with large offsets, inconsistent results during data changes
	// Best for: Small to medium datasets, UI pagination with page numbers
	PaginationStrategyOffset PaginationStrategy = "offset"

	// PaginationStrategyCursor represents cursor-based pagination using tokens
	// Pros: Consistent performance, stable results during data changes
	// Cons: No random access, more complex implementation
	// Best for: Large datasets, real-time feeds, high-performance requirements
	PaginationStrategyCursor PaginationStrategy = "cursor"

	// PaginationStrategyClientSlice represents client-side slicing after fetching larger result sets
	// Pros: Simple implementation, works with any vector database
	// Cons: Higher memory usage, network overhead, limited to smaller datasets
	// Best for: Simple use cases, databases without native pagination support (like Qdrant)
	PaginationStrategyClientSlice PaginationStrategy = "client_slice"

	// PaginationStrategyScroll represents scroll-based pagination (like Elasticsearch scroll API)
	// Pros: Efficient for large datasets, maintains search context
	// Cons: Stateful, requires cleanup, limited concurrent access
	// Best for: Bulk data processing, export operations
	PaginationStrategyScroll PaginationStrategy = "scroll"
)

// Pagination represents unified pagination options supporting both offset and cursor-based pagination
type Pagination struct {
	// Offset-based pagination (traditional page/pagesize)
	Page     int `json:"page,omitempty"`      // Page number (1-based), 0 means no pagination
	PageSize int `json:"page_size,omitempty"` // Number of results per page (default: 10)

	// Cursor-based pagination (for better performance with large datasets)
	Cursor string `json:"cursor,omitempty"` // Cursor token for cursor-based pagination

	// Control options
	IncludeTotal bool `json:"include_total,omitempty"` // Whether to calculate total count (expensive for large datasets)
}

// PaginationResult represents unified pagination response metadata
type PaginationResult struct {
	// Offset-based pagination info
	Page         int   `json:"page,omitempty"`          // Current page number (1-based)
	PageSize     int   `json:"page_size,omitempty"`     // Number of results per page
	Total        int64 `json:"total,omitempty"`         // Total number of matching documents (if IncludeTotal=true)
	TotalPages   int   `json:"total_pages,omitempty"`   // Total number of pages (if IncludeTotal=true)
	HasNext      bool  `json:"has_next,omitempty"`      // Whether there are more pages
	HasPrevious  bool  `json:"has_previous,omitempty"`  // Whether there are previous pages
	NextPage     int   `json:"next_page,omitempty"`     // Next page number (if HasNext=true)
	PreviousPage int   `json:"previous_page,omitempty"` // Previous page number (if HasPrevious=true)

	// Cursor-based pagination info
	Cursor     string `json:"cursor,omitempty"`      // Current cursor position
	NextCursor string `json:"next_cursor,omitempty"` // Cursor for next page (if HasNext=true)
	PrevCursor string `json:"prev_cursor,omitempty"` // Cursor for previous page (if HasPrevious=true)
}

// IsOffsetBased returns true if using offset-based pagination (page/pagesize)
func (p *Pagination) IsOffsetBased() bool {
	return p.Page > 0 && p.PageSize > 0 && p.Cursor == ""
}

// IsCursorBased returns true if using cursor-based pagination
func (p *Pagination) IsCursorBased() bool {
	return p.Cursor != ""
}

// GetOffset calculates the offset for offset-based pagination
func (p *Pagination) GetOffset() int {
	if !p.IsOffsetBased() {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}

// GetLimit returns the limit/pagesize for pagination
func (p *Pagination) GetLimit() int {
	if p.PageSize <= 0 {
		return 10 // default page size
	}
	return p.PageSize
}

// GetStrategy returns the pagination strategy being used
func (p *Pagination) GetStrategy() PaginationStrategy {
	if p.IsCursorBased() {
		return PaginationStrategyCursor
	}
	if p.IsOffsetBased() {
		return PaginationStrategyOffset
	}
	return PaginationStrategyClientSlice // default for non-paginated requests
}

// ===== Vector Store Types =====

// Document represents a document with content and metadata
type Document struct {
	ID           string                 `json:"id,omitempty"`
	Content      string                 `json:"content"`
	Vector       []float64              `json:"vector,omitempty"`        // Dense vector (legacy field for backward compatibility)
	DenseVector  []float64              `json:"dense_vector,omitempty"`  // Dense embedding vector
	SparseVector *SparseVector          `json:"sparse_vector,omitempty"` // Sparse embedding vector
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// HasDenseVector returns true if the document has a dense vector (either Vector or DenseVector fields)
func (d *Document) HasDenseVector() bool {
	return len(d.Vector) > 0 || len(d.DenseVector) > 0
}

// HasSparseVector returns true if the document has a sparse vector
func (d *Document) HasSparseVector() bool {
	return d.SparseVector != nil && len(d.SparseVector.Indices) > 0
}

// GetDenseVector returns the dense vector, prioritizing DenseVector field over legacy Vector field
func (d *Document) GetDenseVector() []float64 {
	if len(d.DenseVector) > 0 {
		return d.DenseVector
	}
	return d.Vector
}

// GetSparseVector returns the sparse vector
func (d *Document) GetSparseVector() *SparseVector {
	return d.SparseVector
}

// SearchResultItem represents a single search result with document and score
type SearchResultItem struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
}

// SearchResult represents unified search results (both paginated and non-paginated)
type SearchResult struct {
	// Results
	Documents []*SearchResultItem `json:"documents"` // Search results

	// Pagination metadata (embedded for backward compatibility)
	PaginationResult `json:",inline"`

	// Search engine features
	QueryTime   int64                   `json:"query_time_ms"`         // Query execution time in milliseconds
	Facets      map[string]*SearchFacet `json:"facets,omitempty"`      // Faceted search results
	Suggestions []string                `json:"suggestions,omitempty"` // Query suggestions for typos/alternatives
	MaxScore    float64                 `json:"max_score,omitempty"`   // Highest score in results
	MinScore    float64                 `json:"min_score,omitempty"`   // Lowest score in results
}

// VectorStoreConfig represents configuration for vector store connection
type VectorStoreConfig struct {
	// Database Connection Configuration
	DatabaseURL    string                 `json:"database_url,omitempty"`    // Database connection URL
	ConnectionPool int                    `json:"connection_pool,omitempty"` // Connection pool size
	Timeout        int                    `json:"timeout,omitempty"`         // Operation timeout in seconds
	ExtraParams    map[string]interface{} `json:"extra_params,omitempty"`    // Database-specific parameters (host, port, api_key, etc.)
}

// Validate validates the vector store connection configuration
func (c *VectorStoreConfig) Validate() error {
	// Basic validation for connection config
	// Most validation will be database-specific and handled by individual drivers
	return nil
}

// CreateCollectionOptions represents configuration for creating a collection
type CreateCollectionOptions struct {
	// Collection Basic Configuration
	CollectionName string         `json:"collection_name" yaml:"collection_name"` // Collection/Table name
	Dimension      int            `json:"dimension" yaml:"dimension"`             // Vector dimension (e.g., 1536 for OpenAI embeddings)
	Distance       DistanceMetric `json:"distance" yaml:"distance"`               // Distance metric
	IndexType      IndexType      `json:"index_type" yaml:"index_type"`           // Index type

	// Index Parameters (for HNSW)
	M              int `json:"m,omitempty" yaml:"m,omitempty"`                             // Number of bidirectional links for each node (HNSW)
	EfConstruction int `json:"ef_construction,omitempty" yaml:"ef_construction,omitempty"` // Size of dynamic candidate list (HNSW)
	EfSearch       int `json:"ef_search,omitempty" yaml:"ef_search,omitempty"`             // Size of dynamic candidate list for search (HNSW)

	// Index Parameters (for IVF)
	NumLists  int `json:"num_lists,omitempty" yaml:"num_lists,omitempty"`   // Number of clusters (IVF)
	NumProbes int `json:"num_probes,omitempty" yaml:"num_probes,omitempty"` // Number of clusters to search (IVF)

	// Sparse Vector Configuration (for hybrid search)
	EnableSparseVectors bool   `json:"enable_sparse_vectors,omitempty" yaml:"enable_sparse_vectors,omitempty"` // Enable sparse vector support for hybrid retrieval
	DenseVectorName     string `json:"dense_vector_name,omitempty" yaml:"dense_vector_name,omitempty"`         // Named vector for dense vectors (default: "dense")
	SparseVectorName    string `json:"sparse_vector_name,omitempty" yaml:"sparse_vector_name,omitempty"`       // Named vector for sparse vectors (default: "sparse")

	// Storage Configuration
	PersistPath string `json:"persist_path,omitempty" yaml:"persist_path,omitempty"` // Path for persistent storage

	// Collection-specific settings
	ExtraParams map[string]interface{} `json:"extra_params,omitempty" yaml:"extra_params,omitempty"` // Collection-specific parameters
}

// Validate validates the collection creation configuration
func (c *CreateCollectionOptions) Validate() error {
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

	// Validate sparse vector configuration
	if c.EnableSparseVectors {
		// Check for naming conflicts
		denseVectorName := c.DenseVectorName
		if denseVectorName == "" {
			denseVectorName = "dense"
		}

		sparseVectorName := c.SparseVectorName
		if sparseVectorName == "" {
			sparseVectorName = "sparse"
		}

		if denseVectorName == sparseVectorName {
			return fmt.Errorf("dense and sparse vector names cannot be the same: %s", denseVectorName)
		}
	}

	return nil
}

// VectorMode represents the vector operation mode for document operations
type VectorMode string

const (
	// VectorModeAuto automatically determines which vectors to use based on what's available in the document
	VectorModeAuto VectorMode = "auto"
	// VectorModeDenseOnly only processes dense vectors, ignores sparse vectors
	VectorModeDenseOnly VectorMode = "dense_only"
	// VectorModeSparseOnly only processes sparse vectors, ignores dense vectors
	VectorModeSparseOnly VectorMode = "sparse_only"
	// VectorModeBoth processes both dense and sparse vectors (requires both to be present)
	VectorModeBoth VectorMode = "both"
)

// String returns the string representation of the vector mode
func (vm VectorMode) String() string {
	return string(vm)
}

// IsValid checks if the vector mode is valid
func (vm VectorMode) IsValid() bool {
	switch vm {
	case VectorModeAuto, VectorModeDenseOnly, VectorModeSparseOnly, VectorModeBoth:
		return true
	default:
		return false
	}
}

// GetSupportedVectorModes returns all supported vector modes
func GetSupportedVectorModes() []VectorMode {
	return []VectorMode{
		VectorModeAuto,
		VectorModeDenseOnly,
		VectorModeSparseOnly,
		VectorModeBoth,
	}
}

// AddDocumentOptions represents options for adding documents
type AddDocumentOptions struct {
	CollectionName string      `json:"collection_name"`
	Documents      []*Document `json:"documents"`            // Documents to add (ID in Document will be used if provided, otherwise auto-generated)
	BatchSize      int         `json:"batch_size,omitempty"` // Batch size for bulk insert
	Timeout        int         `json:"timeout,omitempty"`    // Operation timeout in seconds
	Upsert         bool        `json:"upsert,omitempty"`     // If true, update existing documents with same ID

	// Vector operation mode - determines which vectors to insert/update
	VectorMode VectorMode `json:"vector_mode,omitempty"` // "dense_only", "sparse_only", "both", "auto" (default: "auto")

	// Named vector support (for collections with multiple vectors)
	VectorUsing      string `json:"vector_using,omitempty"`       // Named vector to use for document vectors (e.g., "dense", "sparse") - legacy field
	DenseVectorName  string `json:"dense_vector_name,omitempty"`  // Named vector for dense vectors (default: "dense")
	SparseVectorName string `json:"sparse_vector_name,omitempty"` // Named vector for sparse vectors (default: "sparse")
}

// SearchOptions represents options for similarity search
type SearchOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVector    []float64              `json:"query_vector"`     // Query vector for similarity search
	K              int                    `json:"k,omitempty"`      // Number of documents to return (ignored if using pagination)
	Filter         map[string]interface{} `json:"filter,omitempty"` // Metadata filter

	// Pagination (optional - if not specified, returns top K results)
	Page     int    `json:"page,omitempty"`      // Page number (1-based), 0 means no pagination
	PageSize int    `json:"page_size,omitempty"` // Number of results per page (default: 10)
	Cursor   string `json:"cursor,omitempty"`    // Cursor for cursor-based pagination (alternative to page/pagesize)

	// Return control options
	IncludeVector   bool     `json:"include_vector"`   // Whether to include vector data in results
	IncludeMetadata bool     `json:"include_metadata"` // Whether to include document metadata
	IncludeContent  bool     `json:"include_content"`  // Whether to include document content
	Fields          []string `json:"fields,omitempty"` // Specific fields to retrieve
	IncludeTotal    bool     `json:"include_total"`    // Whether to calculate total count (expensive for pagination)

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
	FacetFields     []string `json:"facet_fields,omitempty"`     // Fields for faceted search
	HighlightFields []string `json:"highlight_fields,omitempty"` // Fields to highlight in results

	// Named vector support (for collections with multiple vectors)
	VectorUsing string `json:"vector_using,omitempty"` // Named vector to use for search (e.g., "dense", "sparse")
}

// MMRSearchOptions represents options for maximal marginal relevance search
type MMRSearchOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVector    []float64              `json:"query_vector"`          // Query vector for similarity search
	K              int                    `json:"k,omitempty"`           // Number of documents to return (ignored if using pagination)
	FetchK         int                    `json:"fetch_k,omitempty"`     // Number of documents to fetch for MMR algorithm
	LambdaMult     float64                `json:"lambda_mult,omitempty"` // Diversity parameter (0-1, 0=max diversity, 1=max similarity)
	Filter         map[string]interface{} `json:"filter,omitempty"`      // Metadata filter

	// Pagination (optional - if not specified, returns top K results)
	Page     int    `json:"page,omitempty"`      // Page number (1-based), 0 means no pagination
	PageSize int    `json:"page_size,omitempty"` // Number of results per page (default: 10)
	Cursor   string `json:"cursor,omitempty"`    // Cursor for cursor-based pagination (alternative to page/pagesize)

	// Return control options
	IncludeVector   bool     `json:"include_vector"`   // Whether to include vector data in results
	IncludeMetadata bool     `json:"include_metadata"` // Whether to include document metadata
	IncludeContent  bool     `json:"include_content"`  // Whether to include document content
	Fields          []string `json:"fields,omitempty"` // Specific fields to retrieve
	IncludeTotal    bool     `json:"include_total"`    // Whether to calculate total count (expensive for pagination)

	// Search parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search
	Timeout     int  `json:"timeout,omitempty"`     // Search timeout in milliseconds

	// Search engine specific options
	MinScore    float64  `json:"min_score,omitempty"`    // Minimum similarity score to include
	MaxResults  int      `json:"max_results,omitempty"`  // Maximum total results to consider
	FacetFields []string `json:"facet_fields,omitempty"` // Fields for faceted search

	// Named vector support (for collections with multiple vectors)
	VectorUsing string `json:"vector_using,omitempty"` // Named vector to use for search (e.g., "dense", "sparse")
}

// ScoreThresholdOptions represents options for similarity search with score threshold
type ScoreThresholdOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVector    []float64              `json:"query_vector"`     // Query vector for similarity search
	ScoreThreshold float64                `json:"score_threshold"`  // Minimum relevance score threshold
	K              int                    `json:"k,omitempty"`      // Number of documents to return (ignored if using pagination)
	Filter         map[string]interface{} `json:"filter,omitempty"` // Metadata filter

	// Pagination (optional - if not specified, returns top K results)
	Page     int    `json:"page,omitempty"`      // Page number (1-based), 0 means no pagination
	PageSize int    `json:"page_size,omitempty"` // Number of results per page (default: 10)
	Cursor   string `json:"cursor,omitempty"`    // Cursor for cursor-based pagination (alternative to page/pagesize)

	// Return control options
	IncludeVector   bool     `json:"include_vector"`   // Whether to include vector data in results
	IncludeMetadata bool     `json:"include_metadata"` // Whether to include document metadata
	IncludeContent  bool     `json:"include_content"`  // Whether to include document content
	Fields          []string `json:"fields,omitempty"` // Specific fields to retrieve
	IncludeTotal    bool     `json:"include_total"`    // Whether to calculate total count (expensive for pagination)

	// Search parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search
	Timeout     int  `json:"timeout,omitempty"`     // Search timeout in milliseconds

	// Search engine specific options
	MaxResults      int      `json:"max_results,omitempty"`      // Maximum total results to consider
	SortBy          []string `json:"sort_by,omitempty"`          // Secondary sorting criteria
	FacetFields     []string `json:"facet_fields,omitempty"`     // Fields for faceted search
	HighlightFields []string `json:"highlight_fields,omitempty"` // Fields to highlight in results

	// Named vector support (for collections with multiple vectors)
	VectorUsing string `json:"vector_using,omitempty"` // Named vector to use for search (e.g., "dense", "sparse")
}

// HybridSearchOptions represents options for hybrid (vector + keyword) search using Qdrant's native Query API
type HybridSearchOptions struct {
	CollectionName string                 `json:"collection_name"`
	QueryVector    []float64              `json:"query_vector,omitempty"` // Dense vector query (optional if only using sparse vector search)
	QuerySparse    *SparseVector          `json:"query_sparse,omitempty"` // Sparse vector query (e.g., from BM25, TF-IDF)
	K              int                    `json:"k,omitempty"`            // Number of documents to return (ignored if using pagination)
	Filter         map[string]interface{} `json:"filter,omitempty"`       // Metadata filter

	// Pagination (optional - if not specified, returns top K results)
	Page     int    `json:"page,omitempty"`      // Page number (1-based), 0 means no pagination
	PageSize int    `json:"page_size,omitempty"` // Number of results per page (default: 10)
	Cursor   string `json:"cursor,omitempty"`    // Cursor for cursor-based pagination (alternative to page/pagesize)

	// Return control options
	IncludeVector   bool     `json:"include_vector"`   // Whether to include vector data in results
	IncludeMetadata bool     `json:"include_metadata"` // Whether to include document metadata
	IncludeContent  bool     `json:"include_content"`  // Whether to include document content
	Fields          []string `json:"fields,omitempty"` // Specific fields to retrieve
	IncludeTotal    bool     `json:"include_total"`    // Whether to calculate total count (expensive for pagination)

	// Hybrid search fusion configuration
	FusionType  FusionType `json:"fusion_type,omitempty"`  // Fusion algorithm: "rrf" (Reciprocal Rank Fusion) or "dbsf" (Distribution-Based Score Fusion)
	VectorUsing string     `json:"vector_using,omitempty"` // Named vector for dense vectors (e.g., "dense")
	SparseUsing string     `json:"sparse_using,omitempty"` // Named vector for sparse vectors (e.g., "sparse")

	// Legacy weight support (will be converted to appropriate fusion)
	VectorWeight  float64 `json:"vector_weight,omitempty"`  // Weight for vector similarity (0-1) - for backward compatibility
	KeywordWeight float64 `json:"keyword_weight,omitempty"` // Weight for keyword relevance (0-1) - for backward compatibility

	// Vector search parameters
	EfSearch    int  `json:"ef_search,omitempty"`   // Dynamic search parameter (HNSW)
	NumProbes   int  `json:"num_probes,omitempty"`  // Number of probes (IVF)
	Rescore     bool `json:"rescore,omitempty"`     // Whether to rescore results
	Approximate bool `json:"approximate,omitempty"` // Whether to use approximate search
	Timeout     int  `json:"timeout,omitempty"`     // Search timeout in milliseconds

	// Search engine specific options
	MinScore        float64  `json:"min_score,omitempty"`        // Minimum combined score
	MaxResults      int      `json:"max_results,omitempty"`      // Maximum total results to consider (default: 1000)
	SortBy          []string `json:"sort_by,omitempty"`          // Secondary sorting criteria
	FacetFields     []string `json:"facet_fields,omitempty"`     // Fields for faceted search
	HighlightFields []string `json:"highlight_fields,omitempty"` // Fields to highlight in results
}

// SparseVector represents a sparse vector with indices and values
type SparseVector struct {
	Indices []uint32  `json:"indices"` // Non-zero indices
	Values  []float32 `json:"values"`  // Non-zero values
}

// FusionType represents the type of fusion algorithm to use
type FusionType string

const (
	// FusionRRF represents Reciprocal Rank Fusion
	FusionRRF FusionType = "rrf"
	// FusionDBSF represents Distribution-Based Score Fusion
	FusionDBSF FusionType = "dbsf"
)

// String returns the string representation of the fusion type
func (ft FusionType) String() string {
	return string(ft)
}

// IsValid checks if the fusion type is valid
func (ft FusionType) IsValid() bool {
	switch ft {
	case FusionRRF, FusionDBSF:
		return true
	default:
		return false
	}
}

// GetType returns the type of the search options
func (h *HybridSearchOptions) GetType() SearchType {
	return SearchTypeHybrid
}

// GetType returns the type of the search options
func (m *MMRSearchOptions) GetType() SearchType {
	return SearchTypeMMR
}

// GetType returns the type of the search options
func (s *ScoreThresholdOptions) GetType() SearchType {
	return SearchTypeScoreThreshold
}

// GetType returns the type of the search options
func (s *SearchOptions) GetType() SearchType {
	return SearchTypeSimilarity
}

// SearchType represents the type of search operation
type SearchType string

const (
	// SearchTypeSimilarity represents similarity-based vector search
	SearchTypeSimilarity SearchType = "similarity"
	// SearchTypeMMR represents maximal marginal relevance search for diversity
	SearchTypeMMR SearchType = "mmr"
	// SearchTypeScoreThreshold represents similarity search with minimum score filtering
	SearchTypeScoreThreshold SearchType = "score_threshold"
	// SearchTypeHybrid represents hybrid search combining vector and keyword search
	SearchTypeHybrid SearchType = "hybrid"
)

// String returns the string representation of the search type
func (st SearchType) String() string {
	return string(st)
}

// IsValid checks if the search type is valid
func (st SearchType) IsValid() bool {
	switch st {
	case SearchTypeSimilarity, SearchTypeMMR, SearchTypeScoreThreshold, SearchTypeHybrid:
		return true
	default:
		return false
	}
}

// GetSupportedSearchTypes returns all supported search types
func GetSupportedSearchTypes() []SearchType {
	return []SearchType{
		SearchTypeSimilarity,
		SearchTypeMMR,
		SearchTypeScoreThreshold,
		SearchTypeHybrid,
	}
}

// SearchOptionsInterface represents options for batch search
type SearchOptionsInterface interface {
	GetType() SearchType
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

// ProgressCallback defines the callback function for progress reporting with flexible payload

// EmbeddingStatus defines the status of embedding process
type EmbeddingStatus string

// Status constants for embedding process
const (
	EmbeddingStatusStarting   EmbeddingStatus = "starting"   // Starting the embedding process
	EmbeddingStatusProcessing EmbeddingStatus = "processing" // Processing embeddings
	EmbeddingStatusCompleted  EmbeddingStatus = "completed"  // Successfully completed
	EmbeddingStatusError      EmbeddingStatus = "error"      // Error occurred
)

// EmbeddingPayload contains context-specific data for different embedding scenarios
type EmbeddingPayload struct {
	// Common fields
	Current int    `json:"current"` // Current progress count
	Total   int    `json:"total"`   // Total items to process
	Message string `json:"message"` // Status message

	// Document embedding specific
	DocumentIndex *int    `json:"document_index,omitempty"` // Index of current document being processed
	DocumentText  *string `json:"document_text,omitempty"`  // Text being processed (truncated if too long)

	// Error specific
	Error error `json:"error,omitempty"` // Error details when Status is StatusError
}

// ExtractionStatus defines the status of extraction process
type ExtractionStatus string

// Status constants for extraction process
const (
	ExtractionStatusStarting   ExtractionStatus = "starting"   // Starting the extraction process
	ExtractionStatusProcessing ExtractionStatus = "processing" // Processing extraction
	ExtractionStatusCompleted  ExtractionStatus = "completed"  // Successfully completed
	ExtractionStatusError      ExtractionStatus = "error"      // Error occurred
)

// EntityStatus defines the lifecycle status of nodes and relationships
type EntityStatus string

// Status constants for entity lifecycle management (applies to both nodes and relationships)
const (
	EntityStatusActive     EntityStatus = "active"     // Active entity in use
	EntityStatusMerged     EntityStatus = "merged"     // Merged into another entity
	EntityStatusDeprecated EntityStatus = "deprecated" // No longer used but preserved
	EntityStatusDraft      EntityStatus = "draft"      // Draft/unconfirmed entity
	EntityStatusReviewed   EntityStatus = "reviewed"   // Human reviewed and confirmed
)

// ExtractionMethod defines the method used for extracting entities and relationships
type ExtractionMethod string

// Extraction method constants based on actual implementation directories
const (
	ExtractionMethodLLM     ExtractionMethod = "llm"     // LLM-based extraction (OpenAI, etc.)
	ExtractionMethodSpacy   ExtractionMethod = "spacy"   // spaCy NER-based extraction
	ExtractionMethodDeppke  ExtractionMethod = "deppke"  // Deppke extraction method
	ExtractionMethodManual  ExtractionMethod = "manual"  // Manual/user-defined extraction
	ExtractionMethodPattern ExtractionMethod = "pattern" // Pattern-based extraction
)

// String returns the string representation of ExtractionMethod
func (em ExtractionMethod) String() string {
	return string(em)
}

// IsValid validates if the extraction method is supported
func (em ExtractionMethod) IsValid() bool {
	switch em {
	case ExtractionMethodLLM, ExtractionMethodSpacy, ExtractionMethodDeppke,
		ExtractionMethodManual, ExtractionMethodPattern:
		return true
	default:
		return false
	}
}

// GetSupportedExtractionMethods returns all supported extraction methods
func GetSupportedExtractionMethods() []ExtractionMethod {
	return []ExtractionMethod{
		ExtractionMethodLLM,
		ExtractionMethodSpacy,
		ExtractionMethodDeppke,
		ExtractionMethodManual,
		ExtractionMethodPattern,
	}
}

// ExtractionPayload contains context-specific data for different extraction scenarios
type ExtractionPayload struct {
	// Common fields
	Current int    `json:"current"` // Current progress count
	Total   int    `json:"total"`   // Total items to process
	Message string `json:"message"` // Status message

	// Document embedding specific
	DocumentIndex *int    `json:"document_index,omitempty"` // Index of current document being processed
	DocumentText  *string `json:"document_text,omitempty"`  // Text being processed (truncated if too long)

	// Error specific
	Error error `json:"error,omitempty"` // Error details when Status is StatusError
}

// ExtractionOptions represents options for extraction
type ExtractionOptions struct {
	Use       Extraction `json:"use"`       // Use the extraction method
	Embedding Embedding  `json:"embedding"` // Embedding function to use for extraction
}

// LLMOptimizer represents the LLM optimizer for extraction
type LLMOptimizer struct {
	Connector string `json:"connector"` // Connector to use for extraction
}

// ExtractionResult represents the result of an extraction process
type ExtractionResult struct {
	Usage         ExtractionUsage `json:"usage"`                   // Combined usage statistics
	Model         string          `json:"model"`                   // Model used for extraction
	Nodes         []Node          `json:"nodes,omitempty"`         // Extracted entities
	Relationships []Relationship  `json:"relationships,omitempty"` // Extracted relationships
}

// ExtractionUsage represents usage statistics for extraction operations
type ExtractionUsage struct {
	TotalTokens  int `json:"total_tokens"`  // Total number of tokens processed
	PromptTokens int `json:"prompt_tokens"` // Number of tokens in the input
	TotalTexts   int `json:"total_texts"`   // Total number of texts processed
}

// ===== Graph Database Types =====

// Node represents a graph node extracted from text (corresponds to Neo4j Node)
type Node struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`                 // Entity name/title
	Type       string                 `json:"type"`                 // Entity type (Person, Organization, Location, etc.)
	Labels     []string               `json:"labels,omitempty"`     // Additional labels/categories
	Properties map[string]interface{} `json:"properties,omitempty"` // Entity properties

	// GraphRAG specific fields
	Description     string    `json:"description,omitempty"`      // Entity description/summary
	Confidence      float64   `json:"confidence,omitempty"`       // Extraction confidence score (0-1)
	EmbeddingVector []float64 `json:"embedding_vector,omitempty"` // Entity embedding for semantic search

	// Source tracking
	SourceDocuments  []string         `json:"source_documents,omitempty"`  // Document IDs where entity was found
	SourceChunks     []string         `json:"source_chunks,omitempty"`     // Chunk IDs where entity was extracted
	ExtractionMethod ExtractionMethod `json:"extraction_method,omitempty"` // Extraction method used

	// Neo4j internal fields (populated when reading from database)
	InternalID int64  `json:"internal_id,omitempty"` // Neo4j internal node ID
	ElementID  string `json:"element_id,omitempty"`  // Neo4j element ID (Neo4j 5.0+)

	// Timestamps
	CreatedAt int64 `json:"created_at,omitempty"` // Unix timestamp when entity was created
	UpdatedAt int64 `json:"updated_at,omitempty"` // Unix timestamp when entity was last updated

	// Version control
	Version int          `json:"version,omitempty"` // Node version number
	Status  EntityStatus `json:"status,omitempty"`  // Node lifecycle status
}

// Relationship represents a graph relationship/edge
type Relationship struct {
	ID         string                 `json:"id,omitempty"`
	Type       string                 `json:"type"`
	StartNode  string                 `json:"start_node"`           // Start node ID
	EndNode    string                 `json:"end_node"`             // End node ID
	Properties map[string]interface{} `json:"properties,omitempty"` // Relationship properties

	// GraphRAG specific fields
	Description     string    `json:"description,omitempty"`      // Relationship description
	Confidence      float64   `json:"confidence,omitempty"`       // Extraction confidence score (0-1)
	Weight          float64   `json:"weight,omitempty"`           // Relationship strength/weight
	EmbeddingVector []float64 `json:"embedding_vector,omitempty"` // Relationship embedding

	// Source tracking
	SourceDocuments  []string         `json:"source_documents,omitempty"`  // Document IDs where relationship was found
	SourceChunks     []string         `json:"source_chunks,omitempty"`     // Chunk IDs where relationship was extracted
	ExtractionMethod ExtractionMethod `json:"extraction_method,omitempty"` // Extraction method used

	// Neo4j internal fields (populated when reading from database)
	InternalID  int64  `json:"internal_id,omitempty"`   // Neo4j internal relationship ID
	ElementID   string `json:"element_id,omitempty"`    // Neo4j element ID (Neo4j 5.0+)
	StartNodeID int64  `json:"start_node_id,omitempty"` // Neo4j internal start node ID
	EndNodeID   int64  `json:"end_node_id,omitempty"`   // Neo4j internal end node ID

	// Timestamps
	CreatedAt int64 `json:"created_at,omitempty"` // Unix timestamp when relationship was created
	UpdatedAt int64 `json:"updated_at,omitempty"` // Unix timestamp when relationship was last updated

	// Version control
	Version int          `json:"version,omitempty"` // Relationship version number
	Status  EntityStatus `json:"status,omitempty"`  // Relationship lifecycle status
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
	GraphName string `json:"graph_name"` // Target graph name

	// Query Type and Content
	QueryType string `json:"query_type"`      // "cypher", "traversal", "path", "analytics", "custom"
	Query     string `json:"query,omitempty"` // Query string (e.g., Cypher query)

	// Traversal-specific options (when QueryType is "traversal")
	TraversalOptions *GraphTraversalOptions `json:"traversal_options,omitempty"`

	// Analytics-specific options (when QueryType is "analytics")
	AnalyticsOptions *GraphAnalyticsOptions `json:"analytics_options,omitempty"`

	// General query parameters
	Parameters map[string]interface{} `json:"parameters,omitempty"` // Query parameters

	// Result control
	Limit      int      `json:"limit,omitempty"`
	Skip       int      `json:"skip,omitempty"`
	OrderBy    []string `json:"order_by,omitempty"`    // Fields to order by
	ReturnType string   `json:"return_type,omitempty"` // "nodes", "relationships", "paths", "records", "all"

	// Performance and safety
	Timeout  int  `json:"timeout,omitempty"`   // Query timeout in seconds
	ReadOnly bool `json:"read_only,omitempty"` // Whether this is a read-only query
	Explain  bool `json:"explain,omitempty"`   // Whether to return query execution plan
	Profile  bool `json:"profile,omitempty"`   // Whether to return detailed profiling info
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
	GraphName  string                 `json:"graph_name"`           // Target graph name
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
	GraphStore  GraphStore  `json:"graph_store"`
	VectorStore VectorStore `json:"vector_store"`
	Embedding   Embedding   `json:"embedding_function"`

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

// RestoreOptions represents options for restoring data (unified for VectorStore and GraphStore)
type RestoreOptions struct {
	CollectionName string                 `json:"collection_name,omitempty"` // For VectorStore (deprecated, use Name)
	GraphName      string                 `json:"graph_name,omitempty"`      // For GraphStore (deprecated, use Name)
	Name           string                 `json:"name"`                      // Unified name field for collection/graph
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

// DocumentMetadataUpdate represents metadata update for a specific document
type DocumentMetadataUpdate struct {
	DocumentID string                 `json:"document_id"`        // Document ID to update
	Metadata   map[string]interface{} `json:"metadata,omitempty"` // Specific metadata for this document (if provided, overrides defaultMetadata)
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
	Filter         map[string]interface{} `json:"filter,omitempty"`    // Metadata filter
	Limit          int                    `json:"limit,omitempty"`     // Number of documents per batch (default: 100)
	ScrollID       string                 `json:"scroll_id,omitempty"` // Scroll ID for continuing pagination
	OrderBy        []string               `json:"order_by,omitempty"`  // Fields to order by
	IncludeVector  bool                   `json:"include_vector"`      // Whether to include vector data
	IncludePayload bool                   `json:"include_payload"`     // Whether to include payload/metadata
	Fields         []string               `json:"fields,omitempty"`    // Specific fields to retrieve
}

// ScrollResult represents scroll-based query results
type ScrollResult struct {
	Documents []*Document `json:"documents"`
	ScrollID  string      `json:"scroll_id,omitempty"` // ID for next scroll request
	HasMore   bool        `json:"has_more"`            // Whether there are more results
}

// SearchFacet represents a facet for faceted search
type SearchFacet struct {
	Field  string           `json:"field"`  // Metadata field name
	Values map[string]int64 `json:"values"` // Value -> count mapping
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

// ===== ID Generation Types =====

// CollectionIDs represents the collection identifiers for vector, graph, and KV store databases
type CollectionIDs struct {
	Vector string `json:"vector"` // Vector database collection ID
	Graph  string `json:"graph"`  // Graph database collection ID (uses Vector as prefix)
	Store  string `json:"store"`  // KV store database collection ID (uses Vector as prefix)
}

// ===== Embedding Result Types =====

// EmbeddingUsage represents usage statistics for embedding operations
type EmbeddingUsage struct {
	TotalTokens  int `json:"total_tokens"`  // Total number of tokens processed
	PromptTokens int `json:"prompt_tokens"` // Number of tokens in the input
	TotalTexts   int `json:"total_texts"`   // Total number of texts processed
}

// EmbeddingType represents the type of embedding
type EmbeddingType string

const (
	// EmbeddingTypeDense represents dense vector embeddings
	EmbeddingTypeDense EmbeddingType = "dense"

	// EmbeddingTypeSparse represents sparse vector embeddings
	EmbeddingTypeSparse EmbeddingType = "sparse"
)

// String returns the string representation of EmbeddingType
func (et EmbeddingType) String() string {
	return string(et)
}

// IsValid validates if the embedding type is supported
func (et EmbeddingType) IsValid() bool {
	switch et {
	case EmbeddingTypeDense, EmbeddingTypeSparse:
		return true
	default:
		return false
	}
}

// GetSupportedEmbeddingTypes returns all supported embedding types
func GetSupportedEmbeddingTypes() []EmbeddingType {
	return []EmbeddingType{
		EmbeddingTypeDense,
		EmbeddingTypeSparse,
	}
}

// EmbeddingResult represents the result of a single embedding operation
type EmbeddingResult struct {
	Usage     EmbeddingUsage `json:"usage"`               // Usage statistics
	Model     string         `json:"model"`               // Model used for embedding
	Type      EmbeddingType  `json:"type"`                // Type of embedding (dense/sparse)
	Embedding []float64      `json:"embedding,omitempty"` // Dense vector (for dense embeddings)
	Indices   []uint32       `json:"indices,omitempty"`   // Sparse vector indices (for sparse embeddings)
	Values    []float32      `json:"values,omitempty"`    // Sparse vector values (for sparse embeddings)
}

// IsDense returns true if this is a dense embedding
func (er *EmbeddingResult) IsDense() bool {
	return er.Type == EmbeddingTypeDense
}

// IsSparse returns true if this is a sparse embedding
func (er *EmbeddingResult) IsSparse() bool {
	return er.Type == EmbeddingTypeSparse
}

// GetDenseEmbedding returns the dense embedding vector
func (er *EmbeddingResult) GetDenseEmbedding() []float64 {
	if er.IsDense() {
		return er.Embedding
	}
	return nil
}

// GetSparseEmbedding returns the sparse embedding indices and values
func (er *EmbeddingResult) GetSparseEmbedding() ([]uint32, []float32) {
	if er.IsSparse() {
		return er.Indices, er.Values
	}
	return nil, nil
}

// EmbeddingResults represents the result of multiple embedding operations
type EmbeddingResults struct {
	Usage            EmbeddingUsage    `json:"usage"`                       // Combined usage statistics
	Model            string            `json:"model"`                       // Model used for embedding
	Type             EmbeddingType     `json:"type"`                        // Type of embedding (dense/sparse)
	Embeddings       [][]float64       `json:"embeddings,omitempty"`        // Dense vectors (for dense embeddings)
	SparseEmbeddings []SparseEmbedding `json:"sparse_embeddings,omitempty"` // Sparse vectors (for sparse embeddings)
}

// SparseEmbedding represents a single sparse embedding
type SparseEmbedding struct {
	Indices []uint32  `json:"indices"` // Non-zero indices
	Values  []float32 `json:"values"`  // Non-zero values
}

// IsDense returns true if this contains dense embeddings
func (ers *EmbeddingResults) IsDense() bool {
	return ers.Type == EmbeddingTypeDense
}

// IsSparse returns true if this contains sparse embeddings
func (ers *EmbeddingResults) IsSparse() bool {
	return ers.Type == EmbeddingTypeSparse
}

// GetDenseEmbeddings returns all dense embedding vectors
func (ers *EmbeddingResults) GetDenseEmbeddings() [][]float64 {
	if ers.IsDense() {
		return ers.Embeddings
	}
	return nil
}

// GetSparseEmbeddings returns all sparse embeddings
func (ers *EmbeddingResults) GetSparseEmbeddings() []SparseEmbedding {
	if ers.IsSparse() {
		return ers.SparseEmbeddings
	}
	return nil
}

// Count returns the number of embeddings
func (ers *EmbeddingResults) Count() int {
	if ers.IsDense() {
		return len(ers.Embeddings)
	}
	return len(ers.SparseEmbeddings)
}

// ===== Graph Database Types (Flexible Design) =====

// GraphStoreConfig represents configuration for graph store (similar to VectorStoreConfig)
type GraphStoreConfig struct {
	// Basic Configuration
	StoreType   string `json:"store_type"`   // "neo4j", "kuzu", etc.
	DatabaseURL string `json:"database_url"` // Database connection URL

	// Performance Settings (common across all stores)
	BatchSize    int `json:"batch_size,omitempty"`    // Default batch size for operations
	QueryTimeout int `json:"query_timeout,omitempty"` // Query timeout in seconds

	// Business Logic Settings (common graph operations)
	DefaultGraphName string `json:"default_graph_name,omitempty"` // Default graph/namespace to use
	AutoCreateGraph  bool   `json:"auto_create_graph,omitempty"`  // Whether to auto-create graphs if they don't exist

	// Schema and Indexing (common across graph databases)
	AutoIndex      bool     `json:"auto_index,omitempty"`       // Whether to automatically create indexes
	IndexNodeProps []string `json:"index_node_props,omitempty"` // Node properties to auto-index (id, name, type, etc.)
	IndexRelProps  []string `json:"index_rel_props,omitempty"`  // Relationship properties to auto-index
	EnforceSchema  bool     `json:"enforce_schema,omitempty"`   // Whether to enforce strict schema validation

	// Transaction Settings (most graph DBs support transactions)
	AutoCommit      bool `json:"auto_commit,omitempty"`      // Whether to auto-commit single operations
	TransactionSize int  `json:"transaction_size,omitempty"` // Max operations per transaction (for batch)

	// Data Consistency (common business requirement)
	AllowDuplicates bool `json:"allow_duplicates,omitempty"`  // Whether to allow duplicate nodes/relationships
	MergeOnConflict bool `json:"merge_on_conflict,omitempty"` // Whether to merge when conflicts occur

	// Store-specific Configuration (delegated to individual drivers)
	DriverConfig map[string]interface{} `json:"driver_config,omitempty"` // Driver-specific configuration
}

// Validate validates the graph store configuration
func (c *GraphStoreConfig) Validate() error {
	if c.StoreType == "" {
		return fmt.Errorf("store type cannot be empty")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("database_url must be provided")
	}
	return nil
}

// GraphConfig represents configuration for a specific graph
type GraphConfig struct {
	Description  string                 `json:"description,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	IndexConfig  *GraphIndexConfig      `json:"index_config,omitempty"`
	SchemaConfig *GraphSchemaConfig     `json:"schema_config,omitempty"`
}

// GraphIndexConfig represents index configuration for a graph
type GraphIndexConfig struct {
	AutoIndex      bool     `json:"auto_index"`                // Whether to automatically create indexes
	NodeIndexes    []string `json:"node_indexes,omitempty"`    // Properties to index for nodes
	RelIndexes     []string `json:"rel_indexes,omitempty"`     // Properties to index for relationships
	FullTextFields []string `json:"fulltext_fields,omitempty"` // Fields for full-text search
	VectorFields   []string `json:"vector_fields,omitempty"`   // Fields for vector similarity search
}

// GraphSchemaConfig represents schema configuration
type GraphSchemaConfig struct {
	Strict      bool                   `json:"strict"` // Whether to enforce strict schema
	NodeLabels  []string               `json:"node_labels,omitempty"`
	RelTypes    []string               `json:"rel_types,omitempty"`
	Constraints []SchemaConstraint     `json:"constraints,omitempty"`
	Defaults    map[string]interface{} `json:"defaults,omitempty"`
}

// GraphNode represents a flexible graph node (dynamic properties)
type GraphNode struct {
	ID         string                 `json:"id"`
	Labels     []string               `json:"labels"`
	Properties map[string]interface{} `json:"properties"`

	// Vector embedding support
	Embedding  []float64            `json:"embedding,omitempty"`  // Primary embedding vector
	Embeddings map[string][]float64 `json:"embeddings,omitempty"` // Multiple named embeddings

	// GraphRAG specific fields (simplified design)
	EntityType  string  `json:"entity_type,omitempty"` // Entity type
	Description string  `json:"description,omitempty"` // Entity description
	Confidence  float64 `json:"confidence,omitempty"`  // Confidence score
	Importance  float64 `json:"importance,omitempty"`  // Importance score (PageRank, etc.)

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`
}

// GraphRelationship represents a flexible graph relationship
type GraphRelationship struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	StartNode  string                 `json:"start_node"`
	EndNode    string                 `json:"end_node"`
	Properties map[string]interface{} `json:"properties"`

	// Vector embedding support
	Embedding  []float64            `json:"embedding,omitempty"`  // Primary embedding vector
	Embeddings map[string][]float64 `json:"embeddings,omitempty"` // Multiple named embeddings

	// GraphRAG specific fields (simplified design)
	Description string  `json:"description,omitempty"` // Relationship description
	Confidence  float64 `json:"confidence,omitempty"`  // Confidence score
	Weight      float64 `json:"weight,omitempty"`      // Relationship weight

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`
}

// AddNodesOptions represents options for adding nodes
type AddNodesOptions struct {
	GraphName string       `json:"graph_name"`
	Nodes     []*GraphNode `json:"nodes"`
	BatchSize int          `json:"batch_size,omitempty"` // Batch size for bulk insert
	Upsert    bool         `json:"upsert,omitempty"`     // If true, update existing nodes with same ID
	Timeout   int          `json:"timeout,omitempty"`    // Operation timeout in seconds
}

// GetNodesOptions represents options for retrieving nodes
type GetNodesOptions struct {
	GraphName string                 `json:"graph_name"`
	IDs       []string               `json:"ids,omitempty"`    // Specific node IDs
	Labels    []string               `json:"labels,omitempty"` // Filter by labels
	Filter    map[string]interface{} `json:"filter,omitempty"` // Property filters

	// Return control
	IncludeProperties bool     `json:"include_properties"` // Whether to include properties
	IncludeMetadata   bool     `json:"include_metadata"`   // Whether to include metadata
	Fields            []string `json:"fields,omitempty"`   // Specific fields to retrieve
	Limit             int      `json:"limit,omitempty"`    // Maximum results
}

// DeleteNodesOptions represents options for deleting nodes
type DeleteNodesOptions struct {
	GraphName  string                 `json:"graph_name"`
	IDs        []string               `json:"ids,omitempty"`         // Specific node IDs
	Filter     map[string]interface{} `json:"filter,omitempty"`      // Filter for bulk delete
	DeleteRels bool                   `json:"delete_rels,omitempty"` // Whether to delete connected relationships
	DryRun     bool                   `json:"dry_run"`               // Preview what would be deleted
	BatchSize  int                    `json:"batch_size,omitempty"`
	Timeout    int                    `json:"timeout,omitempty"`
}

// AddRelationshipsOptions represents options for adding relationships
type AddRelationshipsOptions struct {
	GraphName     string               `json:"graph_name"`
	Relationships []*GraphRelationship `json:"relationships"`
	BatchSize     int                  `json:"batch_size,omitempty"`
	Upsert        bool                 `json:"upsert,omitempty"`
	CreateNodes   bool                 `json:"create_nodes,omitempty"` // Create nodes if they don't exist
	Timeout       int                  `json:"timeout,omitempty"`
}

// GetRelationshipsOptions represents options for retrieving relationships
type GetRelationshipsOptions struct {
	GraphName string                 `json:"graph_name"`
	IDs       []string               `json:"ids,omitempty"`       // Specific relationship IDs
	Types     []string               `json:"types,omitempty"`     // Filter by relationship types
	NodeIDs   []string               `json:"node_ids,omitempty"`  // Filter by connected nodes
	Direction string                 `json:"direction,omitempty"` // "IN", "OUT", "BOTH"
	Filter    map[string]interface{} `json:"filter,omitempty"`    // Property filters

	// Return control
	IncludeProperties bool     `json:"include_properties"`
	IncludeMetadata   bool     `json:"include_metadata"`
	Fields            []string `json:"fields,omitempty"`
	Limit             int      `json:"limit,omitempty"`
}

// DeleteRelationshipsOptions represents options for deleting relationships
type DeleteRelationshipsOptions struct {
	GraphName string                 `json:"graph_name"`
	IDs       []string               `json:"ids,omitempty"`
	Filter    map[string]interface{} `json:"filter,omitempty"`
	DryRun    bool                   `json:"dry_run"`
	BatchSize int                    `json:"batch_size,omitempty"`
	Timeout   int                    `json:"timeout,omitempty"`
}

// DynamicGraphSchema represents a discovered/dynamic graph schema
type DynamicGraphSchema struct {
	NodeLabels        []string                  `json:"node_labels"`
	RelationshipTypes []string                  `json:"relationship_types"`
	NodeProperties    map[string][]PropertyInfo `json:"node_properties"` // label -> property info
	RelProperties     map[string][]PropertyInfo `json:"rel_properties"`  // type -> property info
	Constraints       []SchemaConstraint        `json:"constraints"`
	Indexes           []SchemaIndex             `json:"indexes"`
	Statistics        *GraphSchemaStats         `json:"statistics,omitempty"`
}

// PropertyInfo represents information about a property
type PropertyInfo struct {
	Name         string        `json:"name"`
	Type         string        `json:"type"` // "string", "int", "float", "bool", "array", "object"
	Nullable     bool          `json:"nullable"`
	SampleValues []interface{} `json:"sample_values,omitempty"`
	Count        int64         `json:"count,omitempty"`      // Number of entities with this property
	Uniqueness   float64       `json:"uniqueness,omitempty"` // Uniqueness ratio (0-1)
}

// GraphSchemaStats represents statistics about the graph schema
type GraphSchemaStats struct {
	TotalNodes         int64                  `json:"total_nodes"`
	TotalRelationships int64                  `json:"total_relationships"`
	NodeCounts         map[string]int64       `json:"node_counts"` // label -> count
	RelCounts          map[string]int64       `json:"rel_counts"`  // type -> count
	AvgDegree          float64                `json:"avg_degree"`
	MaxDegree          int                    `json:"max_degree"`
	Density            float64                `json:"density"`
	ExtraStats         map[string]interface{} `json:"extra_stats,omitempty"`
}

// CreateIndexOptions represents options for creating indexes
type CreateIndexOptions struct {
	GraphName   string                 `json:"graph_name"`
	IndexType   string                 `json:"index_type"`       // "BTREE", "FULLTEXT", "VECTOR", etc.
	Target      string                 `json:"target"`           // "NODE" or "RELATIONSHIP"
	Labels      []string               `json:"labels,omitempty"` // Node labels or relationship types
	Properties  []string               `json:"properties"`       // Properties to index
	Name        string                 `json:"name,omitempty"`   // Index name
	Config      map[string]interface{} `json:"config,omitempty"` // Index-specific configuration
	IfNotExists bool                   `json:"if_not_exists"`    // Don't error if index already exists
}

// DropIndexOptions represents options for dropping indexes
type DropIndexOptions struct {
	GraphName string `json:"graph_name"`
	Name      string `json:"name"`      // Index name
	IfExists  bool   `json:"if_exists"` // Don't error if index doesn't exist
}

// GraphStats represents statistics about a graph (similar to VectorStoreStats)
type GraphStats struct {
	TotalNodes         int64                  `json:"total_nodes"`
	TotalRelationships int64                  `json:"total_relationships"`
	NodeLabels         []string               `json:"node_labels"`
	RelationshipTypes  []string               `json:"relationship_types"`
	AvgDegree          float64                `json:"avg_degree"`
	MaxDegree          int                    `json:"max_degree"`
	Density            float64                `json:"density"`
	StorageSize        int64                  `json:"storage_size_bytes,omitempty"`
	MemoryUsage        int64                  `json:"memory_usage_bytes,omitempty"`
	IndexCount         int                    `json:"index_count,omitempty"`
	ExtraStats         map[string]interface{} `json:"extra_stats,omitempty"`
}

// GraphBackupOptions represents options for creating graph backups
type GraphBackupOptions struct {
	GraphName   string                 `json:"graph_name"`
	Format      string                 `json:"format,omitempty"` // "json", "gexf", "graphml", "cypher", etc.
	Compress    bool                   `json:"compress"`
	Filter      map[string]interface{} `json:"filter,omitempty"` // Filter what to backup
	ExtraParams map[string]interface{} `json:"extra_params,omitempty"`
}

// GraphRestoreOptions represents options for restoring graph data
type GraphRestoreOptions struct {
	GraphName   string                 `json:"graph_name"`
	Format      string                 `json:"format,omitempty"` // "json", "gexf", "graphml", "cypher", etc.
	Compress    bool                   `json:"compress"`         // whether to compress the backup file
	Force       bool                   `json:"force"`            // Whether to overwrite existing data
	CreateGraph bool                   `json:"create_graph"`     // Whether to create graph if it doesn't exist
	ExtraParams map[string]interface{} `json:"extra_params,omitempty"`
}

// ==== GraphRag Types =====

// ConverterStatus represents the status of the conversion
type ConverterStatus string

// ConverterStatus values
const (
	ConverterStatusSuccess ConverterStatus = "success"
	ConverterStatusError   ConverterStatus = "error"
	ConverterStatusPending ConverterStatus = "pending"
)

// ConvertResult represents the result of a document conversion operation
type ConvertResult struct {
	Text     string                 `json:"text"`     // Converted text content
	Metadata map[string]interface{} `json:"metadata"` // Additional metadata from conversion (page mappings, structure info, etc.)
}

// SearcherStatus represents the status of the search
type SearcherStatus string

// SearchStatus values
const (
	SearchStatusSuccess SearcherStatus = "success"
	SearchStatusError   SearcherStatus = "error"
	SearchStatusPending SearcherStatus = "pending"
)

// SearcherType represents the type of the searcher
type SearcherType string

// SearcherType values
const (
	SearcherTypeVector SearcherType = "vector"
	SearcherTypeGraph  SearcherType = "graph"
	SearcherTypeHybrid SearcherType = "hybrid"
)

// ===== Reranker Types =====

// RerankerStatus represents the status of the reranker
type RerankerStatus string

// RerankerStatus values
const (
	RerankerStatusSuccess RerankerStatus = "success"
	RerankerStatusError   RerankerStatus = "error"
	RerankerStatusPending RerankerStatus = "pending"
)

// RerankerPayload represents the payload of the reranker
type RerankerPayload struct {
	Status   RerankerStatus `json:"status"`
	Message  string         `json:"message"`
	Progress float64        `json:"progress"`
}

// ===== Scorer Types =====

// ScoreStatus represents the status of the scorer
type ScoreStatus string

// ScoreStatus values
const (
	ScoreStatusSuccess ScoreStatus = "success"
	ScoreStatusError   ScoreStatus = "error"
	ScoreStatusPending ScoreStatus = "pending"
)

// ScorePayload represents the payload of the scorer
type ScorePayload struct {
	Status   ScoreStatus `json:"status"`
	Message  string      `json:"message"`
	Progress float64     `json:"progress"`
}

// ===== Weight Types =====

// WeightStatus represents the status of the weight
type WeightStatus string

// WeightStatus values
const (
	WeightStatusSuccess WeightStatus = "success"
	WeightStatusError   WeightStatus = "error"
	WeightStatusPending WeightStatus = "pending"
)

// WeightPayload represents the payload of the weight
type WeightPayload struct {
	Status   WeightStatus `json:"status"`
	Message  string       `json:"message"`
	Progress float64      `json:"progress"`
}

// ===== Vote Types =====

// VoteStatus represents the status of the vote
type VoteStatus string

// VoteStatus values
const (
	VoteStatusSuccess VoteStatus = "success"
	VoteStatusError   VoteStatus = "error"
	VoteStatusPending VoteStatus = "pending"
)

// VotePayload represents the payload of the vote
type VotePayload struct {
	Status   VoteStatus `json:"status"`
	Message  string     `json:"message"`
	Progress float64    `json:"progress"`
}

// ===== Fetcher Types =====

// FetcherStatus represents the status of the fetcher
type FetcherStatus string

// FetcherStatus values
const (
	FetcherStatusSuccess FetcherStatus = "success"
	FetcherStatusError   FetcherStatus = "error"
	FetcherStatusPending FetcherStatus = "pending"
)

// FetcherPayload represents the payload of the fetcher
type FetcherPayload struct {
	Status   FetcherStatus `json:"status"`
	Message  string        `json:"message"`
	Progress float64       `json:"progress"`
	Bytes    int64         `json:"bytes"`
	URL      string        `json:"url"`
}

// ConverterPayload represents the payload of the conversion
type ConverterPayload struct {
	Status   ConverterStatus `json:"status"`
	Message  string          `json:"message"`
	Progress float64         `json:"progress"`
}

// SearcherPayload represents the payload of the search
type SearcherPayload struct {
	Status   SearcherStatus `json:"status"`
	Message  string         `json:"message"`
	Progress float64        `json:"progress"`
}

// ChatMessage is a message in a chat
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Options represents the options for GraphRag
type Options struct {
	// VectorStore is the vector store to use for storing and searching documents
	VectorStore VectorStore

	// GraphStore is the graph store to use for storing and searching relationships (Optional)
	GraphStore GraphStore
}

// UpsertOptions represents the options for GraphRag
type UpsertOptions struct {
	// Chunking is the chunking model to use for chunking documents
	Chunking        Chunking
	ChunkingOptions *ChunkingOptions // Chunking options (Optional)
	// ChunkingProgress ChunkingProgress // Chunking progress callback (Optional)

	// Embedding is the embedding model to use for embedding documents
	Embedding Embedding
	// EmbeddingProgress EmbeddingProgress // Embedding progress callback (Optional)

	// Extraction is the extraction model to use for extracting documents (Optional)
	Extraction Extraction

	// Progress is the progress callback for the upsert
	Progress UpsertProgress

	// ExtractionProgress ExtractionProgress // Extraction progress callback (Optional)

	// ExtractionEmbedding is the embedding model to use for embedding extracted documents (Optional, default is the same as Embedding)
	// ExtractionEmbedding         Embedding
	// ExtractionEmbeddingProgress EmbeddingProgress // Extraction embedding progress callback (Optional)

	// Fetcher is the fetcher to use for fetching documents from URLs (Optional)
	Fetcher Fetcher

	// Converter is the converter to use for converting documents to text (Optional)
	Converter Converter

	// CollectionID is the collection ID to use for storing the document (Optional)
	CollectionID string `json:"collection_id,omitempty"`

	// DocID is the document ID to use for tracking, if not provided, will auto-generate (Optional)
	DocID string `json:"doc_id,omitempty"`

	// Metadata is the metadata to use for the document (Optional)
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpsertProgressType represents the type of the progress
type UpsertProgressType string

// UpsertProgressType values
const (
	UpsertProgressTypeConverter  UpsertProgressType = "converter"
	UpsertProgressTypeChunking   UpsertProgressType = "chunking"
	UpsertProgressTypeEmbedding  UpsertProgressType = "embedding"
	UpsertProgressTypeExtraction UpsertProgressType = "extraction"
	UpsertProgressTypeFetcher    UpsertProgressType = "fetcher"
)

// UpsertProgressPayload represents the progress of the upsert
type UpsertProgressPayload struct {
	ID       string                 `json:"id"`       // ID of the request
	Progress float64                `json:"progress"` // 0 - 100
	Type     UpsertProgressType     `json:"type"`     // "converter", "chunking", "embedding", "extraction", "fetcher"
	Data     map[string]interface{} `json:"data"`     // Data to display
}

// UpsertCallback is the callback for the upsert progress
type UpsertCallback struct {
	Converter  ConverterProgress
	Chunking   ChunkingProgress
	Embedding  EmbeddingProgress
	Extraction ExtractionProgress
	Fetcher    FetcherProgress
}

// QueryOptions represents the options for querying the graph
type QueryOptions struct {
	CollectionID string `json:"collection_id"`
	DocumentID   string `json:"document_id"`

	// Query is the query to use for searching documents
	Query string `json:"query"` // Query or History at least one of them is required

	// History is the history of messages to use for searching documents
	History []ChatMessage `json:"history"` // History or Query at least one of them is required

	// Filter is the filter to use for searching documents (Optional)
	Filter map[string]interface{} `json:"filter,omitempty"`

	// Embedding is the embedding model to use for embedding documents
	Embedding         Embedding
	EmbeddingProgress EmbeddingProgress // Embedding progress callback (Optional)

	// Extraction is the extraction model to use for extracting documents (Optional)
	Extraction         Extraction
	ExtractionProgress ExtractionProgress // Extraction progress callback (Optional)

	// ExtractionEmbedding is the embedding model to use for embedding extracted documents (Optional, default is the same as Embedding)
	ExtractionEmbedding         Embedding
	ExtractionEmbeddingProgress EmbeddingProgress // Extraction embedding progress callback (Optional)

	// Fetcher is the fetcher to use for fetching documents from URLs (Optional)
	Fetcher Fetcher

	// Converter is the converter to use for converting documents to text (Optional)
	Converter Converter

	// Reranker is the reranker to use for reranking documents (Optional)
	Reranker Reranker

	// Searcher is the searcher to use for searching documents (Optional)
	Searcher Searcher
}

// UpdateWeightOptions represents the options for updating weight
type UpdateWeightOptions struct {
	Compute  WeightCompute
	Progress WeightProgress
}

// UpdateScoreOptions represents the options for updating score
type UpdateScoreOptions struct {
	Compute  ScoreCompute
	Progress ScoreProgress
}

// UpdateVoteOptions represents the options for updating vote
type UpdateVoteOptions struct {
	Compute  VoteCompute
	Progress VoteProgress
	Reaction *SegmentReaction
}

// ScrollVotesOptions represents the options for scrolling votes
type ScrollVotesOptions struct {
	SegmentID string   `json:"segment_id,omitempty"` // Filter by segment ID
	VoteType  VoteType `json:"vote_type,omitempty"`  // Filter by vote type
	Source    string   `json:"source,omitempty"`     // Filter by reaction source
	Scenario  string   `json:"scenario,omitempty"`   // Filter by reaction scenario
	Limit     int      `json:"limit,omitempty"`      // Number of votes per page (default 20, max 100)
	Cursor    string   `json:"cursor,omitempty"`     // Cursor for pagination
}

// VoteScrollResult represents the result of vote listing with scroll pagination
type VoteScrollResult struct {
	Votes      []SegmentVote `json:"votes"`       // List of votes
	NextCursor string        `json:"next_cursor"` // Cursor for next page
	HasMore    bool          `json:"has_more"`    // Whether there are more votes
	Total      int           `json:"total"`       // Total count (if available)
}

// UpdateHitOptions represents the options for updating hit
type UpdateHitOptions struct {
	Reaction *SegmentReaction
}

// VoteRemoval represents a vote to be removed
type VoteRemoval struct {
	SegmentID string `json:"segment_id"` // Segment ID
	VoteID    string `json:"vote_id"`    // Vote ID to remove
}

// HitRemoval represents a hit to be removed
type HitRemoval struct {
	SegmentID string `json:"segment_id"` // Segment ID
	HitID     string `json:"hit_id"`     // Hit ID to remove
}

// ScrollHitsOptions represents the options for scrolling hits
type ScrollHitsOptions struct {
	SegmentID string `json:"segment_id,omitempty"` // Filter by segment ID
	Source    string `json:"source,omitempty"`     // Filter by reaction source
	Scenario  string `json:"scenario,omitempty"`   // Filter by reaction scenario
	Limit     int    `json:"limit,omitempty"`      // Number of hits per page (default 20, max 100)
	Cursor    string `json:"cursor,omitempty"`     // Cursor for pagination
}

// HitScrollResult represents the result of hit listing with scroll pagination
type HitScrollResult struct {
	Hits       []SegmentHit `json:"hits"`        // List of hits
	NextCursor string       `json:"next_cursor"` // Cursor for next page
	HasMore    bool         `json:"has_more"`    // Whether there are more hits
	Total      int          `json:"total"`       // Total count (if available)
}

// Collection represents a collection of documents
type Collection struct {
	ID               string                   `json:"id"`
	Metadata         map[string]interface{}   `json:"metadata"`
	VectorConfig     *VectorStoreConfig       `json:"vector_config"`     // Connection configuration
	CollectionConfig *CreateCollectionOptions `json:"collection_config"` // Collection creation configuration
	GraphStoreConfig *GraphStoreConfig        `json:"graph_store_config"`
}

// CollectionInfo represents the information of a collection
type CollectionInfo struct {
	ID       string                   `json:"id"`
	Metadata map[string]interface{}   `json:"metadata"`
	Config   *CreateCollectionOptions `json:"config"`
}

// CollectionConfig represents the configuration for a collection
type CollectionConfig struct {
	ID       string                   `json:"id"`
	Metadata map[string]interface{}   `json:"metadata"`
	Config   *CreateCollectionOptions `json:"config"`
}

// Segment represents a segment of a document
type Segment struct {
	CollectionID    string                 `json:"collection_id"`
	DocumentID      string                 `json:"document_id"`
	ID              string                 `json:"id"`
	Text            string                 `json:"text"`
	Nodes           []GraphNode            `json:"nodes"`
	Relationships   []GraphRelationship    `json:"relationships"`
	Parents         []string               `json:"parents"`
	Children        []string               `json:"children"`
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Version         int                    `json:"version"`
	Weight          float64                `json:"weight"`
	Score           float64                `json:"score"`
	ScoreDimensions map[string]float64     `json:"score_dimensions,omitempty"`
	Positive        int                    `json:"positive"` // Positive vote count
	Negative        int                    `json:"negative"` // Negative vote count
	Hit             int                    `json:"hit"`      // Hit count for the segment
}

// SegmentText represents a segment of a document
type SegmentText struct {
	ID       string                 `json:"id,omitempty"`
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SegmentTree represents a hierarchical tree structure of segment parents
type SegmentTree struct {
	Segment *Segment     `json:"segment"`          // The segment node
	Parent  *SegmentTree `json:"parent,omitempty"` // Parent node (only one in document hierarchy)
	Depth   int          `json:"depth"`            // Depth in the original document hierarchy (extracted from metadata)
}

// SegmentGraph represents the graph information for a specific segment
type SegmentGraph struct {
	DocID         string              `json:"doc_id"`
	SegmentID     string              `json:"segment_id"`
	Entities      []GraphNode         `json:"entities"`
	Relationships []GraphRelationship `json:"relationships"`
}

// EntityDeduplicationResult contains the result of entity deduplication
type EntityDeduplicationResult struct {
	NormalizedID string   `json:"normalized_id"`
	DocIDs       []string `json:"doc_ids"`
	IsUpdate     bool     `json:"is_update"`
}

// RelationshipDeduplicationResult contains the result of relationship deduplication
type RelationshipDeduplicationResult struct {
	NormalizedID string   `json:"normalized_id"`
	DocIDs       []string `json:"doc_ids"`
	IsUpdate     bool     `json:"is_update"`
}

// SegmentExtractionResult represents the result of extracting entities and relationships from a segment
// Minimal structure containing only statistical information for frontend consumption
type SegmentExtractionResult struct {
	DocID              string `json:"doc_id"`              // Document ID
	SegmentID          string `json:"segment_id"`          // Segment ID
	ExtractionModel    string `json:"extraction_model"`    // Model used for extraction
	EntitiesCount      int    `json:"entities_count"`      // Number of entities extracted
	RelationshipsCount int    `json:"relationships_count"` // Number of relationships extracted
	// Removed fields (frontend doesn't need detailed data):
	// - ExtractedEntities: detailed entity data not needed
	// - ExtractedRelationships: detailed relationship data not needed
	// - Text: can be retrieved via GetSegment if needed
	// - ActualEntityIDs: internal implementation detail
	// - ActualRelationshipIDs: internal implementation detail
	// - EntityDeduplicationResults: internal implementation detail
	// - RelationshipDeduplicationResults: internal implementation detail
}

// SaveExtractionResultsResponse represents the response from SaveExtractionResults
// Contains the actual entities and relationships that were saved to the database
type SaveExtractionResultsResponse struct {
	SavedEntities      []GraphNode         `json:"saved_entities"`      // Actual entities saved (after deduplication)
	SavedRelationships []GraphRelationship `json:"saved_relationships"` // Actual relationships saved (with updated node IDs)
	EntitiesCount      int                 `json:"entities_count"`      // Number of entities saved
	RelationshipsCount int                 `json:"relationships_count"` // Number of relationships saved
	ProcessedCount     int                 `json:"processed_count"`     // Number of extraction results processed
}

// SegmentReaction represents a reaction for a segment
type SegmentReaction struct {
	Source    string                 `json:"source,omitempty"`    // Source of the reaction, e.g. "chat", "api", "bot", etc.
	Scenario  string                 `json:"scenario,omitempty"`  // Scenario of the reaction, e.g. "question", "search", "response", etc.
	Query     string                 `json:"query,omitempty"`     // Query of the reaction, e.g. "What is the capital of France?", etc.
	Candidate string                 `json:"candidate,omitempty"` // Candidate of the reaction, e.g. "Paris", etc.
	Context   map[string]interface{} `json:"context,omitempty"`   // Context of the reaction, e.g. {"user_id": "123", "session_id": "456", "rank": 1, "score": 0.95}
}

// SegmentVote represents a vote for a segment
type SegmentVote struct {
	ID     string   `json:"id"`               // Segment ID
	VoteID string   `json:"vote_id"`          // Unique vote ID
	Vote   VoteType `json:"vote"`             // Vote type, e.g. "positive", "negative"
	HitID  string   `json:"hit_id,omitempty"` // Optional: Hit ID to associate the vote with a specific hit
	*SegmentReaction
}

// SegmentHit represents a hit for a segment
type SegmentHit struct {
	ID    string `json:"id"`     // Segment ID
	HitID string `json:"hit_id"` // Unique hit ID
	*SegmentReaction
}

// SegmentScore represents a score for a segment
type SegmentScore struct {
	ID         string             `json:"id"`
	Score      float64            `json:"score,omitempty"`
	Dimensions map[string]float64 `json:"dimensions,omitempty"`
}

// SegmentWeight represents a weight for a segment
type SegmentWeight struct {
	ID     string  `json:"id"`
	Weight float64 `json:"weight,omitempty"`
}

// ===== Segment Pagination Types =====

// ListSegmentsOptions represents options for listing segments with pagination
type ListSegmentsOptions struct {
	Filter  map[string]interface{} `json:"filter,omitempty"`   // Metadata filter (vote, score, weight, etc.)
	Limit   int                    `json:"limit,omitempty"`    // Number of segments per page (default: 100)
	Offset  int                    `json:"offset,omitempty"`   // Offset for pagination
	OrderBy []string               `json:"order_by,omitempty"` // Fields to order by (score, weight, vote, created_at, etc.)
	Fields  []string               `json:"fields,omitempty"`   // Specific fields to retrieve

	// Include options
	IncludeNodes         bool `json:"include_nodes"`         // Whether to include graph nodes
	IncludeRelationships bool `json:"include_relationships"` // Whether to include graph relationships
	IncludeMetadata      bool `json:"include_metadata"`      // Whether to include segment metadata
}

// PaginatedSegmentsResult represents paginated segment query results
type PaginatedSegmentsResult struct {
	Segments   []Segment `json:"segments"`
	Total      int64     `json:"total"`       // Total number of matching segments (if supported)
	HasMore    bool      `json:"has_more"`    // Whether there are more pages
	NextOffset int       `json:"next_offset"` // Offset for next page
}

// ScrollSegmentsOptions represents options for scrolling through segments (iterator-style)
type ScrollSegmentsOptions struct {
	Filter   map[string]interface{} `json:"filter,omitempty"`    // Metadata filter (vote, score, weight, etc.)
	Limit    int                    `json:"limit,omitempty"`     // Number of segments per batch (default: 100)
	ScrollID string                 `json:"scroll_id,omitempty"` // Scroll ID for continuing pagination
	OrderBy  []string               `json:"order_by,omitempty"`  // Fields to order by (score, weight, vote, created_at, etc.)
	Fields   []string               `json:"fields,omitempty"`    // Specific fields to retrieve

	// Include options
	IncludeNodes         bool `json:"include_nodes"`         // Whether to include graph nodes
	IncludeRelationships bool `json:"include_relationships"` // Whether to include graph relationships
	IncludeMetadata      bool `json:"include_metadata"`      // Whether to include segment metadata
}

// SegmentScrollResult represents scroll-based segment query results
type SegmentScrollResult struct {
	Segments []Segment `json:"segments"`
	ScrollID string    `json:"scroll_id,omitempty"` // ID for next scroll request
	HasMore  bool      `json:"has_more"`            // Whether there are more results
}
