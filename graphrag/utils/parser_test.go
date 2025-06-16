package utils

import (
	"reflect"
	"testing"

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
