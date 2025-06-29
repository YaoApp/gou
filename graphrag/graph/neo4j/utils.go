package neo4j

import (
	"context"
	"sync"
)

// Global semaphore to serialize critical operations and prevent conflicts
var criticalOperationSemaphore = make(chan struct{}, 1)
var criticalOperationOnce sync.Once

// initCriticalOperationSemaphore initializes the critical operation semaphore
func initCriticalOperationSemaphore() {
	criticalOperationOnce.Do(func() {
		// Initialize with one token to allow only one critical operation at a time
		criticalOperationSemaphore <- struct{}{}
	})
}

// executeCriticalOperation executes a critical operation with serialization to prevent conflicts
// This is useful for operations that might cause deadlocks or conflicts when run concurrently,
// such as schema operations, database management, constraint creation/deletion, etc.
func executeCriticalOperation(ctx context.Context, operation func() error) error {
	initCriticalOperationSemaphore()

	// Acquire semaphore (wait for our turn)
	select {
	case <-criticalOperationSemaphore:
		// Got the token, proceed with operation
	case <-ctx.Done():
		return ctx.Err()
	}

	// Ensure we release the semaphore when done
	defer func() {
		criticalOperationSemaphore <- struct{}{}
	}()

	// Execute the operation
	return operation()
}
