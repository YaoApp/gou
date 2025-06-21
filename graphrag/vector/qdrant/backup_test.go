package qdrant

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

func TestBackup(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	collectionName := fmt.Sprintf("test_backup_collection_%d", time.Now().UnixNano())

	// Create test collection and add documents
	setupTestCollection(t, store, collectionName)

	tests := []struct {
		name        string
		opts        *types.BackupOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful backup without compression",
			opts: &types.BackupOptions{
				CollectionName: collectionName,
				Compress:       false,
			},
			expectError: false,
		},
		{
			name: "successful backup with compression",
			opts: &types.BackupOptions{
				CollectionName: collectionName,
				Compress:       true,
			},
			expectError: false,
		},
		{
			name:        "nil options",
			opts:        nil,
			expectError: true,
			errorMsg:    "backup options cannot be nil",
		},
		{
			name: "empty collection name",
			opts: &types.BackupOptions{
				CollectionName: "",
				Compress:       false,
			},
			expectError: true,
			errorMsg:    "collection name cannot be empty",
		},
		{
			name: "non-existent collection",
			opts: &types.BackupOptions{
				CollectionName: "non_existent_collection",
				Compress:       false,
			},
			expectError: true,
			errorMsg:    "does not exist",
		},
		{
			name: "backup with extra params",
			opts: &types.BackupOptions{
				CollectionName: collectionName,
				Compress:       false,
				ExtraParams: map[string]interface{}{
					"timeout": 30,
					"retry":   3,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := store.Backup(ctx, &buf, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify backup data
			if buf.Len() == 0 {
				t.Error("backup data is empty")
			}

			// If compressed, verify it's actually gzipped
			if tt.opts.Compress {
				reader := bytes.NewReader(buf.Bytes())
				gzReader, err := gzip.NewReader(reader)
				if err != nil {
					t.Fatalf("failed to create gzip reader: %v", err)
				}
				defer gzReader.Close()

				content, err := io.ReadAll(gzReader)
				if err != nil {
					t.Fatalf("failed to read compressed content: %v", err)
				}

				if len(content) == 0 {
					t.Error("decompressed content is empty")
				}
			}
		})
	}
}

func TestBackupNilWriter(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	opts := &types.BackupOptions{
		CollectionName: "test_collection",
	}

	err := store.Backup(ctx, nil, opts)
	if err == nil || !strings.Contains(err.Error(), "writer cannot be nil") {
		t.Errorf("expected 'writer cannot be nil' error, got: %v", err)
	}
}

func TestRestore(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	sourceCollection := fmt.Sprintf("source_collection_%d", time.Now().UnixNano())
	targetCollection := fmt.Sprintf("target_collection_%d", time.Now().UnixNano())

	// Create test collection and backup data
	setupTestCollection(t, store, sourceCollection)

	var backupBuf bytes.Buffer
	backupOpts := &types.BackupOptions{
		CollectionName: sourceCollection,
		Compress:       false,
	}
	err := store.Backup(ctx, &backupBuf, backupOpts)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	tests := []struct {
		name        string
		reader      io.Reader
		opts        *types.RestoreOptions
		expectError bool
		errorMsg    string
	}{
		{
			name:   "restore from uncompressed backup",
			reader: bytes.NewReader(backupBuf.Bytes()),
			opts: &types.RestoreOptions{
				CollectionName: targetCollection,
				Force:          false,
			},
			expectError: true, // Because snapshot restore is not fully implemented
			errorMsg:    "not fully implemented",
		},
		{
			name:        "nil options",
			reader:      bytes.NewReader(backupBuf.Bytes()),
			opts:        nil,
			expectError: true,
			errorMsg:    "restore options cannot be nil",
		},
		{
			name:   "empty collection name",
			reader: bytes.NewReader(backupBuf.Bytes()),
			opts: &types.RestoreOptions{
				CollectionName: "",
			},
			expectError: true,
			errorMsg:    "collection name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Restore(ctx, tt.reader, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRestoreNilReader(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	opts := &types.RestoreOptions{
		CollectionName: "test_collection",
	}

	err := store.Restore(ctx, nil, opts)
	if err == nil || !strings.Contains(err.Error(), "reader cannot be nil") {
		t.Errorf("expected 'reader cannot be nil' error, got: %v", err)
	}
}

func TestBackupRestore_Compressed(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	sourceCollection := fmt.Sprintf("source_compressed_%d", time.Now().UnixNano())

	// Create test collection
	setupTestCollection(t, store, sourceCollection)

	// Test compressed backup
	var compressedBuf bytes.Buffer
	backupOpts := &types.BackupOptions{
		CollectionName: sourceCollection,
		Compress:       true,
	}
	err := store.Backup(ctx, &compressedBuf, backupOpts)
	if err != nil {
		t.Fatalf("failed to create compressed backup: %v", err)
	}

	// Verify compressed data
	if compressedBuf.Len() == 0 {
		t.Fatal("compressed backup is empty")
	}

	// Verify it's actually gzipped
	reader := bytes.NewReader(compressedBuf.Bytes())
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	// Read compressed content
	content, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("failed to read compressed content: %v", err)
	}

	if len(content) == 0 {
		t.Error("decompressed content is empty")
	}
}

func TestBackup_NotConnected(t *testing.T) {
	store := &Store{
		connected: false,
	}

	ctx := context.Background()
	opts := &types.BackupOptions{
		CollectionName: "test",
	}

	var buf bytes.Buffer
	err := store.Backup(ctx, &buf, opts)
	if err == nil || !strings.Contains(err.Error(), "not connected") {
		t.Errorf("expected 'not connected' error, got: %v", err)
	}
}

func TestRestore_NotConnected(t *testing.T) {
	store := &Store{
		connected: false,
	}

	ctx := context.Background()
	opts := &types.RestoreOptions{
		CollectionName: "test",
	}

	reader := bytes.NewReader([]byte("test data"))
	err := store.Restore(ctx, reader, opts)
	if err == nil || !strings.Contains(err.Error(), "not connected") {
		t.Errorf("expected 'not connected' error, got: %v", err)
	}
}

// Concurrent backup test
func TestBackup_Concurrent(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	collectionName := fmt.Sprintf("concurrent_backup_test_%d", time.Now().UnixNano())

	// Create test collection
	setupTestCollection(t, store, collectionName)

	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make(chan error, numGoroutines)

	opts := &types.BackupOptions{
		CollectionName: collectionName,
		Compress:       false,
	}

	// Run concurrent backups
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			var buf bytes.Buffer
			err := store.Backup(ctx, &buf, opts)
			if err != nil {
				results <- fmt.Errorf("goroutine %d failed: %w", id, err)
				return
			}
			if buf.Len() == 0 {
				results <- fmt.Errorf("goroutine %d produced empty backup", id)
				return
			}
			results <- nil
		}(i)
	}

	wg.Wait()
	close(results)

	// Check for errors
	for err := range results {
		if err != nil {
			t.Error(err)
		}
	}
}

