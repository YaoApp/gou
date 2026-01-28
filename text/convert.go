package text

import (
	"bytes"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// MarkdownToHTML converts Markdown text to HTML
// Supports GitHub Flavored Markdown (GFM) extensions including:
// - Tables
// - Strikethrough
// - Task lists
// - Autolinks
func MarkdownToHTML(markdown string) (string, error) {
	var buf bytes.Buffer

	gm := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // GitHub Flavored Markdown
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	if err := gm.Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// HTMLToMarkdown converts HTML to Markdown
// Uses goquery-based parsing for reliable HTML handling
func HTMLToMarkdown(htmlContent string) (string, error) {
	return htmltomarkdown.ConvertString(htmlContent)
}
