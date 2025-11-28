package process

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/mcp/types"
)

// ListSamples lists available training samples for a tool or resource from .jsonl files
// itemType: types.SampleTool or types.SampleResource
func (c *Client) ListSamples(ctx context.Context, itemType types.SampleItemType, itemName string) (*types.ListSamplesResponse, error) {
	// Convert DSL.ID dots to slashes (e.g., "foo.bar.hi" → "foo/bar/hi")
	clientPath := strings.ReplaceAll(c.DSL.ID, ".", "/")

	// Build path based on item type
	var samplePath string
	if itemType == types.SampleResource {
		// For resources: mcps/mapping/{client_id}/resources/{resource_name}.jsonl
		samplePath = filepath.Join("mcps", "mapping", clientPath, "resources", itemName+".jsonl")
	} else {
		// For tools: mcps/mapping/{client_id}/schemes/{tool_name}.jsonl
		samplePath = filepath.Join("mcps", "mapping", clientPath, "schemes", itemName+".jsonl")
	}

	// Check if file exists
	exists, err := application.App.Exists(samplePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check sample file: %w", err)
	}

	if !exists {
		// No samples available for this tool
		return &types.ListSamplesResponse{
			Samples: []types.SampleData{},
			Total:   0,
		}, nil
	}

	// Read the .jsonl file
	data, err := application.App.Read(samplePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sample file %s: %w", samplePath, err)
	}

	// Parse .jsonl file (each line is a JSON object)
	samples := []types.SampleData{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	index := 0

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue // Skip empty lines
		}

		var sample types.SampleData
		if err := json.Unmarshal([]byte(line), &sample); err != nil {
			// Log warning but continue processing other samples
			continue
		}

		// Set index and item name (tool or resource)
		sample.Index = index
		sample.ItemName = itemName
		samples = append(samples, sample)
		index++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan sample file: %w", err)
	}

	return &types.ListSamplesResponse{
		Samples: samples,
		Total:   len(samples),
	}, nil
}

// GetSample retrieves a specific sample by index
// itemType: types.SampleTool or types.SampleResource
func (c *Client) GetSample(ctx context.Context, itemType types.SampleItemType, itemName string, index int) (*types.SampleData, error) {
	if index < 0 {
		return nil, fmt.Errorf("invalid sample index: %d", index)
	}

	// Convert DSL.ID dots to slashes (e.g., "foo.bar.hi" → "foo/bar/hi")
	clientPath := strings.ReplaceAll(c.DSL.ID, ".", "/")

	// Build path based on item type
	var samplePath string
	if itemType == types.SampleResource {
		// For resources: mcps/mapping/{client_id}/resources/{resource_name}.jsonl
		samplePath = filepath.Join("mcps", "mapping", clientPath, "resources", itemName+".jsonl")
	} else {
		// For tools: mcps/mapping/{client_id}/schemes/{tool_name}.jsonl
		samplePath = filepath.Join("mcps", "mapping", clientPath, "schemes", itemName+".jsonl")
	}

	// Check if file exists
	exists, err := application.App.Exists(samplePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check sample file: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("no samples found for %s: %s", itemType, itemName)
	}

	// Read the .jsonl file
	data, err := application.App.Read(samplePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sample file %s: %w", samplePath, err)
	}

	// Parse .jsonl file and find the sample at the specified index
	scanner := bufio.NewScanner(bytes.NewReader(data))
	currentIndex := 0

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue // Skip empty lines
		}

		if currentIndex == index {
			var sample types.SampleData
			if err := json.Unmarshal([]byte(line), &sample); err != nil {
				return nil, fmt.Errorf("failed to parse sample at index %d: %w", index, err)
			}

			// Set index and item name
			sample.Index = index
			sample.ItemName = itemName
			return &sample, nil
		}

		currentIndex++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan sample file: %w", err)
	}

	return nil, fmt.Errorf("sample not found at index %d (total samples: %d)", index, currentIndex)
}
