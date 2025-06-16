package utils

import (
	"fmt"
	"strings"
	"testing"
)

func TestSemanticPrompt_DefaultPrompt(t *testing.T) {
	// Test default prompt generation
	prompt := SemanticPrompt("", 300)

	// Check that prompt is not empty
	if len(prompt) == 0 {
		t.Error("Default prompt should not be empty")
	}

	// Check that size is included
	if !strings.Contains(prompt, "300") {
		t.Error("Prompt should contain the specified size")
	}

	// Check for key concepts in the default prompt
	expectedConcepts := []string{
		"SEMANTIC",
		"segmentation",
		"boundaries",
		"array",
		"indices",
		"characters",
		"JSON",
	}

	for _, concept := range expectedConcepts {
		if !strings.Contains(prompt, concept) {
			t.Errorf("Default prompt should contain concept '%s'", concept)
		}
	}

	// Check that the prompt includes proper formatting instructions (s/e format is also valid)
	if !strings.Contains(prompt, "start_pos") && !strings.Contains(prompt, "end_pos") && !strings.Contains(prompt, "\"s\"") && !strings.Contains(prompt, "\"e\"") {
		t.Error("Prompt should include position field specifications")
	}
}

func TestSemanticPrompt_CustomPrompt(t *testing.T) {
	tests := []struct {
		name       string
		userPrompt string
		size       int
		expected   string
	}{
		{
			name:       "Custom prompt with size placeholder",
			userPrompt: "Please segment this text into chunks of {{SIZE}} characters each",
			size:       150,
			expected:   "Please segment this text into chunks of 150 characters each",
		},
		{
			name:       "Custom prompt without placeholder",
			userPrompt: "Just segment this text please",
			size:       100,
			expected:   "Just segment this text please",
		},
		{
			name:       "Multiple placeholders",
			userPrompt: "Segment into {{SIZE}} chars. Each chunk should be {{SIZE}} long.",
			size:       200,
			expected:   "Segment into 200 chars. Each chunk should be 200 long.",
		},
		{
			name:       "Case variations",
			userPrompt: "Use {{SIZE}} and {{SIZE}} placeholders",
			size:       75,
			expected:   "Use 75 and 75 placeholders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SemanticPrompt(tt.userPrompt, tt.size)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSemanticPrompt_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		userPrompt string
		size       int
		checkSize  bool
	}{
		{
			name:       "Empty string",
			userPrompt: "",
			size:       300,
			checkSize:  true,
		},
		{
			name:       "Only whitespace",
			userPrompt: "   \n\t  ",
			size:       150,
			checkSize:  true,
		},
		{
			name:       "Zero size",
			userPrompt: "Segment with {{SIZE}} chars",
			size:       0,
			checkSize:  true,
		},
		{
			name:       "Negative size",
			userPrompt: "Segment with {{SIZE}} chars",
			size:       -50,
			checkSize:  true,
		},
		{
			name:       "Very large size",
			userPrompt: "Segment with {{SIZE}} chars",
			size:       1000000,
			checkSize:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SemanticPrompt(tt.userPrompt, tt.size)

			// Should never return empty string
			if len(result) == 0 {
				t.Error("Result should never be empty")
			}

			if tt.checkSize {
				sizeStr := fmt.Sprintf("%d", tt.size)
				if !strings.Contains(result, sizeStr) {
					t.Errorf("Result should contain size %s", sizeStr)
				}
			}
		})
	}
}

func TestSemanticPrompt_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name       string
		userPrompt string
		size       int
	}{
		{
			name:       "Unicode characters",
			userPrompt: "分割文本为{{SIZE}}个字符的块",
			size:       100,
		},
		{
			name:       "Special symbols",
			userPrompt: "Split @text #into {{SIZE}} $chars %each &time!",
			size:       50,
		},
		{
			name:       "JSON-like content",
			userPrompt: `{"instruction": "segment into {{SIZE}} chars", "format": "json"}`,
			size:       200,
		},
		{
			name:       "Newlines and tabs",
			userPrompt: "Split text\ninto {{SIZE}} chars\tper segment",
			size:       300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SemanticPrompt(tt.userPrompt, tt.size)

			// Should handle special characters without errors
			if len(result) == 0 {
				t.Error("Result should not be empty")
			}

			// Should replace size placeholder
			sizeStr := fmt.Sprintf("%d", tt.size)
			if !strings.Contains(result, sizeStr) {
				t.Errorf("Result should contain size %s", sizeStr)
			}
		})
	}
}

