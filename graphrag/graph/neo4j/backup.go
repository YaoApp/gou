package neo4j

import (
	"context"
	"io"

	"github.com/yaoapp/gou/graphrag/types"
)

// Backup creates a backup of the graph
func (s *Store) Backup(ctx context.Context, writer io.Writer, opts *types.GraphBackupOptions) error {
	// TODO: implement graph backup
	return nil
}

// Restore restores a graph from backup
func (s *Store) Restore(ctx context.Context, reader io.Reader, opts *types.GraphRestoreOptions) error {
	// TODO: implement graph restore
	return nil
}
