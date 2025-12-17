package text

import (
	"regexp"
	"strings"

	"github.com/yaoapp/gou/json"
)

// CodeBlock represents an extracted code block from text
type CodeBlock struct {
	Type    string      `json:"type"`            // Language type: json, html, python, sql, etc.
	Content string      `json:"content"`         // Raw content of the code block
	Data    interface{} `json:"data,omitempty"`  // Parsed data (for JSON/YAML)
	Start   int         `json:"start,omitempty"` // Start position in original text
	End     int         `json:"end,omitempty"`   // End position in original text
}

// Common language indicators for detection
// Note: Order matters - more specific patterns should be checked first
var languageIndicators = map[string][]string{
	"json": {"{", "["},
	"html": {"<!DOCTYPE", "<html", "<div", "<span", "<p>", "<head", "<body", "<script", "<style", "<table", "<form"},
	"xml":  {"<?xml", "<root", "<item", "<node"},
	"yaml": {"---"},
	"sql":  {"SELECT ", "INSERT ", "UPDATE ", "DELETE ", "CREATE ", "DROP ", "ALTER "},

	// Programming languages - use distinctive patterns
	"go":         {"package ", "func ", "import (", "import \""},
	"python":     {"def ", "class ", "if __name__", "#!/usr/bin/env python", "#!/usr/bin/python", "print("},
	"javascript": {"function ", "require(", "module.exports", "console.log"},
	"typescript": {"interface ", ": string", ": number", ": boolean", ": any", "as const"},
	"rust":       {"fn ", "let mut ", "impl ", "println!", "pub fn ", "use std::"},
	"java":       {"public class ", "public static void main", "System.out", "import java."},
	"csharp":     {"using System", "Console.Write", "public class ", "static void Main"},
	"cpp":        {"#include <iostream>", "#include <vector>", "std::", "cout <<", "cin >>"},
	"c":          {"#include <stdio.h>", "#include <stdlib.h>", "printf(", "scanf(", "int main("},
	"ruby":       {"puts ", "attr_accessor", "attr_reader", "def ", "end\n"},
	"php":        {"<?php", "<?=", "echo ", "$_"},
	"shell":      {"#!/bin/bash", "#!/bin/sh", "#!/usr/bin/env bash", "echo $", "export "},
	"css":        {"margin:", "padding:", "display:", "color:", "background:", "font-"},
	"markdown":   {"# ", "## ", "### "},
}

// Extract extracts code blocks from text (typically LLM output)
// It handles:
// 1. Markdown code blocks (```lang ... ```)
// 2. Direct JSON/HTML/Code without markdown wrapper
// 3. JSON is parsed using fault-tolerant parser
// 4. Fallback: returns original text as type "text" if no blocks detected
//
// Returns a slice of CodeBlock with type, content, and parsed data (for JSON)
func Extract(text string) []CodeBlock {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	var blocks []CodeBlock

	// First, try to extract markdown code blocks
	blocks = extractMarkdownBlocks(text)

	// If no markdown blocks found, try to detect direct content
	if len(blocks) == 0 {
		block := detectDirectContent(text)
		if block != nil {
			blocks = append(blocks, *block)
		}
	}

	// Fallback: if still no blocks, return original text as "text" type
	if len(blocks) == 0 {
		blocks = append(blocks, CodeBlock{
			Type:    "text",
			Content: text,
			Start:   0,
			End:     len(text),
		})
	}

	// Parse JSON/YAML blocks
	for i := range blocks {
		if blocks[i].Type == "json" || blocks[i].Type == "yaml" || blocks[i].Type == "yml" {
			parsed, err := json.Parse(blocks[i].Content)
			if err == nil {
				blocks[i].Data = parsed
			}
		}
	}

	return blocks
}

// ExtractFirst extracts and returns only the first code block
// Useful when you expect only one block (common in LLM responses)
func ExtractFirst(text string) *CodeBlock {
	blocks := Extract(text)
	if len(blocks) > 0 {
		return &blocks[0]
	}
	return nil
}

// ExtractJSON extracts the first JSON or YAML block and returns parsed data
// Supports both JSON and YAML as they are both structured data formats
// Returns nil if no valid JSON/YAML found
func ExtractJSON(text string) interface{} {
	blocks := Extract(text)
	for _, block := range blocks {
		if (block.Type == "json" || block.Type == "yaml" || block.Type == "yml") && block.Data != nil {
			return block.Data
		}
	}
	return nil
}

// extractMarkdownBlocks extracts code blocks wrapped in markdown ``` fences
func extractMarkdownBlocks(text string) []CodeBlock {
	var blocks []CodeBlock

	// Pattern: ```lang\n...content...\n```
	// Also handles ``` without language specifier
	pattern := regexp.MustCompile("(?s)```(\\w*)\\s*\\n?(.*?)\\n?```")
	matches := pattern.FindAllStringSubmatchIndex(text, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			lang := strings.ToLower(strings.TrimSpace(text[match[2]:match[3]]))
			content := strings.TrimSpace(text[match[4]:match[5]])

			// If no language specified, try to detect
			if lang == "" {
				lang = detectLanguage(content)
			}

			blocks = append(blocks, CodeBlock{
				Type:    lang,
				Content: content,
				Start:   match[0],
				End:     match[1],
			})
		}
	}

	return blocks
}

// detectDirectContent detects if the entire text is a specific format
// without markdown wrapper
func detectDirectContent(text string) *CodeBlock {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}

	// Detect language
	lang := detectLanguage(trimmed)
	if lang == "" {
		return nil
	}

	return &CodeBlock{
		Type:    lang,
		Content: trimmed,
		Start:   0,
		End:     len(text),
	}
}

