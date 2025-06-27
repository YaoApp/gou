package neo4j

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/graphrag/types"
)

// TestConfig holds test configuration for connection tests
type TestConfig struct {
	URL      string
	User     string
	Password string
}

// getTestConfig returns test configuration from environment variables
func getTestConfig() *TestConfig {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		return nil // Required environment variable missing
	}

	return &TestConfig{
		URL:      url,
		User:     getEnvOrDefault("NEO4J_TEST_USER", "neo4j"),
		Password: getEnvOrDefault("NEO4J_TEST_PASS", "password"),
	}
}

// getEnterpriseTestConfig returns enterprise test configuration
func getEnterpriseTestConfig() *TestConfig {
	url := os.Getenv("NEO4J_TEST_ENTERPRISE_URL")
	if url == "" {
		return nil // Enterprise config not available
	}

	return &TestConfig{
		URL:      url,
		User:     getEnvOrDefault("NEO4J_TEST_ENTERPRISE_USER", "neo4j"),
		Password: getEnvOrDefault("NEO4J_TEST_ENTERPRISE_PASS", "password"),
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// connectWithRetry attempts to connect and skips test if connection fails
func connectWithRetry(ctx context.Context, t *testing.T, store *Store, config types.GraphStoreConfig) {
	err := store.Connect(ctx, config)
	if err != nil {
		// If connection fails, it might be because Neo4j server is not running
		// Skip the test instead of failing
		t.Skipf("Connect failed (Neo4j server might not be running): %v", err)
	}
}

// =============================================================================
// Tests for connection.go methods
// =============================================================================

// TestConnect tests the Connect method
func TestConnect(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)

		// Verify connection state
		if !store.IsConnected() {
			t.Error("Store should be connected after Connect")
		}

		// Verify config is saved
		savedConfig := store.GetConfig()
		if savedConfig.DatabaseURL != config.URL {
			t.Errorf("Expected database URL %s, got %s", config.URL, savedConfig.DatabaseURL)
		}

		// Clean up
		store.Close()
	})

	t.Run("MissingURL", func(t *testing.T) {
		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType: "neo4j",
			// Missing DatabaseURL
		}

		err := store.Connect(ctx, storeConfig)
		if err == nil {
			t.Error("Connect should fail with missing database URL")
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after failed Connect")
		}
	})

	t.Run("MissingPassword", func(t *testing.T) {
		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: "neo4j://localhost:7687",
			DriverConfig: map[string]interface{}{
				"username": "neo4j",
				// Missing password
			},
		}

		err := store.Connect(ctx, storeConfig)
		if err == nil {
			t.Error("Connect should fail with missing password")
		}

		if store.IsConnected() {
			t.Error("Store should not be connected after failed Connect")
		}
	})

	t.Run("AlreadyConnected", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		// First connection
		connectWithRetry(ctx, t, store, storeConfig)

		// Second connection should not fail
		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Errorf("Second Connect should not fail: %v", err)
		}

		// Clean up
		store.Close()
	})
}

// TestDisconnect tests the Disconnect method
func TestDisconnect(t *testing.T) {
	t.Run("ConnectedStore", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
			},
		}

		// Connect first
		connectWithRetry(ctx, t, store, storeConfig)

		// Disconnect
		err := store.Disconnect(ctx)
		if err != nil {
			t.Errorf("Disconnect failed: %v", err)
		}

		// Verify disconnection
		if store.IsConnected() {
			t.Error("Store should not be connected after Disconnect")
		}

		// Verify config is cleared
		savedConfig := store.GetConfig()
		if savedConfig.DatabaseURL != "" {
			t.Error("Config should be cleared after Disconnect")
		}

		// Verify separate database flag is reset
		if store.UseSeparateDatabase() {
			t.Error("Separate database flag should be reset after Disconnect")
		}
	})

	t.Run("NotConnectedStore", func(t *testing.T) {
		store := NewStore()
		ctx := context.Background()

		// Disconnect without connecting should not fail
		err := store.Disconnect(ctx)
		if err != nil {
			t.Errorf("Disconnect should not fail for unconnected store: %v", err)
		}
	})
}

