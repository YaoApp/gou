package utils

import (
	"reflect"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/graphrag/types"
)

func TestParseSemanticToolcall(t *testing.T) {
	parser := NewSemanticParser(true)

	// Test toolcall streaming chunks
	testChunks := []string{
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"segments\": ["}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"s\": 0, \"e\": 50},"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"s\": 50, \"e\": 100}"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"]}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	}

	var positions []types.Position
	var err error
	for _, chunk := range testChunks {
		positions, err = parser.parseSemanticToolcall([]byte(chunk))
		if err != nil {
			t.Errorf("Failed to parse toolcall chunk: %v", err)
		}
	}

	if len(positions) != 2 {
		t.Errorf("Expected 2 positions, got %d", len(positions))
	}

	expectedPositions := []types.Position{
		{StartPos: 0, EndPos: 50},
		{StartPos: 50, EndPos: 100},
	}

	for i, pos := range positions {
		if pos.StartPos != expectedPositions[i].StartPos || pos.EndPos != expectedPositions[i].EndPos {
			t.Errorf("Position %d mismatch: expected %+v, got %+v", i, expectedPositions[i], pos)
		}
	}
}

func TestParseSemanticRegular(t *testing.T) {
	parser := NewSemanticParser(false)

	// Test regular content streaming chunks with proper JSON format
	testChunks := []string{
		`{"choices":[{"delta":{"content":"Here are the segments:\n["}}]}`,
		`{"choices":[{"delta":{"content":"{\"s\": 0, \"e\": 50},"}}]}`,
		`{"choices":[{"delta":{"content":"{\"s\": 50, \"e\": 100}"}}]}`,
		`{"choices":[{"delta":{"content":"]\n\nThese segments represent..."}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
	}

	var err error
	for _, chunk := range testChunks {
		_, err = parser.parseSemanticRegular([]byte(chunk))
		if err != nil {
			// Expected for incomplete JSON chunks during streaming
			t.Logf("Expected error during streaming: %v", err)
		}
	}

	// The final accumulated content should contain valid JSON
	expectedJSON := `[{"s": 0, "e": 50},{"s": 50, "e": 100}]`
	finalPositions, err := parser.ParseSemanticRegular(expectedJSON)
	if err != nil {
		t.Errorf("Failed to parse final regular content: %v", err)
	}

	if len(finalPositions) != 2 {
		t.Errorf("Expected 2 positions, got %d", len(finalPositions))
	}

	expectedPositions := []types.Position{
		{StartPos: 0, EndPos: 50},
		{StartPos: 50, EndPos: 100},
	}

	for i, pos := range finalPositions {
		if pos.StartPos != expectedPositions[i].StartPos || pos.EndPos != expectedPositions[i].EndPos {
			t.Errorf("Position %d mismatch: expected %+v, got %+v", i, expectedPositions[i], pos)
		}
	}
}

func TestParseSemanticRegular_DifferentFormats(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []types.Position
	}{
		{
			name:    "start_pos/end_pos format",
			content: `[{"start_pos": 0, "end_pos": 100}, {"start_pos": 100, "end_pos": 200}]`,
			expected: []types.Position{
				{StartPos: 0, EndPos: 100},
				{StartPos: 100, EndPos: 200},
			},
		},
		{
			name:    "s/e format",
			content: `[{"s": 0, "e": 100}, {"s": 100, "e": 200}]`,
			expected: []types.Position{
				{StartPos: 0, EndPos: 100},
				{StartPos: 100, EndPos: 200},
			},
		},
		{
			name:    "mixed format",
			content: `[{"start_pos": 0, "end_pos": 100}, {"s": 100, "e": 200}]`,
			expected: []types.Position{
				{StartPos: 0, EndPos: 100},
				{StartPos: 100, EndPos: 200},
			},
		},
		{
			name:    "with markdown",
			content: "```json\n[{\"s\": 0, \"e\": 100}]\n```",
			expected: []types.Position{
				{StartPos: 0, EndPos: 100},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewSemanticParser(false)
			positions, err := parser.ParseSemanticRegular(tt.content)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(positions, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, positions)
			}
		})
	}
}

func TestParseSemanticToolcall_Final(t *testing.T) {
	tests := []struct {
		name      string
		arguments string
		expected  []types.Position
		expectErr bool
	}{
		{
			name:      "Valid toolcall arguments",
			arguments: `{"segments": [{"s": 0, "e": 100}, {"s": 100, "e": 200}]}`,
			expected: []types.Position{
				{StartPos: 0, EndPos: 100},
				{StartPos: 100, EndPos: 200},
			},
		},
		{
			name:      "Empty arguments",
			arguments: "",
			expected:  nil,
		},
		{
			name:      "Repaired JSON",
			arguments: `{"segments": [{"s": 0, "e": 100`, // Incomplete but repairable
			expected: []types.Position{
				{StartPos: 0, EndPos: 100},
			},
		},
		{
			name:      "No segments field",
			arguments: `{"other": "data"}`,
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewSemanticParser(true)
			positions, err := parser.ParseSemanticToolcall(tt.arguments)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(positions, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, positions)
			}
		})
	}
}

