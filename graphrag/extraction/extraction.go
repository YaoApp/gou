package extraction

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/yaoapp/gou/graphrag/types"
)

// Extraction represents the extraction process
type Extraction struct {
	Options types.ExtractionOptions
}

// New creates a new extraction instance
func New(options types.ExtractionOptions) *Extraction {
	return &Extraction{Options: options}
}

// Use sets the extraction method to use
func (extra *Extraction) Use(extraction types.Extraction) *Extraction {
	extra.Options.Use = extraction
	return extra
}

// Embedding sets the embedding function to use
func (extra *Extraction) Embedding(embedding types.Embedding) *Extraction {
	extra.Options.Embedding = embedding
	return extra
}

// ExtractDocuments extracts entities and relationships from documents
func (extra *Extraction) ExtractDocuments(ctx context.Context, texts []string, callback ...types.ExtractionProgress) ([]*types.ExtractionResult, error) {
	if extra.Options.Use == nil {
		return nil, fmt.Errorf("no extraction method configured")
	}
	return extra.Options.Use.ExtractDocuments(ctx, texts, callback...)
}

// ExtractQuery extracts entities and relationships from a query
func (extra *Extraction) ExtractQuery(ctx context.Context, text string, callback ...types.ExtractionProgress) (*types.ExtractionResult, error) {
	if extra.Options.Use == nil {
		return nil, fmt.Errorf("no extraction method configured")
	}
	return extra.Options.Use.ExtractQuery(ctx, text, callback...)
}

