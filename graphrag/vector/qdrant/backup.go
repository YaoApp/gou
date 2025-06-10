package qdrant

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
)

// Backup creates a backup of the collection
func (s *Store) Backup(ctx context.Context, opts *types.BackupOptions) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return fmt.Errorf("not connected to Qdrant server")
	}

	// TODO: Implement backup functionality
	return fmt.Errorf("Backup not implemented yet")
}

// Restore restores a collection from backup
func (s *Store) Restore(ctx context.Context, opts *types.RestoreOptions) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return fmt.Errorf("not connected to Qdrant server")
	}

	// TODO: Implement restore functionality
	return fmt.Errorf("Restore not implemented yet")
}
