package graphrag

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// Store key formats (without docID to reduce queries)
const (
	StoreKeyOrigin = "origin_%s" // origin_{docID}
)

// storeSegmentValue stores a value for a segment with the given key format
func (g *GraphRag) storeSegmentValue(docID string, segmentID string, keyFormat string, value interface{}) error {
	if g.Store == nil {
		return fmt.Errorf("store is not configured")
	}

	key := fmt.Sprintf(keyFormat, docID, segmentID)
	err := g.Store.Set(key, value, 0)
	if err != nil {
		return fmt.Errorf("failed to store %s for segment %s: %w", keyFormat, segmentID, err)
	}

	g.Logger.Debugf("Stored %s for segment %s: %v", keyFormat, segmentID, value)
	return nil
}

// getSegmentValue retrieves a value for a segment with the given key format
func (g *GraphRag) getSegmentValue(docID string, segmentID string, keyFormat string) (interface{}, bool) {
	if g.Store == nil {
		return nil, false
	}

	key := fmt.Sprintf(keyFormat, docID, segmentID)
	value, ok := g.Store.Get(key)
	if !ok {
		g.Logger.Debugf("Key %s not found for segment %s", keyFormat, segmentID)
		return nil, false
	}

	return value, true
}

// deleteSegmentValue deletes a value for a segment with the given key format
func (g *GraphRag) deleteSegmentValue(docID string, segmentID string, keyFormat string) error {
	if g.Store == nil {
		return nil // No error if store is not configured
	}

	key := fmt.Sprintf(keyFormat, docID, segmentID)
	err := g.Store.Del(key)
	if err != nil {
		return fmt.Errorf("failed to delete %s for segment %s: %w", keyFormat, segmentID, err)
	}

	g.Logger.Debugf("Deleted %s for segment %s", keyFormat, segmentID)
	return nil
}

// updateSegmentMetadataInVectorBatch updates multiple segment metadata in vector database in batch
func (g *GraphRag) updateSegmentMetadataInVectorBatch(ctx context.Context, docID string, updates []segmentMetadataUpdate) error {
	if g.Vector == nil {
		return fmt.Errorf("vector database is not configured")
	}

	if len(updates) == 0 {
		return nil
	}

	// Extract graphName from docID
	graphName, _ := utils.ExtractCollectionIDFromDocID(docID)
	if graphName == "" {
		graphName = "default"
	}

	collectionIDs, err := utils.GetCollectionIDs(graphName)
	if err != nil {
		return fmt.Errorf("failed to get collection IDs for document %s: %w", docID, err)
	}

	collectionName := collectionIDs.Vector

	// Check if collection exists
	exists, err := g.Vector.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence %s: %w", collectionName, err)
	}
	if !exists {
		return fmt.Errorf("vector collection %s does not exist", collectionName)
	}

	// Prepare metadata updates for the new UpdateMetadata method
	documentUpdates := make([]types.DocumentMetadataUpdate, 0)
	segmentMetadataMap := make(map[string]map[string]interface{})

	// Group updates by segment ID
	for _, update := range updates {
		if segmentMetadataMap[update.SegmentID] == nil {
			segmentMetadataMap[update.SegmentID] = make(map[string]interface{})
		}
		segmentMetadataMap[update.SegmentID][update.MetadataKey] = update.Value
	}

	// Convert to DocumentMetadataUpdate array
	for segmentID, metadata := range segmentMetadataMap {
		documentUpdates = append(documentUpdates, types.DocumentMetadataUpdate{
			DocumentID: segmentID,
			Metadata:   metadata,
		})
	}

	// Use the new UpdateMetadata method for direct metadata updates
	err = g.Vector.UpdateMetadata(ctx, collectionName, documentUpdates, nil)
	if err != nil {
		return fmt.Errorf("failed to update segment metadata in vector store: %w", err)
	}

	g.Logger.Debugf("Updated metadata for %d segments in vector store collection %s", len(documentUpdates), collectionName)

	return nil
}

// segmentMetadataUpdate represents a metadata update for a segment
type segmentMetadataUpdate struct {
	SegmentID   string
	MetadataKey string
	Value       interface{}
}

// structToMap converts any struct to map[string]interface{} for Store operations
func structToMap(data interface{}) (map[string]interface{}, error) {
	// Use JSON marshaling/unmarshaling for reliable conversion
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct to JSON: %w", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	return result, nil
}

// mapToStruct converts map[string]interface{} or any data back to specified struct type
func mapToStruct(data interface{}, target interface{}) error {
	// Use JSON marshaling/unmarshaling for reliable conversion
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	err = json.Unmarshal(jsonData, target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON to struct: %w", err)
	}

	return nil
}

// segmentVoteToMap converts SegmentVote struct to map[string]interface{} for Store operations
func segmentVoteToMap(vote types.SegmentVote) (map[string]interface{}, error) {
	return structToMap(vote)
}

// mapToSegmentVote converts map[string]interface{} back to SegmentVote struct
func mapToSegmentVote(data interface{}) (types.SegmentVote, error) {
	var vote types.SegmentVote
	err := mapToStruct(data, &vote)
	if err != nil {
		return types.SegmentVote{}, err
	}
	return vote, nil
}

// segmentScoreToMap converts SegmentScore struct to map[string]interface{} for Store operations
func segmentScoreToMap(score types.SegmentScore) (map[string]interface{}, error) {
	return structToMap(score)
}

// mapToSegmentScore converts map[string]interface{} back to SegmentScore struct
func mapToSegmentScore(data interface{}) (types.SegmentScore, error) {
	var score types.SegmentScore
	err := mapToStruct(data, &score)
	if err != nil {
		return types.SegmentScore{}, err
	}
	return score, nil
}

// segmentWeightToMap converts SegmentWeight struct to map[string]interface{} for Store operations
func segmentWeightToMap(weight types.SegmentWeight) (map[string]interface{}, error) {
	return structToMap(weight)
}

// mapToSegmentWeight converts map[string]interface{} back to SegmentWeight struct
func mapToSegmentWeight(data interface{}) (types.SegmentWeight, error) {
	var weight types.SegmentWeight
	err := mapToStruct(data, &weight)
	if err != nil {
		return types.SegmentWeight{}, err
	}
	return weight, nil
}

// segmentHitToMap converts SegmentHit struct to map[string]interface{} for Store operations
func segmentHitToMap(hit types.SegmentHit) (map[string]interface{}, error) {
	return structToMap(hit)
}

// mapToSegmentHit converts map[string]interface{} back to SegmentHit struct
func mapToSegmentHit(data interface{}) (types.SegmentHit, error) {
	var hit types.SegmentHit
	err := mapToStruct(data, &hit)
	if err != nil {
		return types.SegmentHit{}, err
	}
	return hit, nil
}
