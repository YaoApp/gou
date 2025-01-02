package qdrant

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/rag/driver"
)

// FileUpload implements the driver.FileUpload interface for Qdrant
type FileUpload struct {
	engine     *Engine
	vectorizer driver.Vectorizer
}

// NewFileUpload creates a new FileUpload instance
func NewFileUpload(engine *Engine, vectorizer driver.Vectorizer) (*FileUpload, error) {
	return &FileUpload{
		engine:     engine,
		vectorizer: vectorizer,
	}, nil
}

// Upload processes content from a reader
func (f *FileUpload) Upload(ctx context.Context, reader io.Reader, opts driver.FileUploadOptions) (*driver.FileUploadResult, error) {
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = 1000 // default chunk size
	}

	scanner := bufio.NewScanner(reader)
	var buffer string
	var documents []*driver.Document
	docID := 1

	// Read content in chunks
	for scanner.Scan() {
		line := scanner.Text()
		buffer += line + "\n"

		if len(buffer) >= opts.ChunkSize {
			// Generate numeric string ID for the document
			docIDStr := fmt.Sprintf("file-%s-chunk-%d", filepath.Base(opts.IndexName), docID)

			// Vectorize the chunk
			embeddings, err := f.vectorizer.Vectorize(ctx, buffer)
			if err != nil {
				return nil, fmt.Errorf("failed to vectorize content chunk %d: %w", docID, err)
			}

			doc := &driver.Document{
				DocID:        docIDStr,
				Content:      buffer,
				ChunkSize:    opts.ChunkSize,
				ChunkOverlap: opts.ChunkOverlap,
				Embeddings:   embeddings,
				Metadata: map[string]interface{}{
					"chunk_number": docID,
					"file_name":    opts.IndexName,
				},
			}
			documents = append(documents, doc)
			buffer = buffer[max(0, len(buffer)-opts.ChunkOverlap):]
			docID++
		}
	}

	// Handle any remaining content
	if len(buffer) > 0 {
		docIDStr := fmt.Sprintf("file-%s-chunk-%d", filepath.Base(opts.IndexName), docID)
		embeddings, err := f.vectorizer.Vectorize(ctx, buffer)
		if err != nil {
			return nil, fmt.Errorf("failed to vectorize final content chunk: %w", err)
		}

		doc := &driver.Document{
			DocID:        docIDStr,
			Content:      buffer,
			ChunkSize:    opts.ChunkSize,
			ChunkOverlap: opts.ChunkOverlap,
			Embeddings:   embeddings,
			Metadata: map[string]interface{}{
				"chunk_number": docID,
				"file_name":    opts.IndexName,
			},
		}
		documents = append(documents, doc)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading content: %w", err)
	}

	result := &driver.FileUploadResult{
		Documents: documents,
	}

	// If async processing is requested, index documents in batch
	if opts.Async && len(documents) > 0 {
		taskID, err := f.engine.IndexBatch(ctx, opts.IndexName, documents)
		if err != nil {
			return nil, fmt.Errorf("error indexing documents: %w", err)
		}
		result.TaskID = taskID
	} else if len(documents) > 0 {
		// Synchronous processing
		for _, doc := range documents {
			if err := f.engine.IndexDoc(ctx, opts.IndexName, doc); err != nil {
				return nil, fmt.Errorf("error indexing document: %w", err)
			}
		}
	}

	return result, nil
}

// UploadFile processes content from a file path
func (f *FileUpload) UploadFile(ctx context.Context, path string, opts driver.FileUploadOptions) (*driver.FileUploadResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Add filename to metadata if not already set
	if opts.IndexName == "" {
		opts.IndexName = filepath.Base(path)
	}

	return f.Upload(ctx, file, opts)
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
