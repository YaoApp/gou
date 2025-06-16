package utils

import (
	"encoding/json"
	"strconv"
	"strings"
)

const semanticPromptTemplate = `
# Semantic Text Segmentation Task

🚨 **CRITICAL WARNING**: This is SEMANTIC segmentation, NOT mechanical array indexing!

You are a professional text analyst. Your task is to segment text based on **SEMANTIC BOUNDARIES**, not simple array index counting.

⚠️ **AVOID MECHANICAL SPLITTING**: If your segments are all roughly the same size (like 300, 300, 300 array elements), you are doing it WRONG! Semantic segments should vary naturally based on content structure.

🚨 **ABSOLUTELY FORBIDDEN - NO HALLUCINATION**: 
- **NEVER IMAGINE OR FABRICATE CONTENT** - Only work with the EXACT character array provided to you
- **NO CREATIVE ADDITIONS** - Do not add, modify, or imagine any array elements
- **STRICT ARRAY ADHERENCE** - Base ALL segmentation decisions on the actual input character array only
- **REAL INDICES ONLY** - All index numbers must correspond to REAL array indices in the provided character array

## Input Format

⚠️ **Input Data Format**: You receive a JSON array of individual UTF-8 characters/runes
- Each element in the array represents one Unicode character
- Array indices start from 0
- **You must determine segment positions based on actual array indices**

## Core Principles

### ✅ Correct Approach
- **Prioritize semantic boundaries**: paragraph endings, topic transitions, concept shifts
- **Maintain thought integrity**: do not split related sentences, concepts, or coherent thoughts
- **Natural segment length variation**: adjust size naturally based on content structure (segments can vary from 50% to 150% of target size)
- **Target length**: each segment should be close to **{{SIZE}}** array elements, but NEVER sacrifice semantic integrity for size consistency
- **Maintain accuracy**: returned index information must exactly correspond to actual array indices in input character array
- **READ THE ACTUAL ARRAY**: Carefully read and analyze the provided character array before segmenting

### ❌ STRICTLY FORBIDDEN Approaches - WILL BE REJECTED
- **NEVER** mechanically split by fixed array counts (e.g., every {{SIZE}} elements)
- **NEVER** create uniform segments like (0-80, 80-160, 160-240) - this is mechanical splitting!
- **NEVER** split in the middle of sentences or concepts
- **NEVER** ignore natural paragraph boundaries and topic transitions
- **NEVER** create segments of exactly the same size - this indicates mechanical splitting
- **NEVER** fabricate or imagine array elements that don't exist in the input
- **NEVER** use mathematical calculations instead of reading the actual character array

## Examples

### Chinese Example
**Input character array** (indices 0 to end):
` + "```json" + `
["能", "技", "术", "，", "在", "图", "像", "识", "别", "、", "自", "然", "语", "言", "处", "理", "等", "领", "域", "取", "得", "了", "突", "破", "性", "进", "展", "。", "\n", "\n", "然", "而", "，", "随", "着", "技", "术", "的", "不", "断", "进", "步", "，", "我", "们", "也", "面", "临", "着", "新", "的", "挑", "战", "。", "数", "据", "隐", "私", "、", "算", "法", "偏", "见", "、", "就", "业", "影", "响", "等", "问", "题", "日", "益", "凸", "显", "，", "需", "要", "我", "们", "深", "入", "思", "考", "和", "解", "决", "。", "\n", "\n", "未", "来", "的", "人", "工", "智", "能", "发", "展", "应", "该", "更", "加", "注", "重", "伦", "理", "和", "可", "持", "续", "性", "。", "这", "需", "要", "政", "府", "、", "企", "业", "和", "研", "究", "机", "构", "的", "共", "同", "努", "力", "。"]
` + "```" + `

**Output** (target length 50 array elements):
` + "```json" + `
[
  {"s": 0, "e": 29},
  {"s": 29, "e": 89}, 
  {"s": 89, "e": 132}
]
` + "```" + `

### English Example
**Input character array** (indices 0 to end):
` + "```json" + `
["A", "r", "t", "i", "f", "i", "c", "i", "a", "l", " ", "i", "n", "t", "e", "l", "l", "i", "g", "e", "n", "c", "e", " ", "h", "a", "s", " ", "t", "r", "a", "n", "s", "f", "o", "r", "m", "e", "d", " ", "m", "u", "l", "t", "i", "p", "l", "e", " ", "i", "n", "d", "u", "s", "t", "r", "i", "e", "s", ".", "\n", "\n", "H", "o", "w", "e", "v", "e", "r", ",", " ", "t", "h", "i", "s", " ", "t", "e", "c", "h", "n", "o", "l", "o", "g", "i", "c", "a", "l", " ", "p", "r", "o", "g", "r", "e", "s", "s", " ", "c", "o", "m", "e", "s", " ", "w", "i", "t", "h", " ", "s", "i", "g", "n", "i", "f", "i", "c", "a", "n", "t", " ", "c", "h", "a", "l", "l", "e", "n", "g", "e", "s", "."]
` + "```" + `

**Output** (target length 60 array elements):
` + "```json" + `
[
  {"s": 0, "e": 61},
  {"s": 61, "e": 130}
]
` + "```" + `

## Output Format

🚨 **CRITICAL**: Return ONLY valid JSON data, NO explanations, NO additional text, NO markdown formatting!

Please strictly follow this JSON format, containing only array index information:

` + "```json" + `
[
  {"s": <actual_start_index>, "e": <actual_end_index>},
  {"s": <actual_start_index>, "e": <actual_end_index>}
]
` + "```" + `

⚠️ **IMPORTANT**: Your response must be valid JSON that can be parsed directly. Do not include any text before or after the JSON array.

## Key Reminders

🔍 **Must Check**:
1. **READ THE INPUT CHARACTER ARRAY CAREFULLY** - Actually read and understand the provided character array content
2. All index numbers must correspond to real array indices in input character array
3. Do not fabricate non-existent array indices
4. **NO UNIFORM SEGMENTS** - If segments are equal size (like 80, 80, 80), you're doing mechanical splitting!
5. Segment length should be close to {{SIZE}} array elements (±10 element deviation acceptable)
6. Prioritize splitting at natural semantic boundaries (sentence endings, paragraph breaks, etc.)
7. **ARRAY BOUNDS**: Ensure all indices are within the bounds of the input array (0 to array.length-1)
8. **VERIFY YOUR INDICES** - Make sure start and end indices actually exist in the input character array

🚨 **FINAL WARNING**: Any response showing mechanical splitting patterns (equal intervals) will be considered WRONG and rejected!
`