func TestParseIncompleteJSON(t *testing.T) {
	parser := NewSemanticParser(true)

	// Test incomplete JSON that needs repair
	incompleteChunk := `{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"segments\": [{\"s\": 0, \"e\": 50"}}]}}]}`

	positions, err := parser.parseSemanticToolcall([]byte(incompleteChunk))
	if err != nil {
		// This is expected for incomplete JSON
		t.Logf("Expected error for incomplete JSON: %v", err)
	}

	// Should handle gracefully without panic
	if len(positions) > 0 {
		t.Logf("Parsed %d positions from incomplete JSON", len(positions))
	}
}

func TestCompleteToolcallJSON(t *testing.T) {
	parser := NewSemanticParser(true)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Complete JSON",
			input:    `{"segments":[{"s":0,"e":300},{"s":300,"e":600}]}`,
			expected: `{"segments":[{"s":0,"e":300},{"s":300,"e":600}]}`,
		},
		{
			name:     "Incomplete last object",
			input:    `{"segments":[{"s":0,"e":300},{"s":300,"e":600},{"s":600,"e"`,
			expected: `{"segments":[{"s":0,"e":300},{"s":300,"e":600}]}`,
		},
		{
			name:     "Incomplete with whitespace",
			input:    `{"segments":[{"s":0,"e":300}, {"s":300,"e":600}, {"s"`,
			expected: `{"segments":[{"s":0,"e":300}, {"s":300,"e":600}]}`,
		},
		{
			name:     "Single incomplete object",
			input:    `{"segments":[{"s":0,"e"`,
			expected: `{"segments":[{"s":0,"e"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.completeJSON(tt.input)
			if result != tt.expected {
				t.Errorf("Expected: %s\nGot: %s", tt.expected, result)
			}
		})
	}
}

func TestParseSemanticPositions(t *testing.T) {
	tests := []struct {
		name       string
		isToolcall bool
		chunk      string
		expectErr  bool
	}{
		{
			name:       "Toolcall chunk",
			isToolcall: true,
			chunk:      `{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"segments\": ["}}]}}]}`,
		},
		{
			name:       "Regular chunk",
			isToolcall: false,
			chunk:      `{"choices":[{"delta":{"content":"[{\"s\": 0, \"e\": 50}]"}}]}`,
		},
		{
			name:       "Empty chunk",
			isToolcall: true,
			chunk:      "",
		},
		{
			name:       "SSE format",
			isToolcall: true,
			chunk:      `data: {"choices":[{"delta":{}}]}`,
		},
		{
			name:       "SSE DONE",
			isToolcall: true,
			chunk:      `data: [DONE]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewSemanticParser(tt.isToolcall)
			positions, err := parser.ParseSemanticPositions([]byte(tt.chunk))

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Positions can be nil for partial chunks
			t.Logf("Parsed %d positions", len(positions))
		})
	}
}