// EmbeddingResults embeds the results
func (extra *Extraction) EmbeddingResults(ctx context.Context, results []*types.ExtractionResult, callback ...types.EmbeddingProgress) error {
	if extra.Options.Embedding == nil {
		return nil
	}

	// Step 1: Deduplicate nodes and relationships
	uniqueNodes, uniqueRelationships, err := extra.Deduplicate(ctx, results)
	if err != nil {
		return fmt.Errorf("failed to deduplicate: %w", err)
	}

	// Step 2: Embed deduplicated nodes and relationships
	var errors []string

	// Embed nodes
	if len(uniqueNodes) > 0 {
		err := extra.EmbeddingNodes(ctx, uniqueNodes, callback...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to embed nodes: %v", err))
		}
	}

	// Embed relationships
	if len(uniqueRelationships) > 0 {
		err := extra.EmbeddingRelationships(ctx, uniqueRelationships, callback...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to embed relationships: %v", err))
		}
	}

	// Step 3: Write embeddings back to original results
	// Create maps for quick lookup of embeddings by ID
	nodeEmbeddingMap := make(map[string][]float64)
	for _, node := range uniqueNodes {
		if len(node.EmbeddingVector) > 0 {
			nodeEmbeddingMap[node.ID] = node.EmbeddingVector
		}
	}

	relEmbeddingMap := make(map[string][]float64)
	for _, rel := range uniqueRelationships {
		if len(rel.EmbeddingVector) > 0 {
			relEmbeddingMap[rel.ID] = rel.EmbeddingVector
		}
	}

	// Write back to results
	for _, result := range results {
		if result == nil {
			continue
		}

		// Update node embeddings
		if result.Nodes != nil {
			for i := range result.Nodes {
				if embedding, exists := nodeEmbeddingMap[result.Nodes[i].ID]; exists {
					result.Nodes[i].EmbeddingVector = embedding
				}
			}
		}

		// Update relationship embeddings
		if result.Relationships != nil {
			for i := range result.Relationships {
				if embedding, exists := relEmbeddingMap[result.Relationships[i].ID]; exists {
					result.Relationships[i].EmbeddingVector = embedding
				}
			}
		}
	}

	// Return all collected errors if any
	if len(errors) > 0 {
		return fmt.Errorf("embedding errors occurred: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Deduplicate deduplicates the results
func (extra *Extraction) Deduplicate(ctx context.Context, results []*types.ExtractionResult) ([]types.Node, []types.Relationship, error) {
	if len(results) == 0 {
		return nil, nil, nil
	}

	// Node deduplication
	nodeMap := make(map[string]types.Node) // key -> canonical node
	nodeIDMap := make(map[string]string)   // old ID -> canonical ID

	// First pass: collect all nodes and identify duplicates
	for _, result := range results {
		if result == nil || result.Nodes == nil {
			continue
		}

		for _, node := range result.Nodes {
			key := createNodeKey(node)
			if existing, exists := nodeMap[key]; exists {
				// Found duplicate, map the old ID to canonical ID
				nodeIDMap[node.ID] = existing.ID
			} else {
				// New unique node
				nodeMap[key] = node
				nodeIDMap[node.ID] = node.ID // Map to itself
			}
		}
	}

	// Update node IDs in results
	for _, result := range results {
		if result == nil || result.Nodes == nil {
			continue
		}

		for i := range result.Nodes {
			if canonicalID, exists := nodeIDMap[result.Nodes[i].ID]; exists {
				result.Nodes[i].ID = canonicalID
			}
		}
	}

	// Update relationship node references
	for _, result := range results {
		if result == nil || result.Relationships == nil {
			continue
		}

		for i := range result.Relationships {
			if canonicalID, exists := nodeIDMap[result.Relationships[i].StartNode]; exists {
				result.Relationships[i].StartNode = canonicalID
			}
			if canonicalID, exists := nodeIDMap[result.Relationships[i].EndNode]; exists {
				result.Relationships[i].EndNode = canonicalID
			}
		}
	}

	// Relationship deduplication
	relMap := make(map[string]types.Relationship) // key -> canonical relationship

	// Collect all relationships and identify duplicates
	for _, result := range results {
		if result == nil || result.Relationships == nil {
			continue
		}

		for _, rel := range result.Relationships {
			key := createRelationshipKey(rel)
			if _, exists := relMap[key]; !exists {
				// New unique relationship
				relMap[key] = rel
			}
		}
	}

	// Convert maps to slices
	uniqueNodes := make([]types.Node, 0, len(nodeMap))
	for _, node := range nodeMap {
		uniqueNodes = append(uniqueNodes, node)
	}

	uniqueRelationships := make([]types.Relationship, 0, len(relMap))
	for _, rel := range relMap {
		uniqueRelationships = append(uniqueRelationships, rel)
	}

	return uniqueNodes, uniqueRelationships, nil
}

// EmbeddingNodes embeds the nodes
func (extra *Extraction) EmbeddingNodes(ctx context.Context, nodes []types.Node, callback ...types.EmbeddingProgress) error {
	if extra.Options.Embedding == nil || len(nodes) == 0 {
		return nil
	}

	// Extract node information for embedding
	var nodeTexts []string
	for _, node := range nodes {
		var textParts []string

		// Add Name
		if node.Name != "" {
			textParts = append(textParts, "Name: "+node.Name)
		}

		// Add Type
		if node.Type != "" {
			textParts = append(textParts, "Type: "+node.Type)
		}

		// Add Labels
		if len(node.Labels) > 0 {
			textParts = append(textParts, "Labels: "+strings.Join(node.Labels, ", "))
		}

		// Add Properties Description
		if len(node.Properties) > 0 {
			var propStrings []string
			for key, value := range node.Properties {
				propStrings = append(propStrings, fmt.Sprintf("%s: %v", key, value))
			}
			textParts = append(textParts, "Properties: "+strings.Join(propStrings, "; "))
		}

		// Add Description
		if node.Description != "" {
			textParts = append(textParts, "Description: "+node.Description)
		}

		// Combine all parts
		nodeText := strings.Join(textParts, ". ")
		nodeTexts = append(nodeTexts, nodeText)
	}

	// Embed the node texts
	embeddings, err := extra.Options.Embedding.EmbedDocuments(ctx, nodeTexts, callback...)
	if err != nil {
		return fmt.Errorf("failed to embed nodes: %w", err)
	}

	// Write embeddings back to nodes (modify the slice in place)
	if embeddings != nil && embeddings.IsDense() {
		denseEmbeddings := embeddings.GetDenseEmbeddings()
		for i, embedding := range denseEmbeddings {
			if i < len(nodes) {
				// Note: This modifies the node in place, but since we're working with a slice of values,
				// we need to return the modified nodes or use pointers
				nodes[i].EmbeddingVector = embedding
			}
		}
	}

	return nil
}

// EmbeddingRelationships embeds the relationships
func (extra *Extraction) EmbeddingRelationships(ctx context.Context, relationships []types.Relationship, callback ...types.EmbeddingProgress) error {
	if extra.Options.Embedding == nil || len(relationships) == 0 {
		return nil
	}

	// Extract relationship information for embedding
	var relationshipTexts []string
	for _, rel := range relationships {
		var textParts []string

		// Add Type
		if rel.Type != "" {
			textParts = append(textParts, "Type: "+rel.Type)
		}

		// Add StartNode and EndNode
		if rel.StartNode != "" && rel.EndNode != "" {
			textParts = append(textParts, fmt.Sprintf("From: %s To: %s", rel.StartNode, rel.EndNode))
		}

		// Add Properties
		if len(rel.Properties) > 0 {
			var propStrings []string
			for key, value := range rel.Properties {
				propStrings = append(propStrings, fmt.Sprintf("%s: %v", key, value))
			}
			textParts = append(textParts, "Properties: "+strings.Join(propStrings, "; "))
		}

		// Add Description
		if rel.Description != "" {
			textParts = append(textParts, "Description: "+rel.Description)
		}

		// Combine all parts
		relText := strings.Join(textParts, ". ")
		relationshipTexts = append(relationshipTexts, relText)
	}

	// Embed the relationship texts
	embeddings, err := extra.Options.Embedding.EmbedDocuments(ctx, relationshipTexts, callback...)
	if err != nil {
		return fmt.Errorf("failed to embed relationships: %w", err)
	}

	// Write embeddings back to relationships (modify the slice in place)
	if embeddings != nil && embeddings.IsDense() {
		denseEmbeddings := embeddings.GetDenseEmbeddings()
		for i, embedding := range denseEmbeddings {
			if i < len(relationships) {
				relationships[i].EmbeddingVector = embedding
			}
		}
	}

	return nil
}

// normalizeProperties creates a normalized string representation of properties for comparison
func normalizeProperties(props map[string]interface{}) string {
	if len(props) == 0 {
		return ""
	}

	// Sort keys to ensure consistent ordering
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build normalized string
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%v", k, props[k]))
	}
	return strings.Join(parts, "|")
}

// createNodeKey creates a unique key for node deduplication
func createNodeKey(node types.Node) string {
	var parts []string

	parts = append(parts, "name:"+node.Name)
	parts = append(parts, "type:"+node.Type)

	// Sort labels for consistent comparison
	if len(node.Labels) > 0 {
		labels := make([]string, len(node.Labels))
		copy(labels, node.Labels)
		sort.Strings(labels)
		parts = append(parts, "labels:"+strings.Join(labels, ","))
	}

	// Add normalized properties
	if len(node.Properties) > 0 {
		parts = append(parts, "props:"+normalizeProperties(node.Properties))
	}

	return strings.Join(parts, "|")
}

// createRelationshipKey creates a unique key for relationship deduplication
func createRelationshipKey(rel types.Relationship) string {
	var parts []string

	parts = append(parts, "type:"+rel.Type)
	parts = append(parts, "start:"+rel.StartNode)
	parts = append(parts, "end:"+rel.EndNode)

	// Add normalized properties
	if len(rel.Properties) > 0 {
		parts = append(parts, "props:"+normalizeProperties(rel.Properties))
	}

	return strings.Join(parts, "|")
}
