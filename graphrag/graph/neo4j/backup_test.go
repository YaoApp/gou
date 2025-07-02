package neo4j

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// GlobalTestCleanup ensures clean state before running backup tests
func init() {
	// This will run once when the package is loaded
	cleanupExistingTestDatabases()
}

// cleanupExistingTestDatabases removes all existing test databases
func cleanupExistingTestDatabases() {
	url := os.Getenv("NEO4J_TEST_ENTERPRISE_URL")
	user := os.Getenv("NEO4J_TEST_ENTERPRISE_USER")
	pass := os.Getenv("NEO4J_TEST_ENTERPRISE_PASS")

	if url == "" {
		return // Skip if enterprise URL not available
	}

	store := NewStore()
	config := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: fmt.Sprintf("%s?username=%s&password=%s", url, user, pass),
		DriverConfig: map[string]interface{}{
			"url":      url,
			"username": user,
			"password": pass,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := store.Connect(ctx, config)
	if err != nil {
		return // Skip cleanup if can't connect
	}
	defer store.Disconnect(ctx)

	store.SetUseSeparateDatabase(true)
	store.SetIsEnterpriseEdition(true)

	// List and drop test databases
	databases, err := store.listSeparateDatabaseGraphs(ctx)
	if err != nil {
		return
	}

	for _, dbName := range databases {
		if strings.Contains(dbName, "test") && dbName != "neo4j" && dbName != "system" {
			store.dropSeparateDatabaseGraph(ctx, dbName)
		}
	}
}

// TestEnvironment holds the test environment configuration
type TestEnvironment struct {
	Store             *Store
	GraphName         string
	TestNodes         []*types.GraphNode
	TestRelationships []*types.GraphRelationship
	CleanupFuncs      []func() error
}

// setupTestEnvironment sets up test environment for backup tests
func setupTestEnvironment(t *testing.T, useSeparateDatabase bool) *TestEnvironment {
	t.Helper()

	// Choose the appropriate Neo4j instance based on storage mode
	var url, user, pass string
	if useSeparateDatabase {
		// Use Enterprise Edition for separate database tests
		url = os.Getenv("NEO4J_TEST_ENTERPRISE_URL")
		user = os.Getenv("NEO4J_TEST_ENTERPRISE_USER")
		pass = os.Getenv("NEO4J_TEST_ENTERPRISE_PASS")
		if url == "" {
			t.Skip("NEO4J_TEST_ENTERPRISE_URL not set, skipping enterprise tests")
		}
	} else {
		// Use Community Edition for label-based tests
		url = os.Getenv("NEO4J_TEST_URL")
		user = os.Getenv("NEO4J_TEST_USER")
		pass = os.Getenv("NEO4J_TEST_PASS")
		if url == "" {
			t.Skip("NEO4J_TEST_URL not set, skipping Neo4j tests")
		}
	}

	// Create store
	store := NewStore()
	config := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: fmt.Sprintf("%s?username=%s&password=%s", url, user, pass),
		DriverConfig: map[string]interface{}{
			"url":      url,
			"username": user,
			"password": pass,
		},
	}

	// Connect to Neo4j
	ctx := context.Background()
	err := store.Connect(ctx, config)
	if err != nil {
		t.Fatalf("failed to connect to Neo4j: %v", err)
	}

	// Set storage mode
	store.SetUseSeparateDatabase(useSeparateDatabase)
	if useSeparateDatabase {
		store.SetIsEnterpriseEdition(true)
		// Clean up existing test databases to prevent limit issues
		CleanupAllTestDatabases(t, store)
	}

	// Generate unique graph name based on storage mode
	// Use shorter names to avoid limit issues
	timestamp := time.Now().UnixNano() % 1000000 // Use shorter timestamp
	var graphName string
	if useSeparateDatabase {
		graphName = fmt.Sprintf("testdb%d", timestamp)
	} else {
		graphName = fmt.Sprintf("test_backup_%d", timestamp)
	}

	// Create test data
	testNodes := CreateTestNodes(10)
	testRelationships := CreateTestRelationshipsWithNodes(
		[]string{"test_node_0", "test_node_1", "test_node_2"},
		[]string{"test_node_3", "test_node_4", "test_node_5"},
		"RELATED_TO",
	)

	env := &TestEnvironment{
		Store:             store,
		GraphName:         graphName,
		TestNodes:         testNodes,
		TestRelationships: testRelationships,
		CleanupFuncs:      []func() error{},
	}

	// Add cleanup function to drop graph and disconnect
	env.CleanupFuncs = append(env.CleanupFuncs, func() error {
		// Drop test graph if it exists
		if exists, _ := store.GraphExists(ctx, graphName); exists {
			store.DropGraph(ctx, graphName)
		}
		return store.Disconnect(ctx)
	})

	return env
}