func TestExtractJSONArray(t *testing.T) {
	parser := NewSemanticParser(false)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean JSON",
			input:    `[{"s": 0, "e": 100}]`,
			expected: `[{"s": 0, "e": 100}]`,
		},
		{
			name:     "With markdown",
			input:    "```json\n[{\"s\": 0, \"e\": 100}]\n```",
			expected: `[{"s": 0, "e": 100}]`,
		},
		{
			name:     "With newlines",
			input:    "[\n{\"s\": 0, \"e\": 100}\n]",
			expected: `[{"s": 0, "e": 100}]`,
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "Too short",
			input:    "[]",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractJSONArray(tt.input)
			if result != tt.expected {
				t.Errorf("Expected: %s\nGot: %s", tt.expected, result)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	parser := NewSemanticParser(false)

	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{"int", 42, 42},
		{"int32", int32(42), 42},
		{"int64", int64(42), 42},
		{"float32", float32(42.5), 42},
		{"float64", float64(42.7), 42},
		{"string", "invalid", -1},
		{"nil", nil, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.toInt(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkParseSemanticToolcall(b *testing.B) {
	parser := NewSemanticParser(true)
	chunk := `{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"segments\": [{\"s\": 0, \"e\": 50}]}"}}]}}]}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.parseSemanticToolcall([]byte(chunk))
	}
}

func BenchmarkParseSemanticRegular(b *testing.B) {
	parser := NewSemanticParser(false)
	chunk := `{"choices":[{"delta":{"content":"[{\"s\": 0, \"e\": 50}]"}}]}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.parseSemanticRegular([]byte(chunk))
	}
}

func BenchmarkCompleteJSON(b *testing.B) {
	parser := NewSemanticParser(true)
	json := `{"segments":[{"s":0,"e":300},{"s":300,"e":600},{"s":600,"e"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.completeJSON(json)
	}
}

// ===== EXTRACTION PARSER TESTS =====

func TestNewExtractionParser(t *testing.T) {
	parser := NewExtractionParser()

	if parser == nil {
		t.Fatal("NewExtractionParser() returned nil")
	}

	if !parser.Toolcall {
		t.Error("Expected extraction parser to have Toolcall=true")
	}

	if parser.Content != "" {
		t.Error("Expected empty Content initially")
	}

	if parser.Arguments != "" {
		t.Error("Expected empty Arguments initially")
	}
}

func TestParseExtractionToolcall(t *testing.T) {
	tests := []struct {
		name          string
		arguments     string
		expectedNodes int
		expectedRels  int
		expectError   bool
		description   string
	}{
		{
			name: "Valid complete extraction",
			arguments: `{
				"entities": [
					{
						"id": "john_smith",
						"name": "John Smith",
						"type": "PERSON",
						"description": "A software engineer",
						"confidence": 0.9
					},
					{
						"id": "google",
						"name": "Google",
						"type": "ORGANIZATION",
						"description": "Technology company",
						"confidence": 0.95
					}
				],
				"relationships": [
					{
						"start_node": "john_smith",
						"end_node": "google",
						"type": "WORKS_FOR",
						"description": "Employment relationship",
						"confidence": 0.8
					}
				]
			}`,
			expectedNodes: 2,
			expectedRels:  1,
			expectError:   false,
			description:   "Should parse complete valid extraction JSON",
		},
		{
			name: "Only entities, no relationships",
			arguments: `{
				"entities": [
					{
						"id": "alice",
						"name": "Alice",
						"type": "PERSON",
						"description": "A person",
						"confidence": 0.8
					}
				],
				"relationships": []
			}`,
			expectedNodes: 1,
			expectedRels:  0,
			expectError:   false,
			description:   "Should handle extraction with only entities",
		},
		{
			name: "Empty extraction",
			arguments: `{
				"entities": [],
				"relationships": []
			}`,
			expectedNodes: 0,
			expectedRels:  0,
			expectError:   false,
			description:   "Should handle empty extraction",
		},
		{
			name: "Incomplete JSON that needs repair",
			arguments: `{
				"entities": [
					{
						"id": "bob",
						"name": "Bob",
						"type": "PERSON",
						"description": "A person",
						"confidence": 0.7
					}
				],
				"relationships": [
					{
						"start_node": "bob",
						"end_node": "company",
						"type": "WORKS_FOR"`,
			expectedNodes: 1,
			expectedRels:  0, // Incomplete relationship cannot be repaired (missing end_node)
			expectError:   false,
			description:   "Should repair incomplete JSON",
		},
		{
			name:        "Invalid JSON",
			arguments:   `{invalid json}`,
			expectError: true,
			description: "Should return error for invalid JSON",
		},
		{
			name:          "Empty arguments",
			arguments:     "",
			expectedNodes: 0,
			expectedRels:  0,
			expectError:   false, // Returns nil, nil, nil - not an error
			description:   "Should handle empty arguments gracefully",
		},
		{
			name: "Missing required fields",
			arguments: `{
				"entities": [
					{
						"name": "No ID",
						"type": "PERSON"
					}
				],
				"relationships": []
			}`,
			expectedNodes: 0, // Changed from 1 to 0 - entities without ID should be skipped
			expectedRels:  0,
			expectError:   false,
			description:   "Should skip entities with missing required fields (ID, Name)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewExtractionParser()
			nodes, relationships, err := parser.ParseExtractionToolcall(tt.arguments)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: Expected error but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: Unexpected error: %v", tt.description, err)
				return
			}

			if len(nodes) != tt.expectedNodes {
				t.Errorf("%s: Expected %d nodes, got %d", tt.description, tt.expectedNodes, len(nodes))
			}

			if len(relationships) != tt.expectedRels {
				t.Errorf("%s: Expected %d relationships, got %d", tt.description, tt.expectedRels, len(relationships))
			}

			// Validate node structure (only for tests that expect valid nodes)
			if tt.name != "Missing required fields" {
				for i, node := range nodes {
					if node.ID == "" {
						t.Errorf("%s: Node %d has empty ID", tt.description, i)
					}
					if node.Name == "" {
						t.Errorf("%s: Node %d has empty Name", tt.description, i)
					}
					if node.Type == "" {
						t.Errorf("%s: Node %d has empty Type", tt.description, i)
					}
					if node.ExtractionMethod != types.ExtractionMethodLLM {
						t.Errorf("%s: Node %d has wrong extraction method", tt.description, i)
					}
				}
			}

			// Validate relationship structure (only for tests that expect valid relationships)
			if tt.name != "Incomplete JSON that needs repair" {
				for i, rel := range relationships {
					if rel.StartNode == "" {
						t.Errorf("%s: Relationship %d has empty StartNode", tt.description, i)
					}
					if rel.EndNode == "" {
						t.Errorf("%s: Relationship %d has empty EndNode", tt.description, i)
					}
					if rel.Type == "" {
						t.Errorf("%s: Relationship %d has empty Type", tt.description, i)
					}
					if rel.ExtractionMethod != types.ExtractionMethodLLM {
						t.Errorf("%s: Relationship %d has wrong extraction method", tt.description, i)
					}
				}
			}
		})
	}
}

func TestParseExtractionEntities(t *testing.T) {
	tests := []struct {
		name        string
		chunk       string
		expectError bool
		description string
	}{
		{
			name:        "Valid toolcall chunk",
			chunk:       `{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"entities\": ["}}]}}]}`,
			expectError: false,
			description: "Should handle valid streaming toolcall chunk",
		},
		{
			name:        "Chunk with entity data",
			chunk:       `{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"id\": \"test\", \"name\": \"Test\""}}]}}]}`,
			expectError: false,
			description: "Should handle chunk with partial entity data",
		},
		{
			name:        "Empty chunk",
			chunk:       "",
			expectError: false,
			description: "Should handle empty chunk gracefully",
		},
		{
			name:        "SSE format chunk",
			chunk:       `data: {"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"test"}}]}}]}`,
			expectError: false,
			description: "Should handle SSE format",
		},
		{
			name:        "SSE DONE",
			chunk:       `data: [DONE]`,
			expectError: false,
			description: "Should handle SSE DONE signal",
		},
		{
			name:        "Invalid JSON",
			chunk:       `{invalid json}`,
			expectError: false, // Should not error, just return empty results
			description: "Should handle invalid JSON gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewExtractionParser()
			nodes, relationships, err := parser.ParseExtractionEntities([]byte(tt.chunk))

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: Expected error but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: Unexpected error: %v", tt.description, err)
				return
			}

			// Results can be empty for partial chunks - this is expected
			t.Logf("%s: Parsed %d nodes, %d relationships", tt.description, len(nodes), len(relationships))
		})
	}
}

