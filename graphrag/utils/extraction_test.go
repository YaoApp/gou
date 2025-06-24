package utils

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractionPrompt(t *testing.T) {
	tests := []struct {
		name        string
		userPrompt  string
		expectEmpty bool
		description string
	}{
		{
			name:        "Empty user prompt",
			userPrompt:  "",
			expectEmpty: false,
			description: "Should return default template when user prompt is empty",
		},
		{
			name:        "Whitespace only user prompt",
			userPrompt:  "   \n\t   ",
			expectEmpty: false,
			description: "Should return default template when user prompt is only whitespace",
		},
		{
			name:        "Valid user prompt",
			userPrompt:  "Extract entities from this text",
			expectEmpty: false,
			description: "Should return user prompt when provided",
		},
		{
			name:        "Complex user prompt",
			userPrompt:  "Please extract all entities and relationships with high accuracy. Focus on technical terms.",
			expectEmpty: false,
			description: "Should return complex user prompt unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractionPrompt(tt.userPrompt)

			if result == "" {
				t.Errorf("%s: ExtractionPrompt returned empty string", tt.description)
				return
			}

			// Check if we got the user prompt back or the default template
			trimmedUserPrompt := strings.TrimSpace(tt.userPrompt)
			if trimmedUserPrompt != "" {
				// Should return user prompt
				if result != tt.userPrompt {
					t.Errorf("%s: Expected user prompt '%s', got '%s'", tt.description, tt.userPrompt, result)
				}
			} else {
				// Should return default template
				if result != extractionPromptTemplate {
					t.Errorf("%s: Expected default template, got different content", tt.description)
				}

				// Verify template contains key elements
				if !strings.Contains(result, "Entity and Relationship Extraction Task") {
					t.Errorf("%s: Default template missing main title", tt.description)
				}
				if !strings.Contains(result, "CRITICAL INSTRUCTIONS") {
					t.Errorf("%s: Default template missing critical instructions", tt.description)
				}
				if !strings.Contains(result, "Entity Types") {
					t.Errorf("%s: Default template missing entity types section", tt.description)
				}
				if !strings.Contains(result, "Relationship Types") {
					t.Errorf("%s: Default template missing relationship types section", tt.description)
				}
			}
		})
	}
}

func TestExtractionPromptTemplate(t *testing.T) {
	// Test that the extraction prompt template contains essential elements
	template := extractionPromptTemplate

	essentialElements := []string{
		"Entity and Relationship Extraction Task",
		"CRITICAL INSTRUCTIONS",
		"Core Principles",
		"Required Actions",
		"STRICTLY FORBIDDEN",
		"Entity Types",
		"Relationship Types",
		"Example",
		"Quality Requirements",
		"Output Format",
		"Key Reminders",
		"PERSON", "ORGANIZATION", "LOCATION", // Entity types
		"WORKS_FOR", "LOCATED_IN", "PART_OF", // Relationship types
		"NO HALLUCINATION",
		"confidence scores",
		"unique ID",
	}

	for _, element := range essentialElements {
		if !strings.Contains(template, element) {
			t.Errorf("Extraction prompt template missing essential element: %s", element)
		}
	}

	// Check template structure
	if len(template) < 1000 {
		t.Error("Extraction prompt template seems too short")
	}

	// Verify it contains instructions about function calls
	if !strings.Contains(template, "function calls") {
		t.Error("Template should mention function calls for output format")
	}

	// Verify it emphasizes accuracy
	if !strings.Contains(template, "accuracy") {
		t.Error("Template should emphasize accuracy")
	}
}

func TestGetExtractionToolcall(t *testing.T) {
	toolcall := GetExtractionToolcall()

	if toolcall == nil {
		t.Fatal("GetExtractionToolcall returned nil")
	}

	if len(toolcall) == 0 {
		t.Fatal("GetExtractionToolcall returned empty slice")
	}

	// Should have exactly one function
	if len(toolcall) != 1 {
		t.Errorf("Expected 1 function in toolcall, got %d", len(toolcall))
	}

	function := toolcall[0]

	// Check function structure
	if function["type"] != "function" {
		t.Errorf("Expected type 'function', got %v", function["type"])
	}

	functionDef, ok := function["function"].(map[string]interface{})
	if !ok {
		t.Fatal("Function definition is not a map")
	}

	// Check function name
	if functionDef["name"] != "extract_entities_and_relationships" {
		t.Errorf("Expected function name 'extract_entities_and_relationships', got %v", functionDef["name"])
	}

	// Check description exists
	if description, ok := functionDef["description"].(string); !ok || description == "" {
		t.Error("Function description is missing or empty")
	}

	// Check parameters structure
	parameters, ok := functionDef["parameters"].(map[string]interface{})
	if !ok {
		t.Fatal("Parameters is not a map")
	}

	if parameters["type"] != "object" {
		t.Errorf("Expected parameters type 'object', got %v", parameters["type"])
	}

	properties, ok := parameters["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties is not a map")
	}

	// Check entities property
	entities, ok := properties["entities"].(map[string]interface{})
	if !ok {
		t.Fatal("Entities property is missing or not a map")
	}

	if entities["type"] != "array" {
		t.Errorf("Expected entities type 'array', got %v", entities["type"])
	}

	// Check relationships property
	relationships, ok := properties["relationships"].(map[string]interface{})
	if !ok {
		t.Fatal("Relationships property is missing or not a map")
	}

	if relationships["type"] != "array" {
		t.Errorf("Expected relationships type 'array', got %v", relationships["type"])
	}

	// Check required fields
	required, ok := parameters["required"].([]interface{})
	if !ok {
		t.Fatal("Required fields is not an array")
	}

	requiredFields := make([]string, len(required))
	for i, field := range required {
		requiredFields[i] = field.(string)
	}

	expectedRequired := []string{"entities", "relationships"}
	for _, expectedField := range expectedRequired {
		found := false
		for _, actualField := range requiredFields {
			if actualField == expectedField {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required field '%s' is missing", expectedField)
		}
	}
}

