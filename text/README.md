# Text Processing

Text processing utilities including code block extraction and format conversion.

## Features

- Extract markdown code blocks (` ```lang ... ``` `)
- Auto-detect language when not specified
- Parse JSON/YAML with fault-tolerant parser
- Fallback to `text` type when no pattern detected
- Support 15+ languages: Go, Python, JavaScript, TypeScript, Rust, Java, C#, C/C++, Ruby, PHP, Shell, SQL, HTML, CSS, Markdown, etc.
- **Markdown to HTML conversion** (GitHub Flavored Markdown support)
- **HTML to Markdown conversion**

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

// Markdown to HTML
html, err := text.MarkdownToHTML("# Hello\n\nThis is **bold** text.")
// Returns: "<h1 id="hello">Hello</h1>\n<p>This is <strong>bold</strong> text.</p>\n"

// HTML to Markdown
md, err := text.HTMLToMarkdown("<h1>Hello</h1><p>This is <strong>bold</strong> text.</p>")
// Returns: "# Hello\n\nThis is **bold** text."
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

// Markdown to HTML (supports GFM: tables, strikethrough, task lists)
const html = Process("text.MarkdownToHTML", "# Hello\n\nThis is **bold** text.");
// Returns: "<h1 id="hello">Hello</h1>\n<p>This is <strong>bold</strong> text.</p>\n"

// HTML to Markdown
const md = Process("text.HTMLToMarkdown", "<h1>Hello</h1><p>This is <strong>bold</strong> text.</p>");
// Returns: "# Hello\n\nThis is **bold** text."
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

## Markdown/HTML Conversion

### Markdown to HTML

Supports GitHub Flavored Markdown (GFM) including:

- Tables
- Strikethrough (`~~text~~`)
- Task lists (`- [x] done`)
- Autolinks

```javascript
// Basic conversion
const html = Process("text.MarkdownToHTML", "# Title\n\nParagraph with **bold**.");

// GFM table
const tableHtml = Process("text.MarkdownToHTML", `
| Name | Age |
|------|-----|
| John | 30  |
`);

// GFM task list
const taskHtml = Process("text.MarkdownToHTML", `
- [x] Task done
- [ ] Task pending
`);
```

### HTML to Markdown

Converts HTML back to clean Markdown:

```javascript
// Basic conversion
const md = Process("text.HTMLToMarkdown", "<h1>Title</h1><p>Text</p>");
// Returns: "# Title\n\nText"

// Links and images
const md2 = Process("text.HTMLToMarkdown", '<a href="https://example.com">Link</a>');
// Returns: "[Link](https://example.com)"

// Lists
const md3 = Process("text.HTMLToMarkdown", "<ul><li>Item 1</li><li>Item 2</li></ul>");
// Returns: "- Item 1\n- Item 2"
```