func TestTryParseExtractionToolcall(t *testing.T) {
	tests := []struct {
		name          string
		arguments     string
		expectedNodes int
		expectedRels  int
		expectSuccess bool
		description   string
	}{
		{
			name: "Complete valid JSON",
			arguments: `{
				"entities": [{"id": "test", "name": "Test", "type": "PERSON", "description": "Test person", "confidence": 0.9}],
				"relationships": []
			}`,
			expectedNodes: 1,
			expectedRels:  0,
			expectSuccess: true,
			description:   "Should parse complete valid JSON",
		},
		{
			name: "Incomplete but repairable JSON",
			arguments: `{
				"entities": [
					{"id": "test1", "name": "Test1", "type": "PERSON", "description": "Test", "confidence": 0.9},
					{"id": "test2", "name": "Test2"`,
			expectedNodes: 2, // Should repair incomplete entities by removing incomplete fields
			expectedRels:  0,
			expectSuccess: true,
			description:   "Should repair incomplete JSON and include all repairable entities",
		},
		{
			name:          "Empty arguments",
			arguments:     "",
			expectedNodes: 0,
			expectedRels:  0,
			expectSuccess: true, // Returns nil, nil, nil (not an error)
			description:   "Should return nil for empty arguments",
		},
		{
			name:          "Invalid JSON",
			arguments:     `{completely invalid}`,
			expectedNodes: 0,
			expectedRels:  0,
			expectSuccess: false,
			description:   "Should fail on invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewExtractionParser()
			// Set the arguments in the parser
			parser.Arguments = tt.arguments

			nodes, relationships, err := parser.tryParseExtractionToolcall()

			success := err == nil
			if success != tt.expectSuccess {
				t.Errorf("%s: Expected success=%v, got %v (error: %v)", tt.description, tt.expectSuccess, success, err)
			}

			if tt.expectSuccess {
				if len(nodes) != tt.expectedNodes {
					t.Errorf("%s: Expected %d nodes, got %d", tt.description, tt.expectedNodes, len(nodes))
				}
				if len(relationships) != tt.expectedRels {
					t.Errorf("%s: Expected %d relationships, got %d", tt.description, tt.expectedRels, len(relationships))
				}
			}
		})
	}
}

func TestCompleteExtractionJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		description string
	}{
		{
			name:        "Complete JSON",
			input:       `{"entities":[{"id":"test","name":"Test"}],"relationships":[]}`,
			expected:    `{"entities":[{"id":"test","name":"Test"}],"relationships":[]}`,
			description: "Should return complete JSON unchanged",
		},
		{
			name:        "Incomplete entity",
			input:       `{"entities":[{"id":"test1","name":"Test1"},{"id":"test2","name"`,
			expected:    `{"entities":[{"id":"test1","name":"Test1"}],"relationships":[]}`,
			description: "Should remove incomplete entity and add missing relationships array",
		},
		{
			name:        "Incomplete relationship",
			input:       `{"entities":[],"relationships":[{"start_node":"a","end_node":"b"},{"start_node"`,
			expected:    `{"entities":[],"relationships":[{"start_node":"a","end_node":"b"}]}`,
			description: "Should remove incomplete relationship (cannot repair without end_node)",
		},
		{
			name:        "Missing relationships array",
			input:       `{"entities":[{"id":"test","name":"Test"}]}`,
			expected:    `{"entities":[{"id":"test","name":"Test"}],"relationships":[]}`,
			description: "Should add missing relationships array for extraction JSON",
		},
		{
			name:        "Missing entities array",
			input:       `{"relationships":[]}`,
			expected:    `{"entities":[],"relationships":[]}`,
			description: "Should add missing entities array for extraction JSON",
		},
		{
			name:        "Empty object",
			input:       `{}`,
			expected:    `{"entities":[],"relationships":[]}`,
			description: "Should add missing arrays to empty object for extraction JSON",
		},
		{
			name:        "Incomplete with nested objects",
			input:       `{"entities":[{"id":"test","properties":{"key":"value","incomplete"`,
			expected:    `{"entities":[],"relationships":[]}`,
			description: "Should remove incomplete nested objects and add relationships array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewExtractionParser()
			result := parser.completeExtractionJSON(tt.input)

			// Parse both expected and actual results to compare structure, not string order
			var expectedObj, resultObj map[string]interface{}
			expectedErr := jsoniter.UnmarshalFromString(tt.expected, &expectedObj)
			resultErr := jsoniter.UnmarshalFromString(result, &resultObj)

			if expectedErr != nil {
				t.Errorf("Failed to parse expected JSON: %v", expectedErr)
				return
			}
			if resultErr != nil {
				t.Errorf("Failed to parse result JSON: %v", resultErr)
				return
			}

			// Compare the parsed structures
			if !equalMaps(expectedObj, resultObj) {
				t.Errorf("%s:\nExpected: %s\nGot:      %s", tt.description, tt.expected, result)
			}
		})
	}
}

