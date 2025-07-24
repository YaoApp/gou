package graphrag

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
)

// TestUpdateStoreFunctions tests UpdateVote, UpdateScore, UpdateWeight with vector and vector+store configurations
func TestUpdateStoreFunctions(t *testing.T) {
	// Setup connector as in document_test.go
	prepareAddFileConnector(t)

	configs := GetTestConfigs()
	testConfigs := []string{"vector", "vector+store"}

	for _, configName := range testConfigs {
		config := configs[configName]
		if config == nil {
			t.Skipf("Config %s not found", configName)
		}

		t.Run(fmt.Sprintf("Config_%s", configName), func(t *testing.T) {
			// Create GraphRag instance
			g, err := New(config)
			if err != nil {
				t.Skipf("Failed to create GraphRag instance for %s: %v", configName, err)
			}

			ctx := context.Background()

			// Create collection using utility from collection_test.go
			// Replace + with _ to make collection name valid
			safeName := strings.ReplaceAll(configName, "+", "_")
			collectionID := fmt.Sprintf("update_store_collection_%s_%d", safeName, time.Now().Unix())
			collection := types.CollectionConfig{
				ID: collectionID,
				Metadata: map[string]interface{}{
					"type": "update_store_test",
				},
				Config: &types.CreateCollectionOptions{
					CollectionName: fmt.Sprintf("%s_vector", collectionID),
					Dimension:      1536,
					Distance:       types.DistanceCosine,
					IndexType:      types.IndexTypeHNSW,
				},
			}

			// Create collection
			_, err = g.CreateCollection(ctx, collection)
			if err != nil {
				t.Skipf("Failed to create test collection for %s: %v", configName, err)
			}

			// Cleanup collection after test
			defer func() {
				removed, err := g.RemoveCollection(ctx, collectionID)
				if err != nil {
					t.Logf("Warning: Failed to cleanup collection %s: %v", collectionID, err)
				} else if removed {
					t.Logf("Successfully cleaned up collection: %s", collectionID)
				}
			}()

			// Test UpdateVote
			t.Run("Update_Vote", func(t *testing.T) {
				// Test with empty arrays
				votes := []types.SegmentVote{}
				updatedCount, err := g.UpdateVote(ctx, votes)
				if err != nil {
					t.Errorf("Config %s: UpdateVote with empty array should not fail: %v", configName, err)
				}
				if updatedCount != 0 {
					t.Errorf("Config %s: Expected 0 updated votes, got %d", configName, updatedCount)
				}

				// Test with nil array
				updatedCount, err = g.UpdateVote(ctx, nil)
				if err != nil {
					t.Errorf("Config %s: UpdateVote with nil array should not fail: %v", configName, err)
				}
				if updatedCount != 0 {
					t.Errorf("Config %s: Expected 0 updated votes, got %d", configName, updatedCount)
				}

				t.Logf("Config %s: UpdateVote tests passed", configName)
			})

			// Test UpdateScore
			t.Run("Update_Score", func(t *testing.T) {
				scores := []types.SegmentScore{}
				updatedCount, err := g.UpdateScore(ctx, scores)
				if err != nil {
					t.Errorf("Config %s: UpdateScore with empty array should not fail: %v", configName, err)
				}
				if updatedCount != 0 {
					t.Errorf("Config %s: Expected 0 updated scores, got %d", configName, updatedCount)
				}

				// Test with nil array
				updatedCount, err = g.UpdateScore(ctx, nil)
				if err != nil {
					t.Errorf("Config %s: UpdateScore with nil array should not fail: %v", configName, err)
				}
				if updatedCount != 0 {
					t.Errorf("Config %s: Expected 0 updated scores, got %d", configName, updatedCount)
				}

				t.Logf("Config %s: UpdateScore tests passed", configName)
			})

			// Test UpdateWeight
			t.Run("Update_Weight", func(t *testing.T) {
				weights := []types.SegmentWeight{}
				updatedCount, err := g.UpdateWeight(ctx, weights)
				if err != nil {
					t.Errorf("Config %s: UpdateWeight with empty array should not fail: %v", configName, err)
				}
				if updatedCount != 0 {
					t.Errorf("Config %s: Expected 0 updated weights, got %d", configName, updatedCount)
				}

				// Test with nil array
				updatedCount, err = g.UpdateWeight(ctx, nil)
				if err != nil {
					t.Errorf("Config %s: UpdateWeight with nil array should not fail: %v", configName, err)
				}
				if updatedCount != 0 {
					t.Errorf("Config %s: Expected 0 updated weights, got %d", configName, updatedCount)
				}

				t.Logf("Config %s: UpdateWeight tests passed", configName)
			})

			// Verify storage strategy
			t.Run("Storage_Strategy", func(t *testing.T) {
				if strings.Contains(configName, "store") {
					if g.Store == nil {
						t.Errorf("Config %s: Store should be configured but it is not", configName)
					} else {
						t.Logf("Config %s: Store is configured - using dual storage strategy", configName)
					}
				} else {
					if g.Store != nil {
						t.Errorf("Config %s: Store should not be configured but it is", configName)
					} else {
						t.Logf("Config %s: Store is not configured - using Vector DB only strategy", configName)
					}
				}

				if g.Vector == nil {
					t.Errorf("Config %s: Vector should always be configured", configName)
				} else {
					t.Logf("Config %s: Vector is configured", configName)
				}
			})
		})
	}
}
