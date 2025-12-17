# Text Extract

Extract code blocks from text, especially useful for parsing LLM outputs.

## Features

- Extract markdown code blocks (` ```lang ... ``` `)
- Auto-detect language when not specified
- Parse JSON/YAML with fault-tolerant parser
- Fallback to `text` type when no pattern detected
- Support 15+ languages: Go, Python, JavaScript, TypeScript, Rust, Java, C#, C/C++, Ruby, PHP, Shell, SQL, HTML, CSS, Markdown, etc.

## Go Usage

```go
import "github.com/yaoapp/gou/text"

// Extract all code blocks
blocks := text.Extract(llmResponse)
// Returns: []CodeBlock{{Type: "json", Content: "...", Data: {...}}, ...}

// Extract first block
block := text.ExtractFirst(llmResponse)

// Extract JSON/YAML data directly
data := text.ExtractJSON(llmResponse)

// Extract by type
sqlBlocks := text.ExtractByType(llmResponse, "sql")
```

## JavaScript Usage (Yao Process)

````javascript
// Extract all code blocks
const blocks = Process("text.Extract", llmResponse);
// Returns: [{type: "json", content: "...", data: {...}}, ...]

// Extract first block
const block = Process("text.ExtractFirst", llmResponse);

// Extract JSON/YAML data directly (most common)
const data = Process("text.ExtractJSON", llmResponse);
// Works with: ```json {...} ```, ```yaml ... ```, or raw JSON/YAML

// Extract by type
const sqlBlocks = Process("text.ExtractByType", llmResponse, "sql");
````

## Supported Types

| Category  | Types                                                                                                    |
| --------- | -------------------------------------------------------------------------------------------------------- |
| Data      | `json`, `yaml`, `xml`, `html`, `sql`, `css`, `markdown`                                                  |
| Languages | `go`, `python`, `javascript`, `typescript`, `rust`, `java`, `csharp`, `c`, `cpp`, `ruby`, `php`, `shell` |
| Fallback  | `text` (when no pattern detected)                                                                        |

## CodeBlock Structure

```typescript
interface CodeBlock {
  type: string; // "json", "python", "sql", "text", etc.
  content: string; // Raw content
  data?: any; // Parsed data (JSON/YAML only)
  start?: number; // Start position
  end?: number; // End position
}
```

## Examples

```javascript
// LLM returns markdown-wrapped JSON
const response = `Here is the result:
\`\`\`json
{"keywords": ["apple", "banana"]}
\`\`\``;

const data = Process("text.ExtractJSON", response);
// Returns: {keywords: ["apple", "banana"]}

// LLM returns raw JSON (also works)
const data2 = Process("text.ExtractJSON", '{"name": "test"}');
// Returns: {name: "test"}

// LLM returns YAML (also works)
const data3 = Process("text.ExtractJSON", "name: test\nvalue: 123");
// Returns: {name: "test", value: 123}

// Plain text fallback
const blocks = Process("text.Extract", "Hello, how are you?");
// Returns: [{type: "text", content: "Hello, how are you?"}]
```
