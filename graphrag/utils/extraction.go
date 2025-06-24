package utils

import (
	"encoding/json"
	"strings"
)

const extractionPromptTemplate = `
# Entity and Relationship Extraction Task

🚨 **CRITICAL INSTRUCTIONS**: You are an expert knowledge graph extraction system. Your task is to extract entities and relationships from the provided text with high accuracy and completeness.

🔥 **MANDATORY FIELD REQUIREMENTS** - ABSOLUTELY REQUIRED:
- Every entity MUST have non-empty "labels" array (minimum 1 item)
- Every entity MUST have non-empty "properties" object (minimum 1 property)  
- Every relationship MUST have non-empty "properties" object (minimum 1 property)
- Every relationship MUST have "weight" value (0.0-1.0)
- FAILURE TO PROVIDE THESE FIELDS WILL RESULT IN EXTRACTION ERROR

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
- **🔥 REQUIRED labels**: ALWAYS provide at least one label/category for each entity
- **🔥 REQUIRED properties**: ALWAYS provide at least one property (even if just {"extracted": true})

### High-Quality Relationships
- **Clear semantics**: Relationship type should clearly describe the connection
- **Bidirectional awareness**: Consider if relationships are directional
- **Contextual descriptions**: Explain the relationship in context
- **Confidence scoring**: Rate based on how explicitly stated the relationship is
- **🔥 REQUIRED properties**: ALWAYS provide at least one property (even if just {"extracted": true})
- **🔥 REQUIRED weight**: ALWAYS assign a weight between 0.0-1.0 based on relationship strength

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
9. **🌐 Language consistency**: **CRITICAL** - Use the SAME LANGUAGE as the input text for all entity names, descriptions, and relationship descriptions. If the input is in Chinese, output in Chinese. If in English, output in English. If in Japanese, output in Japanese, etc. DO NOT translate or change the language of the extracted content.

🚨 **FINAL REMINDER**: Your extractions will be used for knowledge graph construction. Accuracy, completeness, and LANGUAGE CONSISTENCY are paramount!

🔥 **CRITICAL FIELD REQUIREMENTS**: 
- Every entity MUST have "labels" and "properties" fields (non-empty)
- Every relationship MUST have "properties" and "weight" fields (non-empty)
- If you cannot determine specific labels/properties, use generic ones like ["entity"] or {"type": "general"}

🚨 **PROPERTIES FIELD EXAMPLES - MANDATORY TO INCLUDE**:
For entities: {"domain": "software", "category": "tool"} or {"type": "technology", "purpose": "development"}
For relationships: {"context": "usage", "method": "application"} or {"type": "functional", "strength": "strong"}

🔥 **STRICT JSON FORMAT REQUIREMENT**:
EVERY entity MUST follow this exact format:
{
  "id": "...",
  "name": "...",
  "type": "...",
  "description": "...",
  "confidence": 0.9,
  "labels": ["label1", "label2"],
  "properties": {"key1": "value1", "key2": "value2"}
}

EVERY relationship MUST follow this exact format:
{
  "start_node": "...",
  "end_node": "...",
  "type": "...",
  "description": "...",
  "confidence": 0.9,
  "properties": {"key1": "value1", "key2": "value2"},
  "weight": 0.9
}
`