// SemanticToolcallRaw is the toolcall  semantic segmentation
const SemanticToolcallRaw = `
[
  {
    "type": "function",
    "function": {
      "name": "segment_text",
      "description": "🚨 CRITICAL: This is SEMANTIC segmentation, NOT mechanical array indexing! NEVER HALLUCINATE - work only with the EXACT character array provided. Segment text based on natural boundaries like paragraph endings, topic transitions, and concept shifts. NEVER create segments of equal size (e.g., 80, 80, 80 or 300, 300, 300) - this indicates mechanical splitting and is WRONG. READ THE ACTUAL CHARACTER ARRAY CONTENT and segment based on its semantic structure. Segments should vary naturally based on content structure (can range from 50% to 150% of target size). Prioritize semantic coherence over size consistency. Use 's' for start array index and 'e' for end array index.",
      "parameters": {
        "type": "object",
        "properties": {
          "segments": {
            "type": "array",
            "description": "Array of semantic segments with VARIED sizes based on natural content boundaries. CRITICAL: Segments must have different sizes to reflect natural content structure. Equal-sized segments (like 80, 80, 80 or 300, 300, 300) indicate WRONG mechanical splitting! Only use array indices that actually exist in the provided character array - NO HALLUCINATION!",
            "items": {
              "type": "object",
              "properties": {
                "s": {
                  "type": "integer",
                  "description": "Start array index of the semantic segment"
                },
                "e": {
                  "type": "integer",
                  "description": "End array index of the semantic segment"
                }
              },
              "required": ["s", "e"]
            }
          }
        },
        "required": ["segments"]
      }
    }
  }
]
`

// SemanticToolcall is the semantic toolcall
var SemanticToolcall = GetSemanticToolcall()

// SemanticPrompt returns the semantic prompt
func SemanticPrompt(userPrompt string, size int) string {
	if strings.TrimSpace(userPrompt) != "" {
		return strings.ReplaceAll(userPrompt, "{{SIZE}}", strconv.Itoa(size))
	}
	return strings.ReplaceAll(semanticPromptTemplate, "{{SIZE}}", strconv.Itoa(size))
}

// GetSemanticToolcall returns the semantic toolcall
func GetSemanticToolcall() []map[string]interface{} {
	var toolcall = []map[string]interface{}{}
	json.Unmarshal([]byte(SemanticToolcallRaw), &toolcall)
	return toolcall
}