// cleanupTestEnvironment cleans up test environment
func cleanupTestEnvironment(env *TestEnvironment, t *testing.T) {
	t.Helper()
	for _, cleanup := range env.CleanupFuncs {
		if err := cleanup(); err != nil {
			t.Logf("cleanup error: %v", err)
		}
	}
}

// setupTestGraph creates a test graph with nodes and relationships
func setupTestGraph(t *testing.T, env *TestEnvironment) {
	t.Helper()
	ctx := context.Background()

	// Create graph
	err := env.Store.CreateGraph(ctx, env.GraphName)
	if err != nil {
		t.Fatalf("failed to create test graph: %v", err)
	}

	// Add nodes
	if len(env.TestNodes) > 0 {
		addNodesOpts := &types.AddNodesOptions{
			GraphName: env.GraphName,
			Nodes:     env.TestNodes,
			BatchSize: 5,
		}
		_, err = env.Store.AddNodes(ctx, addNodesOpts)
		if err != nil {
			t.Fatalf("failed to add test nodes: %v", err)
		}
	}

	// Add relationships
	if len(env.TestRelationships) > 0 {
		addRelsOpts := &types.AddRelationshipsOptions{
			GraphName:     env.GraphName,
			Relationships: env.TestRelationships,
			BatchSize:     5,
		}
		_, err = env.Store.AddRelationships(ctx, addRelsOpts)
		if err != nil {
			t.Fatalf("failed to add test relationships: %v", err)
		}
	}
}

