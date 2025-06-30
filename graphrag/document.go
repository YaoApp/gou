package graphrag

import (
	"context"
	"io"

	"github.com/yaoapp/gou/graphrag/types"
)

// AddFile adds a file to a collection
func (g *GraphRag) AddFile(ctx context.Context, file string, options *types.UpsertOptions) (string, error) {
	// TODO: Implement AddFile
	return "", nil
}

// AddText adds a text to a collection
func (g *GraphRag) AddText(ctx context.Context, text string, options *types.UpsertOptions) (string, error) {
	// TODO: Implement AddText
	return "", nil
}

// AddURL adds a URL to a collection
func (g *GraphRag) AddURL(ctx context.Context, url string, options *types.UpsertOptions) (string, error) {
	// TODO: Implement AddURL
	return "", nil
}

// AddStream adds a stream to a collection
func (g *GraphRag) AddStream(ctx context.Context, stream io.ReadSeeker, options *types.UpsertOptions) (string, error) {
	// TODO: Implement AddStream
	return "", nil
}

// RemoveDocs removes documents by IDs
func (g *GraphRag) RemoveDocs(ctx context.Context, ids []string) (int, error) {
	// TODO: Implement RemoveDocs
	return 0, nil
}