// detectLanguage attempts to detect the language/format of content
func detectLanguage(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}

	// Check first few characters for common patterns
	upper := strings.ToUpper(trimmed)

	// JSON detection (starts with { or [)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "json"
	}

	// HTML/XML detection
	if strings.HasPrefix(trimmed, "<") {
		// PHP detection first
		if strings.HasPrefix(trimmed, "<?php") || strings.HasPrefix(trimmed, "<?=") {
			return "php"
		}
		for _, indicator := range languageIndicators["xml"] {
			if strings.HasPrefix(upper, strings.ToUpper(indicator)) {
				return "xml"
			}
		}
		for _, indicator := range languageIndicators["html"] {
			if strings.HasPrefix(upper, strings.ToUpper(indicator)) {
				return "html"
			}
		}
		// Generic XML/HTML-like
		if strings.Contains(trimmed, ">") {
			return "html"
		}
	}

	// YAML detection (starts with ---)
	if strings.HasPrefix(trimmed, "---") {
		return "yaml"
	}

	// Shell script detection (shebang) - check early
	if strings.HasPrefix(trimmed, "#!/") {
		if strings.Contains(trimmed, "python") {
			return "python"
		}
		if strings.Contains(trimmed, "bash") || strings.Contains(trimmed, "/sh") {
			return "shell"
		}
		if strings.Contains(trimmed, "node") {
			return "javascript"
		}
		return "shell" // Default shebang to shell
	}

	// SQL detection (case insensitive)
	for _, indicator := range languageIndicators["sql"] {
		if strings.HasPrefix(upper, indicator) {
			return "sql"
		}
	}

	// C/C++ detection (#include) - check early as it's distinctive
	if strings.HasPrefix(trimmed, "#include") {
		if strings.Contains(trimmed, "<iostream>") || strings.Contains(trimmed, "<vector>") || strings.Contains(trimmed, "std::") {
			return "cpp"
		}
		if strings.Contains(trimmed, "<stdio.h>") || strings.Contains(trimmed, "<stdlib.h>") {
			return "c"
		}
		return "c" // Default #include to C
	}

	// Markdown detection (# heading at start)
	if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
		return "markdown"
	}

	// Get first few lines for multi-line detection
	lines := strings.Split(trimmed, "\n")
	firstLines := lines
	if len(firstLines) > 5 {
		firstLines = lines[:5]
	}
	firstContent := strings.Join(firstLines, "\n")

	// Go detection - very distinctive patterns
	if strings.HasPrefix(trimmed, "package ") || strings.Contains(firstContent, "\npackage ") {
		return "go"
	}
	if strings.HasPrefix(trimmed, "func ") || strings.Contains(firstContent, "\nfunc ") {
		return "go"
	}
	if strings.HasPrefix(trimmed, "import (") || strings.HasPrefix(trimmed, "import \"") {
		return "go"
	}

	// TypeScript detection - check before JavaScript (more specific)
	for _, indicator := range languageIndicators["typescript"] {
		if strings.Contains(firstContent, indicator) {
			return "typescript"
		}
	}

	// Java detection - distinctive patterns
	if strings.Contains(firstContent, "public class ") || strings.Contains(firstContent, "public static void main") {
		return "java"
	}
	if strings.Contains(firstContent, "System.out.") || strings.HasPrefix(trimmed, "import java.") {
		return "java"
	}

	// C# detection
	if strings.HasPrefix(trimmed, "using System") || strings.Contains(firstContent, "Console.Write") {
		return "csharp"
	}
	if strings.Contains(firstContent, "static void Main") {
		return "csharp"
	}

	// Rust detection
	if strings.HasPrefix(trimmed, "fn ") || strings.Contains(firstContent, "pub fn ") {
		return "rust"
	}
	if strings.Contains(firstContent, "println!") || strings.Contains(firstContent, "let mut ") {
		return "rust"
	}

	// Python detection
	if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "class ") {
		return "python"
	}
	if strings.Contains(firstContent, "if __name__") || strings.Contains(firstContent, "print(") {
		return "python"
	}

	// JavaScript detection
	if strings.HasPrefix(trimmed, "function ") || strings.Contains(firstContent, "console.log") {
		return "javascript"
	}
	if strings.Contains(firstContent, "require(") || strings.Contains(firstContent, "module.exports") {
		return "javascript"
	}

	// Ruby detection
	if strings.Contains(firstContent, "puts ") || strings.Contains(firstContent, "attr_accessor") {
		return "ruby"
	}

	// Shell detection
	if strings.Contains(firstContent, "echo $") || strings.HasPrefix(trimmed, "export ") {
		return "shell"
	}

	// CSS detection (needs { } structure)
	for _, indicator := range languageIndicators["css"] {
		if strings.Contains(trimmed, indicator) {
			if strings.Contains(trimmed, "{") && strings.Contains(trimmed, "}") {
				return "css"
			}
		}
	}

	// Check for YAML-style content (key: value without JSON braces)
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		// YAML pattern: key: value (not JSON, not URL, not TypeScript type annotation)
		if strings.Contains(firstLine, ": ") &&
			!strings.HasPrefix(firstLine, "{") &&
			!strings.Contains(firstLine, "\":") &&
			!strings.Contains(firstLine, "://") &&
			!strings.Contains(firstLine, ": string") &&
			!strings.Contains(firstLine, ": number") {
			return "yaml"
		}
	}

	return ""
}

// ExtractByType extracts all blocks of a specific type
func ExtractByType(text string, blockType string) []CodeBlock {
	blocks := Extract(text)
	var filtered []CodeBlock
	blockType = strings.ToLower(blockType)

	for _, block := range blocks {
		if strings.ToLower(block.Type) == blockType {
			filtered = append(filtered, block)
		}
	}

	return filtered
}