// TestBackup_Basic tests basic backup functionality
func TestBackup_Basic(t *testing.T) {
	tests := []struct {
		name                string
		useSeparateDatabase bool
	}{
		{"LabelBased", false},
		{"SeparateDatabase", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture initial goroutine and memory state
			initialGoroutines := captureGoroutineState()
			initialMemory := captureMemoryStats()

			env := setupTestEnvironment(t, tt.useSeparateDatabase)
			defer cleanupTestEnvironment(env, t)

			setupTestGraph(t, env)

			ctx := context.Background()

			// Test JSON format backup without compression
			t.Run("JSON_NoCompression", func(t *testing.T) {
				var buf bytes.Buffer
				opts := &types.GraphBackupOptions{
					GraphName: env.GraphName,
					Format:    "json",
					Compress:  false,
				}

				err := env.Store.Backup(ctx, &buf, opts)
				if err != nil {
					t.Fatalf("backup failed: %v", err)
				}

				if buf.Len() == 0 {
					t.Error("backup data is empty")
				}

				// Verify backup data structure
				var backupData BackupData
				err = json.Unmarshal(buf.Bytes(), &backupData)
				if err != nil {
					t.Fatalf("failed to parse backup JSON: %v", err)
				}

				if backupData.Format != "json" {
					t.Errorf("expected format 'json', got '%s'", backupData.Format)
				}
				if backupData.GraphName != env.GraphName {
					t.Errorf("expected graph name '%s', got '%s'", env.GraphName, backupData.GraphName)
				}
				if len(backupData.Nodes) == 0 {
					t.Error("no nodes in backup data")
				}
				if len(backupData.Relationships) == 0 {
					t.Error("no relationships in backup data")
				}
			})

			// Test JSON format backup with compression
			t.Run("JSON_WithCompression", func(t *testing.T) {
				var buf bytes.Buffer
				opts := &types.GraphBackupOptions{
					GraphName: env.GraphName,
					Format:    "json",
					Compress:  true,
				}

				err := env.Store.Backup(ctx, &buf, opts)
				if err != nil {
					t.Fatalf("backup failed: %v", err)
				}

				if buf.Len() == 0 {
					t.Error("backup data is empty")
				}

				// Verify it's compressed
				data := buf.Bytes()
				if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
					t.Error("backup data is not gzipped")
				}

				// Decompress and verify
				reader := bytes.NewReader(data)
				gzReader, err := gzip.NewReader(reader)
				if err != nil {
					t.Fatalf("failed to create gzip reader: %v", err)
				}
				defer gzReader.Close()

				decompressed, err := io.ReadAll(gzReader)
				if err != nil {
					t.Fatalf("failed to decompress: %v", err)
				}

				var backupData BackupData
				err = json.Unmarshal(decompressed, &backupData)
				if err != nil {
					t.Fatalf("failed to parse decompressed JSON: %v", err)
				}

				if backupData.Format != "json" {
					t.Errorf("expected format 'json', got '%s'", backupData.Format)
				}
			})

			// Test Cypher format backup
			t.Run("Cypher_Format", func(t *testing.T) {
				var buf bytes.Buffer
				opts := &types.GraphBackupOptions{
					GraphName: env.GraphName,
					Format:    "cypher",
					Compress:  false,
				}

				err := env.Store.Backup(ctx, &buf, opts)
				if err != nil {
					t.Fatalf("backup failed: %v", err)
				}

				if buf.Len() == 0 {
					t.Error("backup data is empty")
				}

				cypherScript := buf.String()
				if !strings.Contains(cypherScript, "CREATE (n") {
					t.Error("cypher script should contain CREATE statements")
				}
				if !strings.Contains(cypherScript, "Neo4j Graph Backup") {
					t.Error("cypher script should contain header comment")
				}
			})

			// Check for memory leaks and goroutine leaks
			finalGoroutines := captureGoroutineState()
			finalMemory := captureMemoryStats()

			leaked, _ := analyzeGoroutineChanges(initialGoroutines, finalGoroutines)
			memoryGrowth := calculateMemoryGrowth(initialMemory, finalMemory)

			// Filter out system goroutines
			var appLeaked []GoroutineInfo
			for _, g := range leaked {
				if !g.IsSystem {
					appLeaked = append(appLeaked, g)
				}
			}

			if len(appLeaked) > 2 { // Allow some tolerance
				t.Errorf("detected %d potentially leaked goroutines", len(appLeaked))
				for _, g := range appLeaked {
					t.Logf("Leaked goroutine: ID=%d, State=%s, Function=%s", g.ID, g.State, g.Function)
				}
			}

			// Memory growth check (allow 10MB tolerance)
			if memoryGrowth.HeapAllocGrowth > 10*1024*1024 {
				t.Errorf("significant memory growth detected: %d bytes", memoryGrowth.HeapAllocGrowth)
			}
		})
	}
}