// Non-toolcall JSON format instructions
const extractionJSONFormatInstructions = `

## 🚨 CRITICAL JSON OUTPUT FORMAT REQUIREMENTS 🚨

You MUST return your response as a valid JSON object with the following EXACT structure. Do NOT include any other text, explanations, or markdown formatting. Only return the JSON:

{
  "entities": [
    {
      "id": "unique_entity_id",
      "name": "Entity Name",
      "type": "ENTITY_TYPE",
      "description": "Detailed description of the entity",
      "confidence": 0.95,
      "labels": ["technology", "software", "tool"],
      "properties": {
        "domain": "software_development",
        "purpose": "application_building",
        "approach": "generative_programming"
      }
    }
  ],
  "relationships": [
    {
      "start_node": "source_entity_id",
      "end_node": "target_entity_id", 
      "type": "RELATIONSHIP_TYPE",
      "description": "Detailed description of the relationship",
      "confidence": 0.90,
      "properties": {
        "context": "development_process",
        "method": "automated_generation",
        "benefit": "faster_development"
      },
      "weight": 0.85
    }
  ]
}

### MANDATORY FIELD REQUIREMENTS:

**For each entity:**
- ✅ "id": MUST be a unique, descriptive identifier (e.g., "john_smith_engineer", "google_company")
- ✅ "name": MUST be the exact name as it appears in the text
- ✅ "type": MUST be one of: PERSON, ORGANIZATION, LOCATION, CONCEPT, EVENT, OBJECT, DATE, TECHNOLOGY, or similar
- ✅ "description": MUST provide context and details about the entity
- ✅ "confidence": MUST be a number between 0.0 and 1.0
- 🚨 "labels": MANDATORY array of category labels/tags for the entity (minimum 1 item required)
- 🚨 "properties": MANDATORY object with key-value properties and attributes (minimum 1 property required)

**For each relationship:**
- ✅ "start_node": MUST exactly match an entity "id" from the entities array
- ✅ "end_node": MUST exactly match an entity "id" from the entities array  
- ✅ "type": MUST be descriptive (e.g., "WORKS_FOR", "LOCATED_IN", "PART_OF", "CREATES")
- ✅ "description": MUST explain the relationship with context
- ✅ "confidence": MUST be a number between 0.0 and 1.0
- 🚨 "properties": MANDATORY object with key-value properties and attributes (minimum 1 property required)
- 🚨 "weight": MANDATORY number (0.0-1.0) representing relationship strength/importance

### ❌ CRITICAL VALIDATION RULES:
- NO empty strings ("") for any field
- NO missing required fields
- NO relationships with non-existent entity IDs
- NO duplicate entity IDs
- NO invalid JSON syntax
- NO additional text outside the JSON

### 🌐 LANGUAGE CONSISTENCY:
- Use the SAME LANGUAGE as the input text for ALL entity names, descriptions, and relationship descriptions
- If input is Chinese, output Chinese. If English, output English. DO NOT translate!

🚨 **CRITICAL REQUIREMENTS CHECKLIST**:
✅ Every entity has "labels" array with at least 1 item
✅ Every entity has "properties" object with at least 1 key-value pair  
✅ Every relationship has "properties" object with at least 1 key-value pair
✅ Every relationship has "weight" number between 0.0-1.0

**EXTRACTION WILL FAIL if any of these fields are missing!**

### 📝 **EXAMPLE FOR REFERENCE**:
For text: "John works at Google as a software engineer"

REQUIRED OUTPUT FORMAT:
{
  "entities": [
    {
      "id": "john_person",
      "name": "John",
      "type": "PERSON", 
      "description": "A person mentioned in the text",
      "confidence": 0.9,
      "labels": ["person", "employee"],
      "properties": {
        "role": "software engineer",
        "mentioned_context": "workplace"
      }
    },
    {
      "id": "google_company",
      "name": "Google",
      "type": "ORGANIZATION",
      "description": "Technology company mentioned as employer",
      "confidence": 0.95,
      "labels": ["company", "technology", "employer"],
      "properties": {
        "industry": "technology",
        "type": "corporation"
      }
    }
  ],
  "relationships": [
    {
      "start_node": "john_person",
      "end_node": "google_company",
      "type": "WORKS_FOR",
      "description": "John is employed by Google",
      "confidence": 0.9,
      "properties": {
        "employment_type": "full_time",
        "role_description": "software engineer"
      },
      "weight": 0.8
    }
  ]
}

🚨 **FINAL WARNING**: Return ONLY the JSON object. Any additional text will cause parsing errors!
`

// ExtractionToolcallRaw is the toolcall for entity and relationship extraction
const ExtractionToolcallRaw = `
[
  {
    "type": "function",
    "function": {
      "name": "extract_entities_and_relationships",
      "description": "Extract entities and relationships from text for knowledge graph construction. CRITICAL: Only extract information that is explicitly stated or strongly implied in the text. NO HALLUCINATION allowed. Provide accurate confidence scores and detailed descriptions. 🚨 ABSOLUTE REQUIREMENT: Every entity MUST have 'labels' array (at least 1 item) and 'properties' object (at least 1 key-value pair). Every relationship MUST have 'properties' object (at least 1 key-value pair) and 'weight' number. PROPERTIES FIELD IS MANDATORY - examples: entity properties {\"domain\": \"software\", \"type\": \"tool\"}, relationship properties {\"context\": \"usage\", \"strength\": \"high\"}. These fields are NOT optional - omitting them will cause extraction failure.",
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
                },
                "labels": {
                  "type": "array",
                  "description": "MANDATORY: Entity labels/categories. Must provide at least one label. Cannot be empty.",
                  "items": {
                    "type": "string"
                  },
                  "minItems": 1
                },
                "properties": {
                  "type": "object",
                  "description": "MANDATORY: Entity properties and attributes as key-value pairs. Must provide at least one property. Cannot be empty. Example: {\"domain\": \"software\", \"purpose\": \"development\"}",
                  "additionalProperties": true,
                  "minProperties": 1,
                  "default": {"category": "entity"}
                }
              },
              "required": ["id", "name", "type", "description", "confidence", "labels", "properties"]
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
                },
                "properties": {
                  "type": "object",
                  "description": "MANDATORY: Relationship properties and attributes as key-value pairs. Must provide at least one property. Cannot be empty. Example: {\"context\": \"development\", \"method\": \"tool_usage\"}",
                  "additionalProperties": true,
                  "minProperties": 1,
                  "default": {"type": "relationship"}
                },
                "weight": {
                  "type": "number",
                  "description": "MANDATORY: Relationship strength/weight (0.0-1.0), representing the strength or importance of this relationship. Must be provided. Cannot be null or empty.",
                  "minimum": 0.0,
                  "maximum": 1.0
                }
              },
              "required": ["start_node", "end_node", "type", "description", "confidence", "properties", "weight"]
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

// ExtractionPrompt returns the extraction prompt, with JSON format instructions for non-toolcall mode
func ExtractionPrompt(userPrompt string) string {
	if strings.TrimSpace(userPrompt) != "" {
		return userPrompt
	}
	return extractionPromptTemplate
}

// ExtractionPromptWithJSONFormat returns the extraction prompt with JSON format instructions for non-toolcall mode
func ExtractionPromptWithJSONFormat(userPrompt string) string {
	basePrompt := ExtractionPrompt(userPrompt)
	return basePrompt + extractionJSONFormatInstructions
}

// GetExtractionToolcall returns the extraction toolcall
func GetExtractionToolcall() []map[string]interface{} {
	var toolcall = []map[string]interface{}{}
	json.Unmarshal([]byte(ExtractionToolcallRaw), &toolcall)
	return toolcall
}
