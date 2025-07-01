package utils

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/graphrag/types"
)

// GenCollectionIDs generates meaningful collection IDs for vector, graph, and KV store databases
// vectorName: base name for the vector collection
// Returns CollectionIDs with Vector, Graph, and Store (Graph and Store use Vector as prefix)
func GenCollectionIDs(vectorName string) types.CollectionIDs {
	// Clean and normalize the vector name
	vectorName = strings.TrimSpace(vectorName)
	if vectorName == "" {
		vectorName = "default"
	}

	// Generate timestamp with nanoseconds for better uniqueness
	timestamp := time.Now().UnixNano()

	// Add random component for extra uniqueness
	randomBytes := make([]byte, 2)
	rand.Read(randomBytes)

	// Create meaningful vector ID with both timestamp and random component
	vectorID := fmt.Sprintf("%s_vector_%d_%x",
		strings.ToLower(strings.ReplaceAll(vectorName, " ", "_")),
		timestamp,
		randomBytes)

	// Create graph ID using vector ID as prefix
	graphID := fmt.Sprintf("%s_graph", vectorID)

	// Create store ID using vector ID as prefix
	storeID := fmt.Sprintf("%s_store", vectorID)

	return types.CollectionIDs{
		Vector: vectorID,
		Graph:  graphID,
		Store:  storeID,
	}
}

// GenDocID generates a UUID for document identification
func GenDocID() string {
	return uuid.New().String()
}

// GenChunkID generates a UUID for chunk identification
func GenChunkID() string {
	return uuid.New().String()
}

// GenShortID generates a shorter ID using timestamp and random bytes
// Useful for cases where shorter IDs are preferred
func GenShortID() string {
	timestamp := time.Now().Unix()

	// Generate 4 random bytes
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)

	return fmt.Sprintf("%x_%x", timestamp, randomBytes)
}

// BatchGenDocIDs generates multiple document IDs at once
func BatchGenDocIDs(count int) []string {
	if count <= 0 {
		return []string{}
	}

	docIDs := make([]string, count)
	for i := 0; i < count; i++ {
		docIDs[i] = GenDocID()
	}
	return docIDs
}

// BatchGenChunkIDs generates multiple chunk IDs at once
func BatchGenChunkIDs(count int) []string {
	if count <= 0 {
		return []string{}
	}

	chunkIDs := make([]string, count)
	for i := 0; i < count; i++ {
		chunkIDs[i] = GenChunkID()
	}
	return chunkIDs
}

// IsValidUUID checks if a string is a valid UUID format
func IsValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
