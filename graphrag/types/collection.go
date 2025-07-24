package types

import (
	"encoding/json"
	"fmt"
)

// SerializeCollection serializes a collection to JSON string
func SerializeCollection(collection Collection) (string, error) {
	data, err := json.Marshal(collection)
	if err != nil {
		return "", fmt.Errorf("failed to serialize collection: %w", err)
	}
	return string(data), nil
}

// SerializeCollectionConfig serializes a collection configuration to JSON string
func SerializeCollectionConfig(collection CollectionConfig) (string, error) {
	data, err := json.Marshal(collection)
	if err != nil {
		return "", fmt.Errorf("failed to serialize collection config: %w", err)
	}
	return string(data), nil
}

// DeserializeCollectionConfig deserializes a JSON string to collection configuration
func DeserializeCollectionConfig(data string) (CollectionConfig, error) {
	var collection CollectionConfig
	err := json.Unmarshal([]byte(data), &collection)
	if err != nil {
		return collection, fmt.Errorf("failed to deserialize collection config: %w", err)
	}
	return collection, nil
}

// DeserializeCollection deserializes a JSON string to collection
func DeserializeCollection(data string) (Collection, error) {
	var collection Collection
	err := json.Unmarshal([]byte(data), &collection)
	if err != nil {
		return collection, fmt.Errorf("failed to deserialize collection: %w", err)
	}
	return collection, nil
}

// ValidateCollectionConfig validates a collection configuration
func ValidateCollectionConfig(collection CollectionConfig) error {
	if collection.ID == "" {
		return fmt.Errorf("collection ID cannot be empty")
	}

	if collection.Config != nil {
		if err := collection.Config.Validate(); err != nil {
			return fmt.Errorf("invalid collection config: %w", err)
		}
	}

	return nil
}

// ValidateCollection validates a collection configuration
func ValidateCollection(collection Collection) error {
	if collection.ID == "" {
		return fmt.Errorf("collection ID cannot be empty")
	}

	// Validate VectorConfig if provided
	if collection.VectorConfig != nil {
		if err := collection.VectorConfig.Validate(); err != nil {
			return fmt.Errorf("invalid vector config: %w", err)
		}
	}

	// Validate CollectionConfig if provided
	if collection.CollectionConfig != nil {
		if err := collection.CollectionConfig.Validate(); err != nil {
			return fmt.Errorf("invalid collection config: %w", err)
		}
	}

	// Validate GraphStoreConfig if provided
	if collection.GraphStoreConfig != nil {
		if err := collection.GraphStoreConfig.Validate(); err != nil {
			return fmt.Errorf("invalid graph store config: %w", err)
		}
	}

	return nil
}

// CloneCollection creates a deep copy of a collection
func CloneCollection(original Collection) (Collection, error) {
	// Use JSON serialization for deep copy
	serialized, err := SerializeCollection(original)
	if err != nil {
		return Collection{}, fmt.Errorf("failed to serialize for cloning: %w", err)
	}

	return DeserializeCollection(serialized)
}