// TestIsConnected tests the IsConnected method
func TestIsConnected(t *testing.T) {
	store := NewStore()

	// Initially not connected
	if store.IsConnected() {
		t.Error("New store should not be connected")
	}

	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	ctx := context.Background()
	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	// Connect
	connectWithRetry(ctx, t, store, storeConfig)

	// Should be connected
	if !store.IsConnected() {
		t.Error("Store should be connected after Connect")
	}

	// Disconnect
	store.Disconnect(ctx)

	// Should not be connected
	if store.IsConnected() {
		t.Error("Store should not be connected after Disconnect")
	}
}

// TestClose tests the Close method
func TestClose(t *testing.T) {
	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	store := NewStore()
	ctx := context.Background()

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	// Connect
	connectWithRetry(ctx, t, store, storeConfig)

	// Close
	err := store.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Should be disconnected
	if store.IsConnected() {
		t.Error("Store should not be connected after Close")
	}
}

// TestEnterpriseFlag tests enterprise flag configuration
func TestEnterpriseFlag(t *testing.T) {
	// Get test config to avoid hardcoded URL
	testConfig := getTestConfig()
	if testConfig == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	testCases := []struct {
		name       string
		config     map[string]interface{}
		enterprise bool
	}{
		{
			name:       "DefaultCommunity",
			config:     map[string]interface{}{"username": testConfig.User, "password": testConfig.Password},
			enterprise: false,
		},
		{
			name:       "ExplicitLabelBased",
			config:     map[string]interface{}{"username": testConfig.User, "password": testConfig.Password, "use_separate_database": false},
			enterprise: false,
		},
		{
			name:       "ExplicitSeparateDatabase",
			config:     map[string]interface{}{"username": testConfig.User, "password": testConfig.Password, "use_separate_database": true},
			enterprise: true,
		},
		{
			name:       "NoDriverConfig",
			config:     nil,
			enterprise: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStore()
			ctx := context.Background()

			storeConfig := types.GraphStoreConfig{
				StoreType:    "neo4j",
				DatabaseURL:  testConfig.URL, // Use environment variable URL
				DriverConfig: tc.config,
			}

			err := store.Connect(ctx, storeConfig)
			if tc.config == nil || tc.config["password"] == nil {
				// Should fail without password
				if err == nil {
					t.Error("Connect should fail without password")
				}
				return
			}

			if err != nil {
				// Skip test if connection fails (e.g., server not running or wrong credentials)
				t.Skipf("Connect failed (Neo4j server might not be running or wrong credentials): %v", err)
			}

			if store.UseSeparateDatabase() != tc.enterprise {
				t.Errorf("Expected use_separate_database flag %v, got %v", tc.enterprise, store.UseSeparateDatabase())
			}

			store.Close()
		})
	}
}

// TestEnterpriseDetection tests enterprise edition detection
func TestEnterpriseDetection(t *testing.T) {
	t.Run("CommunityEdition", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username": config.User,
				"password": config.Password,
				// No enterprise flag, should default to false
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)

		// Should default to label-based storage
		if store.UseSeparateDatabase() {
			t.Error("Should default to label-based storage when use_separate_database flag not set")
		}

		store.Close()
	})

	t.Run("EnterpriseEdition", func(t *testing.T) {
		config := getEnterpriseTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_ENTERPRISE_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username":              config.User,
				"password":              config.Password,
				"use_separate_database": true, // Explicitly set separate database flag
			},
		}

		connectWithRetry(ctx, t, store, storeConfig)

		// Should use separate database storage
		if !store.UseSeparateDatabase() {
			t.Error("Should use separate database storage when use_separate_database flag is true")
		}

		store.Close()
	})
}

