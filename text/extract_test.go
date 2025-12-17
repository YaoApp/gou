package text

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtract_MarkdownJSON(t *testing.T) {
	text := "Here is the result:\n```json\n{\"keywords\": [\"apple\", \"banana\"]}\n```"
	blocks := Extract(text)

	assert.Len(t, blocks, 1)
	assert.Equal(t, "json", blocks[0].Type)
	assert.Equal(t, `{"keywords": ["apple", "banana"]}`, blocks[0].Content)
	assert.NotNil(t, blocks[0].Data)

	data, ok := blocks[0].Data.(map[string]interface{})
	assert.True(t, ok)
	keywords, ok := data["keywords"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, keywords, 2)
}

func TestExtract_MarkdownMultipleBlocks(t *testing.T) {
	text := `
Here is JSON:
` + "```json\n{\"key\": \"value\"}\n```" + `

And here is SQL:
` + "```sql\nSELECT * FROM users\n```"

	blocks := Extract(text)

	assert.Len(t, blocks, 2)
	assert.Equal(t, "json", blocks[0].Type)
	assert.Equal(t, "sql", blocks[1].Type)
	assert.Equal(t, "SELECT * FROM users", blocks[1].Content)
}

func TestExtract_DirectJSON(t *testing.T) {
	text := `{"need_search": false, "confidence": 0.99}`
	blocks := Extract(text)

	assert.Len(t, blocks, 1)
	assert.Equal(t, "json", blocks[0].Type)
	assert.NotNil(t, blocks[0].Data)

	data, ok := blocks[0].Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, false, data["need_search"])
}

func TestExtract_DirectJSONArray(t *testing.T) {
	text := `["apple", "banana", "cherry"]`
	blocks := Extract(text)

	assert.Len(t, blocks, 1)
	assert.Equal(t, "json", blocks[0].Type)
	assert.NotNil(t, blocks[0].Data)
}

func TestExtract_DirectHTML(t *testing.T) {
	text := `<div class="container"><h1>Hello</h1></div>`
	blocks := Extract(text)

	assert.Len(t, blocks, 1)
	assert.Equal(t, "html", blocks[0].Type)
}

func TestExtract_DirectSQL(t *testing.T) {
	text := `SELECT id, name FROM users WHERE active = 1`
	blocks := Extract(text)

	assert.Len(t, blocks, 1)
	assert.Equal(t, "sql", blocks[0].Type)
}

func TestExtract_MarkdownNoLanguage(t *testing.T) {
	text := "```\n{\"auto\": \"detect\"}\n```"
	blocks := Extract(text)

	assert.Len(t, blocks, 1)
	assert.Equal(t, "json", blocks[0].Type)
	assert.NotNil(t, blocks[0].Data)
}

func TestExtract_BrokenJSON(t *testing.T) {
	// JSON with trailing comma (common LLM error)
	text := "```json\n{\"key\": \"value\",}\n```"
	blocks := Extract(text)

	assert.Len(t, blocks, 1)
	assert.Equal(t, "json", blocks[0].Type)
	// Should be repaired and parsed
	assert.NotNil(t, blocks[0].Data)
}

func TestExtractFirst(t *testing.T) {
	text := "```json\n{\"first\": true}\n```\n```json\n{\"second\": true}\n```"
	block := ExtractFirst(text)

	assert.NotNil(t, block)
	assert.Equal(t, "json", block.Type)

	data, ok := block.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, data["first"])
}

func TestExtractJSON(t *testing.T) {
	text := "The result is:\n```json\n{\"keywords\": [\"test\"]}\n```"
	data := ExtractJSON(text)

	assert.NotNil(t, data)
	m, ok := data.(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, m, "keywords")
}

func TestExtractJSON_NoJSON(t *testing.T) {
	text := "Just plain text without any code blocks"
	data := ExtractJSON(text)

	assert.Nil(t, data)
}

func TestExtractJSON_YAML(t *testing.T) {
	text := "```yaml\nkeywords:\n  - apple\n  - banana\n```"
	data := ExtractJSON(text)

	assert.NotNil(t, data)
	m, ok := data.(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, m, "keywords")
}

