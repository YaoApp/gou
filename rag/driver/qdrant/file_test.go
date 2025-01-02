package qdrant

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/rag/driver"
)

func TestFileUploader_Upload(t *testing.T) {
	// Create a test engine
	engine, err := NewEngine(getTestConfig(t))
	assert.NoError(t, err)
	defer engine.Close()

	indexName := fmt.Sprintf("test_upload_%d", time.Now().UnixNano())

	// Create a test index
	err = engine.CreateIndex(context.Background(), driver.IndexConfig{
		Name:   indexName,
		Driver: "qdrant",
	})
	assert.NoError(t, err)
	defer engine.DeleteIndex(context.Background(), indexName)

	uploader, err := NewFileUpload(engine, engine.vectorizer)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		content string
		opts    driver.FileUploadOptions
		wantErr bool
	}{
		{
			name:    "Basic upload",
			content: "This is a test document",
			opts: driver.FileUploadOptions{
				IndexName: indexName,
				ChunkSize: 100,
			},
			wantErr: false,
		},
		{
			name:    "Async upload",
			content: "This is an async test document",
			opts: driver.FileUploadOptions{
				IndexName: indexName,
				ChunkSize: 100,
				Async:     true,
			},
			wantErr: false,
		},
		{
			name:    "Upload with chunks",
			content: strings.Repeat("Test content ", 100),
			opts: driver.FileUploadOptions{
				IndexName:    indexName,
				ChunkSize:    100,
				ChunkOverlap: 20,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			result, err := uploader.Upload(context.Background(), reader, tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Greater(t, len(result.Documents), 0)

			if tt.opts.Async {
				assert.NotEmpty(t, result.TaskID)
				// Check task status
				taskInfo, err := engine.GetTaskInfo(context.Background(), result.TaskID)
				assert.NoError(t, err)
				assert.NotNil(t, taskInfo)
			}
		})
	}
}

func TestFileUploader_UploadFile(t *testing.T) {
	// Create a test engine
	engine, err := NewEngine(getTestConfig(t))
	assert.NoError(t, err)
	defer engine.Close()

	indexName := fmt.Sprintf("test_file_upload_%d", time.Now().UnixNano())

	// Create a test index
	err = engine.CreateIndex(context.Background(), driver.IndexConfig{
		Name:   indexName,
		Driver: "qdrant",
	})
	assert.NoError(t, err)
	defer engine.DeleteIndex(context.Background(), indexName)

	uploader, err := NewFileUpload(engine, engine.vectorizer)
	assert.NoError(t, err)

	// Create a temporary test file
	content := "This is a test file content\nWith multiple lines\nFor testing purposes"
	tmpfile, err := os.CreateTemp("", "test_upload_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)

	tests := []struct {
		name    string
		opts    driver.FileUploadOptions
		wantErr bool
	}{
		{
			name: "Basic file upload",
			opts: driver.FileUploadOptions{
				IndexName: indexName,
				ChunkSize: 100,
			},
			wantErr: false,
		},
		{
			name: "Async file upload",
			opts: driver.FileUploadOptions{
				IndexName: indexName,
				ChunkSize: 100,
				Async:     true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := uploader.UploadFile(context.Background(), tmpfile.Name(), tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Greater(t, len(result.Documents), 0)

			if tt.opts.Async {
				assert.NotEmpty(t, result.TaskID)
				// Check task status
				taskInfo, err := engine.GetTaskInfo(context.Background(), result.TaskID)
				assert.NoError(t, err)
				assert.NotNil(t, taskInfo)
			}
		})
	}
}