// TestRestore_Basic tests basic restore functionality
func TestRestore_Basic(t *testing.T) {
	tests := []struct {
		name                string
		useSeparateDatabase bool
	}{
		{"LabelBased", false},
		{"SeparateDatabase", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnvironment(t, tt.useSeparateDatabase)
			defer cleanupTestEnvironment(env, t)

			setupTestGraph(t, env)

			ctx := context.Background()

			// Create backup first
			var backupBuf bytes.Buffer
			backupOpts := &types.GraphBackupOptions{
				GraphName: env.GraphName,
				Format:    "json",
				Compress:  false,
			}

			err := env.Store.Backup(ctx, &backupBuf, backupOpts)
			if err != nil {
				t.Fatalf("backup failed: %v", err)
			}

			// Test restore to new graph
			var newGraphName string
			if tt.useSeparateDatabase {
				timestamp := time.Now().UnixNano() % 1000000
				newGraphName = fmt.Sprintf("testdb%d", timestamp+1)
			} else {
				newGraphName = env.GraphName + "_restored"
			}

			t.Run("JSON_Restore", func(t *testing.T) {
				restoreOpts := &types.GraphRestoreOptions{
					GraphName:   newGraphName,
					Format:      "json",
					CreateGraph: true,
					Force:       false,
				}

				err := env.Store.Restore(ctx, bytes.NewReader(backupBuf.Bytes()), restoreOpts)
				if err != nil {
					// Skip test if database limit is reached
					if strings.Contains(err.Error(), "DatabaseLimitReached") {
						t.Skip("Database limit reached, skipping test")
					}
					t.Fatalf("restore failed: %v", err)
				}

				// Verify restored graph exists
				exists, err := env.Store.GraphExists(ctx, newGraphName)
				if err != nil {
					t.Fatalf("failed to check graph existence: %v", err)
				}
				if !exists {
					t.Error("restored graph does not exist")
				}

				// Verify graph has data
				stats, err := env.Store.DescribeGraph(ctx, newGraphName)
				if err != nil {
					t.Fatalf("failed to get graph stats: %v", err)
				}
				if stats.TotalNodes == 0 {
					t.Error("restored graph has no nodes")
				}
				if stats.TotalRelationships == 0 {
					t.Error("restored graph has no relationships")
				}

				// Cleanup restored graph
				env.Store.DropGraph(ctx, newGraphName)
			})

			// Test compressed restore
			t.Run("Compressed_Restore", func(t *testing.T) {
				// Create compressed backup
				var compressedBuf bytes.Buffer
				compressedOpts := &types.GraphBackupOptions{
					GraphName: env.GraphName,
					Format:    "json",
					Compress:  true,
				}

				err := env.Store.Backup(ctx, &compressedBuf, compressedOpts)
				if err != nil {
					t.Fatalf("compressed backup failed: %v", err)
				}

				var compressedGraphName string
				if tt.useSeparateDatabase {
					timestamp := time.Now().UnixNano() % 1000000
					compressedGraphName = fmt.Sprintf("testdb%d", timestamp+2)
				} else {
					compressedGraphName = env.GraphName + "_compressed"
				}
				restoreOpts := &types.GraphRestoreOptions{
					GraphName:   compressedGraphName,
					Format:      "json",
					CreateGraph: true,
					Force:       false,
				}

				err = env.Store.Restore(ctx, bytes.NewReader(compressedBuf.Bytes()), restoreOpts)
				if err != nil {
					// Skip test if database limit is reached
					if strings.Contains(err.Error(), "DatabaseLimitReached") {
						t.Skip("Database limit reached, skipping test")
					}
					t.Fatalf("compressed restore failed: %v", err)
				}

				// Verify restored graph
				exists, err := env.Store.GraphExists(ctx, compressedGraphName)
				if err != nil {
					t.Fatalf("failed to check compressed graph existence: %v", err)
				}
				if !exists {
					t.Error("compressed restored graph does not exist")
				}

				// Cleanup
				env.Store.DropGraph(ctx, compressedGraphName)
			})

			// Test Cypher restore
			t.Run("Cypher_Restore", func(t *testing.T) {
				// Create Cypher backup
				var cypherBuf bytes.Buffer
				cypherOpts := &types.GraphBackupOptions{
					GraphName: env.GraphName,
					Format:    "cypher",
					Compress:  false,
				}

				err := env.Store.Backup(ctx, &cypherBuf, cypherOpts)
				if err != nil {
					t.Fatalf("cypher backup failed: %v", err)
				}

				var cypherGraphName string
				if tt.useSeparateDatabase {
					timestamp := time.Now().UnixNano() % 1000000
					cypherGraphName = fmt.Sprintf("testdb%d", timestamp+3)
				} else {
					cypherGraphName = env.GraphName + "_cypher"
				}
				restoreOpts := &types.GraphRestoreOptions{
					GraphName:   cypherGraphName,
					Format:      "cypher",
					CreateGraph: true,
					Force:       false,
				}

				err = env.Store.Restore(ctx, bytes.NewReader(cypherBuf.Bytes()), restoreOpts)
				if err != nil {
					// Skip test if database limit is reached
					if strings.Contains(err.Error(), "DatabaseLimitReached") {
						t.Skip("Database limit reached, skipping test")
					}
					t.Fatalf("cypher restore failed: %v", err)
				}

				// Verify restored graph
				exists, err := env.Store.GraphExists(ctx, cypherGraphName)
				if err != nil {
					t.Fatalf("failed to check cypher graph existence: %v", err)
				}
				if !exists {
					t.Error("cypher restored graph does not exist")
				}

				// Cleanup
				env.Store.DropGraph(ctx, cypherGraphName)
			})
		})
	}
}