// equalMaps compares two maps recursively
func equalMaps(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	for key, valueA := range a {
		valueB, exists := b[key]
		if !exists {
			return false
		}

		if !equalValues(valueA, valueB) {
			return false
		}
	}

	return true
}

// equalValues compares two interface{} values recursively
func equalValues(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch va := a.(type) {
	case map[string]interface{}:
		vb, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		return equalMaps(va, vb)
	case []interface{}:
		vb, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for i, itemA := range va {
			if !equalValues(itemA, vb[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

func TestExtractionStreamingIntegration(t *testing.T) {
	parser := NewExtractionParser()

	// Simulate streaming chunks that build up a complete extraction
	streamingChunks := []string{
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"\"entities\": ["}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"\"id\": \"john\","}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"\"name\": \"John Smith\","}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"\"type\": \"PERSON\","}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"\"description\": \"A person\","}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"\"confidence\": 0.9"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"}"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"],"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"\"relationships\": []"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
	}

	var finalNodes []types.Node
	var finalRelationships []types.Relationship

	// Process each chunk
	for i, chunk := range streamingChunks {
		nodes, relationships, err := parser.ParseExtractionEntities([]byte(chunk))
		if err != nil {
			t.Logf("Chunk %d parsing error (may be expected): %v", i, err)
		}

		if len(nodes) > 0 || len(relationships) > 0 {
			finalNodes = nodes
			finalRelationships = relationships
			t.Logf("Chunk %d: Found %d entities, %d relationships", i, len(nodes), len(relationships))
		}
	}

	// Final parsing of accumulated arguments
	if parser.Arguments != "" {
		nodes, relationships, err := parser.ParseExtractionToolcall(parser.Arguments)
		if err != nil {
			t.Errorf("Failed to parse final accumulated arguments: %v", err)
		} else {
			finalNodes = nodes
			finalRelationships = relationships
		}
	}

	// Verify final results
	if len(finalNodes) != 1 {
		t.Errorf("Expected 1 final entity, got %d", len(finalNodes))
	}

	if len(finalRelationships) != 0 {
		t.Errorf("Expected 0 final relationships, got %d", len(finalRelationships))
	}

	if len(finalNodes) > 0 {
		node := finalNodes[0]
		if node.ID != "john" {
			t.Errorf("Expected entity ID 'john', got '%s'", node.ID)
		}
		if node.Name != "John Smith" {
			t.Errorf("Expected entity name 'John Smith', got '%s'", node.Name)
		}
		if node.Type != "PERSON" {
			t.Errorf("Expected entity type 'PERSON', got '%s'", node.Type)
		}
	}
}

// Benchmark tests for extraction parsing
func BenchmarkParseExtractionToolcall(b *testing.B) {
	parser := NewExtractionParser()
	arguments := `{
		"entities": [
			{"id": "john", "name": "John Smith", "type": "PERSON", "description": "A person", "confidence": 0.9},
			{"id": "google", "name": "Google", "type": "ORGANIZATION", "description": "Tech company", "confidence": 0.95}
		],
		"relationships": [
			{"start_node": "john", "end_node": "google", "type": "WORKS_FOR", "description": "Employment", "confidence": 0.8}
		]
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = parser.ParseExtractionToolcall(arguments)
	}
}

func BenchmarkParseExtractionEntities(b *testing.B) {
	parser := NewExtractionParser()
	chunk := `{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"entities\": [{\"id\": \"test\", \"name\": \"Test\"}]}"}}]}}]}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = parser.ParseExtractionEntities([]byte(chunk))
	}
}

func BenchmarkCompleteExtractionJSON(b *testing.B) {
	parser := NewExtractionParser()
	json := `{"entities":[{"id":"test1","name":"Test1"},{"id":"test2","name":"Test2"},{"id":"incomplete`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.completeExtractionJSON(json)
	}
}

func BenchmarkTryParseExtractionToolcall(b *testing.B) {
	parser := NewExtractionParser()
	arguments := `{"entities":[{"id":"test","name":"Test","type":"PERSON"}],"relationships":[]}`
	parser.Arguments = arguments

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = parser.tryParseExtractionToolcall()
	}
}