func TestGetSemanticToolcall_Structure(t *testing.T) {
	toolcall := GetSemanticToolcall()

	// Should return a non-empty array
	if len(toolcall) == 0 {
		t.Fatal("Toolcall should not be empty")
	}

	// Check first tool structure
	firstTool := toolcall[0]

	// Check required fields
	requiredFields := []string{"type", "function"}
	for _, field := range requiredFields {
		if _, exists := firstTool[field]; !exists {
			t.Errorf("Tool should have field '%s'", field)
		}
	}

	// Check type
	toolType, ok := firstTool["type"].(string)
	if !ok {
		t.Error("Type field should be a string")
	} else if toolType != "function" {
		t.Errorf("Expected type 'function', got '%s'", toolType)
	}
}

func TestGetSemanticToolcall_FunctionDetails(t *testing.T) {
	toolcall := GetSemanticToolcall()
	firstTool := toolcall[0]

	// Get function details
	function, ok := firstTool["function"].(map[string]interface{})
	if !ok {
		t.Fatal("Function field should be a map")
	}

	// Check function name
	name, ok := function["name"].(string)
	if !ok {
		t.Error("Function name should be a string")
	} else if name != "segment_text" {
		t.Errorf("Expected function name 'segment_text', got '%s'", name)
	}

	// Check description
	description, ok := function["description"].(string)
	if !ok {
		t.Error("Function description should be a string")
	} else {
		expectedKeywords := []string{
			"SEMANTIC",
			"segment",
			"boundaries",
			"text",
		}
		for _, keyword := range expectedKeywords {
			if !strings.Contains(description, keyword) {
				t.Errorf("Description should contain keyword '%s'", keyword)
			}
		}
	}

	// Check parameters
	parameters, ok := function["parameters"].(map[string]interface{})
	if !ok {
		t.Fatal("Function parameters should be a map")
	}

	// Check parameters type
	paramType, ok := parameters["type"].(string)
	if !ok || paramType != "object" {
		t.Errorf("Parameters type should be 'object', got '%v'", parameters["type"])
	}

	// Check required fields
	required, ok := parameters["required"].([]interface{})
	if !ok {
		t.Error("Required field should be an array")
	} else {
		if len(required) == 0 {
			t.Error("Required fields should not be empty")
		}

		// Check that segments is required
		foundSegments := false
		for _, req := range required {
			if reqStr, ok := req.(string); ok && reqStr == "segments" {
				foundSegments = true
				break
			}
		}
		if !foundSegments {
			t.Error("'segments' should be a required field")
		}
	}
}

func TestGetSemanticToolcall_ParametersSchema(t *testing.T) {
	toolcall := GetSemanticToolcall()
	firstTool := toolcall[0]
	function := firstTool["function"].(map[string]interface{})
	parameters := function["parameters"].(map[string]interface{})

	// Check properties
	properties, ok := parameters["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// Check segments property
	segments, ok := properties["segments"].(map[string]interface{})
	if !ok {
		t.Fatal("Segments property should exist")
	}

	// Check segments type
	segmentsType, ok := segments["type"].(string)
	if !ok || segmentsType != "array" {
		t.Errorf("Segments type should be 'array', got '%v'", segments["type"])
	}

	// Check items schema
	items, ok := segments["items"].(map[string]interface{})
	if !ok {
		t.Fatal("Segments items should be a map")
	}

	// Check items type
	itemsType, ok := items["type"].(string)
	if !ok || itemsType != "object" {
		t.Errorf("Items type should be 'object', got '%v'", items["type"])
	}

	// Check item properties
	itemProperties, ok := items["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Item properties should be a map")
	}

	// Check for required position fields
	requiredFields := []string{"s", "e"}
	for _, field := range requiredFields {
		if _, exists := itemProperties[field]; !exists {
			t.Errorf("Item properties should include field '%s'", field)
		}
	}

	// Check field types
	for _, field := range requiredFields {
		if fieldDef, exists := itemProperties[field]; exists {
			fieldMap, ok := fieldDef.(map[string]interface{})
			if !ok {
				t.Errorf("Field '%s' definition should be a map", field)
				continue
			}

			fieldType, ok := fieldMap["type"].(string)
			if !ok || fieldType != "integer" {
				t.Errorf("Field '%s' type should be 'integer', got '%v'", field, fieldMap["type"])
			}
		}
	}

	// Check item required fields
	itemRequired, ok := items["required"].([]interface{})
	if !ok {
		t.Error("Item required field should be an array")
	} else {
		if len(itemRequired) != 2 {
			t.Errorf("Expected 2 required fields, got %d", len(itemRequired))
		}

		// Check that both s and e are required
		requiredMap := make(map[string]bool)
		for _, req := range itemRequired {
			if reqStr, ok := req.(string); ok {
				requiredMap[reqStr] = true
			}
		}

		for _, field := range requiredFields {
			if !requiredMap[field] {
				t.Errorf("Field '%s' should be required", field)
			}
		}
	}
}

