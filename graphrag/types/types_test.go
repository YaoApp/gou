package types

import (
	"testing"
)

func TestVectorStoreConfig_ValidateSparseVectors(t *testing.T) {
	tests := []struct {
		name    string
		config  VectorStoreConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid sparse vector config with different names",
			config: VectorStoreConfig{
				Dimension:           128,
				Distance:            DistanceCosine,
				IndexType:           IndexTypeHNSW,
				CollectionName:      "test_collection",
				EnableSparseVectors: true,
				DenseVectorName:     "dense_vec",
				SparseVectorName:    "sparse_vec",
			},
			wantErr: false,
		},
		{
			name: "valid sparse vector config with default names",
			config: VectorStoreConfig{
				Dimension:           128,
				Distance:            DistanceCosine,
				IndexType:           IndexTypeHNSW,
				CollectionName:      "test_collection",
				EnableSparseVectors: true,
				// DenseVectorName and SparseVectorName are empty, should use defaults
			},
			wantErr: false,
		},
		{
			name: "invalid sparse vector config - same names",
			config: VectorStoreConfig{
				Dimension:           128,
				Distance:            DistanceCosine,
				IndexType:           IndexTypeHNSW,
				CollectionName:      "test_collection",
				EnableSparseVectors: true,
				DenseVectorName:     "same_name",
				SparseVectorName:    "same_name",
			},
			wantErr: true,
			errMsg:  "dense and sparse vector names cannot be the same: same_name",
		},
		{
			name: "invalid sparse vector config - one empty defaults to same as other",
			config: VectorStoreConfig{
				Dimension:           128,
				Distance:            DistanceCosine,
				IndexType:           IndexTypeHNSW,
				CollectionName:      "test_collection",
				EnableSparseVectors: true,
				DenseVectorName:     "sparse", // This will conflict with default sparse name
				SparseVectorName:    "",       // This will default to "sparse"
			},
			wantErr: true,
			errMsg:  "dense and sparse vector names cannot be the same: sparse",
		},
		{
			name: "sparse vectors disabled - no validation needed",
			config: VectorStoreConfig{
				Dimension:           128,
				Distance:            DistanceCosine,
				IndexType:           IndexTypeHNSW,
				CollectionName:      "test_collection",
				EnableSparseVectors: false,
				DenseVectorName:     "same_name",
				SparseVectorName:    "same_name", // Should not cause error when sparse vectors disabled
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}