// Memory leak test
func TestBackup_MemoryLeak(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	collectionName := fmt.Sprintf("memory_leak_test_%d", time.Now().UnixNano())

	// Create test collection
	setupTestCollection(t, store, collectionName)

	opts := &types.BackupOptions{
		CollectionName: collectionName,
		Compress:       false,
	}

	// Get initial memory stats
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.GC() // Double GC to be more reliable
	runtime.ReadMemStats(&m1)

	// Run multiple backups
	const iterations = 100
	for i := 0; i < iterations; i++ {
		var buf bytes.Buffer
		err := store.Backup(ctx, &buf, opts)
		if err != nil {
			t.Fatalf("backup iteration %d failed: %v", i, err)
		}
		// Don't accumulate the data
		buf.Reset()

		// Periodic GC to help detect real leaks
		if i%20 == 0 {
			runtime.GC()
		}
	}

	// Force garbage collection and check memory
	runtime.GC()
	runtime.GC() // Double GC to be more reliable
	runtime.ReadMemStats(&m2)

	// Calculate memory change more safely
	if m2.Alloc > m1.Alloc {
		memIncrease := float64(m2.Alloc-m1.Alloc) / float64(m1.Alloc)
		if memIncrease > 0.5 { // Allow 50% increase
			t.Errorf("potential memory leak detected: memory increased by %.2f%% (from %d to %d bytes)",
				memIncrease*100, m1.Alloc, m2.Alloc)
		}
	} else {
		// Memory decreased or stayed the same - this is good
		t.Logf("Memory usage decreased or stayed stable: from %d to %d bytes", m1.Alloc, m2.Alloc)
	}
}

