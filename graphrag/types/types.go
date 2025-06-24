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

	// Sparse Vector Configuration (for hybrid search)
	EnableSparseVectors bool   `json:"enable_sparse_vectors,omitempty"` // Enable sparse vector support for hybrid retrieval
	DenseVectorName     string `json:"dense_vector_name,omitempty"`     // Named vector for dense vectors (default: "dense")
	SparseVectorName    string `json:"sparse_vector_name,omitempty"`    // Named vector for sparse vectors (default: "sparse")

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
	Use          Extraction   `json:"use"`           // Use the extraction method
	Embedding    Embedding    `json:"embedding"`     // Embedding function to use for extraction
	LLMOptimizer LLMOptimizer `json:"llm_optimizer"` // LLM optimizer for extraction, deduplication, optimization, etc. if not provided, will not be used
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

// RestoreOptions represents options for restoring data
type RestoreOptions struct {
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
