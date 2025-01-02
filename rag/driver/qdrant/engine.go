package qdrant

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"github.com/yaoapp/gou/rag/driver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// stringToUint64ID converts a string ID to uint64 using FNV-1a hash
func stringToUint64ID(s string) uint64 {
	h := uint64(14695981039346656037) // FNV offset basis
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211 // FNV prime
	}
	return h
}

// Engine implements the driver.Engine interface using Qdrant as the vector store backend
type Engine struct {
	client     *qdrant.Client
	vectorizer driver.Vectorizer
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	closed     bool
	mu         sync.RWMutex // Protects the closed field
}

// Config represents the configuration options for the Qdrant engine.
// For local storage, run a local Qdrant server and point Host to "localhost".
// The data will be stored in the Qdrant server's configured storage path.
type Config struct {
	Host       string // Host address of Qdrant server, e.g., "localhost"
	Port       uint32 // Port number of Qdrant server, default is 6334 for gRPC
	APIKey     string // Optional API key for authentication
	Vectorizer driver.Vectorizer
}

// NewEngine creates a new instance of the Qdrant engine with the given configuration
func NewEngine(config Config) (*Engine, error) {
	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Configure gRPC connection with keepalive
	keepaliveParams := grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	})

	// Create Qdrant client
	client, err := qdrant.NewClient(
		&qdrant.Config{
			Host:   config.Host,
			Port:   int(config.Port),
			APIKey: config.APIKey,
			GrpcOptions: []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				keepaliveParams,
			},
		},
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	return &Engine{
		client:     client,
		vectorizer: config.Vectorizer,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// CreateIndex creates a new vector collection in Qdrant with the given configuration
func (e *Engine) CreateIndex(ctx context.Context, config driver.IndexConfig) error {
	// Get vector dimension from vectorizer
	dims, err := e.getVectorDimension()
	if err != nil {
		return fmt.Errorf("failed to get vector dimension: %w", err)
	}

	// Create collection
	params := &qdrant.VectorParams{
		Size:     uint64(dims),
		Distance: qdrant.Distance_Cosine,
	}

	err = e.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: config.Name,
		VectorsConfig:  qdrant.NewVectorsConfig(params),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	return nil
}

// DeleteIndex removes a vector collection from Qdrant
func (e *Engine) DeleteIndex(ctx context.Context, name string) error {
	err := e.client.DeleteCollection(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	return nil
}

// ListIndexes returns a list of all vector collection names in Qdrant
func (e *Engine) ListIndexes(ctx context.Context) ([]string, error) {
	collections, err := e.client.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return collections, nil
}

// IndexDoc adds or updates a document in the specified vector collection
func (e *Engine) IndexDoc(ctx context.Context, indexName string, doc *driver.Document) error {
	if err := e.checkContext(ctx); err != nil {
		return err
	}

	var embeddings []float32
	var err error

	if doc.Embeddings == nil {
		embeddings, err = e.vectorizer.Vectorize(ctx, doc.Content)
		if err != nil {
			return fmt.Errorf("failed to vectorize document: %w", err)
		}
	} else {
		embeddings = doc.Embeddings
		// Check vector dimension
		dims, err := e.getVectorDimension()
		if err != nil {
			return fmt.Errorf("failed to get vector dimension: %w", err)
		}
		if len(embeddings) != dims {
			return fmt.Errorf("dimension mismatch: expected %d, got %d", dims, len(embeddings))
		}
	}

	point := &qdrant.PointStruct{
		Id:      qdrant.NewIDNum(stringToUint64ID(doc.DocID)),
		Vectors: qdrant.NewVectors(embeddings...),
		Payload: map[string]*qdrant.Value{
			"content":     qdrant.NewValueString(doc.Content),
			"original_id": qdrant.NewValueString(doc.DocID),
		},
	}

	if doc.Metadata != nil {
		payload := make(map[string]*qdrant.Value)
		for k, v := range doc.Metadata {
			switch val := v.(type) {
			case string:
				payload[k] = qdrant.NewValueString(val)
			case float64:
				payload[k] = qdrant.NewValueDouble(val)
			case bool:
				payload[k] = qdrant.NewValueBool(val)
			case []string:
				values := make([]*qdrant.Value, len(val))
				for i, s := range val {
					values[i] = qdrant.NewValueString(s)
				}
				payload[k] = &qdrant.Value{
					Kind: &qdrant.Value_ListValue{
						ListValue: &qdrant.ListValue{
							Values: values,
						},
					},
				}
			case map[string]interface{}:
				if nested, err := qdrant.NewStruct(val); err == nil {
					payload[k] = qdrant.NewValueStruct(nested)
				}
			}
		}
		point.Payload["metadata"] = qdrant.NewValueStruct(&qdrant.Struct{Fields: payload})
	}

	_, err = e.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: indexName,
		Points:         []*qdrant.PointStruct{point},
	})
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	return nil
}