// Stress test with large data
func TestBackup_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	collectionName := fmt.Sprintf("stress_test_collection_%d", time.Now().UnixNano())

	// Create large test collection
	setupLargeTestCollection(t, store, collectionName, 1000)

	// Test with and without compression
	tests := []struct {
		name     string
		compress bool
	}{
		{"uncompressed", false},
		{"compressed", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			var buf bytes.Buffer

			opts := &types.BackupOptions{
				CollectionName: collectionName,
				Compress:       tt.compress,
			}

			err := store.Backup(ctx, &buf, opts)
			if err != nil {
				t.Fatalf("stress test backup failed: %v", err)
			}

			duration := time.Since(start)
			t.Logf("Backup completed in %v, data size: %d bytes", duration, buf.Len())

			if buf.Len() == 0 {
				t.Error("stress test produced empty backup")
			}
		})
	}
}

// Context cancellation test
func TestBackup_ContextCancellation(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	collectionName := fmt.Sprintf("context_cancel_test_%d", time.Now().UnixNano())
	setupTestCollection(t, store, collectionName)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	opts := &types.BackupOptions{
		CollectionName: collectionName,
		Compress:       false,
	}

	start := time.Now()
	err := store.Backup(ctx, &buf, opts)
	duration := time.Since(start)

	// Should fail quickly due to cancelled context
	if err == nil {
		t.Error("expected error due to cancelled context")
	}

	if duration > time.Second {
		t.Errorf("backup took too long with cancelled context: %v", duration)
	}

	if !strings.Contains(err.Error(), "context") {
		t.Errorf("expected context-related error, got: %v", err)
	}
}