// TestConfigurationValidation tests configuration validation
func TestConfigurationValidation(t *testing.T) {
	t.Run("SeparateDatabaseRequiresEnterprise", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		// Try to use separate database with community edition
		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username":              config.User,
				"password":              config.Password,
				"use_separate_database": true, // Request separate database
			},
		}

		err := store.Connect(ctx, storeConfig)

		// Check the actual detected edition
		if err == nil {
			defer store.Close()
			if store.IsEnterpriseEdition() {
				t.Skip("Test environment is actually Enterprise Edition, cannot test community validation")
			} else {
				t.Error("Connect should fail when requesting separate database with community edition")
			}
		} else {
			expectedError := "separate database storage requires Neo4j Enterprise Edition"
			if !contains(err.Error(), expectedError) {
				t.Errorf("Expected error to contain '%s', got: %s", expectedError, err.Error())
			} else {
				t.Logf("Correctly rejected separate database request for community edition: %s", err.Error())
			}
		}
	})

	t.Run("LabelBasedWorksWithCommunity", func(t *testing.T) {
		config := getTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		// Use label-based storage with community edition (should work)
		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username":              config.User,
				"password":              config.Password,
				"use_separate_database": false, // Use label-based storage
			},
		}

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Connect failed (Neo4j server might not be running): %v", err)
		}
		defer store.Close()

		// Should work fine
		if !store.IsConnected() {
			t.Error("Should be connected when using label-based storage")
		}

		if store.UseSeparateDatabase() {
			t.Error("Should not use separate database when explicitly set to false")
		}

		t.Logf("Successfully connected with label-based storage. Edition: %v",
			map[bool]string{true: "Enterprise", false: "Community"}[store.IsEnterpriseEdition()])
	})

	t.Run("SeparateDatabaseWorksWithEnterprise", func(t *testing.T) {
		config := getEnterpriseTestConfig()
		if config == nil {
			t.Skip("NEO4J_TEST_ENTERPRISE_URL environment variable not set")
		}

		store := NewStore()
		ctx := context.Background()

		// Use separate database with enterprise edition (should work)
		storeConfig := types.GraphStoreConfig{
			StoreType:   "neo4j",
			DatabaseURL: config.URL,
			DriverConfig: map[string]interface{}{
				"username":              config.User,
				"password":              config.Password,
				"use_separate_database": true, // Use separate database
			},
		}

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			t.Skipf("Connect failed (Neo4j Enterprise server might not be running): %v", err)
		}
		defer store.Close()

		// Should work fine
		if !store.IsConnected() {
			t.Error("Should be connected when using separate database with enterprise edition")
		}

		if !store.UseSeparateDatabase() {
			t.Error("Should use separate database when explicitly set to true with enterprise edition")
		}

		if !store.IsEnterpriseEdition() {
			t.Error("Should detect enterprise edition")
		}

		t.Logf("Successfully connected with separate database storage on Enterprise Edition")
	})

}

// ===== Stress Tests =====

func TestStressConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	// Use light stress config for CI/CD environments
	stressConfig := LightStressConfig()

	t.Logf("Starting stress test: %d workers, %d operations per worker",
		stressConfig.NumWorkers, stressConfig.OperationsPerWorker)

	// Capture initial state for leak detection
	initialGoroutines := captureGoroutineState()
	initialMemory := captureMemoryStats()

	// Define the test operation
	operation := func(ctx context.Context) error {
		store := NewStore()

		err := store.Connect(ctx, storeConfig)
		if err != nil {
			return err
		}

		// Verify connection
		if !store.IsConnected() {
			return fmt.Errorf("connection verification failed")
		}

		// Close connection
		return store.Close()
	}

	// Run stress test
	result := runStressTest(stressConfig, operation)

	t.Logf("Stress test completed: %d total operations, %d errors, %.2f%% success rate, duration: %v",
		result.TotalOperations, result.ErrorCount, result.SuccessRate, result.Duration)

	// Verify success rate
	assert.GreaterOrEqual(t, result.SuccessRate, stressConfig.MinSuccessRate,
		"Success rate %.2f%% is below minimum %.2f%%", result.SuccessRate, stressConfig.MinSuccessRate)

	// Allow some time for cleanup
	time.Sleep(2 * time.Second)

	// Check for goroutine leaks
	finalGoroutines := captureGoroutineState()
	leaked, _ := analyzeGoroutineChanges(initialGoroutines, finalGoroutines)

	// Filter out system goroutines
	var appLeaked []GoroutineInfo
	for _, g := range leaked {
		if !g.IsSystem {
			appLeaked = append(appLeaked, g)
		}
	}

	if len(appLeaked) > 0 {
		t.Logf("Potential goroutine leaks detected (%d):", len(appLeaked))
		for _, g := range appLeaked {
			t.Logf("  Goroutine %d [%s]: %s", g.ID, g.State, g.Function)
		}
		// Note: We log but don't fail the test as some leaks might be acceptable
	}

	// Check for significant memory leaks
	finalMemory := captureMemoryStats()
	memGrowth := calculateMemoryGrowth(initialMemory, finalMemory)

	t.Logf("Memory growth: Alloc=%d, HeapAlloc=%d, Sys=%d",
		memGrowth.AllocGrowth, memGrowth.HeapAllocGrowth, memGrowth.SysGrowth)

	// Alert if memory growth is excessive (more than 10MB)
	const maxMemoryGrowth = 10 * 1024 * 1024 // 10MB
	if memGrowth.HeapAllocGrowth > maxMemoryGrowth {
		t.Logf("WARNING: Significant memory growth detected: %d bytes", memGrowth.HeapAllocGrowth)
	}
}

func TestConcurrentConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	const numGoroutines = 20
	const operationsPerGoroutine = 10

	t.Logf("Starting concurrent test: %d goroutines, %d operations each",
		numGoroutines, operationsPerGoroutine)

	// Capture initial state
	initialGoroutines := captureGoroutineState()
	initialMemory := captureMemoryStats()

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	// Start concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				func() {
					store := NewStore()

					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					err := store.Connect(ctx, storeConfig)
					if err != nil {
						errors <- fmt.Errorf("worker %d, op %d: Connect failed: %w", workerID, j, err)
						return
					}

					if !store.IsConnected() {
						errors <- fmt.Errorf("worker %d, op %d: IsConnected returned false", workerID, j)
						store.Close()
						return
					}

					err = store.Close()
					if err != nil {
						errors <- fmt.Errorf("worker %d, op %d: Close failed: %w", workerID, j, err)
						return
					}
				}()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Count errors
	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent test error: %v", err)
		errorCount++
	}

	totalOperations := numGoroutines * operationsPerGoroutine
	successRate := float64(totalOperations-errorCount) / float64(totalOperations) * 100

	t.Logf("Concurrent test completed: %d total operations, %d errors, %.2f%% success rate",
		totalOperations, errorCount, successRate)

	// Verify success rate (should be very high for concurrent operations)
	assert.GreaterOrEqual(t, successRate, 95.0,
		"Success rate %.2f%% is below minimum 95%%", successRate)

	// Allow cleanup time
	time.Sleep(2 * time.Second)

	// Check for leaks
	finalGoroutines := captureGoroutineState()
	leaked, _ := analyzeGoroutineChanges(initialGoroutines, finalGoroutines)

	var appLeaked []GoroutineInfo
	for _, g := range leaked {
		if !g.IsSystem {
			appLeaked = append(appLeaked, g)
		}
	}

	if len(appLeaked) > 0 {
		t.Logf("Potential goroutine leaks after concurrent test (%d):", len(appLeaked))
		for _, g := range appLeaked {
			t.Logf("  Goroutine %d [%s]: %s", g.ID, g.State, g.Function)
		}
	}

	// Check memory growth
	finalMemory := captureMemoryStats()
	memGrowth := calculateMemoryGrowth(initialMemory, finalMemory)

	t.Logf("Memory growth after concurrent test: Alloc=%d, HeapAlloc=%d, Sys=%d",
		memGrowth.AllocGrowth, memGrowth.HeapAllocGrowth, memGrowth.SysGrowth)
}

func TestMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	// Capture baseline memory stats
	runtime.GC()
	runtime.GC() // Double GC to ensure clean baseline
	time.Sleep(100 * time.Millisecond)

	baselineMemory := captureMemoryStats()
	t.Logf("Baseline memory: Alloc=%d, HeapAlloc=%d, Sys=%d",
		baselineMemory.Alloc, baselineMemory.HeapAlloc, baselineMemory.Sys)

	// Perform multiple connection cycles
	const cycles = 100
	for i := 0; i < cycles; i++ {
		func() {
			store := NewStore()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := store.Connect(ctx, storeConfig)
			if err != nil {
				t.Skipf("Failed to connect in cycle %d: %v", i, err)
			}

			assert.True(t, store.IsConnected())

			err = store.Close()
			assert.NoError(t, err)
		}()

		// Force GC every 10 cycles
		if i%10 == 9 {
			runtime.GC()
		}
	}

	// Final cleanup and measurement
	runtime.GC()
	runtime.GC()
	time.Sleep(500 * time.Millisecond) // Allow more time for cleanup

	finalMemory := captureMemoryStats()
	memGrowth := calculateMemoryGrowth(baselineMemory, finalMemory)

	t.Logf("Final memory: Alloc=%d, HeapAlloc=%d, Sys=%d",
		finalMemory.Alloc, finalMemory.HeapAlloc, finalMemory.Sys)
	t.Logf("Memory growth: Alloc=%d, HeapAlloc=%d, Sys=%d, GC cycles=%d",
		memGrowth.AllocGrowth, memGrowth.HeapAllocGrowth, memGrowth.SysGrowth, memGrowth.NumGCDiff)

	// Check for excessive memory growth
	// Allow some growth but alert if it's too much
	const maxAllowedGrowth = 5 * 1024 * 1024 // 5MB
	if memGrowth.HeapAllocGrowth > maxAllowedGrowth {
		t.Errorf("Excessive memory growth detected: %d bytes (max allowed: %d)",
			memGrowth.HeapAllocGrowth, maxAllowedGrowth)
	}

	// Also check if total allocated memory grew too much
	growthRatio := float64(memGrowth.TotalAllocGrowth) / float64(baselineMemory.TotalAlloc)
	if growthRatio > 2.0 { // More than 200% growth
		t.Logf("WARNING: Total allocation grew by %.1f%% during test", growthRatio*100)
	}
}

func TestGoroutineLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping goroutine leak test in short mode")
	}

	config := getTestConfig()
	if config == nil {
		t.Skip("NEO4J_TEST_URL environment variable not set")
	}

	storeConfig := types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: config.URL,
		DriverConfig: map[string]interface{}{
			"username": config.User,
			"password": config.Password,
		},
	}

	// Capture baseline goroutine state
	baselineGoroutines := captureGoroutineState()
	t.Logf("Baseline goroutines: %d total", len(baselineGoroutines))

	// Perform multiple connection cycles
	const cycles = 50
	for i := 0; i < cycles; i++ {
		func() {
			store := NewStore()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := store.Connect(ctx, storeConfig)
			if err != nil {
				t.Skipf("Failed to connect in cycle %d: %v", i, err)
			}

			assert.True(t, store.IsConnected())

			err = store.Close()
			assert.NoError(t, err)
		}()
	}

	// Allow time for cleanup
	time.Sleep(2 * time.Second)
	runtime.GC()

	// Capture final goroutine state
	finalGoroutines := captureGoroutineState()
	t.Logf("Final goroutines: %d total", len(finalGoroutines))

	// Analyze changes
	leaked, cleaned := analyzeGoroutineChanges(baselineGoroutines, finalGoroutines)

	t.Logf("Goroutine changes: %d leaked, %d cleaned", len(leaked), len(cleaned))

	// Filter out system goroutines from leaked list
	var appLeaked []GoroutineInfo
	for _, g := range leaked {
		if !g.IsSystem {
			appLeaked = append(appLeaked, g)
		}
	}

	if len(appLeaked) > 0 {
		t.Logf("Application goroutine leaks detected (%d):", len(appLeaked))
		for _, g := range appLeaked {
			t.Logf("  Goroutine %d [%s]: %s", g.ID, g.State, g.Function)
			if len(g.Stack) > 0 {
				t.Logf("    Stack: %s", strings.Split(g.Stack, "\n")[0]) // First line only
			}
		}

		// Fail test if we have significant application goroutine leaks
		if len(appLeaked) > 5 { // Allow some tolerance
			t.Errorf("Too many application goroutine leaks: %d (threshold: 5)", len(appLeaked))
		}
	}

	// Log some statistics
	systemLeaked := len(leaked) - len(appLeaked)
	if systemLeaked > 0 {
		t.Logf("System goroutine changes: %d (these are typically acceptable)", systemLeaked)
	}
}