func TestParseExtractionRegular(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectNodes int
		expectRels  int
		expectError bool
		description string
	}{
		{
			name: "Valid JSON with entities and relationships",
			content: `{
				"entities": [
					{
						"id": "john_smith",
						"name": "John Smith",
						"type": "PERSON",
						"description": "A person",
						"confidence": 0.9
					}
				],
				"relationships": [
					{
						"start_node": "john_smith",
						"end_node": "google",
						"type": "WORKS_FOR",
						"description": "Employment relationship",
						"confidence": 0.8
					}
				]
			}`,
			expectNodes: 1,
			expectRels:  1,
			expectError: false,
			description: "Should parse valid JSON with entities and relationships",
		},
		{
			name:        "JSON in markdown blocks",
			content:     "```json\n{\n\"entities\": [],\n\"relationships\": []\n}\n```",
			expectNodes: 0,
			expectRels:  0,
			expectError: false,
			description: "Should extract JSON from markdown blocks",
		},
		{
			name:        "Empty content",
			content:     "",
			expectNodes: 0,
			expectRels:  0,
			expectError: false,
			description: "Should handle empty content gracefully",
		},
		{
			name:        "No valid JSON",
			content:     "This is just text with no JSON",
			expectNodes: 0,
			expectRels:  0,
			expectError: true,
			description: "Should error when no JSON found",
		},
		{
			name:        "Invalid JSON",
			content:     "{invalid json",
			expectNodes: 0,
			expectRels:  0,
			expectError: true,
			description: "Should error on invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewExtractionParser()
			parser.SetToolcall(false) // Set to non-toolcall mode

			nodes, relationships, err := parser.ParseExtractionRegular(tt.content)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: Expected error but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: Unexpected error: %v", tt.description, err)
				return
			}

			if len(nodes) != tt.expectNodes {
				t.Errorf("%s: Expected %d nodes, got %d", tt.description, tt.expectNodes, len(nodes))
			}

			if len(relationships) != tt.expectRels {
				t.Errorf("%s: Expected %d relationships, got %d", tt.description, tt.expectRels, len(relationships))
			}

			// Validate node structure if any exist
			for i, node := range nodes {
				if node.ID == "" {
					t.Errorf("%s: Node %d has empty ID", tt.description, i)
				}
				if node.ExtractionMethod == "" {
					t.Errorf("%s: Node %d has empty ExtractionMethod", tt.description, i)
				}
			}

			// Validate relationship structure if any exist
			for i, rel := range relationships {
				if rel.StartNode == "" {
					t.Errorf("%s: Relationship %d has empty StartNode", tt.description, i)
				}
				if rel.EndNode == "" {
					t.Errorf("%s: Relationship %d has empty EndNode", tt.description, i)
				}
			}
		})
	}
}

func TestSetToolcall(t *testing.T) {
	parser := NewExtractionParser()

	// Initially should be true (toolcall mode)
	if !parser.Toolcall {
		t.Error("Expected initial toolcall mode to be true")
	}

	// Set to false (non-toolcall mode)
	parser.SetToolcall(false)
	if parser.Toolcall {
		t.Error("Expected toolcall mode to be false after SetToolcall(false)")
	}

	// Set back to true
	parser.SetToolcall(true)
	if !parser.Toolcall {
		t.Error("Expected toolcall mode to be true after SetToolcall(true)")
	}
}

func TestParseExtractionEntitiesMode(t *testing.T) {
	// Test toolcall mode
	t.Run("toolcall mode", func(t *testing.T) {
		parser := NewExtractionParser()
		parser.SetToolcall(true)

		chunk := `{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"entities\": []}"}}]}}]}`
		nodes, relationships, err := parser.ParseExtractionEntities([]byte(chunk))

		if err != nil {
			t.Errorf("Unexpected error in toolcall mode: %v", err)
		}

		// Results can be empty for partial chunks
		t.Logf("Toolcall mode: %d nodes, %d relationships", len(nodes), len(relationships))
	})

	// Test non-toolcall mode
	t.Run("non-toolcall mode", func(t *testing.T) {
		parser := NewExtractionParser()
		parser.SetToolcall(false)

		chunk := `{"choices":[{"delta":{"content":"{\"entities\": [], \"relationships\": []}"}}]}`
		nodes, relationships, err := parser.ParseExtractionEntities([]byte(chunk))

		if err != nil {
			t.Errorf("Unexpected error in non-toolcall mode: %v", err)
		}

		// Results can be empty for partial chunks
		t.Logf("Non-toolcall mode: %d nodes, %d relationships", len(nodes), len(relationships))
	})
}

// Test for non-toolcall mode error scenarios and edge cases
func TestParseExtractionRegularErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectNodes int
		expectRels  int
		expectError bool
		description string
	}{
		{
			name:        "Malformed JSON with extra commas",
			content:     `{"entities": [{"id": "test", "name": "Test",}], "relationships": []}`,
			expectNodes: 1,
			expectRels:  0,
			expectError: false,
			description: "Should handle and repair malformed JSON with trailing commas",
		},
		{
			name:        "JSON without quotes on keys",
			content:     `{entities: [{id: "test", name: "Test"}], relationships: []}`,
			expectNodes: 1,
			expectRels:  0,
			expectError: false,
			description: "Should handle JSON with unquoted keys",
		},
		{
			name:        "Incomplete JSON structure",
			content:     `{"entities": [{"id": "test", "name": "Test"`,
			expectNodes: 1, // Changed from 0 to 1 to match toolcall behavior
			expectRels:  0,
			expectError: false,
			description: "Should handle incomplete JSON gracefully and extract valid entities",
		},
		{
			name:        "JSON with Chinese content",
			content:     `{"entities": [{"id": "张三", "name": "张三", "type": "人物", "description": "一个人", "confidence": 0.9}], "relationships": []}`,
			expectNodes: 1,
			expectRels:  0,
			expectError: false,
			description: "Should handle Chinese content correctly",
		},
		{
			name:        "Multiple JSON objects in content",
			content:     `{"entities": [{"id": "test", "name": "Test", "type": "PERSON", "description": "Test person", "confidence": 0.9}], "relationships": []}`,
			expectNodes: 1,
			expectRels:  0,
			expectError: false,
			description: "Should extract from JSON objects with complete structure",
		},
		{
			name:        "JSON in mixed text content",
			content:     `Here are the extracted entities: {"entities": [{"id": "test", "name": "Test"}], "relationships": []} That's all.`,
			expectNodes: 1,
			expectRels:  0,
			expectError: false,
			description: "Should extract JSON from mixed text content",
		},
		{
			name:        "Empty arrays",
			content:     `{"entities": [], "relationships": []}`,
			expectNodes: 0,
			expectRels:  0,
			expectError: false,
			description: "Should handle empty arrays correctly",
		},
		{
			name:        "Missing confidence fields",
			content:     `{"entities": [{"id": "test", "name": "Test", "type": "PERSON"}], "relationships": []}`,
			expectNodes: 1,
			expectRels:  0,
			expectError: false,
			description: "Should handle missing optional fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewExtractionParser()
			parser.SetToolcall(false)

			nodes, relationships, err := parser.ParseExtractionRegular(tt.content)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: Expected error but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: Unexpected error: %v", tt.description, err)
				return
			}

			if len(nodes) != tt.expectNodes {
				t.Errorf("%s: Expected %d nodes, got %d", tt.description, tt.expectNodes, len(nodes))
			}

			if len(relationships) != tt.expectRels {
				t.Errorf("%s: Expected %d relationships, got %d", tt.description, tt.expectRels, len(relationships))
			}

			// Log actual results for debugging
			t.Logf("%s: Got %d nodes, %d relationships", tt.description, len(nodes), len(relationships))
		})
	}
}