// TestBackup_ErrorConditions tests error handling
func TestBackup_ErrorConditions(t *testing.T) {
	env := setupTestEnvironment(t, false)
	defer cleanupTestEnvironment(env, t)

	setupTestGraph(t, env)

	ctx := context.Background()

	tests := []struct {
		name          string
		store         *Store
		writer        io.Writer
		opts          *types.GraphBackupOptions
		expectError   bool
		errorContains string
	}{
		{
			name:          "not_connected",
			store:         &Store{connected: false},
			writer:        &bytes.Buffer{},
			opts:          &types.GraphBackupOptions{GraphName: "test"},
			expectError:   true,
			errorContains: "not connected",
		},
		{
			name:          "nil_options",
			store:         env.Store,
			writer:        &bytes.Buffer{},
			opts:          nil,
			expectError:   true,
			errorContains: "backup options cannot be nil",
		},
		{
			name:          "empty_graph_name",
			store:         env.Store,
			writer:        &bytes.Buffer{},
			opts:          &types.GraphBackupOptions{GraphName: ""},
			expectError:   true,
			errorContains: "graph name cannot be empty",
		},
		{
			name:          "nil_writer",
			store:         env.Store,
			writer:        nil,
			opts:          &types.GraphBackupOptions{GraphName: "test"},
			expectError:   true,
			errorContains: "writer cannot be nil",
		},
		{
			name:          "non_existent_graph",
			store:         env.Store,
			writer:        &bytes.Buffer{},
			opts:          &types.GraphBackupOptions{GraphName: "nonexistentgraph"},
			expectError:   true,
			errorContains: "does not exist",
		},
		{
			name:          "unsupported_format",
			store:         env.Store,
			writer:        &bytes.Buffer{},
			opts:          &types.GraphBackupOptions{GraphName: env.GraphName, Format: "xml"},
			expectError:   true,
			errorContains: "unsupported backup format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.store.Backup(ctx, tt.writer, tt.opts)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestRestore_ErrorConditions tests restore error handling
func TestRestore_ErrorConditions(t *testing.T) {
	env := setupTestEnvironment(t, false)
	defer cleanupTestEnvironment(env, t)

	setupTestGraph(t, env)

	ctx := context.Background()

	// Create valid backup data for tests
	var validBackup bytes.Buffer
	backupOpts := &types.GraphBackupOptions{
		GraphName: env.GraphName,
		Format:    "json",
		Compress:  false,
	}
	err := env.Store.Backup(ctx, &validBackup, backupOpts)
	if err != nil {
		t.Fatalf("failed to create test backup: %v", err)
	}

	tests := []struct {
		name          string
		store         *Store
		reader        io.Reader
		opts          *types.GraphRestoreOptions
		expectError   bool
		errorContains string
	}{
		{
			name:          "not_connected",
			store:         &Store{connected: false},
			reader:        bytes.NewReader(validBackup.Bytes()),
			opts:          &types.GraphRestoreOptions{GraphName: "test"},
			expectError:   true,
			errorContains: "not connected",
		},
		{
			name:          "nil_options",
			store:         env.Store,
			reader:        bytes.NewReader(validBackup.Bytes()),
			opts:          nil,
			expectError:   true,
			errorContains: "restore options cannot be nil",
		},
		{
			name:          "empty_graph_name",
			store:         env.Store,
			reader:        bytes.NewReader(validBackup.Bytes()),
			opts:          &types.GraphRestoreOptions{GraphName: ""},
			expectError:   true,
			errorContains: "graph name cannot be empty",
		},
		{
			name:          "nil_reader",
			store:         env.Store,
			reader:        nil,
			opts:          &types.GraphRestoreOptions{GraphName: "test"},
			expectError:   true,
			errorContains: "reader cannot be nil",
		},
		{
			name:          "graph_exists_no_force",
			store:         env.Store,
			reader:        bytes.NewReader(validBackup.Bytes()),
			opts:          &types.GraphRestoreOptions{GraphName: env.GraphName, Force: false},
			expectError:   true,
			errorContains: "already exists",
		},
		{
			name:          "empty_data",
			store:         env.Store,
			reader:        bytes.NewReader([]byte{}),
			opts:          &types.GraphRestoreOptions{GraphName: "testempty"},
			expectError:   true,
			errorContains: "empty data",
		},
		{
			name:          "invalid_json",
			store:         env.Store,
			reader:        bytes.NewReader([]byte("invalid json")),
			opts:          &types.GraphRestoreOptions{GraphName: "testinvalid", CreateGraph: true},
			expectError:   true,
			errorContains: "failed to parse JSON",
		},
		{
			name:          "unsupported_format",
			store:         env.Store,
			reader:        bytes.NewReader(validBackup.Bytes()),
			opts:          &types.GraphRestoreOptions{GraphName: "testformat", Format: "xml", CreateGraph: true},
			expectError:   true,
			errorContains: "unsupported restore format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.store.Restore(ctx, tt.reader, tt.opts)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestBackupRestore_StressTest tests concurrent backup/restore operations
func TestBackupRestore_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	tests := []struct {
		name                string
		useSeparateDatabase bool
		config              StressTestConfig
	}{
		{"LabelBased_Light", false, LightStressConfig()},
		{"SeparateDatabase_Light", true, SeparateDBStressConfig()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture initial state
			initialGoroutines := captureGoroutineState()
			initialMemory := captureMemoryStats()

			env := setupTestEnvironment(t, tt.useSeparateDatabase)
			defer cleanupTestEnvironment(env, t)

			setupTestGraph(t, env)

			ctx := context.Background()

			// Create test backup data
			var backupBuf bytes.Buffer
			backupOpts := &types.GraphBackupOptions{
				GraphName: env.GraphName,
				Format:    "json",
				Compress:  false,
			}
			err := env.Store.Backup(ctx, &backupBuf, backupOpts)
			if err != nil {
				t.Fatalf("failed to create test backup: %v", err)
			}

			// Test concurrent backup operations
			t.Run("ConcurrentBackups", func(t *testing.T) {
				operation := func(ctx context.Context) error {
					var buf bytes.Buffer
					opts := &types.GraphBackupOptions{
						GraphName: env.GraphName,
						Format:    "json",
						Compress:  false,
					}
					return env.Store.Backup(ctx, &buf, opts)
				}

				result := runStressTest(tt.config, operation)

				t.Logf("Backup stress test: %d operations, %.2f%% success rate, %d errors in %v",
					result.TotalOperations, result.SuccessRate, result.ErrorCount, result.Duration)

				if result.SuccessRate < tt.config.MinSuccessRate {
					t.Errorf("success rate too low: %.2f%% < %.2f%%", result.SuccessRate, tt.config.MinSuccessRate)
				}
			})

			// Test concurrent restore operations
			t.Run("ConcurrentRestores", func(t *testing.T) {
				// Pre-generate all graph names to avoid lock contention
				totalOps := tt.config.NumWorkers * tt.config.OperationsPerWorker
				graphNames := make([]string, totalOps)
				for i := 0; i < totalOps; i++ {
					timestamp := time.Now().UnixNano() % 1000000
					if env.Store.useSeparateDatabase {
						graphNames[i] = fmt.Sprintf("testdb%d", timestamp+int64(i)+100)
					} else {
						graphNames[i] = fmt.Sprintf("%s_stress_%d", env.GraphName, i)
					}
				}

				nameCounter := int64(0)
				operation := func(ctx context.Context) error {
					// Get next graph name atomically
					idx := atomic.AddInt64(&nameCounter, 1) - 1
					if idx >= int64(len(graphNames)) {
						return fmt.Errorf("graph name index out of range")
					}
					graphName := graphNames[idx]

					defer func() {
						// Cleanup restored graph
						if exists, _ := env.Store.GraphExists(ctx, graphName); exists {
							env.Store.DropGraph(ctx, graphName)
						}
					}()

					opts := &types.GraphRestoreOptions{
						GraphName:   graphName,
						Format:      "json",
						CreateGraph: true,
						Force:       false,
					}
					err := env.Store.Restore(ctx, bytes.NewReader(backupBuf.Bytes()), opts)

					// Skip database limit errors in stress tests
					if err != nil && strings.Contains(err.Error(), "DatabaseLimitReached") {
						return nil // Don't count as error
					}
					return err
				}

				result := runStressTest(tt.config, operation)

				t.Logf("Restore stress test: %d operations, %.2f%% success rate, %d errors in %v",
					result.TotalOperations, result.SuccessRate, result.ErrorCount, result.Duration)

				// Use more lenient success rate for separate database mode due to limits
				minSuccessRate := tt.config.MinSuccessRate
				if tt.useSeparateDatabase {
					minSuccessRate = 30.0 // Lower threshold for separate database mode
				}

				if result.SuccessRate < minSuccessRate {
					t.Errorf("success rate too low: %.2f%% < %.2f%%", result.SuccessRate, minSuccessRate)
				}
			})

			// Check for resource leaks after stress test
			finalGoroutines := captureGoroutineState()
			finalMemory := captureMemoryStats()

			leaked, _ := analyzeGoroutineChanges(initialGoroutines, finalGoroutines)
			memoryGrowth := calculateMemoryGrowth(initialMemory, finalMemory)

			// Filter out system goroutines
			var appLeaked []GoroutineInfo
			for _, g := range leaked {
				if !g.IsSystem {
					appLeaked = append(appLeaked, g)
				}
			}

			if len(appLeaked) > 5 { // Allow some tolerance for stress tests
				t.Errorf("detected %d potentially leaked goroutines after stress test", len(appLeaked))
				for i, g := range appLeaked {
					if i < 3 { // Log first few for debugging
						t.Logf("Leaked goroutine: ID=%d, State=%s, Function=%s", g.ID, g.State, g.Function)
					}
				}
			}

			// Memory growth check (allow 50MB tolerance for stress tests)
			if memoryGrowth.HeapAllocGrowth > 50*1024*1024 {
				t.Errorf("significant memory growth detected after stress test: %d bytes", memoryGrowth.HeapAllocGrowth)
			}
		})
	}
}

// TestBackupRestore_ForceOverwrite tests force overwrite functionality
func TestBackupRestore_ForceOverwrite(t *testing.T) {
	env := setupTestEnvironment(t, false)
	defer cleanupTestEnvironment(env, t)

	setupTestGraph(t, env)

	ctx := context.Background()

	// Create backup
	var backupBuf bytes.Buffer
	backupOpts := &types.GraphBackupOptions{
		GraphName: env.GraphName,
		Format:    "json",
		Compress:  false,
	}
	err := env.Store.Backup(ctx, &backupBuf, backupOpts)
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	// Test force overwrite
	restoreOpts := &types.GraphRestoreOptions{
		GraphName: env.GraphName,
		Format:    "json",
		Force:     true,
	}

	err = env.Store.Restore(ctx, bytes.NewReader(backupBuf.Bytes()), restoreOpts)
	if err != nil {
		t.Fatalf("force restore failed: %v", err)
	}

	// Verify graph still exists
	exists, err := env.Store.GraphExists(ctx, env.GraphName)
	if err != nil {
		t.Fatalf("failed to check graph existence: %v", err)
	}
	if !exists {
		t.Error("graph should exist after force restore")
	}
}

// TestBackupRestore_WithFilters tests backup with filters
func TestBackupRestore_WithFilters(t *testing.T) {
	env := setupTestEnvironment(t, false)
	defer cleanupTestEnvironment(env, t)

	setupTestGraph(t, env)

	ctx := context.Background()

	// Test backup with node filter
	var buf bytes.Buffer
	opts := &types.GraphBackupOptions{
		GraphName: env.GraphName,
		Format:    "json",
		Compress:  false,
		Filter: map[string]interface{}{
			"nodes": "n.name CONTAINS 'Node 0'",
		},
	}

	err := env.Store.Backup(ctx, &buf, opts)
	if err != nil {
		t.Fatalf("filtered backup failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("filtered backup data is empty")
	}

	// Verify backup data
	var backupData BackupData
	err = json.Unmarshal(buf.Bytes(), &backupData)
	if err != nil {
		t.Fatalf("failed to parse filtered backup JSON: %v", err)
	}

	// Should have fewer nodes due to filter
	if len(backupData.Nodes) >= len(env.TestNodes) {
		t.Error("filter should have reduced the number of backed up nodes")
	}
}

// TestBackupRestore_ExtraParams tests backup/restore with extra parameters
func TestBackupRestore_ExtraParams(t *testing.T) {
	env := setupTestEnvironment(t, false)
	defer cleanupTestEnvironment(env, t)

	setupTestGraph(t, env)

	ctx := context.Background()

	// Test backup with extra params
	var buf bytes.Buffer
	opts := &types.GraphBackupOptions{
		GraphName: env.GraphName,
		Format:    "json",
		Compress:  false,
		ExtraParams: map[string]interface{}{
			"version":    "1.0",
			"created_by": "test",
			"custom":     true,
		},
	}

	err := env.Store.Backup(ctx, &buf, opts)
	if err != nil {
		t.Fatalf("backup with extra params failed: %v", err)
	}

	// Verify extra params are in metadata
	var backupData BackupData
	err = json.Unmarshal(buf.Bytes(), &backupData)
	if err != nil {
		t.Fatalf("failed to parse backup JSON: %v", err)
	}

	if backupData.Metadata["version"] != "1.0" {
		t.Error("extra param 'version' not found in metadata")
	}
	if backupData.Metadata["created_by"] != "test" {
		t.Error("extra param 'created_by' not found in metadata")
	}
	if backupData.Metadata["custom"] != true {
		t.Error("extra param 'custom' not found in metadata")
	}
}

// TestBackupHelperFunctions tests individual backup-related helper functions
func TestBackupHelperFunctions(t *testing.T) {
	env := setupTestEnvironment(t, false)
	defer cleanupTestEnvironment(env, t)

	// Test getStorageType
	t.Run("getStorageType", func(t *testing.T) {
		env.Store.SetUseSeparateDatabase(false)
		if env.Store.getStorageType() != "label_based" {
			t.Error("expected 'label_based' for non-separate database")
		}

		env.Store.SetUseSeparateDatabase(true)
		if env.Store.getStorageType() != "separate_database" {
			t.Error("expected 'separate_database' for separate database")
		}
	})

	// Test singleByteReader
	t.Run("singleByteReader", func(t *testing.T) {
		data := []byte("test data")
		reader := &singleByteReader{data: data}

		buf := make([]byte, 4)
		n, err := reader.Read(buf)
		if err != nil {
			t.Fatalf("first read failed: %v", err)
		}
		if n != 4 {
			t.Errorf("expected 4 bytes read, got %d", n)
		}
		if string(buf) != "test" {
			t.Errorf("expected 'test', got '%s'", string(buf))
		}

		// Read remaining
		buf2 := make([]byte, 10)
		n, err = reader.Read(buf2)
		if n != 5 {
			t.Errorf("expected 5 bytes read, got %d", n)
		}
		if err != io.EOF {
			t.Errorf("expected EOF, got %v", err)
		}
		if string(buf2[:n]) != " data" {
			t.Errorf("expected ' data', got '%s'", string(buf2[:n]))
		}

		// Read past end
		n, err = reader.Read(buf2)
		if n != 0 {
			t.Errorf("expected 0 bytes read, got %d", n)
		}
		if err != io.EOF {
			t.Errorf("expected EOF, got %v", err)
		}
	})
}

// TestContextCancellation tests context cancellation during backup/restore
func TestContextCancellation(t *testing.T) {
	env := setupTestEnvironment(t, false)
	defer cleanupTestEnvironment(env, t)

	setupTestGraph(t, env)

	t.Run("BackupCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		var buf bytes.Buffer
		opts := &types.GraphBackupOptions{
			GraphName: env.GraphName,
			Format:    "json",
			Compress:  false,
		}

		err := env.Store.Backup(ctx, &buf, opts)
		if err == nil {
			t.Error("expected error due to cancelled context")
		}
	})

	t.Run("RestoreCancellation", func(t *testing.T) {
		// First create a backup
		var backupBuf bytes.Buffer
		backupOpts := &types.GraphBackupOptions{
			GraphName: env.GraphName,
			Format:    "json",
			Compress:  false,
		}
		err := env.Store.Backup(context.Background(), &backupBuf, backupOpts)
		if err != nil {
			t.Fatalf("failed to create backup: %v", err)
		}

		// Now try restore with cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		opts := &types.GraphRestoreOptions{
			GraphName: func() string {
				if env.Store.useSeparateDatabase {
					return env.GraphName + "-cancelled"
				}
				return env.GraphName + "_cancelled"
			}(),
			Format:      "json",
			CreateGraph: true,
		}

		err = env.Store.Restore(ctx, bytes.NewReader(backupBuf.Bytes()), opts)
		if err == nil {
			t.Error("expected error due to cancelled context")
		}
	})
}