func TestExtractionToolcallRaw(t *testing.T) {
	// Test that the raw JSON is valid
	var toolcall []map[string]interface{}
	err := json.Unmarshal([]byte(ExtractionToolcallRaw), &toolcall)
	if err != nil {
		t.Fatalf("ExtractionToolcallRaw contains invalid JSON: %v", err)
	}

	if len(toolcall) != 1 {
		t.Errorf("Expected 1 function in raw toolcall, got %d", len(toolcall))
	}

	// Verify the JSON structure matches what GetExtractionToolcall returns
	parsedToolcall := GetExtractionToolcall()
	if len(parsedToolcall) != len(toolcall) {
		t.Error("Parsed toolcall length doesn't match raw toolcall")
	}

	// Deep comparison would be complex, so just check key fields
	rawFunction := toolcall[0]["function"].(map[string]interface{})
	parsedFunction := parsedToolcall[0]["function"].(map[string]interface{})

	if rawFunction["name"] != parsedFunction["name"] {
		t.Error("Function name mismatch between raw and parsed toolcall")
	}
}

func TestExtractionToolcallEntitySchema(t *testing.T) {
	toolcall := GetExtractionToolcall()
	function := toolcall[0]["function"].(map[string]interface{})
	parameters := function["parameters"].(map[string]interface{})
	properties := parameters["properties"].(map[string]interface{})
	entities := properties["entities"].(map[string]interface{})

	// Check entities array items schema
	items, ok := entities["items"].(map[string]interface{})
	if !ok {
		t.Fatal("Entities items schema is missing")
	}

	itemProperties, ok := items["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Entity item properties are missing")
	}

	// Check required entity fields
	expectedEntityFields := []string{"id", "name", "type", "description", "confidence", "labels", "properties"}
	for _, field := range expectedEntityFields {
		if _, exists := itemProperties[field]; !exists {
			t.Errorf("Entity schema missing required field: %s", field)
		}
	}

	// Check confidence field constraints
	confidence, ok := itemProperties["confidence"].(map[string]interface{})
	if !ok {
		t.Fatal("Confidence field schema is missing")
	}

	if confidence["type"] != "number" {
		t.Error("Confidence field should be of type number")
	}

	if confidence["minimum"] != 0.0 {
		t.Error("Confidence minimum should be 0.0")
	}

	if confidence["maximum"] != 1.0 {
		t.Error("Confidence maximum should be 1.0")
	}

	// Check labels field constraints
	labels, ok := itemProperties["labels"].(map[string]interface{})
	if !ok {
		t.Fatal("Labels field schema is missing")
	}

	if labels["type"] != "array" {
		t.Error("Labels field should be of type array")
	}

	if labels["minItems"] != 1.0 {
		t.Error("Labels minItems should be 1")
	}

	// Check properties field constraints
	propertiesField, ok := itemProperties["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties field schema is missing")
	}

	if propertiesField["type"] != "object" {
		t.Error("Properties field should be of type object")
	}

	if propertiesField["minProperties"] != 1.0 {
		t.Error("Properties minProperties should be 1")
	}

	// Check required fields for entities
	entityRequired, ok := items["required"].([]interface{})
	if !ok {
		t.Fatal("Entity required fields are missing")
	}

	if len(entityRequired) != len(expectedEntityFields) {
		t.Errorf("Expected %d required entity fields, got %d", len(expectedEntityFields), len(entityRequired))
	}
}

