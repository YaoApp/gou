package graphrag

import (
	"context"
	"io"
)

// Backup backs up a collection
func (g *GraphRag) Backup(ctx context.Context, writer io.Writer, id string) error {
	// TODO: Implement Backup
	return nil
}

// Restore restores a collection
func (g *GraphRag) Restore(ctx context.Context, reader io.Reader, id string) error {
	// TODO: Implement Restore
	return nil
}
