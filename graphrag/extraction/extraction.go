package extraction

import (
	"context"

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

// LLMOptimizer sets the LLM optimizer to use
func (extra *Extraction) LLMOptimizer(llmOptimizer types.LLMOptimizer) *Extraction {
	extra.Options.LLMOptimizer = llmOptimizer
	return extra
}

// Embedding sets the embedding function to use
func (extra *Extraction) Embedding(embedding types.Embedding) *Extraction {
	extra.Options.Embedding = embedding
	return extra
}

// ExtractDocuments extracts entities and relationships from documents
func (extra *Extraction) ExtractDocuments(ctx context.Context, texts []string, callback ...types.ExtractionProgress) ([]*types.ExtractionResult, error) {
	return extra.Options.Use.ExtractDocuments(ctx, texts, callback...)
}

// ExtractQuery extracts entities and relationships from a query
func (extra *Extraction) ExtractQuery(ctx context.Context, text string, callback ...types.ExtractionProgress) (*types.ExtractionResult, error) {
	return extra.Options.Use.ExtractQuery(ctx, text, callback...)
}

// // Deduplicate deduplicates the extracted entities and relationships
// func (extra *Extraction) Deduplicate(ctx context.Context, entities []types.Node, relationships []types.Relationship, callback ...types.ExtractionProgress) (*types.ExtractionResult, error) {
// 	return nil, nil
// }

// // Optimize optimizes the extracted entities and relationships
// func (extra *Extraction) Optimize(ctx context.Context, entities []types.Node, relationships []types.Relationship, callback ...types.ExtractionProgress) (*types.ExtractionResult, error) {
// 	return nil, nil
// }