func TestExtractionToolcallRelationshipSchema(t *testing.T) {
	toolcall := GetExtractionToolcall()
	function := toolcall[0]["function"].(map[string]interface{})
	parameters := function["parameters"].(map[string]interface{})
	properties := parameters["properties"].(map[string]interface{})
	relationships := properties["relationships"].(map[string]interface{})

	// Check relationships array items schema
	items, ok := relationships["items"].(map[string]interface{})
	if !ok {
		t.Fatal("Relationships items schema is missing")
	}

	itemProperties, ok := items["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Relationship item properties are missing")
	}

	// Check required relationship fields
	expectedRelationshipFields := []string{"start_node", "end_node", "type", "description", "confidence", "properties", "weight"}
	for _, field := range expectedRelationshipFields {
		if _, exists := itemProperties[field]; !exists {
			t.Errorf("Relationship schema missing required field: %s", field)
		}
	}

	// Check confidence field constraints
	confidence, ok := itemProperties["confidence"].(map[string]interface{})
	if !ok {
		t.Fatal("Relationship confidence field schema is missing")
	}

	if confidence["type"] != "number" {
		t.Error("Relationship confidence field should be of type number")
	}

	if confidence["minimum"] != 0.0 {
		t.Error("Relationship confidence minimum should be 0.0")
	}

	if confidence["maximum"] != 1.0 {
		t.Error("Relationship confidence maximum should be 1.0")
	}

	// Check properties field constraints for relationships
	propertiesField, ok := itemProperties["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Relationship properties field schema is missing")
	}

	if propertiesField["type"] != "object" {
		t.Error("Relationship properties field should be of type object")
	}

	if propertiesField["minProperties"] != 1.0 {
		t.Error("Relationship properties minProperties should be 1")
	}

	// Check weight field constraints
	weight, ok := itemProperties["weight"].(map[string]interface{})
	if !ok {
		t.Fatal("Weight field schema is missing")
	}

	if weight["type"] != "number" {
		t.Error("Weight field should be of type number")
	}

	if weight["minimum"] != 0.0 {
		t.Error("Weight minimum should be 0.0")
	}

	if weight["maximum"] != 1.0 {
		t.Error("Weight maximum should be 1.0")
	}

	// Check required fields for relationships
	relationshipRequired, ok := items["required"].([]interface{})
	if !ok {
		t.Fatal("Relationship required fields are missing")
	}

	if len(relationshipRequired) != len(expectedRelationshipFields) {
		t.Errorf("Expected %d required relationship fields, got %d", len(expectedRelationshipFields), len(relationshipRequired))
	}
}

func TestExtractionToolcallDescriptions(t *testing.T) {
	toolcall := GetExtractionToolcall()
	function := toolcall[0]["function"].(map[string]interface{})

	// Check function description contains key phrases
	description := function["description"].(string)
	keyPhrases := []string{
		"Extract entities and relationships",
		"knowledge graph",
		"CRITICAL",
		"NO HALLUCINATION",
		"confidence scores",
	}

	for _, phrase := range keyPhrases {
		if !strings.Contains(description, phrase) {
			t.Errorf("Function description missing key phrase: %s", phrase)
		}
	}

	// Check parameter descriptions
	parameters := function["parameters"].(map[string]interface{})
	properties := parameters["properties"].(map[string]interface{})

	entities := properties["entities"].(map[string]interface{})
	entitiesDesc := entities["description"].(string)
	if !strings.Contains(entitiesDesc, "unique ID") {
		t.Error("Entities description should mention unique ID requirement")
	}

	relationships := properties["relationships"].(map[string]interface{})
	relationshipsDesc := relationships["description"].(string)
	if !strings.Contains(relationshipsDesc, "entities list") {
		t.Error("Relationships description should mention entities list requirement")
	}
}

func TestExtractionToolcallGlobalVariable(t *testing.T) {
	// Test that the global ExtractionToolcall variable is properly initialized
	if ExtractionToolcall == nil {
		t.Fatal("Global ExtractionToolcall is nil")
	}

	if len(ExtractionToolcall) == 0 {
		t.Fatal("Global ExtractionToolcall is empty")
	}

	// Should be the same as what GetExtractionToolcall returns
	generated := GetExtractionToolcall()
	if len(ExtractionToolcall) != len(generated) {
		t.Error("Global ExtractionToolcall length differs from GetExtractionToolcall")
	}

	// Check that both have the same function name
	globalFunc := ExtractionToolcall[0]["function"].(map[string]interface{})
	generatedFunc := generated[0]["function"].(map[string]interface{})

	if globalFunc["name"] != generatedFunc["name"] {
		t.Error("Global ExtractionToolcall function name differs from GetExtractionToolcall")
	}
}

// Benchmark tests
func BenchmarkExtractionPrompt(b *testing.B) {
	userPrompt := "Extract entities and relationships from this text"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExtractionPrompt(userPrompt)
	}
}

func BenchmarkExtractionPromptEmpty(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExtractionPrompt("")
	}
}

func BenchmarkGetExtractionToolcall(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetExtractionToolcall()
	}
}

func BenchmarkExtractionToolcallJSON(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var toolcall []map[string]interface{}
		_ = json.Unmarshal([]byte(ExtractionToolcallRaw), &toolcall)
	}
}