// Search performs a vector similarity search in the specified collection
func (e *Engine) Search(ctx context.Context, indexName string, vector []float32, opts driver.VectorSearchOptions) ([]driver.SearchResult, error) {
	if err := e.checkContext(ctx); err != nil {
		return nil, err
	}

	// Create a new context with timeout
	searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	e.wg.Add(1)
	defer e.wg.Done()

	var searchVector []float32
	var err error

	if vector == nil && opts.QueryText != "" {
		searchVector, err = e.vectorizer.Vectorize(searchCtx, opts.QueryText)
		if err != nil {
			return nil, fmt.Errorf("failed to vectorize query: %w", err)
		}
	} else {
		searchVector = vector
	}

	points, err := e.client.Query(searchCtx, &qdrant.QueryPoints{
		CollectionName: indexName,
		Query:          qdrant.NewQuery(searchVector...),
		Limit:          qdrant.PtrOf(uint64(opts.TopK)),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	results := make([]driver.SearchResult, len(points))
	for i, point := range points {
		content := point.Payload["content"].GetStringValue()
		originalID := point.Payload["original_id"].GetStringValue()
		var metadata map[string]interface{}
		if metadataValue := point.Payload["metadata"]; metadataValue != nil {
			if metadataStruct := metadataValue.GetStructValue(); metadataStruct != nil {
				metadata = convertStructToMap(metadataStruct)
			}
		}

		results[i] = driver.SearchResult{
			DocID:    originalID,
			Score:    float64(point.Score),
			Content:  content,
			Metadata: metadata,
		}
	}
	return results, nil
}

// GetDocument retrieves a document by its ID from the specified collection
func (e *Engine) GetDocument(ctx context.Context, indexName string, DocID string) (*driver.Document, error) {
	points, err := e.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: indexName,
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(stringToUint64ID(DocID))},
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(true),
	})
	if err != nil {
		if strings.Contains(err.Error(), "doesn't exist") {
			return nil, fmt.Errorf("collection doesn't exist: %w", err)
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("document not found")
	}

	point := points[0]
	content := point.Payload["content"].GetStringValue()
	originalID := point.Payload["original_id"].GetStringValue()
	var metadata map[string]interface{}
	if metadataValue := point.Payload["metadata"]; metadataValue != nil {
		if metadataStruct := metadataValue.GetStructValue(); metadataStruct != nil {
			metadata = convertStructToMap(metadataStruct)
		}
	}

	return &driver.Document{
		DocID:      originalID,
		Content:    content,
		Metadata:   metadata,
		Embeddings: point.Vectors.GetVector().Data,
	}, nil
}