// Test enhanced validation and empty field handling
func TestParseExtractionRegularEnhancedValidation(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectNodes int
		expectRels  int
		expectError bool
		description string
	}{
		{
			name: "Valid entities and relationships with all fields",
			content: `{
				"entities": [
					{
						"id": "john_smith",
						"name": "John Smith", 
						"type": "PERSON",
						"description": "A software engineer",
						"confidence": 0.9
					},
					{
						"id": "google_company",
						"name": "Google",
						"type": "ORGANIZATION", 
						"description": "Technology company",
						"confidence": 0.95
					}
				],
				"relationships": [
					{
						"start_node": "john_smith",
						"end_node": "google_company",
						"type": "WORKS_FOR",
						"description": "Employment relationship",
						"confidence": 0.8
					}
				]
			}`,
			expectNodes: 2,
			expectRels:  1,
			expectError: false,
			description: "Should parse valid entities and relationships correctly",
		},
		{
			name: "Entities with empty fields should be skipped",
			content: `{
				"entities": [
					{
						"id": "",
						"name": "Empty ID",
						"type": "PERSON",
						"description": "Should be skipped",
						"confidence": 0.9
					},
					{
						"id": "valid_entity",
						"name": "",
						"type": "PERSON", 
						"description": "Should be kept with default type",
						"confidence": 0.9
					},
					{
						"id": "another_valid",
						"name": "Valid Name",
						"type": "",
						"description": "Should be kept with default type",
						"confidence": 0.9
					},
					{
						"id": "good_entity",
						"name": "Good Entity",
						"type": "PERSON",
						"description": "Valid entity",
						"confidence": 0.9
					}
				],
				"relationships": []
			}`,
			expectNodes: 2, // Changed from 1 to 2 - entities with empty type get default value
			expectRels:  0,
			expectError: false,
			description: "Should skip entities with empty required fields (ID, Name), but allow empty type with default",
		},
		{
			name: "Relationships with empty fields should be skipped",
			content: `{
				"entities": [
					{
						"id": "entity1",
						"name": "Entity 1",
						"type": "PERSON",
						"description": "First entity",
						"confidence": 0.9
					},
					{
						"id": "entity2", 
						"name": "Entity 2",
						"type": "ORGANIZATION",
						"description": "Second entity",
						"confidence": 0.9
					}
				],
				"relationships": [
					{
						"start_node": "",
						"end_node": "entity2",
						"type": "WORKS_FOR",
						"description": "Should be skipped",
						"confidence": 0.8
					},
					{
						"start_node": "entity1",
						"end_node": "",
						"type": "WORKS_FOR", 
						"description": "Should be skipped",
						"confidence": 0.8
					},
					{
						"start_node": "entity1",
						"end_node": "entity2",
						"type": "",
						"description": "Should be kept with default type",
						"confidence": 0.8
					},
					{
						"start_node": "entity1",
						"end_node": "entity2",
						"type": "WORKS_FOR",
						"description": "Valid relationship",
						"confidence": 0.8
					}
				]
			}`,
			expectNodes: 2,
			expectRels:  2, // Changed from 1 to 2 - relationships with empty type get default value
			expectError: false,
			description: "Should skip relationships with empty required fields (StartNode, EndNode), but allow empty type with default",
		},
		{
			name: "Relationships with non-existent entity IDs should be kept for backward compatibility",
			content: `{
				"entities": [
					{
						"id": "entity1",
						"name": "Entity 1",
						"type": "PERSON",
						"description": "First entity",
						"confidence": 0.9
					}
				],
				"relationships": [
					{
						"start_node": "entity1",
						"end_node": "non_existent",
						"type": "WORKS_FOR",
						"description": "Should be kept - end entity doesn't exist but relationship is valid",
						"confidence": 0.8
					},
					{
						"start_node": "another_non_existent",
						"end_node": "entity1", 
						"type": "WORKS_FOR",
						"description": "Should be kept - start entity doesn't exist but relationship is valid",
						"confidence": 0.8
					}
				]
			}`,
			expectNodes: 1,
			expectRels:  2, // Both relationships should be kept for backward compatibility
			expectError: false,
			description: "Should keep relationships with non-existent entity references for backward compatibility",
		},
		{
			name: "Default values for missing optional fields",
			content: `{
				"entities": [
					{
						"id": "minimal_entity",
						"name": "Minimal Entity",
						"type": "PERSON"
					}
				],
				"relationships": [
					{
						"start_node": "minimal_entity",
						"end_node": "minimal_entity",
						"type": "SELF_REFERENCE"
					}
				]
			}`,
			expectNodes: 1,
			expectRels:  1,
			expectError: false,
			description: "Should provide default values for missing description and confidence",
		},
		{
			name: "Chinese content validation",
			content: `{
				"entities": [
					{
						"id": "张三_工程师",
						"name": "张三",
						"type": "人物",
						"description": "一名软件工程师",
						"confidence": 0.9
					},
					{
						"id": "谷歌_公司",
						"name": "谷歌",
						"type": "组织",
						"description": "科技公司",
						"confidence": 0.95
					}
				],
				"relationships": [
					{
						"start_node": "张三_工程师",
						"end_node": "谷歌_公司",
						"type": "工作于",
						"description": "雇佣关系",
						"confidence": 0.8
					}
				]
			}`,
			expectNodes: 2,
			expectRels:  1,
			expectError: false,
			description: "Should handle Chinese content correctly",
		},
		{
			name: "Whitespace trimming",
			content: `{
				"entities": [
					{
						"id": "  spaced_entity  ",
						"name": "  Spaced Name  ",
						"type": "  PERSON  ",
						"description": "  Description with spaces  ",
						"confidence": 0.9
					}
				],
				"relationships": []
			}`,
			expectNodes: 1,
			expectRels:  0,
			expectError: false,
			description: "Should trim whitespace from all string fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewExtractionParser()
			parser.SetToolcall(false)

			nodes, relationships, err := parser.ParseExtractionRegular(tt.content)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: Expected error but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: Unexpected error: %v", tt.description, err)
				return
			}

			if len(nodes) != tt.expectNodes {
				t.Errorf("%s: Expected %d nodes, got %d", tt.description, tt.expectNodes, len(nodes))
			}

			if len(relationships) != tt.expectRels {
				t.Errorf("%s: Expected %d relationships, got %d", tt.description, tt.expectRels, len(relationships))
			}

			// Validate that all returned entities have non-empty required fields
			for i, node := range nodes {
				if strings.TrimSpace(node.ID) == "" {
					t.Errorf("%s: Node %d has empty ID", tt.description, i)
				}
				if strings.TrimSpace(node.Name) == "" {
					t.Errorf("%s: Node %d has empty Name", tt.description, i)
				}
				if strings.TrimSpace(node.Type) == "" {
					t.Errorf("%s: Node %d has empty Type", tt.description, i)
				}
				if node.Confidence < 0.0 || node.Confidence > 1.0 {
					t.Errorf("%s: Node %d has invalid confidence: %f", tt.description, i, node.Confidence)
				}
			}

			// Validate that all returned relationships have non-empty required fields
			for i, rel := range relationships {
				if strings.TrimSpace(rel.StartNode) == "" {
					t.Errorf("%s: Relationship %d has empty StartNode", tt.description, i)
				}
				if strings.TrimSpace(rel.EndNode) == "" {
					t.Errorf("%s: Relationship %d has empty EndNode", tt.description, i)
				}
				if strings.TrimSpace(rel.Type) == "" {
					t.Errorf("%s: Relationship %d has empty Type", tt.description, i)
				}
				if rel.Confidence < 0.0 || rel.Confidence > 1.0 {
					t.Errorf("%s: Relationship %d has invalid confidence: %f", tt.description, i, rel.Confidence)
				}

				// Note: We allow relationships to reference non-existent entities for backward compatibility
				// This can happen when LLM extracts relationships but misses some entities
			}
		})
	}
}

