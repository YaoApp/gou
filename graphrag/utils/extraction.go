package utils

import (
	"encoding/json"
	"strings"
)

const extractionPromptTemplate = `
# Knowledge Graph Extraction

Extract entities and relationships from the provided text for knowledge graph construction.

## Core Rules
- **Accuracy First**: Only extract explicitly stated or clearly implied information
- **No Hallucination**: Never invent facts not present in the text
- **Language Consistency**: Use the same language as the user input text for ALL entity names, types, descriptions, and labels
- **Confidence Scoring**: Rate extractions based on textual evidence strength

## Entity Types
Extract entity types based on the actual content. Common examples include but are not limited to:
- People, organizations, locations
- Concepts, events, objects
- Dates, technologies, products
- Any meaningful entities that appear in the text

## Relationship Types  
Extract relationship types that accurately describe the connections found in the text. Examples include but are not limited to:
- Action relationships (works for, creates, uses)
- Spatial relationships (located in, part of)
- Temporal relationships (happens before, during)
- Conceptual relationships (related to, causes)
- Any meaningful relationships that appear in the text

## Confidence Scoring Guidelines

### High Confidence (0.8-1.0)
- Explicitly stated facts with clear textual evidence
- Direct quotes or definitive statements
- Well-known entities with unambiguous references

### Medium Confidence (0.5-0.79)
- Implied relationships with reasonable inference
- Entities mentioned but requiring context interpretation
- Common sense connections clearly supported by text

### Low Confidence (0.3-0.49)
- Weak textual evidence requiring significant interpretation
- Ambiguous references or unclear context
- Tentative connections with limited support

### Minimal Confidence (0.0-0.29)
- Highly speculative or barely supported by text
- Usually excluded from final extraction

## Output Requirements

### Entities
- **id**: Unique descriptive identifier
- **name**: Exact name from text
- **type**: Entity classification based on content
- **description**: Contextual explanation
- **confidence**: Score based on textual evidence (0.0-1.0)
- **labels**: Optional categorization tags (can be empty array)
- **props**: Optional attributes (can be empty object)

### Relationships
- **start_node/end_node**: Entity IDs
- **type**: Relationship classification based on actual connection in text
- **description**: Contextual explanation
- **confidence**: Score based on relationship evidence (0.0-1.0)
- **props**: Optional attributes (can be empty object)
- **weight**: Optional relationship strength (0.0-1.0, defaults to confidence if not specified)

Focus on extracting high-quality, well-supported facts rather than comprehensive coverage.

## Examples

### English Example
**Text**: "John works at Google as a software engineer"
**Output**:
- **Entities**: 
  - John (person): {"role": "software engineer"}
  - Google (organization): {"industry": "technology"}
- **Relationship**: 
  - John works_for Google: {"position": "software engineer"}

**JSON Format**:
{
  "entities": [
    {
      "id": "john_person",
      "name": "John",
      "type": "person", 
      "description": "Software engineer mentioned in text",
      "confidence": 0.9,
      "labels": ["employee"],
      "props": {"role": "software engineer"}
    },
    {
      "id": "google_company",
      "name": "Google",
      "type": "organization",
      "description": "Technology company",
      "confidence": 0.95,
      "labels": ["technology", "company"],
      "props": {"industry": "technology"}
    }
  ],
  "relationships": [
    {
      "start_node": "john_person",
      "end_node": "google_company", 
      "type": "works_for",
      "description": "Employment relationship",
      "confidence": 0.9,
      "props": {"position": "software engineer"},
      "weight": 0.9
    }
  ]
}

### Chinese Example  
**Text**: "张三在腾讯担任高级工程师"
**Output**:
- **Entities**:
  - 张三 (人员): {"职位": "高级工程师"}
  - 腾讯 (公司): {"行业": "科技"}
- **Relationship**:
  - 张三 在工作 腾讯: {"职位": "高级工程师"}

**JSON Format**:
{
  "entities": [
    {
      "id": "zhangsan_person",
      "name": "张三",
      "type": "人员",
      "description": "文中提到的高级工程师", 
      "confidence": 0.9,
      "labels": ["员工"],
      "props": {"职位": "高级工程师"}
    },
    {
      "id": "tencent_company",
      "name": "腾讯", 
      "type": "公司",
      "description": "科技公司",
      "confidence": 0.95,
      "labels": ["科技公司", "企业"],
      "props": {"industry": "科技"}
    }
  ],
  "relationships": [
    {
      "start_node": "zhangsan_person",
      "end_node": "tencent_company",
      "type": "在工作",
      "description": "雇佣关系", 
      "confidence": 0.9,
      "props": {"职位": "高级工程师"},
      "weight": 0.9
    }
  ]
}
`