// Test all error conditions and edge cases for Backup function
func TestBackup_ErrorConditions(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	collectionName := fmt.Sprintf("test_backup_error_%d", time.Now().UnixNano())

	// Create test collection
	setupTestCollection(t, store, collectionName)

	tests := []struct {
		name          string
		store         *Store
		ctx           context.Context
		writer        io.Writer
		opts          *types.BackupOptions
		expectError   bool
		errorContains string
	}{
		{
			name:          "not connected store",
			store:         &Store{connected: false},
			ctx:           ctx,
			writer:        &bytes.Buffer{},
			opts:          &types.BackupOptions{CollectionName: collectionName},
			expectError:   true,
			errorContains: "not connected",
		},
		{
			name:          "nil options",
			store:         store,
			ctx:           ctx,
			writer:        &bytes.Buffer{},
			opts:          nil,
			expectError:   true,
			errorContains: "backup options cannot be nil",
		},
		{
			name:          "empty collection name",
			store:         store,
			ctx:           ctx,
			writer:        &bytes.Buffer{},
			opts:          &types.BackupOptions{CollectionName: ""},
			expectError:   true,
			errorContains: "collection name cannot be empty",
		},
		{
			name:          "nil writer",
			store:         store,
			ctx:           ctx,
			writer:        nil,
			opts:          &types.BackupOptions{CollectionName: collectionName},
			expectError:   true,
			errorContains: "writer cannot be nil",
		},
		{
			name:          "non-existent collection",
			store:         store,
			ctx:           ctx,
			writer:        &bytes.Buffer{},
			opts:          &types.BackupOptions{CollectionName: "non_existent_collection_12345"},
			expectError:   true,
			errorContains: "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.store.Backup(tt.ctx, tt.writer, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
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

// Test all error conditions and edge cases for Restore function
func TestRestore_ErrorConditions(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	sourceCollection := fmt.Sprintf("source_error_%d", time.Now().UnixNano())
	targetCollection := fmt.Sprintf("target_error_%d", time.Now().UnixNano())

	// Create source collection and backup data
	setupTestCollection(t, store, sourceCollection)

	var backupBuf bytes.Buffer
	backupOpts := &types.BackupOptions{
		CollectionName: sourceCollection,
		Compress:       false,
	}
	err := store.Backup(ctx, &backupBuf, backupOpts)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Create target collection for force test
	setupTestCollection(t, store, targetCollection)

	tests := []struct {
		name          string
		store         *Store
		ctx           context.Context
		reader        io.Reader
		opts          *types.RestoreOptions
		expectError   bool
		errorContains string
	}{
		{
			name:          "not connected store",
			store:         &Store{connected: false},
			ctx:           ctx,
			reader:        bytes.NewReader(backupBuf.Bytes()),
			opts:          &types.RestoreOptions{CollectionName: "test_restore"},
			expectError:   true,
			errorContains: "not connected",
		},
		{
			name:          "nil options",
			store:         store,
			ctx:           ctx,
			reader:        bytes.NewReader(backupBuf.Bytes()),
			opts:          nil,
			expectError:   true,
			errorContains: "restore options cannot be nil",
		},
		{
			name:          "empty collection name",
			store:         store,
			ctx:           ctx,
			reader:        bytes.NewReader(backupBuf.Bytes()),
			opts:          &types.RestoreOptions{CollectionName: ""},
			expectError:   true,
			errorContains: "collection name cannot be empty",
		},
		{
			name:          "nil reader",
			store:         store,
			ctx:           ctx,
			reader:        nil,
			opts:          &types.RestoreOptions{CollectionName: "test_restore"},
			expectError:   true,
			errorContains: "reader cannot be nil",
		},
		{
			name:          "collection exists without force",
			store:         store,
			ctx:           ctx,
			reader:        bytes.NewReader(backupBuf.Bytes()),
			opts:          &types.RestoreOptions{CollectionName: targetCollection, Force: false},
			expectError:   true,
			errorContains: "already exists",
		},
		{
			name:          "collection exists with force",
			store:         store,
			ctx:           ctx,
			reader:        bytes.NewReader(backupBuf.Bytes()),
			opts:          &types.RestoreOptions{CollectionName: targetCollection, Force: true},
			expectError:   true,
			errorContains: "not fully implemented", // Expected since restore is not fully implemented
		},
		{
			name:          "empty data reader",
			store:         store,
			ctx:           ctx,
			reader:        strings.NewReader(""), // Use empty string reader instead of empty bytes
			opts:          &types.RestoreOptions{CollectionName: "new_collection_empty"},
			expectError:   true,
			errorContains: "failed to read data", // This will be the actual error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.store.Restore(tt.ctx, tt.reader, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
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

// Test gzip compression and decompression paths
func TestBackupRestore_GzipPaths(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	collectionName := fmt.Sprintf("gzip_test_%d", time.Now().UnixNano())

	// Create test collection
	setupTestCollection(t, store, collectionName)

	t.Run("compressed backup", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &types.BackupOptions{
			CollectionName: collectionName,
			Compress:       true,
		}

		err := store.Backup(ctx, &buf, opts)
		if err != nil {
			t.Fatalf("backup failed: %v", err)
		}

		// Verify it's actually gzipped
		reader := bytes.NewReader(buf.Bytes())
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			t.Fatalf("failed to create gzip reader: %v", err)
		}
		defer gzReader.Close()

		content, err := io.ReadAll(gzReader)
		if err != nil {
			t.Fatalf("failed to read compressed content: %v", err)
		}

		if len(content) == 0 {
			t.Error("decompressed content is empty")
		}

		// Test restore with gzipped data
		restoreOpts := &types.RestoreOptions{
			CollectionName: fmt.Sprintf("restore_gzip_%d", time.Now().UnixNano()),
			Force:          false,
		}

		err = store.Restore(ctx, bytes.NewReader(buf.Bytes()), restoreOpts)
		if err == nil {
			t.Error("expected error since restore is not fully implemented")
		} else if !strings.Contains(err.Error(), "not fully implemented") {
			t.Errorf("expected 'not fully implemented' error, got: %v", err)
		}
	})

	t.Run("uncompressed backup", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &types.BackupOptions{
			CollectionName: collectionName,
			Compress:       false,
		}

		err := store.Backup(ctx, &buf, opts)
		if err != nil {
			t.Fatalf("backup failed: %v", err)
		}

		// Verify it's not gzipped
		if buf.Len() < 2 {
			t.Fatal("backup data too short")
		}

		data := buf.Bytes()
		if data[0] == 0x1f && data[1] == 0x8b {
			t.Error("data should not be gzipped")
		}

		// Test restore with uncompressed data
		restoreOpts := &types.RestoreOptions{
			CollectionName: fmt.Sprintf("restore_uncompressed_%d", time.Now().UnixNano()),
			Force:          false,
		}

		err = store.Restore(ctx, bytes.NewReader(buf.Bytes()), restoreOpts)
		if err == nil {
			t.Error("expected error since restore is not fully implemented")
		} else if !strings.Contains(err.Error(), "not fully implemented") {
			t.Errorf("expected 'not fully implemented' error, got: %v", err)
		}
	})
}

// Test singleByteReader helper
func TestSingleByteReader(t *testing.T) {
	data := []byte("hello world")

	t.Run("read all at once", func(t *testing.T) {
		reader := &singleByteReader{data: data}
		buf := make([]byte, len(data))
		n, err := reader.Read(buf)
		if err != io.EOF {
			t.Errorf("expected EOF, got %v", err)
		}
		if n != len(data) {
			t.Errorf("expected to read %d bytes, got %d", len(data), n)
		}
		if !bytes.Equal(buf, data) {
			t.Errorf("expected %s, got %s", string(data), string(buf))
		}
	})

	t.Run("read in chunks", func(t *testing.T) {
		reader := &singleByteReader{data: data}
		var result []byte
		buf := make([]byte, 3)

		for {
			n, err := reader.Read(buf)
			if n > 0 {
				result = append(result, buf[:n]...)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		if !bytes.Equal(result, data) {
			t.Errorf("expected %s, got %s", string(data), string(result))
		}
	})

	t.Run("read after EOF", func(t *testing.T) {
		reader := &singleByteReader{data: []byte("test")}
		buf := make([]byte, 10)

		// First read should get all data
		n, err := reader.Read(buf)
		if err != io.EOF {
			t.Errorf("expected EOF, got %v", err)
		}
		if n != 4 {
			t.Errorf("expected 4 bytes, got %d", n)
		}

		// Second read should return 0, EOF
		n, err = reader.Read(buf)
		if err != io.EOF {
			t.Errorf("expected EOF, got %v", err)
		}
		if n != 0 {
			t.Errorf("expected 0 bytes, got %d", n)
		}
	})

	t.Run("empty data", func(t *testing.T) {
		reader := &singleByteReader{data: []byte{}}
		buf := make([]byte, 10)
		n, err := reader.Read(buf)
		if err != io.EOF {
			t.Errorf("expected EOF, got %v", err)
		}
		if n != 0 {
			t.Errorf("expected 0 bytes, got %d", n)
		}
	})
}

// Test backup with various extra params
func TestBackup_ExtraParams(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	collectionName := fmt.Sprintf("extra_params_test_%d", time.Now().UnixNano())

	// Create test collection
	setupTestCollection(t, store, collectionName)

	tests := []struct {
		name        string
		extraParams map[string]interface{}
	}{
		{
			name:        "nil extra params",
			extraParams: nil,
		},
		{
			name:        "empty extra params",
			extraParams: map[string]interface{}{},
		},
		{
			name: "various extra params",
			extraParams: map[string]interface{}{
				"timeout":     30,
				"retry_count": 3,
				"compression": "gzip",
				"metadata":    map[string]string{"version": "1.0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := &types.BackupOptions{
				CollectionName: collectionName,
				Compress:       false,
				ExtraParams:    tt.extraParams,
			}

			err := store.Backup(ctx, &buf, opts)
			if err != nil {
				t.Fatalf("backup failed: %v", err)
			}

			if buf.Len() == 0 {
				t.Error("backup data is empty")
			}
		})
	}
}

// Test restore with various extra params
func TestRestore_ExtraParams(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()
	sourceCollection := fmt.Sprintf("restore_extra_params_%d", time.Now().UnixNano())

	// Create source collection and backup
	setupTestCollection(t, store, sourceCollection)
	var backupBuf bytes.Buffer
	backupOpts := &types.BackupOptions{
		CollectionName: sourceCollection,
		Compress:       false,
	}
	err := store.Backup(ctx, &backupBuf, backupOpts)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	tests := []struct {
		name        string
		extraParams map[string]interface{}
	}{
		{
			name:        "nil extra params",
			extraParams: nil,
		},
		{
			name:        "empty extra params",
			extraParams: map[string]interface{}{},
		},
		{
			name: "various extra params",
			extraParams: map[string]interface{}{
				"timeout":       30,
				"verify_data":   true,
				"restore_mode":  "full",
				"backup_format": "snapshot",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetCollection := fmt.Sprintf("target_extra_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
			opts := &types.RestoreOptions{
				CollectionName: targetCollection,
				Force:          false,
				ExtraParams:    tt.extraParams,
			}

			err := store.Restore(ctx, bytes.NewReader(backupBuf.Bytes()), opts)
			// We expect this to fail since restore is not fully implemented
			if err == nil {
				t.Error("expected error since restore is not fully implemented")
			} else if !strings.Contains(err.Error(), "not fully implemented") {
				t.Errorf("expected 'not fully implemented' error, got: %v", err)
			}
		})
	}
}

// Test backup/restore with malformed gzip data
func TestRestore_MalformedGzipData(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	ctx := context.Background()

	t.Run("malformed gzip header", func(t *testing.T) {
		// Create data that looks like gzip but isn't
		malformedData := []byte{0x1f, 0x8b, 0x00, 0x00} // Gzip magic but incomplete

		opts := &types.RestoreOptions{
			CollectionName: fmt.Sprintf("malformed_gzip_%d", time.Now().UnixNano()),
			Force:          false,
		}

		err := store.Restore(ctx, bytes.NewReader(malformedData), opts)
		if err == nil {
			t.Error("expected error for malformed gzip data")
		} else if !strings.Contains(err.Error(), "failed to create gzip reader") {
			t.Errorf("expected gzip reader error, got: %v", err)
		}
	})

	t.Run("gzip data too short", func(t *testing.T) {
		// Create very short data that doesn't have gzip magic
		shortData := []byte{0x01}

		opts := &types.RestoreOptions{
			CollectionName: fmt.Sprintf("short_data_%d", time.Now().UnixNano()),
			Force:          false,
		}

		err := store.Restore(ctx, bytes.NewReader(shortData), opts)
		if err == nil {
			t.Error("expected error since restore is not fully implemented")
		} else if !strings.Contains(err.Error(), "not fully implemented") {
			t.Errorf("expected 'not fully implemented' error, got: %v", err)
		}
	})
}

// Test context cancellation during backup/restore operations
func TestBackupRestore_ContextCancellation(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestEnvironment(env, t)
	store := env.Store

	collectionName := fmt.Sprintf("context_cancel_%d", time.Now().UnixNano())
	setupTestCollection(t, store, collectionName)

	t.Run("backup with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		var buf bytes.Buffer
		opts := &types.BackupOptions{
			CollectionName: collectionName,
			Compress:       false,
		}

		err := store.Backup(ctx, &buf, opts)
		if err == nil {
			t.Error("expected error due to cancelled context")
		}
	})

	t.Run("restore with cancelled context", func(t *testing.T) {
		// First create a backup
		var backupBuf bytes.Buffer
		backupOpts := &types.BackupOptions{
			CollectionName: collectionName,
			Compress:       false,
		}
		err := store.Backup(context.Background(), &backupBuf, backupOpts)
		if err != nil {
			t.Fatalf("failed to create backup: %v", err)
		}

		// Now try to restore with cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		opts := &types.RestoreOptions{
			CollectionName: fmt.Sprintf("cancelled_restore_%d", time.Now().UnixNano()),
			Force:          false,
		}

		err = store.Restore(ctx, bytes.NewReader(backupBuf.Bytes()), opts)
		if err == nil {
			t.Error("expected error due to cancelled context")
		}
	})
}

func setupTestCollection(t *testing.T, store *Store, collectionName string) {
	ctx := context.Background()

	// Create collection
	config := types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: collectionName,
		M:              16,
		EfConstruction: 100,
		Timeout:        30,
	}

	err := store.CreateCollection(ctx, &config)
	if err != nil {
		t.Fatalf("failed to create test collection: %v", err)
	}

	// Add some test documents
	documents := []*types.Document{
		{
			ID:      "doc1",
			Content: "Test document 1",
			Vector:  generateRandomVector(128),
			Metadata: map[string]interface{}{
				"title": "Document 1",
				"type":  "test",
			},
		},
		{
			ID:      "doc2",
			Content: "Test document 2",
			Vector:  generateRandomVector(128),
			Metadata: map[string]interface{}{
				"title": "Document 2",
				"type":  "test",
			},
		},
	}

	opts := &types.AddDocumentOptions{
		CollectionName: collectionName,
		Documents:      documents,
		BatchSize:      10,
	}

	_, err = store.AddDocuments(ctx, opts)
	if err != nil {
		t.Fatalf("failed to add test documents: %v", err)
	}
}

func setupLargeTestCollection(t *testing.T, store *Store, collectionName string, numDocs int) {
	ctx := context.Background()

	// Create collection
	config := types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: collectionName,
		M:              16,
		EfConstruction: 100,
		Timeout:        30,
	}

	err := store.CreateCollection(ctx, &config)
	if err != nil {
		t.Fatalf("failed to create large test collection: %v", err)
	}

	// Add documents in batches
	batchSize := 50
	for i := 0; i < numDocs; i += batchSize {
		var docs []*types.Document
		end := i + batchSize
		if end > numDocs {
			end = numDocs
		}

		for j := i; j < end; j++ {
			docs = append(docs, &types.Document{
				ID:      fmt.Sprintf("doc_%d", j),
				Content: fmt.Sprintf("Large test document %d with more content to increase size", j),
				Vector:  generateRandomVector(128),
				Metadata: map[string]interface{}{
					"title": fmt.Sprintf("Document %d", j),
					"type":  "large_test",
					"batch": i / batchSize,
				},
			})
		}

		opts := &types.AddDocumentOptions{
			CollectionName: collectionName,
			Documents:      docs,
			BatchSize:      batchSize,
		}

		_, err = store.AddDocuments(ctx, opts)
		if err != nil {
			t.Fatalf("failed to add document batch: %v", err)
		}
	}
}

func generateRandomVector(dimension int) []float64 {
	vector := make([]float64, dimension)
	for i := range vector {
		vector[i] = float64(i%10) / 10.0
	}
	return vector
}