// Close releases any resources held by the engine
func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	e.mu.Unlock()

	var errs []error

	// Cancel context to stop any ongoing operations
	if e.cancel != nil {
		e.cancel()
	}

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines finished
	case <-time.After(10 * time.Second): // Increase timeout duration
		errs = append(errs, fmt.Errorf("timeout waiting for goroutines to finish"))
	}

	// Close client first
	if e.client != nil {
		if err := e.client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close qdrant client: %w", err))
		}
		e.client = nil
	}

	// Close vectorizer
	if e.vectorizer != nil {
		if err := e.vectorizer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close vectorizer: %w", err))
		}
		e.vectorizer = nil
	}

	// Return combined errors if any
	if len(errs) > 0 {
		errMsgs := make([]string, len(errs))
		for i, err := range errs {
			errMsgs[i] = err.Error()
		}
		return fmt.Errorf("multiple errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// Helper function to get vector dimension from vectorizer
func (e *Engine) getVectorDimension() (int, error) {
	// We'll vectorize an empty string to get the dimension
	vec, err := e.vectorizer.Vectorize(context.Background(), "")
	if err != nil {
		return 0, err
	}
	return len(vec), nil
}

// Helper function to convert Qdrant Struct to map
func convertStructToMap(s *qdrant.Struct) map[string]interface{} {
	if s == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range s.Fields {
		switch x := v.Kind.(type) {
		case *qdrant.Value_StringValue:
			result[k] = x.StringValue
		case *qdrant.Value_DoubleValue:
			result[k] = x.DoubleValue
		case *qdrant.Value_BoolValue:
			result[k] = x.BoolValue
		case *qdrant.Value_ListValue:
			if x.ListValue != nil {
				list := make([]interface{}, len(x.ListValue.Values))
				for i, lv := range x.ListValue.Values {
					switch lx := lv.Kind.(type) {
					case *qdrant.Value_StringValue:
						list[i] = lx.StringValue
					case *qdrant.Value_DoubleValue:
						list[i] = lx.DoubleValue
					case *qdrant.Value_BoolValue:
						list[i] = lx.BoolValue
					case *qdrant.Value_StructValue:
						list[i] = convertStructToMap(lx.StructValue)
					}
				}
				result[k] = list
			}
		case *qdrant.Value_StructValue:
			result[k] = convertStructToMap(x.StructValue)
		}
	}
	return result
}

// IndexBatch adds or updates multiple documents in batch
func (e *Engine) IndexBatch(ctx context.Context, indexName string, docs []*driver.Document) (string, error) {
	if err := e.checkContext(ctx); err != nil {
		return "", err
	}

	if len(docs) == 0 {
		return "", fmt.Errorf("empty document batch")
	}

	// Create a new context with timeout
	indexCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	e.wg.Add(1)
	defer e.wg.Done()

	points := make([]*qdrant.PointStruct, len(docs))
	for i, doc := range docs {
		var embeddings []float32
		var err error

		if doc.Embeddings == nil {
			embeddings, err = e.vectorizer.Vectorize(indexCtx, doc.Content)
			if err != nil {
				return "", fmt.Errorf("failed to vectorize document %s: %w", doc.DocID, err)
			}
		} else {
			embeddings = doc.Embeddings
		}

		point := &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(stringToUint64ID(doc.DocID)),
			Vectors: qdrant.NewVectors(embeddings...),
			Payload: map[string]*qdrant.Value{
				"content":     qdrant.NewValueString(doc.Content),
				"original_id": qdrant.NewValueString(doc.DocID),
			},
		}

		if doc.Metadata != nil {
			metadataStruct, err := qdrant.NewStruct(doc.Metadata)
			if err != nil {
				return "", fmt.Errorf("failed to convert metadata for document %s: %w", doc.DocID, err)
			}
			point.Payload["metadata"] = qdrant.NewValueStruct(metadataStruct)
		}

		points[i] = point
	}

	// Perform batch upsert
	_, err := e.client.Upsert(indexCtx, &qdrant.UpsertPoints{
		CollectionName: indexName,
		Points:         points,
		Wait:           qdrant.PtrOf(false), // Async operation
	})
	if err != nil {
		return "", fmt.Errorf("failed to batch index documents: %w", err)
	}

	// Generate a unique task ID since Qdrant doesn't provide one
	taskID := fmt.Sprintf("batch-index-%d", time.Now().UnixNano())
	return taskID, nil
}

// DeleteDoc removes a document from the specified collection
func (e *Engine) DeleteDoc(ctx context.Context, indexName string, DocID string) error {
	_, err := e.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: indexName,
		Wait:           qdrant.PtrOf(true), // Synchronous operation
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{qdrant.NewIDNum(stringToUint64ID(DocID))},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

// DeleteBatch removes multiple documents in batch
func (e *Engine) DeleteBatch(ctx context.Context, indexName string, DocIDs []string) (string, error) {
	if err := e.checkContext(ctx); err != nil {
		return "", err
	}

	if len(DocIDs) == 0 {
		return "", fmt.Errorf("empty batch")
	}

	e.wg.Add(1)
	defer e.wg.Done()

	pointIDs := make([]*qdrant.PointId, len(DocIDs))
	for i, id := range DocIDs {
		pointIDs[i] = qdrant.NewIDNum(stringToUint64ID(id))
	}

	_, err := e.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: indexName,
		Wait:           qdrant.PtrOf(false), // Async operation
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: pointIDs,
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to batch delete documents: %w", err)
	}

	// Generate a unique task ID since Qdrant doesn't provide one
	taskID := fmt.Sprintf("batch-delete-%d", time.Now().UnixNano())
	return taskID, nil
}

// GetTaskInfo retrieves information about an asynchronous task
func (e *Engine) GetTaskInfo(ctx context.Context, taskID string) (*driver.TaskInfo, error) {
	// Parse the task type and timestamp from the taskID
	parts := strings.Split(taskID, "-")
	if len(parts) != 3 || (parts[0] != "batch" && parts[1] != "index" && parts[1] != "delete") {
		return nil, fmt.Errorf("invalid task ID format")
	}

	timestamp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid task ID format")
	}

	// For now, we'll consider tasks as completed after 5 seconds
	elapsed := time.Since(time.Unix(0, timestamp))
	status := driver.StatusRunning
	if elapsed > 5*time.Second {
		status = driver.StatusComplete
	}

	return &driver.TaskInfo{
		TaskID:    taskID,
		Status:    status,
		Created:   timestamp,
		Updated:   time.Now().UnixNano(),
		Total:     0, // We don't track these metrics yet
		Processed: 0,
		Failed:    0,
	}, nil
}

