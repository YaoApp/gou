package graphrag

import (
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/store"
)

// GraphRag is the main struct for the GraphRag system
type GraphRag struct {
	Config *Config
}

// Config is the configuration for the GraphRag instance
type Config struct {
	Logger types.Logger
	Vector types.VectorStore // Vector Store for Embedding, Search, Rerank
	Graph  types.GraphStore  // Graph Store for GraphRAG
	Store  store.Store       // For Collection Metadata, Vote, Score, Weight history etc.
}

// New creates a new GraphRag instance
func New(config *Config) *GraphRag {
	return &GraphRag{Config: config}
}

// WithVector sets the vector store for the GraphRag instance
func (g *GraphRag) WithVector(vector types.VectorStore) *GraphRag {
	g.Config.Vector = vector
	return g
}

// WithGraph sets the graph store for the GraphRag instance
func (g *GraphRag) WithGraph(graph types.GraphStore) *GraphRag {
	g.Config.Graph = graph
	return g
}

// WithStore sets the store for the GraphRag instance
func (g *GraphRag) WithStore(store store.Store) *GraphRag {
	g.Config.Store = store
	return g
}

// WithLogger sets the logger for the GraphRag instance
func (g *GraphRag) WithLogger(logger types.Logger) *GraphRag {
	g.Config.Logger = logger
	return g
}