// Non-toolcall JSON format instructions
const extractionJSONFormatInstructions = `

## JSON Output Format

Return a valid JSON object with this structure:

{
  "entities": [
    {
      "id": "unique_entity_id",
      "name": "Entity Name",
      "type": "dynamic_based_on_content",
      "description": "Description of the entity",
      "confidence": 0.85,
      "labels": ["optional", "category", "tags"],
      "props": {
        "optional_key": "optional_value"
      }
    }
  ],
  "relationships": [
    {
      "start_node": "source_entity_id",
      "end_node": "target_entity_id",
      "type": "dynamic_based_on_content", 
      "description": "Description of relationship",
      "confidence": 0.90,
      "props": {
        "optional_key": "optional_value"
      },
      "weight": 0.85
    }
  ]
}

**Requirements:**
- All listed fields are required
- labels[] and props{} can be empty but must be present
- weight defaults to confidence value if not specified
- Use same language as user input text for all text fields

Return only the JSON object.
`

// ExtractionToolcallRaw is the optimized toolcall for entity and relationship extraction
const ExtractionToolcallRaw = `
[
  {
    "type": "function",
    "function": {
      "name": "extract_entities_and_relationships",
      "description": "Extract entities and relationships from text for knowledge graph construction. Extract only explicitly stated or clearly implied information. Apply confidence scoring based on textual evidence strength. IMPORTANT: Extract meaningful attributes in 'props' field when mentioned in text - avoid empty props unless no attributes are available. Use 'props' instead of 'properties' to avoid JSON Schema conflicts. CRITICAL: Use the same language as user input text for ALL outputs (entity names, types, descriptions, labels, relationship types).",
      "parameters": {
        "type": "object",
        "properties": {
          "entities": {
            "type": "array",
            "description": "List of extracted entities with confidence scoring based on textual evidence",
            "items": {
              "type": "object",
              "properties": {
                "id": {
                  "type": "string",
                  "description": "Unique descriptive identifier for the entity"
                },
                "name": {
                  "type": "string", 
                  "description": "Entity name as it appears in the text (use same language as user input)"
                },
                "type": {
                  "type": "string",
                  "description": "Entity type based on content in same language as user input (e.g., person/人员, organization/公司, location/地点, etc.)"
                },
                "description": {
                  "type": "string",
                  "description": "Contextual description of the entity from the text (use same language as user input)"
                },
                "confidence": {
                  "type": "number",
                  "description": "Confidence score (0.0-1.0): 0.8-1.0 (explicit facts), 0.5-0.79 (reasonable inference), 0.3-0.49 (weak evidence), 0.0-0.29 (speculative)",
                  "minimum": 0.0,
                  "maximum": 1.0
                },
                "labels": {
                  "type": "array",
                  "description": "Optional category labels/tags. Can be empty array if no meaningful categorization available.",
                  "items": {
                    "type": "string"
                  },
                  "default": []
                },
                "props": {
                  "type": "object", 
                  "description": "Entity attributes as key-value pairs. Include meaningful properties when available from text (e.g., {\"industry\": \"technology\"}, {\"role\": \"engineer\"}, {\"location\": \"headquarters\"}). Can be empty object only if no specific attributes mentioned.",
                  "default": {}
                }
              },
              "required": ["id", "name", "type", "description", "confidence", "labels", "props"]
            }
          },
          "relationships": {
            "type": "array",
            "description": "List of extracted relationships with confidence scoring based on relationship evidence",
            "items": {
              "type": "object",
              "properties": {
                "start_node": {
                  "type": "string",
                  "description": "ID of source entity (must match an entity ID)"
                },
                "end_node": {
                  "type": "string", 
                  "description": "ID of target entity (must match an entity ID)"
                },
                "type": {
                  "type": "string",
                  "description": "Relationship type based on actual connection in text, use same language as user input (e.g., works_for/在工作, located_in/位于, part_of/属于, etc.)"
                },
                "description": {
                  "type": "string",
                  "description": "Contextual description of the relationship from the text (use same language as user input)"
                },
                "confidence": {
                  "type": "number",
                  "description": "Confidence score (0.0-1.0): 0.8-1.0 (explicit), 0.5-0.79 (implied), 0.3-0.49 (weak), 0.0-0.29 (speculative)",
                  "minimum": 0.0,
                  "maximum": 1.0
                },
                "props": {
                  "type": "object",
                  "description": "Relationship attributes as key-value pairs. Include meaningful properties when available from text (e.g., {\"duration\": \"5 years\"}, {\"type\": \"full_time\"}, {\"since\": \"2020\"}). Can be empty object only if no specific attributes mentioned.",
                  "default": {}
                },
                "weight": {
                  "type": "number",
                  "description": "Optional relationship strength/importance (0.0-1.0). Defaults to confidence value if not specified.",
                  "minimum": 0.0,
                  "maximum": 1.0
                }
              },
              "required": ["start_node", "end_node", "type", "description", "confidence", "props"]
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