func TestExtractJSON_DirectYAML(t *testing.T) {
	text := "name: test\nvalue: 123"
	data := ExtractJSON(text)

	assert.NotNil(t, data)
	m, ok := data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "test", m["name"])
}

func TestExtractByType(t *testing.T) {
	text := `
` + "```json\n{\"a\": 1}\n```" + `
` + "```sql\nSELECT 1\n```" + `
` + "```json\n{\"b\": 2}\n```"

	jsonBlocks := ExtractByType(text, "json")
	assert.Len(t, jsonBlocks, 2)

	sqlBlocks := ExtractByType(text, "sql")
	assert.Len(t, sqlBlocks, 1)
}

func TestExtract_EmptyText(t *testing.T) {
	blocks := Extract("")
	assert.Nil(t, blocks)

	blocks = Extract("   ")
	assert.Nil(t, blocks)
}

func TestExtract_PlainText_Fallback(t *testing.T) {
	text := "This is just plain text without any code blocks"
	blocks := Extract(text)

	assert.Len(t, blocks, 1)
	assert.Equal(t, "text", blocks[0].Type)
	assert.Equal(t, text, blocks[0].Content)
	assert.Nil(t, blocks[0].Data)
}

func TestExtractFirst_PlainText_Fallback(t *testing.T) {
	text := "Hello, how are you?"
	block := ExtractFirst(text)

	assert.NotNil(t, block)
	assert.Equal(t, "text", block.Type)
	assert.Equal(t, text, block.Content)
}

func TestExtract_YAML(t *testing.T) {
	text := "```yaml\nname: test\nvalue: 123\n```"
	blocks := Extract(text)

	assert.Len(t, blocks, 1)
	assert.Equal(t, "yaml", blocks[0].Type)
	assert.NotNil(t, blocks[0].Data)
}

func TestExtract_DirectYAML(t *testing.T) {
	text := "---\nname: test\nvalue: 123"
	blocks := Extract(text)

	assert.Len(t, blocks, 1)
	assert.Equal(t, "yaml", blocks[0].Type)
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Data formats
		{`{"key": "value"}`, "json"},
		{`[1, 2, 3]`, "json"},
		{`<html><body></body></html>`, "html"},
		{`<!DOCTYPE html>`, "html"},
		{`<div>content</div>`, "html"},
		{`<?xml version="1.0"?>`, "xml"},
		{`SELECT * FROM users`, "sql"},
		{`INSERT INTO table VALUES (1)`, "sql"},
		{`---\nkey: value`, "yaml"},
		{`name: value`, "yaml"},

		// Programming languages
		{`package main`, "go"},
		{`func main() {}`, "go"},
		{`import "fmt"`, "go"},
		{`def hello():`, "python"},
		{`print("Hello")`, "python"},
		{`#!/usr/bin/env python`, "python"},
		{`function hello() {}`, "javascript"},
		{`console.log("hi")`, "javascript"},
		{`require("fs")`, "javascript"},
		{`interface User { name: string; }`, "typescript"},
		{`const x: number = 1`, "typescript"},
		{`fn main() {}`, "rust"},
		{`println!("Hello")`, "rust"},
		{`let mut x = 1`, "rust"},
		{`public class Main {}`, "java"},
		{`System.out.println("hi")`, "java"},
		{`using System;`, "csharp"},
		{`Console.WriteLine("hi")`, "csharp"},
		{`#include <stdio.h>`, "c"},
		{`#include <iostream>`, "cpp"},
		{`#include <vector>`, "cpp"},
		{`#!/bin/bash`, "shell"},
		{`echo $HOME`, "shell"},
		{`<?php echo "Hello"; ?>`, "php"},
		{`.container { margin: 0; }`, "css"},
		{`# Heading`, "markdown"},
		{`## Sub Heading`, "markdown"},

		// Plain text (no detection)
		{`plain text without patterns`, ""},
		{`Hello, how are you?`, ""},
	}

	for _, tt := range tests {
		name := tt.input
		if len(name) > 30 {
			name = name[:30]
		}
		name = strings.ReplaceAll(name, "\n", "\\n")
		t.Run(name, func(t *testing.T) {
			result := detectLanguage(tt.input)
			assert.Equal(t, tt.expected, result, "input: %s", tt.input)
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