// Test extraction prompt with JSON format
func TestExtractionPromptWithJSONFormat(t *testing.T) {
	// Test default prompt with JSON format
	promptWithJSON := ExtractionPromptWithJSONFormat("")

	if !strings.Contains(promptWithJSON, "JSON Output Format") {
		t.Error("Expected prompt to contain JSON format requirements")
	}

	if !strings.Contains(promptWithJSON, `"entities"`) {
		t.Error("Expected prompt to contain entities structure")
	}

	if !strings.Contains(promptWithJSON, `"relationships"`) {
		t.Error("Expected prompt to contain relationships structure")
	}

	if !strings.Contains(promptWithJSON, "Requirements:") {
		t.Error("Expected prompt to contain field requirements")
	}

	if !strings.Contains(promptWithJSON, "same language as user input text") {
		t.Error("Expected prompt to contain language consistency instructions")
	}

	// Test custom prompt with JSON format
	customPrompt := "Custom extraction prompt"
	customWithJSON := ExtractionPromptWithJSONFormat(customPrompt)

	if !strings.Contains(customWithJSON, customPrompt) {
		t.Error("Expected custom prompt to be included")
	}

	if !strings.Contains(customWithJSON, "JSON Output Format") {
		t.Error("Expected custom prompt to also contain JSON format requirements")
	}
}
