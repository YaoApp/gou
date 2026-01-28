package text

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdownToHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "basic heading",
			input: "# Hello World",
			contains: []string{
				"<h1",
				"Hello World",
				"</h1>",
			},
		},
		{
			name:  "bold text",
			input: "This is **bold** text",
			contains: []string{
				"<strong>bold</strong>",
			},
		},
		{
			name:  "italic text",
			input: "This is *italic* text",
			contains: []string{
				"<em>italic</em>",
			},
		},
		{
			name:  "link",
			input: "[Example](https://example.com)",
			contains: []string{
				`<a href="https://example.com"`,
				"Example",
				"</a>",
			},
		},
		{
			name:  "code block",
			input: "```go\nfunc main() {}\n```",
			contains: []string{
				"<pre>",
				"<code",
				"func main()",
				"</code>",
				"</pre>",
			},
		},
		{
			name:  "GFM table",
			input: "| A | B |\n|---|---|\n| 1 | 2 |",
			contains: []string{
				"<table>",
				"<th>A</th>",
				"<td>1</td>",
				"</table>",
			},
		},
		{
			name:  "GFM strikethrough",
			input: "~~deleted~~",
			contains: []string{
				"<del>deleted</del>",
			},
		},
		{
			name:  "GFM task list",
			input: "- [x] Done\n- [ ] Todo",
			contains: []string{
				`<input checked=""`,
				`<input `,
				"disabled",
				"type=\"checkbox\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MarkdownToHTML(tt.input)
			assert.NoError(t, err)
			for _, s := range tt.contains {
				assert.True(t, strings.Contains(result, s), "Expected %q to contain %q, got: %s", tt.input, s, result)
			}
		})
	}
}

func TestHTMLToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "heading",
			input: "<h1>Hello World</h1>",
			contains: []string{
				"# Hello World",
			},
		},
		{
			name:  "bold",
			input: "<p>This is <strong>bold</strong> text</p>",
			contains: []string{
				"**bold**",
			},
		},
		{
			name:  "italic",
			input: "<p>This is <em>italic</em> text</p>",
			contains: []string{
				"*italic*",
			},
		},
		{
			name:  "link",
			input: `<a href="https://example.com">Example</a>`,
			contains: []string{
				"[Example](https://example.com)",
			},
		},
		{
			name:  "unordered list",
			input: "<ul><li>Item 1</li><li>Item 2</li></ul>",
			contains: []string{
				"- Item 1",
				"- Item 2",
			},
		},
		{
			name:  "ordered list",
			input: "<ol><li>First</li><li>Second</li></ol>",
			contains: []string{
				"1. First",
				"2. Second",
			},
		},
		{
			name:  "code",
			input: "<code>inline code</code>",
			contains: []string{
				"`inline code`",
			},
		},
		{
			name:  "blockquote",
			input: "<blockquote>Quote text</blockquote>",
			contains: []string{
				"> Quote text",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HTMLToMarkdown(tt.input)
			assert.NoError(t, err)
			for _, s := range tt.contains {
				assert.True(t, strings.Contains(result, s), "Expected HTML %q to convert to markdown containing %q, got: %s", tt.input, s, result)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that markdown -> html -> markdown preserves content
	originalMD := "# Title\n\nThis is **bold** and *italic* text.\n\n- Item 1\n- Item 2"

	html, err := MarkdownToHTML(originalMD)
	assert.NoError(t, err)

	backToMD, err := HTMLToMarkdown(html)
	assert.NoError(t, err)

	// Check key elements are preserved
	assert.Contains(t, backToMD, "Title")
	assert.Contains(t, backToMD, "**bold**")
	assert.Contains(t, backToMD, "*italic*")
	assert.Contains(t, backToMD, "Item 1")
	assert.Contains(t, backToMD, "Item 2")
}
