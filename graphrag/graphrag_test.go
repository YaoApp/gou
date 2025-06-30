package graphrag

import (
	"testing"

	"github.com/yaoapp/gou/graphrag/graph/neo4j"
	"github.com/yaoapp/gou/graphrag/vector/qdrant"
	"github.com/yaoapp/kun/log"
)

// ==== Test Functions ====

func TestNew(t *testing.T) {
	configs := GetTestConfigs()

	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid config with vector only",
			config:  configs["vector"],
			wantErr: false,
		},
		{
			name:    "valid config with vector and graph",
			config:  configs["vector+graph"],
			wantErr: false,
		},
		{
			name:    "valid config with all components",
			config:  configs["vector+graph+logger"],
			wantErr: false,
		},
		{
			name:        "invalid config without vector",
			config:      configs["invalid"],
			wantErr:     true,
			errContains: "vector store is required",
		},
		{
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			errContains: "vector store is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graphRag, err := New(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("New() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && err.Error() != tt.errContains {
					t.Errorf("New() error = %v, want error containing %v", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("New() unexpected error = %v", err)
				return
			}

			if graphRag == nil {
				t.Error("New() returned nil GraphRag instance")
				return
			}

			// Verify vector store is set
			if graphRag.Vector == nil {
				t.Error("New() GraphRag.Vector is nil")
			}

			// Verify logger is set (should have default if not provided)
			if graphRag.Logger == nil {
				t.Error("New() GraphRag.Logger is nil")
			}

			// Verify optional components
			if tt.config != nil {
				if tt.config.Graph != nil && graphRag.Graph == nil {
					t.Error("New() GraphRag.Graph should be set but is nil")
				}
				if tt.config.Store != nil && graphRag.Store == nil {
					t.Error("New() GraphRag.Store should be set but is nil")
				}
			}
		})
	}
}

func TestGraphRag_WithGraph(t *testing.T) {
	config := GetTestConfigs()["vector"]
	graphRag, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create GraphRag instance: %v", err)
	}

	graphStore := neo4j.NewStore()
	result := graphRag.WithGraph(graphStore)

	// Should return the same instance
	if result != graphRag {
		t.Error("WithGraph() should return the same GraphRag instance")
	}

	// Should set the graph store
	if graphRag.Graph != graphStore {
		t.Error("WithGraph() failed to set graph store")
	}
}

func TestGraphRag_WithStore(t *testing.T) {
	config := GetTestConfigs()["vector"]
	graphRag, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create GraphRag instance: %v", err)
	}

	// Test with nil store first (should work)
	result := graphRag.WithStore(nil)

	// Should return the same instance
	if result != graphRag {
		t.Error("WithStore() should return the same GraphRag instance")
	}

	// Should set the store (even if nil)
	if graphRag.Store != nil {
		t.Error("WithStore(nil) failed to set store to nil")
	}
}

func TestGraphRag_WithLogger(t *testing.T) {
	config := GetTestConfigs()["vector"]
	graphRag, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create GraphRag instance: %v", err)
	}

	logger := log.StandardLogger()
	result := graphRag.WithLogger(logger)

	// Should return the same instance
	if result != graphRag {
		t.Error("WithLogger() should return the same GraphRag instance")
	}

	// Should set the logger
	if graphRag.Logger != logger {
		t.Error("WithLogger() failed to set logger")
	}
}

func TestGraphRag_FluentInterface(t *testing.T) {
	config := GetTestConfigs()["vector"]
	graphRag, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create GraphRag instance: %v", err)
	}

	graphStore := neo4j.NewStore()
	logger := log.StandardLogger()

	// Test fluent interface chaining
	result := graphRag.WithGraph(graphStore).WithStore(nil).WithLogger(logger)

	// Should return the same instance
	if result != graphRag {
		t.Error("Fluent interface should return the same GraphRag instance")
	}

	// Verify all components are set
	if graphRag.Graph != graphStore {
		t.Error("Graph store not set correctly in fluent interface")
	}
	if graphRag.Store != nil {
		t.Error("Store not set correctly in fluent interface")
	}
	if graphRag.Logger != logger {
		t.Error("Logger not set correctly in fluent interface")
	}
}

func TestConfig_Validation(t *testing.T) {
	vectorStore := qdrant.NewStore()
	graphStore := neo4j.NewStore()
	logger := log.StandardLogger()

	tests := []struct {
		name   string
		config *Config
		valid  bool
	}{
		{
			name:   "complete config",
			config: &Config{Vector: vectorStore, Graph: graphStore, Store: nil, Logger: logger},
			valid:  true,
		},
		{
			name:   "minimal valid config",
			config: &Config{Vector: vectorStore},
			valid:  true,
		},
		{
			name:   "config without vector",
			config: &Config{Graph: graphStore, Store: nil, Logger: logger},
			valid:  false,
		},
		{
			name:   "empty config",
			config: &Config{},
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if tt.valid && err != nil {
				t.Errorf("Expected valid config but got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected invalid config to return error but got nil")
			}
		})
	}
}
