package graphrag

import (
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/log"
)

// GraphRag is the main struct for the GraphRag system
type GraphRag struct {
	Vector types.VectorStore
	Graph  types.GraphStore
	Store  store.Store
	Logger types.Logger
	System string // System Collection name for internal metadata
}

// Config is the configuration for the GraphRag instance
type Config struct {
	Logger types.Logger
	Vector types.VectorStore // Vector Store for Embedding, Search, Rerank
	Graph  types.GraphStore  // Graph Store for GraphRAG
	Store  store.Store       // For Collection Metadata, Vote, Score, Weight history etc.
	System string            // System Collection name for internal metadata
}

// New creates a new GraphRag instance
func New(config *Config) (*GraphRag, error) {

	// Validate config
	if config == nil || config.Vector == nil {
		return nil, fmt.Errorf("vector store is required")
	}

	// Set default logger
	if config.Logger == nil {
		config.Logger = log.StandardLogger()
	}

	// Set default system collection name
	system := config.System
	if system == "" {
		system = "__graphrag_system"
	}

	// Create GraphRag instance
	return &GraphRag{
		Vector: config.Vector,
		Graph:  config.Graph,
		Store:  config.Store,
		Logger: config.Logger,
		System: system,
	}, nil
}

// WithGraph sets the graph store for the GraphRag instance
func (g *GraphRag) WithGraph(graph types.GraphStore) *GraphRag {
	g.Graph = graph
	return g
}

// WithStore sets the store for the GraphRag instance
func (g *GraphRag) WithStore(store store.Store) *GraphRag {
	g.Store = store
	return g
}

// WithLogger sets the logger for the GraphRag instance
func (g *GraphRag) WithLogger(logger types.Logger) *GraphRag {
	g.Logger = logger
	return g
}

// WithSystem sets the system collection name for the GraphRag instance
func (g *GraphRag) WithSystem(system string) *GraphRag {
	g.System = system
	return g
}
