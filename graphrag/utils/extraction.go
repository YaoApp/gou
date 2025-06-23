package utils

import (
	"encoding/json"
	"strings"
)

const extractionPromptTemplate = `
# Entity and Relationship Extraction Task

🚨 **CRITICAL INSTRUCTIONS**: You are an expert knowledge graph extraction system. Your task is to extract entities and relationships from the provided text with high accuracy and completeness.

## Core Principles

### ✅ Required Actions
- **Extract ALL relevant entities**: Identify all important entities including people, organizations, locations, concepts, events, objects, etc.
- **Extract ALL relationships**: Identify all meaningful relationships between entities
- **Maintain accuracy**: Only extract information that is explicitly stated or strongly implied in the text
- **Provide descriptions**: Include clear, concise descriptions for entities and relationships
- **Assign confidence scores**: Rate your confidence in each extraction (0.0-1.0)
- **Categorize properly**: Assign appropriate types to entities and relationships

### ❌ STRICTLY FORBIDDEN
- **NO HALLUCINATION**: Never invent information not present in the text
- **NO SPECULATION**: Don't extract relationships that aren't clearly indicated
- **NO GENERIC ENTITIES**: Avoid overly broad or meaningless entity names
- **NO DUPLICATE ENTITIES**: Each entity should have a unique ID and name

## Entity Types
Common entity types include but are not limited to:
- **PERSON**: Individual people, characters
- **ORGANIZATION**: Companies, institutions, groups
- **LOCATION**: Places, cities, countries, buildings
- **EVENT**: Happenings, incidents, activities
- **CONCEPT**: Ideas, theories, principles
- **OBJECT**: Physical items, products, tools
- **DATE**: Temporal references
- **TECHNOLOGY**: Software, systems, methods

## Relationship Types
Common relationship types include but are not limited to:
- **WORKS_FOR**: Person works for organization
- **LOCATED_IN**: Entity is located in location
- **PART_OF**: Entity is part of another entity
- **RELATED_TO**: General relationship between entities
- **CAUSES**: One entity causes another
- **USES**: Entity uses another entity
- **CREATES**: Entity creates another entity
- **LEADS**: Entity leads another entity
- **PARTICIPATES_IN**: Entity participates in event

## Example

**Input Text**: "John Smith works for Google in Mountain View. He is the lead engineer of the AI team that developed the search algorithm."

**Expected Output**:
- **Entities**: John Smith (PERSON), Google (ORGANIZATION), Mountain View (LOCATION), AI team (ORGANIZATION), search algorithm (TECHNOLOGY)
- **Relationships**: John Smith WORKS_FOR Google, Google LOCATED_IN Mountain View, John Smith LEADS AI team, AI team CREATES search algorithm

## Quality Requirements

### High-Quality Entities
- **Specific names**: Use exact names from text, not generic terms
- **Proper types**: Assign the most specific appropriate type
- **Rich descriptions**: Provide context and details
- **High confidence**: Only extract entities you're confident about

### High-Quality Relationships
- **Clear semantics**: Relationship type should clearly describe the connection
- **Bidirectional awareness**: Consider if relationships are directional
- **Contextual descriptions**: Explain the relationship in context
- **Confidence scoring**: Rate based on how explicitly stated the relationship is

## Output Format

🚨 **CRITICAL**: Use the provided function calls to structure your output. Each entity and relationship must be properly formatted with all required fields.

## Key Reminders

1. **Read carefully**: Analyze the entire text before extracting
2. **Be comprehensive**: Don't miss important entities or relationships
3. **Stay grounded**: Only extract what's actually in the text
4. **Maintain consistency**: Use consistent naming and typing
5. **Quality over quantity**: Better to extract fewer high-quality items than many low-quality ones
6. **Consider context**: Understand the domain and context of the text
7. **Validate confidence**: Be honest about your confidence levels
8. **Unique identification**: Each entity should have a unique, descriptive ID

🚨 **FINAL REMINDER**: Your extractions will be used for knowledge graph construction. Accuracy and completeness are paramount!
`

// ExtractionToolcallRaw is the toolcall for entity and relationship extraction
const ExtractionToolcallRaw = `
[
  {
    "type": "function",
    "function": {
      "name": "extract_entities_and_relationships",
      "description": "Extract entities and relationships from text for knowledge graph construction. CRITICAL: Only extract information that is explicitly stated or strongly implied in the text. NO HALLUCINATION allowed. Provide accurate confidence scores and detailed descriptions.",
      "parameters": {
        "type": "object",
        "properties": {
          "entities": {
            "type": "array",
            "description": "List of extracted entities with their properties. Each entity must have a unique ID and proper type classification.",
            "items": {
              "type": "object",
              "properties": {
                "id": {
                  "type": "string",
                  "description": "Unique identifier for the entity (use descriptive names, e.g., 'john_smith_google_engineer')"
                },
                "name": {
                  "type": "string",
                  "description": "The actual name or title of the entity as it appears in the text"
                },
                "type": {
                  "type": "string",
                  "description": "Entity type (e.g., PERSON, ORGANIZATION, LOCATION, CONCEPT, EVENT, OBJECT, DATE, TECHNOLOGY)"
                },
                "description": {
                  "type": "string",
                  "description": "Detailed description of the entity including context from the text"
                },
                "confidence": {
                  "type": "number",
                  "description": "Confidence score for this entity extraction (0.0-1.0)",
                  "minimum": 0.0,
                  "maximum": 1.0
                }
              },
              "required": ["id", "name", "type", "description", "confidence"]
            }
          },
          "relationships": {
            "type": "array",
            "description": "List of extracted relationships between entities. Each relationship must connect two entities that exist in the entities list.",
            "items": {
              "type": "object",
              "properties": {
                "start_node": {
                  "type": "string",
                  "description": "ID of the source entity (must match an entity ID from the entities list)"
                },
                "end_node": {
                  "type": "string",
                  "description": "ID of the target entity (must match an entity ID from the entities list)"
                },
                "type": {
                  "type": "string",
                  "description": "Relationship type (e.g., WORKS_FOR, LOCATED_IN, PART_OF, RELATED_TO, CAUSES, USES, CREATES, LEADS)"
                },
                "description": {
                  "type": "string",
                  "description": "Detailed description of the relationship including context from the text"
                },
                "confidence": {
                  "type": "number",
                  "description": "Confidence score for this relationship extraction (0.0-1.0)",
                  "minimum": 0.0,
                  "maximum": 1.0
                }
              },
              "required": ["start_node", "end_node", "type", "description", "confidence"]
            }
          }
        },
        "required": ["entities", "relationships"]
      }
    }
  }
]
`

// ExtractionToolcall is the extraction toolcall
var ExtractionToolcall = GetExtractionToolcall()

// ExtractionPrompt returns the extraction prompt
func ExtractionPrompt(userPrompt string) string {
	if strings.TrimSpace(userPrompt) != "" {
		return userPrompt
	}
	return extractionPromptTemplate
}

// GetExtractionToolcall returns the extraction toolcall
func GetExtractionToolcall() []map[string]interface{} {
	var toolcall = []map[string]interface{}{}
	json.Unmarshal([]byte(ExtractionToolcallRaw), &toolcall)
	return toolcall
}