// ListTasks returns a list of all tasks for the specified collection
func (e *Engine) ListTasks(ctx context.Context, indexName string) ([]*driver.TaskInfo, error) {
	// Since Qdrant doesn't provide direct task listing,
	// we'll return an empty list for now
	return []*driver.TaskInfo{}, nil
}

// CancelTask cancels an ongoing asynchronous task
func (e *Engine) CancelTask(ctx context.Context, taskID string) error {
	// Since Qdrant doesn't provide direct task cancellation,
	// we'll just return success for now
	return nil
}

// checkContext checks if the engine is closed and context is valid
func (e *Engine) checkContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("nil context")
	}

	e.mu.RLock()
	closed := e.closed
	e.mu.RUnlock()
	if closed {
		return fmt.Errorf("engine is closed")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.ctx.Done():
		return fmt.Errorf("engine context is cancelled")
	default:
		return nil
	}
}

// HasDocument checks if a document exists in the specified collection
func (e *Engine) HasDocument(ctx context.Context, indexName string, DocID string) (bool, error) {
	if err := e.checkContext(ctx); err != nil {
		return false, err
	}

	points, err := e.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: indexName,
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(stringToUint64ID(DocID))},
		WithPayload:    qdrant.NewWithPayload(false), // We don't need payload, just checking existence
	})
	if err != nil {
		if strings.Contains(err.Error(), "doesn't exist") {
			return false, nil // Collection doesn't exist
		}
		return false, fmt.Errorf("failed to check document existence: %w", err)
	}

	return len(points) > 0, nil
}

// HasIndex checks if a collection exists
func (e *Engine) HasIndex(ctx context.Context, name string) (bool, error) {
	if err := e.checkContext(ctx); err != nil {
		return false, err
	}

	collections, err := e.client.ListCollections(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list collections: %w", err)
	}

	for _, collection := range collections {
		if collection == name {
			return true, nil
		}
	}
	return false, nil
}

// GetMetadata retrieves only the metadata of a document by its ID from the specified collection
func (e *Engine) GetMetadata(ctx context.Context, indexName string, DocID string) (map[string]interface{}, error) {
	if err := e.checkContext(ctx); err != nil {
		return nil, err
	}

	points, err := e.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: indexName,
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(stringToUint64ID(DocID))},
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(false), // Don't fetch vectors to save memory
	})
	if err != nil {
		if strings.Contains(err.Error(), "doesn't exist") {
			return nil, fmt.Errorf("collection doesn't exist: %w", err)
		}
		return nil, fmt.Errorf("failed to get document metadata: %w", err)
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("document not found")
	}

	point := points[0]
	if metadataValue := point.Payload["metadata"]; metadataValue != nil {
		if metadataStruct := metadataValue.GetStructValue(); metadataStruct != nil {
			return convertStructToMap(metadataStruct), nil
		}
	}

	return make(map[string]interface{}), nil
}