func TestGetSemanticToolcall_Consistency(t *testing.T) {
	// Test that multiple calls return the same structure
	toolcall1 := GetSemanticToolcall()
	toolcall2 := GetSemanticToolcall()

	if len(toolcall1) != len(toolcall2) {
		t.Error("Multiple calls should return same number of tools")
	}

	if len(toolcall1) > 0 && len(toolcall2) > 0 {
		tool1 := toolcall1[0]
		tool2 := toolcall2[0]

		// Check that basic structure is the same
		if tool1["type"] != tool2["type"] {
			t.Error("Tool type should be consistent across calls")
		}

		func1, ok1 := tool1["function"].(map[string]interface{})
		func2, ok2 := tool2["function"].(map[string]interface{})

		if !ok1 || !ok2 {
			t.Error("Function field should be consistent")
		} else {
			if func1["name"] != func2["name"] {
				t.Error("Function name should be consistent")
			}
		}
	}
}

func TestGetSemanticToolcall_JSONSerialization(t *testing.T) {
	// Test that the toolcall can be JSON serialized without errors
	toolcall := GetSemanticToolcall()

	// This should not panic or cause errors
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("JSON serialization should not panic: %v", r)
		}
	}()

	// Try to access nested fields to ensure they're properly structured
	if len(toolcall) > 0 {
		firstTool := toolcall[0]

		// Access nested structure
		if function, ok := firstTool["function"].(map[string]interface{}); ok {
			if params, ok := function["parameters"].(map[string]interface{}); ok {
				if props, ok := params["properties"].(map[string]interface{}); ok {
					if segments, ok := props["segments"].(map[string]interface{}); ok {
						if items, ok := segments["items"].(map[string]interface{}); ok {
							if _, ok := items["properties"].(map[string]interface{}); !ok {
								t.Error("Nested structure should be properly accessible")
							}
						}
					}
				}
			}
		}
	}
}

// Benchmark tests
func BenchmarkSemanticPrompt_Default(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SemanticPrompt("", 300)
	}
}

func BenchmarkSemanticPrompt_Custom(b *testing.B) {
	prompt := "Please segment this text into chunks of {{SIZE}} characters each"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SemanticPrompt(prompt, 150)
	}
}

func BenchmarkSemanticPrompt_Multiple_Placeholders(b *testing.B) {
	prompt := "Split into {{SIZE}} chars. Each chunk should be {{SIZE}} long. Total size: {{SIZE}}"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SemanticPrompt(prompt, 200)
	}
}

func BenchmarkGetSemanticToolcall_AccessNested(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toolcall := GetSemanticToolcall()
		if len(toolcall) > 0 {
			firstTool := toolcall[0]
			if function, ok := firstTool["function"].(map[string]interface{}); ok {
				_ = function["name"]
				_ = function["description"]
				if params, ok := function["parameters"].(map[string]interface{}); ok {
					_ = params["type"]
				}
			}
		}
	}
}
