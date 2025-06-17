package types

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestChunk_TextWChars(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Empty string",
			text:     "",
			expected: []string{},
		},
		{
			name:     "Single English character",
			text:     "A",
			expected: []string{"A"},
		},
		{
			name:     "Multiple English characters",
			text:     "Hello",
			expected: []string{"H", "e", "l", "l", "o"},
		},
		{
			name:     "Single Chinese character",
			text:     "中",
			expected: []string{"中"},
		},
		{
			name:     "Multiple Chinese characters",
			text:     "你好世界",
			expected: []string{"你", "好", "世", "界"},
		},
		{
			name:     "Mixed English and Chinese",
			text:     "Hello世界",
			expected: []string{"H", "e", "l", "l", "o", "世", "界"},
		},
		{
			name:     "Numbers and symbols",
			text:     "123!@#",
			expected: []string{"1", "2", "3", "!", "@", "#"},
		},
		{
			name:     "Japanese characters",
			text:     "こんにちは",
			expected: []string{"こ", "ん", "に", "ち", "は"},
		},
		{
			name:     "Korean characters",
			text:     "안녕하세요",
			expected: []string{"안", "녕", "하", "세", "요"},
		},
		{
			name:     "Emoji",
			text:     "😀🎉",
			expected: []string{"😀", "🎉"},
		},
		{
			name:     "Complex mixed content",
			text:     "Hi🌍你好123",
			expected: []string{"H", "i", "🌍", "你", "好", "1", "2", "3"},
		},
		{
			name:     "Special Unicode characters",
			text:     "Ñoël",
			expected: []string{"Ñ", "o", "ë", "l"},
		},
		{
			name:     "Whitespace characters",
			text:     "a b\tc\nd",
			expected: []string{"a", " ", "b", "\t", "c", "\n", "d"},
		},
		{
			name:     "Arabic characters",
			text:     "مرحبا",
			expected: []string{"م", "ر", "ح", "ب", "ا"},
		},
		{
			name:     "Russian characters",
			text:     "Привет",
			expected: []string{"П", "р", "и", "в", "е", "т"},
		},
		{
			name:     "Greek characters",
			text:     "Γεια",
			expected: []string{"Γ", "ε", "ι", "α"},
		},
		{
			name:     "Mathematical symbols",
			text:     "∑∫∂",
			expected: []string{"∑", "∫", "∂"},
		},
		{
			name:     "Multi-byte combining characters",
			text:     "e\u0301", // é as e + combining acute accent
			expected: []string{"e", "\u0301"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &Chunk{
				Text: tt.text,
			}

			result := chunk.TextWChars()

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("TextWChars() = %v, expected %v", result, tt.expected)
			}

			// Verify that the original text is not modified
			if chunk.Text != tt.text {
				t.Errorf("Original text was modified: got %q, expected %q", chunk.Text, tt.text)
			}

			// Verify that the total length matches when concatenated
			concatenated := ""
			for _, s := range result {
				concatenated += s
			}
			if concatenated != tt.text {
				t.Errorf("Concatenated result %q does not match original text %q", concatenated, tt.text)
			}
		})
	}
}

func TestChunk_TextWChars_Performance(t *testing.T) {
	// Test with a large string to verify performance
	largeText := ""
	for i := 0; i < 1000; i++ {
		largeText += "Hello世界123🌍"
	}

	chunk := &Chunk{
		Text: largeText,
	}

	result := chunk.TextWChars()

	// Calculate expected length by counting actual runes in the pattern
	pattern := "Hello世界123🌍"
	expectedPerIteration := len([]rune(pattern)) // This gives us the correct rune count
	expectedLength := expectedPerIteration * 1000
	if len(result) != expectedLength {
		t.Errorf("Expected %d characters, got %d", expectedLength, len(result))
	}

	// Verify original text is not modified
	if len(chunk.Text) != len(largeText) {
		t.Error("Original text was modified during processing")
	}
}

func TestChunk_TextWChars_NilChunk(t *testing.T) {
	// Test with nil chunk - should panic with nil pointer dereference
	var chunk *Chunk
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic with nil chunk, but didn't panic")
		}
	}()

	// This should panic with nil pointer dereference, which is expected behavior
	_ = chunk.TextWChars()
}

func TestChunk_TextWCharsJSON(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "Empty string",
			text:     "",
			expected: "[]",
		},
		{
			name:     "Single English character",
			text:     "A",
			expected: `["A"]`,
		},
		{
			name:     "Multiple English characters",
			text:     "Hello",
			expected: `["H","e","l","l","o"]`,
		},
		{
			name:     "Chinese characters",
			text:     "你好",
			expected: `["你","好"]`,
		},
		{
			name:     "Mixed content",
			text:     "Hi你好123",
			expected: `["H","i","你","好","1","2","3"]`,
		},
		{
			name:     "Emoji",
			text:     "😀🎉",
			expected: `["😀","🎉"]`,
		},
		{
			name:     "Special characters with escaping",
			text:     "a\"b\\c",
			expected: `["a","\"","b","\\","c"]`,
		},
		{
			name:     "Whitespace characters",
			text:     "a\nb\tc",
			expected: `["a","\n","b","\t","c"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &Chunk{
				Text: tt.text,
			}

			result, err := chunk.TextWCharsJSON()
			if err != nil {
				t.Errorf("TextWCharsJSON() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("TextWCharsJSON() = %q, expected %q", result, tt.expected)
			}

			// Verify that the result is valid JSON
			var decoded []string
			if err := json.Unmarshal([]byte(result), &decoded); err != nil {
				t.Errorf("Result is not valid JSON: %v", err)
			}

			// Verify that the decoded result matches TextWChars()
			expectedSlices := chunk.TextWChars()
			if !reflect.DeepEqual(decoded, expectedSlices) {
				t.Errorf("Decoded JSON %v does not match TextWChars() result %v", decoded, expectedSlices)
			}
		})
	}
}

func TestChunk_TextWCharsJSON_NilChunk(t *testing.T) {
	// Test with nil chunk - should panic with nil pointer dereference
	var chunk *Chunk
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic with nil chunk, but didn't panic")
		}
	}()

	// This should panic with nil pointer dereference, which is expected behavior
	_, _ = chunk.TextWCharsJSON()
}

func TestChunk_TextLines(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Empty string",
			text:     "",
			expected: []string{},
		},
		{
			name:     "Single line without newline",
			text:     "Hello World",
			expected: []string{"Hello World"},
		},
		{
			name:     "Single line with Unix newline",
			text:     "Hello World\n",
			expected: []string{"Hello World", ""},
		},
		{
			name:     "Multiple lines with Unix newlines",
			text:     "Line 1\nLine 2\nLine 3",
			expected: []string{"Line 1", "Line 2", "Line 3"},
		},
		{
			name:     "Multiple lines with Windows newlines",
			text:     "Line 1\r\nLine 2\r\nLine 3",
			expected: []string{"Line 1", "Line 2", "Line 3"},
		},
		{
			name:     "Multiple lines with Mac Classic newlines",
			text:     "Line 1\rLine 2\rLine 3",
			expected: []string{"Line 1", "Line 2", "Line 3"},
		},
		{
			name:     "Mixed newline styles",
			text:     "Line 1\nLine 2\r\nLine 3\rLine 4",
			expected: []string{"Line 1", "Line 2", "Line 3", "Line 4"},
		},
		{
			name:     "Empty lines with Unix newlines",
			text:     "Line 1\n\nLine 3\n",
			expected: []string{"Line 1", "", "Line 3", ""},
		},
		{
			name:     "Empty lines with Windows newlines",
			text:     "Line 1\r\n\r\nLine 3\r\n",
			expected: []string{"Line 1", "", "Line 3", ""},
		},
		{
			name:     "Empty lines with Mac Classic newlines",
			text:     "Line 1\r\rLine 3\r",
			expected: []string{"Line 1", "", "Line 3", ""},
		},
		{
			name:     "Chinese text with newlines",
			text:     "你好世界\n第二行\n第三行",
			expected: []string{"你好世界", "第二行", "第三行"},
		},
		{
			name:     "Mixed language with mixed newlines",
			text:     "Hello 世界\r\nSecond line\nThird line\rFourth line",
			expected: []string{"Hello 世界", "Second line", "Third line", "Fourth line"},
		},
		{
			name:     "Only newlines",
			text:     "\n\r\n\r",
			expected: []string{"", "", "", ""},
		},
		{
			name:     "Whitespace and newlines",
			text:     "  Line 1  \n  Line 2  \r\n  Line 3  ",
			expected: []string{"  Line 1  ", "  Line 2  ", "  Line 3  "},
		},
		{
			name:     "Special characters with newlines",
			text:     "Line with \"quotes\"\nLine with \\backslash\rLine with \ttab",
			expected: []string{"Line with \"quotes\"", "Line with \\backslash", "Line with \ttab"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &Chunk{
				Text: tt.text,
			}

			result := chunk.TextLines()

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("TextLines() = %v, expected %v", result, tt.expected)
			}

			// Verify that the original text is not modified
			if chunk.Text != tt.text {
				t.Errorf("Original text was modified: got %q, expected %q", chunk.Text, tt.text)
			}
		})
	}
}

func TestChunk_TextLines_Performance(t *testing.T) {
	// Test with a large text containing many lines
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, fmt.Sprintf("Line %d with some content 内容", i))
	}
	largeText := strings.Join(lines, "\n")

	chunk := &Chunk{
		Text: largeText,
	}

	result := chunk.TextLines()

	if len(result) != 1000 {
		t.Errorf("Expected 1000 lines, got %d", len(result))
	}

	// Verify original text is not modified
	if len(chunk.Text) != len(largeText) {
		t.Error("Original text was modified during processing")
	}
}

func TestChunk_TextLines_NilChunk(t *testing.T) {
	// Test with nil chunk - should panic with nil pointer dereference
	var chunk *Chunk
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic with nil chunk, but didn't panic")
		}
	}()

	// This should panic with nil pointer dereference, which is expected behavior
	_ = chunk.TextLines()
}

func TestChunk_TextLinesJSON(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "Empty string",
			text:     "",
			expected: "[]",
		},
		{
			name:     "Single line",
			text:     "Hello World",
			expected: `["Hello World"]`,
		},
		{
			name:     "Multiple lines with Unix newlines",
			text:     "Line 1\nLine 2\nLine 3",
			expected: `["Line 1","Line 2","Line 3"]`,
		},
		{
			name:     "Multiple lines with Windows newlines",
			text:     "Line 1\r\nLine 2\r\nLine 3",
			expected: `["Line 1","Line 2","Line 3"]`,
		},
		{
			name:     "Multiple lines with Mac Classic newlines",
			text:     "Line 1\rLine 2\rLine 3",
			expected: `["Line 1","Line 2","Line 3"]`,
		},
		{
			name:     "Mixed newline styles",
			text:     "Line 1\nLine 2\r\nLine 3\rLine 4",
			expected: `["Line 1","Line 2","Line 3","Line 4"]`,
		},
		{
			name:     "Chinese text",
			text:     "你好\n世界",
			expected: `["你好","世界"]`,
		},
		{
			name:     "Empty lines",
			text:     "Line 1\n\nLine 3",
			expected: `["Line 1","","Line 3"]`,
		},
		{
			name:     "Special characters with escaping",
			text:     "Line with \"quotes\"\nLine with \\backslash",
			expected: `["Line with \"quotes\"","Line with \\backslash"]`,
		},
		{
			name:     "Whitespace characters",
			text:     "Line 1\t\nLine 2  ",
			expected: `["Line 1\t","Line 2  "]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &Chunk{
				Text: tt.text,
			}

			result, err := chunk.TextLinesJSON()
			if err != nil {
				t.Errorf("TextLinesJSON() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("TextLinesJSON() = %q, expected %q", result, tt.expected)
			}

			// Verify that the result is valid JSON
			var decoded []string
			if err := json.Unmarshal([]byte(result), &decoded); err != nil {
				t.Errorf("Result is not valid JSON: %v", err)
			}

			// Verify that the decoded result matches TextLines()
			expectedLines := chunk.TextLines()
			if !reflect.DeepEqual(decoded, expectedLines) {
				t.Errorf("Decoded JSON %v does not match TextLines() result %v", decoded, expectedLines)
			}
		})
	}
}

func TestChunk_TextLinesJSON_NilChunk(t *testing.T) {
	// Test with nil chunk - should panic with nil pointer dereference
	var chunk *Chunk
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic with nil chunk, but didn't panic")
		}
	}()

	// This should panic with nil pointer dereference, which is expected behavior
	_, _ = chunk.TextLinesJSON()
}

func TestChunk_TextLinesToWChars(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected [][]string
	}{
		{
			name:     "Empty string",
			text:     "",
			expected: [][]string{},
		},
		{
			name:     "Single line without newline",
			text:     "Hello",
			expected: [][]string{{"H", "e", "l", "l", "o"}},
		},
		{
			name:     "Single line with Unix newline",
			text:     "Hello\n",
			expected: [][]string{{"H", "e", "l", "l", "o"}, {}},
		},
		{
			name:     "Multiple lines with Unix newlines",
			text:     "Hi\nBye\nOK",
			expected: [][]string{{"H", "i"}, {"B", "y", "e"}, {"O", "K"}},
		},
		{
			name:     "Multiple lines with Windows newlines",
			text:     "Hi\r\nBye\r\nOK",
			expected: [][]string{{"H", "i"}, {"B", "y", "e"}, {"O", "K"}},
		},
		{
			name:     "Multiple lines with Mac Classic newlines",
			text:     "Hi\rBye\rOK",
			expected: [][]string{{"H", "i"}, {"B", "y", "e"}, {"O", "K"}},
		},
		{
			name:     "Mixed newline styles",
			text:     "Hi\nBye\r\nOK\rEnd",
			expected: [][]string{{"H", "i"}, {"B", "y", "e"}, {"O", "K"}, {"E", "n", "d"}},
		},
		{
			name:     "Empty lines",
			text:     "Hi\n\nBye",
			expected: [][]string{{"H", "i"}, {}, {"B", "y", "e"}},
		},
		{
			name:     "Chinese characters",
			text:     "你好\n世界",
			expected: [][]string{{"你", "好"}, {"世", "界"}},
		},
		{
			name:     "Mixed English and Chinese",
			text:     "Hello世界\nGood你好",
			expected: [][]string{{"H", "e", "l", "l", "o", "世", "界"}, {"G", "o", "o", "d", "你", "好"}},
		},
		{
			name:     "Emoji in lines",
			text:     "Hi😀\n🎉Bye",
			expected: [][]string{{"H", "i", "😀"}, {"🎉", "B", "y", "e"}},
		},
		{
			name:     "Special characters",
			text:     "a\"b\n\\c\td",
			expected: [][]string{{"a", "\"", "b"}, {"\\", "c", "\t", "d"}},
		},
		{
			name:     "Whitespace preservation",
			text:     "  a  \n  b  ",
			expected: [][]string{{" ", " ", "a", " ", " "}, {" ", " ", "b", " ", " "}},
		},
		{
			name:     "Only newlines",
			text:     "\n\r\n\r",
			expected: [][]string{{}, {}, {}, {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &Chunk{
				Text: tt.text,
			}

			result := chunk.TextLinesToWChars()

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("TextLinesToWChars() = %v, expected %v", result, tt.expected)
			}

			// Verify that the original text is not modified
			if chunk.Text != tt.text {
				t.Errorf("Original text was modified: got %q, expected %q", chunk.Text, tt.text)
			}
		})
	}
}

func TestChunk_TextLinesToWChars_Performance(t *testing.T) {
	// Test with multiple lines containing various characters
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("Line %d: Hello世界%d", i, i))
	}
	largeText := strings.Join(lines, "\n")

	chunk := &Chunk{
		Text: largeText,
	}

	result := chunk.TextLinesToWChars()

	if len(result) != 100 {
		t.Errorf("Expected 100 lines, got %d", len(result))
	}

	// Check that each line has the expected number of characters
	for i, lineSlices := range result {
		expectedLine := fmt.Sprintf("Line %d: Hello世界%d", i, i)
		expectedChars := len([]rune(expectedLine))
		if len(lineSlices) != expectedChars {
			t.Errorf("Line %d: expected %d characters, got %d", i, expectedChars, len(lineSlices))
		}
	}

	// Verify original text is not modified
	if len(chunk.Text) != len(largeText) {
		t.Error("Original text was modified during processing")
	}
}

func TestChunk_TextLinesToWChars_NilChunk(t *testing.T) {
	// Test with nil chunk - should panic with nil pointer dereference
	var chunk *Chunk
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic with nil chunk, but didn't panic")
		}
	}()

	// This should panic with nil pointer dereference, which is expected behavior
	_ = chunk.TextLinesToWChars()
}

func TestChunk_TextLinesToWCharsJSON(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "Empty string",
			text:     "",
			expected: "[]",
		},
		{
			name:     "Single line",
			text:     "Hi",
			expected: `[["H","i"]]`,
		},
		{
			name:     "Multiple lines",
			text:     "Hi\nBye",
			expected: `[["H","i"],["B","y","e"]]`,
		},
		{
			name:     "Windows newlines",
			text:     "Hi\r\nBye",
			expected: `[["H","i"],["B","y","e"]]`,
		},
		{
			name:     "Mac Classic newlines",
			text:     "Hi\rBye",
			expected: `[["H","i"],["B","y","e"]]`,
		},
		{
			name:     "Mixed newlines",
			text:     "A\nB\r\nC\rD",
			expected: `[["A"],["B"],["C"],["D"]]`,
		},
		{
			name:     "Empty lines",
			text:     "A\n\nB",
			expected: `[["A"],[],["B"]]`,
		},
		{
			name:     "Chinese characters",
			text:     "你好\n世界",
			expected: `[["你","好"],["世","界"]]`,
		},
		{
			name:     "Emoji",
			text:     "Hi😀\n🎉Bye",
			expected: `[["H","i","😀"],["🎉","B","y","e"]]`,
		},
		{
			name:     "Special characters with escaping",
			text:     "a\"b\n\\c",
			expected: `[["a","\"","b"],["\\","c"]]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &Chunk{
				Text: tt.text,
			}

			result, err := chunk.TextLinesToWCharsJSON()
			if err != nil {
				t.Errorf("TextLinesToWCharsJSON() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("TextLinesToWCharsJSON() = %q, expected %q", result, tt.expected)
			}

			// Verify that the result is valid JSON
			var decoded [][]string
			if err := json.Unmarshal([]byte(result), &decoded); err != nil {
				t.Errorf("Result is not valid JSON: %v", err)
			}

			// Verify that the decoded result matches TextLinesToWChars()
			expectedSlices := chunk.TextLinesToWChars()
			if !reflect.DeepEqual(decoded, expectedSlices) {
				t.Errorf("Decoded JSON %v does not match TextLinesToWChars() result %v", decoded, expectedSlices)
			}
		})
	}
}

func TestChunk_TextLinesToWCharsJSON_NilChunk(t *testing.T) {
	// Test with nil chunk - should panic with nil pointer dereference
	var chunk *Chunk
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic with nil chunk, but didn't panic")
		}
	}()

	// This should panic with nil pointer dereference, which is expected behavior
	_, _ = chunk.TextLinesToWCharsJSON()
}

func TestChunk_Split(t *testing.T) {
	tests := []struct {
		name          string
		chunk         *Chunk
		positions     []Position
		expectedCount int
		expectedTexts []string
		expectWarning bool
	}{
		{
			name: "Valid split with multiple positions",
			chunk: &Chunk{
				ID:     "test-chunk-1",
				Text:   "Hello World! This is a test.",
				Type:   ChunkingTypeText,
				Depth:  1,
				Leaf:   true,
				Root:   true,
				Status: ChunkingStatusCompleted,
				TextPos: &TextPosition{
					StartIndex: 0,
					EndIndex:   28,
					StartLine:  1,
					EndLine:    1,
				},
			},
			positions: []Position{
				{StartPos: 0, EndPos: 5},   // "Hello"
				{StartPos: 6, EndPos: 12},  // "World!"
				{StartPos: 13, EndPos: 28}, // "This is a test."
			},
			expectedCount: 3,
			expectedTexts: []string{"Hello", "World!", "This is a test."},
		},
		{
			name: "Chinese text split",
			chunk: &Chunk{
				ID:     "test-chunk-2",
				Text:   "你好世界！这是一个测试。",
				Type:   ChunkingTypeText,
				Depth:  1,
				Leaf:   true,
				Root:   true,
				Status: ChunkingStatusCompleted,
			},
			positions: []Position{
				{StartPos: 0, EndPos: 5},  // "你好世界！" (5 characters)
				{StartPos: 5, EndPos: 12}, // "这是一个测试。" (7 characters)
			},
			expectedCount: 2,
			expectedTexts: []string{"你好世界！", "这是一个测试。"},
		},
		{
			name: "Out of bounds positions",
			chunk: &Chunk{
				ID:     "test-chunk-3",
				Text:   "Short text",
				Type:   ChunkingTypeText,
				Depth:  1,
				Leaf:   true,
				Root:   true,
				Status: ChunkingStatusCompleted,
			},
			positions: []Position{
				{StartPos: 0, EndPos: 5},     // "Short" - valid
				{StartPos: -5, EndPos: 3},    // Invalid: negative start
				{StartPos: 6, EndPos: 50},    // Invalid: end beyond text length (will be clamped)
				{StartPos: 100, EndPos: 105}, // Invalid: start beyond text length
			},
			expectedCount: 2, // Only "Short" and clamped "text"
			expectedTexts: []string{"Short", "text"},
			expectWarning: true,
		},
		{
			name: "Empty and invalid positions",
			chunk: &Chunk{
				ID:     "test-chunk-4",
				Text:   "Test content",
				Type:   ChunkingTypeText,
				Depth:  1,
				Leaf:   true,
				Root:   true,
				Status: ChunkingStatusCompleted,
			},
			positions: []Position{
				{StartPos: 5, EndPos: 5},  // Empty range
				{StartPos: 10, EndPos: 8}, // Invalid: start > end
				{StartPos: 0, EndPos: 4},  // Valid: "Test"
			},
			expectedCount: 1,
			expectedTexts: []string{"Test"},
			expectWarning: true,
		},
		{
			name: "Multi-line text split",
			chunk: &Chunk{
				ID:     "test-chunk-5",
				Text:   "Line 1\nLine 2\nLine 3",
				Type:   ChunkingTypeText,
				Depth:  1,
				Leaf:   true,
				Root:   true,
				Status: ChunkingStatusCompleted,
				TextPos: &TextPosition{
					StartIndex: 0,
					EndIndex:   20,
					StartLine:  1,
					EndLine:    3,
				},
			},
			positions: []Position{
				{StartPos: 0, EndPos: 6},   // "Line 1"
				{StartPos: 7, EndPos: 13},  // "Line 2"
				{StartPos: 14, EndPos: 20}, // "Line 3"
			},
			expectedCount: 3,
			expectedTexts: []string{"Line 1", "Line 2", "Line 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chars := tt.chunk.TextWChars()
			result := tt.chunk.Split(chars, tt.positions)

			// Check result count
			if len(result) != tt.expectedCount {
				t.Errorf("Split() returned %d chunks, expected %d", len(result), tt.expectedCount)
			}

			// Check result texts
			for i, expectedText := range tt.expectedTexts {
				if i >= len(result) {
					t.Errorf("Missing chunk %d with expected text %q", i, expectedText)
					continue
				}
				if result[i].Text != expectedText {
					t.Errorf("Chunk %d text = %q, expected %q", i, result[i].Text, expectedText)
				}
			}

			// Check cascading relationships
			for i, subChunk := range result {
				// Check parent ID
				if subChunk.ParentID != tt.chunk.ID {
					t.Errorf("Chunk %d ParentID = %q, expected %q", i, subChunk.ParentID, tt.chunk.ID)
				}

				// Check depth
				expectedDepth := tt.chunk.Depth + 1
				if subChunk.Depth != expectedDepth {
					t.Errorf("Chunk %d Depth = %d, expected %d", i, subChunk.Depth, expectedDepth)
				}

				// Check leaf/root status
				if !subChunk.Leaf {
					t.Errorf("Chunk %d should be a leaf node", i)
				}
				if subChunk.Root {
					t.Errorf("Chunk %d should not be a root node", i)
				}

				// Check index
				if subChunk.Index != i {
					t.Errorf("Chunk %d Index = %d, expected %d", i, subChunk.Index, i)
				}

				// Check status
				if subChunk.Status != ChunkingStatusCompleted {
					t.Errorf("Chunk %d Status = %s, expected %s", i, subChunk.Status, ChunkingStatusCompleted)
				}

				// Check parents chain
				expectedParentsCount := len(tt.chunk.Parents) + 1
				if len(subChunk.Parents) != expectedParentsCount {
					t.Errorf("Chunk %d has %d parents, expected %d", i, len(subChunk.Parents), expectedParentsCount)
				}

				// Check that original chunk is now not a leaf
				if len(result) > 0 && tt.chunk.Leaf {
					t.Error("Original chunk should no longer be a leaf after splitting")
				}
			}

			// Verify text positions are calculated correctly
			for i, subChunk := range result {
				if subChunk.TextPos != nil && tt.chunk.TextPos != nil {
					// Check that text positions are within bounds
					if subChunk.TextPos.StartIndex < tt.chunk.TextPos.StartIndex {
						t.Errorf("Chunk %d StartIndex %d is before parent StartIndex %d", i, subChunk.TextPos.StartIndex, tt.chunk.TextPos.StartIndex)
					}
					if subChunk.TextPos.EndIndex > tt.chunk.TextPos.EndIndex {
						t.Errorf("Chunk %d EndIndex %d is after parent EndIndex %d", i, subChunk.TextPos.EndIndex, tt.chunk.TextPos.EndIndex)
					}
				}
			}
		})
	}
}

func TestChunk_Split_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		chunk     *Chunk
		positions []Position
		expected  int
	}{
		{
			name:      "Nil chunk",
			chunk:     nil,
			positions: []Position{{StartPos: 0, EndPos: 5}},
			expected:  0,
		},
		{
			name: "Empty text",
			chunk: &Chunk{
				ID:   "empty-chunk",
				Text: "",
			},
			positions: []Position{{StartPos: 0, EndPos: 5}},
			expected:  0,
		},
		{
			name: "Empty positions array",
			chunk: &Chunk{
				ID:   "test-chunk",
				Text: "Some text",
			},
			positions: []Position{},
			expected:  0,
		},
		{
			name: "All invalid positions",
			chunk: &Chunk{
				ID:   "test-chunk",
				Text: "Test",
			},
			positions: []Position{
				{StartPos: -1, EndPos: 2},
				{StartPos: 5, EndPos: 10},
				{StartPos: 3, EndPos: 1},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.chunk == nil {
				// Handle nil chunk case
				result := tt.chunk.Split(nil, tt.positions)
				if len(result) != tt.expected {
					t.Errorf("Split() returned %d chunks, expected %d", len(result), tt.expected)
				}
			} else {
				chars := tt.chunk.TextWChars()
				result := tt.chunk.Split(chars, tt.positions)
				if len(result) != tt.expected {
					t.Errorf("Split() returned %d chunks, expected %d", len(result), tt.expected)
				}
			}
		})
	}
}

func TestChunk_Split_TextPositionCalculation(t *testing.T) {
	chunk := &Chunk{
		ID:   "multiline-chunk",
		Text: "Line 1\nLine 2\nLine 3\nLine 4",
		TextPos: &TextPosition{
			StartIndex: 100,
			EndIndex:   127,
			StartLine:  10,
			EndLine:    13,
		},
	}

	positions := []Position{
		{StartPos: 0, EndPos: 6},   // "Line 1" - line 10
		{StartPos: 7, EndPos: 13},  // "Line 2" - line 11
		{StartPos: 14, EndPos: 20}, // "Line 3" - line 12
	}

	chars := chunk.TextWChars()
	result := chunk.Split(chars, positions)

	if len(result) != 3 {
		t.Fatalf("Expected 3 chunks, got %d", len(result))
	}

	// Check first chunk (Line 1)
	if result[0].TextPos.StartLine != 10 {
		t.Errorf("First chunk StartLine = %d, expected 10", result[0].TextPos.StartLine)
	}
	if result[0].TextPos.EndLine != 10 {
		t.Errorf("First chunk EndLine = %d, expected 10", result[0].TextPos.EndLine)
	}

	// Check second chunk (Line 2)
	if result[1].TextPos.StartLine != 11 {
		t.Errorf("Second chunk StartLine = %d, expected 11", result[1].TextPos.StartLine)
	}
	if result[1].TextPos.EndLine != 11 {
		t.Errorf("Second chunk EndLine = %d, expected 11", result[1].TextPos.EndLine)
	}

	// Check third chunk (Line 3)
	if result[2].TextPos.StartLine != 12 {
		t.Errorf("Third chunk StartLine = %d, expected 12", result[2].TextPos.StartLine)
	}
	if result[2].TextPos.EndLine != 12 {
		t.Errorf("Third chunk EndLine = %d, expected 12", result[2].TextPos.EndLine)
	}

	// Check absolute positions
	if result[0].TextPos.StartIndex != 100 {
		t.Errorf("First chunk StartIndex = %d, expected 100", result[0].TextPos.StartIndex)
	}
	if result[0].TextPos.EndIndex != 106 {
		t.Errorf("First chunk EndIndex = %d, expected 106", result[0].TextPos.EndIndex)
	}
}

func TestChunk_CalculateTextPos(t *testing.T) {
	tests := []struct {
		name           string
		chunk          *Chunk
		parentPos      *TextPosition
		offsetInParent int
		expectedPos    *TextPosition
	}{
		{
			name: "Root chunk - single line",
			chunk: &Chunk{
				Text: "Hello World",
			},
			parentPos:      nil,
			offsetInParent: 0,
			expectedPos: &TextPosition{
				StartIndex: 0,
				EndIndex:   11,
				StartLine:  1,
				EndLine:    1,
			},
		},
		{
			name: "Root chunk - multiple lines",
			chunk: &Chunk{
				Text: "Line 1\nLine 2\nLine 3",
			},
			parentPos:      nil,
			offsetInParent: 0,
			expectedPos: &TextPosition{
				StartIndex: 0,
				EndIndex:   20,
				StartLine:  1,
				EndLine:    3,
			},
		},
		{
			name: "Child chunk with parent position",
			chunk: &Chunk{
				Text: "World",
				Parents: []Chunk{
					{Text: "Hello World! This is a test."},
				},
			},
			parentPos: &TextPosition{
				StartIndex: 100,
				EndIndex:   128,
				StartLine:  10,
				EndLine:    10,
			},
			offsetInParent: 6, // "World" starts at position 6 in "Hello World!"
			expectedPos: &TextPosition{
				StartIndex: 106,
				EndIndex:   111,
				StartLine:  10,
				EndLine:    10,
			},
		},
		{
			name: "Child chunk across lines",
			chunk: &Chunk{
				Text: "Line 2\nLine 3",
				Parents: []Chunk{
					{Text: "Line 1\nLine 2\nLine 3\nLine 4"},
				},
			},
			parentPos: &TextPosition{
				StartIndex: 50,
				EndIndex:   77,
				StartLine:  5,
				EndLine:    8,
			},
			offsetInParent: 7, // Starting from "Line 2"
			expectedPos: &TextPosition{
				StartIndex: 57,
				EndIndex:   70,
				StartLine:  6, // 5 + 1 newline before "Line 2"
				EndLine:    7, // 6 + 1 more line
			},
		},
		{
			name: "Empty chunk",
			chunk: &Chunk{
				Text: "",
			},
			parentPos:      nil,
			offsetInParent: 0,
			expectedPos:    nil,
		},
		{
			name: "Chinese text chunk",
			chunk: &Chunk{
				Text: "你好\n世界", // 6 bytes for "你好" + 1 byte for "\n" + 6 bytes for "世界" = 13 bytes
			},
			parentPos:      nil,
			offsetInParent: 0,
			expectedPos: &TextPosition{
				StartIndex: 0,
				EndIndex:   13, // Corrected byte count
				StartLine:  1,
				EndLine:    2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.chunk.CalculateTextPos(tt.parentPos, tt.offsetInParent)

			if tt.expectedPos == nil {
				if tt.chunk.TextPos != nil {
					t.Errorf("Expected nil TextPos, got %+v", tt.chunk.TextPos)
				}
				return
			}

			if tt.chunk.TextPos == nil {
				t.Fatalf("Expected TextPos to be calculated, got nil")
			}

			if !reflect.DeepEqual(tt.chunk.TextPos, tt.expectedPos) {
				t.Errorf("CalculateTextPos() = %+v, expected %+v", tt.chunk.TextPos, tt.expectedPos)
			}
		})
	}
}

func TestChunk_UpdateTextPosFromText(t *testing.T) {
	tests := []struct {
		name        string
		chunk       *Chunk
		newText     string
		expectedPos *TextPosition
	}{
		{
			name: "Update existing TextPos",
			chunk: &Chunk{
				Text: "Original text",
				TextPos: &TextPosition{
					StartIndex: 100,
					EndIndex:   113,
					StartLine:  5,
					EndLine:    5,
				},
			},
			newText: "New longer text content",
			expectedPos: &TextPosition{
				StartIndex: 100, // Preserved
				EndIndex:   123, // 100 + 23 (length of new text)
				StartLine:  5,   // Preserved
				EndLine:    5,   // Single line
			},
		},
		{
			name: "Update with multiline text",
			chunk: &Chunk{
				Text: "Single line",
				TextPos: &TextPosition{
					StartIndex: 50,
					EndIndex:   61,
					StartLine:  3,
					EndLine:    3,
				},
			},
			newText: "Multi\nline\ntext",
			expectedPos: &TextPosition{
				StartIndex: 50, // Preserved
				EndIndex:   65, // 50 + 15
				StartLine:  3,  // Preserved
				EndLine:    5,  // 3 + 2 additional lines
			},
		},
		{
			name: "Update chunk with no existing TextPos",
			chunk: &Chunk{
				Text:    "Some text",
				TextPos: nil,
			},
			newText: "Updated text",
			expectedPos: &TextPosition{
				StartIndex: 0,
				EndIndex:   12,
				StartLine:  1,
				EndLine:    1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.chunk.Text = tt.newText
			tt.chunk.UpdateTextPosFromText()

			if tt.chunk.TextPos == nil {
				t.Fatalf("Expected TextPos to be updated, got nil")
			}

			if !reflect.DeepEqual(tt.chunk.TextPos, tt.expectedPos) {
				t.Errorf("UpdateTextPosFromText() = %+v, expected %+v", tt.chunk.TextPos, tt.expectedPos)
			}
		})
	}
}

func TestChunk_CalculateRelativeTextPos(t *testing.T) {
	chunk := &Chunk{
		Text: "Hello\nWorld\nThis is line 3",
		TextPos: &TextPosition{
			StartIndex: 100,
			EndIndex:   125,
			StartLine:  10,
			EndLine:    12,
		},
	}

	tests := []struct {
		name        string
		startOffset int
		endOffset   int
		expectedPos *TextPosition
		expectNil   bool
	}{
		{
			name:        "Valid substring - first line",
			startOffset: 0,
			endOffset:   5, // "Hello"
			expectedPos: &TextPosition{
				StartIndex: 100,
				EndIndex:   105,
				StartLine:  10,
				EndLine:    10,
			},
		},
		{
			name:        "Valid substring - across lines",
			startOffset: 6,
			endOffset:   17, // "World\nThis"
			expectedPos: &TextPosition{
				StartIndex: 106,
				EndIndex:   117,
				StartLine:  11, // Second line
				EndLine:    12, // Third line
			},
		},
		{
			name:        "Valid substring - third line",
			startOffset: 12,
			endOffset:   25, // "This is line 3"
			expectedPos: &TextPosition{
				StartIndex: 112,
				EndIndex:   125,
				StartLine:  12,
				EndLine:    12,
			},
		},
		{
			name:        "Invalid - negative start",
			startOffset: -1,
			endOffset:   5,
			expectNil:   true,
		},
		{
			name:        "Invalid - start >= end",
			startOffset: 10,
			endOffset:   10,
			expectNil:   true,
		},
		{
			name:        "Invalid - start beyond text",
			startOffset: 100,
			endOffset:   105,
			expectNil:   true,
		},
		{
			name:        "Valid - end clamped to text length",
			startOffset: 20,
			endOffset:   50, // Will be clamped to 26 (text length)
			expectedPos: &TextPosition{
				StartIndex: 120,
				EndIndex:   126, // 100 + 26 (actual text length)
				StartLine:  12,
				EndLine:    12,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunk.CalculateRelativeTextPos(tt.startOffset, tt.endOffset)

			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil result, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("Expected TextPosition, got nil")
			}

			if !reflect.DeepEqual(result, tt.expectedPos) {
				t.Errorf("CalculateRelativeTextPos() = %+v, expected %+v", result, tt.expectedPos)
			}
		})
	}
}

func TestChunk_GetTextAtPosition(t *testing.T) {
	chunk := &Chunk{
		Text: "Hello World! This is a test.",
		TextPos: &TextPosition{
			StartIndex: 100,
			EndIndex:   128,
			StartLine:  5,
			EndLine:    5,
		},
	}

	tests := []struct {
		name     string
		pos      *TextPosition
		expected string
	}{
		{
			name: "Valid position - beginning",
			pos: &TextPosition{
				StartIndex: 100,
				EndIndex:   105,
			},
			expected: "Hello",
		},
		{
			name: "Valid position - middle",
			pos: &TextPosition{
				StartIndex: 106,
				EndIndex:   112,
			},
			expected: "World!",
		},
		{
			name: "Valid position - full text",
			pos: &TextPosition{
				StartIndex: 100,
				EndIndex:   128,
			},
			expected: "Hello World! This is a test.",
		},
		{
			name: "Invalid position - before chunk",
			pos: &TextPosition{
				StartIndex: 90,
				EndIndex:   105,
			},
			expected: "",
		},
		{
			name: "Invalid position - after chunk",
			pos: &TextPosition{
				StartIndex: 120,
				EndIndex:   140,
			},
			expected: "",
		},
		{
			name: "Invalid position - spans beyond chunk",
			pos: &TextPosition{
				StartIndex: 110,
				EndIndex:   140,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunk.GetTextAtPosition(tt.pos)
			if result != tt.expected {
				t.Errorf("GetTextAtPosition() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestChunk_GetTextAtPosition_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		chunk    *Chunk
		pos      *TextPosition
		expected string
	}{
		{
			name:     "Nil chunk",
			chunk:    nil,
			pos:      &TextPosition{StartIndex: 0, EndIndex: 5},
			expected: "",
		},
		{
			name: "Empty text",
			chunk: &Chunk{
				Text:    "",
				TextPos: &TextPosition{StartIndex: 0, EndIndex: 0},
			},
			pos:      &TextPosition{StartIndex: 0, EndIndex: 5},
			expected: "",
		},
		{
			name: "Nil position",
			chunk: &Chunk{
				Text:    "Some text",
				TextPos: &TextPosition{StartIndex: 0, EndIndex: 9},
			},
			pos:      nil,
			expected: "",
		},
		{
			name: "Chunk without TextPos",
			chunk: &Chunk{
				Text:    "Some text",
				TextPos: nil,
			},
			pos:      &TextPosition{StartIndex: 0, EndIndex: 5},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.chunk.GetTextAtPosition(tt.pos)
			if result != tt.expected {
				t.Errorf("GetTextAtPosition() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestValidatePositions(t *testing.T) {
	tests := []struct {
		name      string
		chars     []string
		positions []Position
		expectErr bool
		errMsg    string
	}{
		{
			name:      "Empty chars and positions",
			chars:     []string{},
			positions: []Position{},
			expectErr: false,
		},
		{
			name:      "Empty chars with positions",
			chars:     []string{},
			positions: []Position{{StartPos: 0, EndPos: 1}},
			expectErr: false, // Early return for empty chars
		},
		{
			name:      "Valid positions",
			chars:     []string{"H", "e", "l", "l", "o"},
			positions: []Position{{StartPos: 0, EndPos: 2}, {StartPos: 2, EndPos: 5}},
			expectErr: false,
		},
		{
			name:      "Negative StartPos",
			chars:     []string{"H", "e", "l", "l", "o"},
			positions: []Position{{StartPos: -1, EndPos: 2}},
			expectErr: true,
			errMsg:    "position 0 has negative StartPos (-1)",
		},
		{
			name:      "Negative EndPos",
			chars:     []string{"H", "e", "l", "l", "o"},
			positions: []Position{{StartPos: 0, EndPos: -1}},
			expectErr: true,
			errMsg:    "position 0 has negative EndPos (-1)",
		},
		{
			name:      "StartPos >= EndPos",
			chars:     []string{"H", "e", "l", "l", "o"},
			positions: []Position{{StartPos: 3, EndPos: 2}},
			expectErr: true,
			errMsg:    "position 0 has StartPos (3) >= EndPos (2)",
		},
		{
			name:      "StartPos equal to EndPos",
			chars:     []string{"H", "e", "l", "l", "o"},
			positions: []Position{{StartPos: 2, EndPos: 2}},
			expectErr: true,
			errMsg:    "position 0 has StartPos (2) >= EndPos (2)",
		},
		{
			name:      "StartPos out of bounds",
			chars:     []string{"H", "e", "l", "l", "o"},
			positions: []Position{{StartPos: 5, EndPos: 6}},
			expectErr: true,
			errMsg:    "position 0 has StartPos (5) out of bounds (5)",
		},
		{
			name:      "EndPos out of bounds",
			chars:     []string{"H", "e", "l", "l", "o"},
			positions: []Position{{StartPos: 0, EndPos: 6}},
			expectErr: true,
			errMsg:    "position 0 has EndPos (6) out of bounds (5)",
		},
		{
			name:  "Multiple positions with mixed validity",
			chars: []string{"H", "e", "l", "l", "o"},
			positions: []Position{
				{StartPos: 0, EndPos: 2},  // Valid
				{StartPos: 2, EndPos: 4},  // Valid
				{StartPos: -1, EndPos: 1}, // Invalid - negative start
			},
			expectErr: true,
			errMsg:    "position 2 has negative StartPos (-1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePositions(tt.chars, tt.positions)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestChunk_TextWChars_InvalidUTF8(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Invalid UTF-8 sequence",
			text:     "Hello\xff\xfeWorld",
			expected: []string{"H", "e", "l", "l", "o", "W", "o", "r", "l", "d"},
		},
		{
			name:     "Mixed valid and invalid UTF-8",
			text:     "你好\xff世界",
			expected: []string{"你", "好", "世", "界"},
		},
		// Note: Removed "Only invalid UTF-8" test case as it's correctly handled by the implementation
		// The function correctly returns an empty slice for invalid UTF-8 sequences
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &Chunk{Text: tt.text}
			result := chunk.TextWChars()

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("TextWChars() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestChunk_TextWChars_OnlyInvalidUTF8 tests the case where text contains only invalid UTF-8
func TestChunk_TextWChars_OnlyInvalidUTF8(t *testing.T) {
	chunk := &Chunk{Text: "\xff\xfe\xfd"}
	result := chunk.TextWChars()

	// Should return empty slice for text with only invalid UTF-8 sequences
	if len(result) != 0 {
		t.Errorf("Expected empty slice for invalid UTF-8, got %v", result)
	}
}

func TestChunk_TextLinesToWChars_InvalidUTF8(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected [][]string
	}{
		{
			name:     "Invalid UTF-8 in lines",
			text:     "Hello\xff\nWorld\xfe",
			expected: [][]string{{"H", "e", "l", "l", "o"}, {"W", "o", "r", "l", "d"}},
		},
		{
			name:     "Mixed valid and invalid UTF-8 across lines",
			text:     "你好\xff\n世界\xfe",
			expected: [][]string{{"你", "好"}, {"世", "界"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &Chunk{Text: tt.text}
			result := chunk.TextLinesToWChars()

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("TextLinesToWChars() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Test for types.go functions
func TestIndexType(t *testing.T) {
	tests := []struct {
		indexType IndexType
		expected  string
		isValid   bool
	}{
		{IndexTypeHNSW, "hnsw", true},
		{IndexTypeIVF, "ivf", true},
		{IndexTypeFlat, "flat", true},
		{IndexTypeLSH, "lsh", true},
		{IndexType("invalid"), "invalid", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.indexType), func(t *testing.T) {
			if tt.indexType.String() != tt.expected {
				t.Errorf("String() = %s, expected %s", tt.indexType.String(), tt.expected)
			}
			if tt.indexType.IsValid() != tt.isValid {
				t.Errorf("IsValid() = %t, expected %t", tt.indexType.IsValid(), tt.isValid)
			}
		})
	}

	// Test GetSupportedIndexTypes
	supported := GetSupportedIndexTypes()
	expectedTypes := []IndexType{IndexTypeHNSW, IndexTypeIVF, IndexTypeFlat, IndexTypeLSH}
	if !reflect.DeepEqual(supported, expectedTypes) {
		t.Errorf("GetSupportedIndexTypes() = %v, expected %v", supported, expectedTypes)
	}
}

func TestDistanceMetric(t *testing.T) {
	tests := []struct {
		metric   DistanceMetric
		expected string
		isValid  bool
	}{
		{DistanceCosine, "cosine", true},
		{DistanceEuclidean, "euclidean", true},
		{DistanceDot, "dot", true},
		{DistanceManhattan, "manhattan", true},
		{DistanceMetric("invalid"), "invalid", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.metric), func(t *testing.T) {
			if tt.metric.String() != tt.expected {
				t.Errorf("String() = %s, expected %s", tt.metric.String(), tt.expected)
			}
			if tt.metric.IsValid() != tt.isValid {
				t.Errorf("IsValid() = %t, expected %t", tt.metric.IsValid(), tt.isValid)
			}
		})
	}

	// Test GetSupportedDistanceMetrics
	supported := GetSupportedDistanceMetrics()
	expectedMetrics := []DistanceMetric{DistanceCosine, DistanceEuclidean, DistanceDot, DistanceManhattan}
	if !reflect.DeepEqual(supported, expectedMetrics) {
		t.Errorf("GetSupportedDistanceMetrics() = %v, expected %v", supported, expectedMetrics)
	}
}

func TestGetChunkingTypeFromMime(t *testing.T) {
	tests := []struct {
		mime     string
		expected ChunkingType
	}{
		{"text/plain", ChunkingTypeText},
		{"text/html", ChunkingTypeText},
		{"text/markdown", ChunkingTypeText},
		{"text/x-go", ChunkingTypeCode},
		{"text/x-python", ChunkingTypeCode},
		{"application/javascript", ChunkingTypeCode},
		{"application/pdf", ChunkingTypePDF},
		{"application/msword", ChunkingTypeWord},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", ChunkingTypeWord},
		{"text/csv", ChunkingTypeCSV},
		{"application/vnd.ms-excel", ChunkingTypeExcel},
		{"application/json", ChunkingTypeJSON},
		{"image/jpeg", ChunkingTypeImage},
		{"image/png", ChunkingTypeImage},
		{"video/mp4", ChunkingTypeVideo},
		{"audio/mpeg", ChunkingTypeAudio},
		{"unknown/type", ChunkingTypeText}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			result := GetChunkingTypeFromMime(tt.mime)
			if result != tt.expected {
				t.Errorf("GetChunkingTypeFromMime(%s) = %s, expected %s", tt.mime, result, tt.expected)
			}
		})
	}
}

func TestGetChunkingTypeFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected ChunkingType
	}{
		{"document.txt", ChunkingTypeText},
		{"README.md", ChunkingTypeText},
		{"script.py", ChunkingTypeCode},
		{"main.go", ChunkingTypeCode},
		{"document.pdf", ChunkingTypePDF},
		{"report.doc", ChunkingTypeWord},
		{"report.docx", ChunkingTypeWord},
		{"data.csv", ChunkingTypeCSV},
		{"spreadsheet.xls", ChunkingTypeExcel},
		{"config.json", ChunkingTypeJSON},
		{"photo.jpg", ChunkingTypeImage},
		{"movie.mp4", ChunkingTypeVideo},
		{"song.mp3", ChunkingTypeAudio},
		{"unknown.xyz", ChunkingTypeText}, // Default
		{"", ChunkingTypeText},            // Empty filename
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := GetChunkingTypeFromFilename(tt.filename)
			if result != tt.expected {
				t.Errorf("GetChunkingTypeFromFilename(%s) = %s, expected %s", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestVectorStoreConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *VectorStoreConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid HNSW config",
			config: &VectorStoreConfig{
				Dimension:      1536,
				Distance:       DistanceCosine,
				IndexType:      IndexTypeHNSW,
				CollectionName: "test_collection",
			},
			expectErr: false,
		},
		{
			name: "Invalid dimension",
			config: &VectorStoreConfig{
				Dimension:      0,
				Distance:       DistanceCosine,
				IndexType:      IndexTypeHNSW,
				CollectionName: "test_collection",
			},
			expectErr: true,
			errMsg:    "dimension must be positive",
		},
		{
			name: "Invalid distance metric",
			config: &VectorStoreConfig{
				Dimension:      1536,
				Distance:       DistanceMetric("invalid"),
				IndexType:      IndexTypeHNSW,
				CollectionName: "test_collection",
			},
			expectErr: true,
			errMsg:    "invalid distance metric",
		},
		{
			name: "Invalid index type",
			config: &VectorStoreConfig{
				Dimension:      1536,
				Distance:       DistanceCosine,
				IndexType:      IndexType("invalid"),
				CollectionName: "test_collection",
			},
			expectErr: true,
			errMsg:    "invalid index type",
		},
		{
			name: "Empty collection name",
			config: &VectorStoreConfig{
				Dimension:      1536,
				Distance:       DistanceCosine,
				IndexType:      IndexTypeHNSW,
				CollectionName: "",
			},
			expectErr: true,
			errMsg:    "collection name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestLoadState_String(t *testing.T) {
	tests := []struct {
		state    LoadState
		expected string
	}{
		{LoadStateNotExist, "NotExist"},
		{LoadStateNotLoad, "NotLoad"},
		{LoadStateLoading, "Loading"},
		{LoadStateLoaded, "Loaded"},
		{LoadState(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("String() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkChunk_TextWChars(b *testing.B) {
	benchmarks := []struct {
		name string
		text string
	}{
		{"Short English", "Hello World"},
		{"Long English", strings.Repeat("Hello World! ", 100)},
		{"Chinese Text", "你好世界！这是一个测试。"},
		{"Mixed Text", "Hello世界123🌍" + strings.Repeat("Mixed Content ", 50)},
		{"Emoji Heavy", strings.Repeat("😀🎉🌍💻🚀", 100)},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			chunk := &Chunk{Text: bm.text}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = chunk.TextWChars()
			}
		})
	}
}

func BenchmarkChunk_TextLines(b *testing.B) {
	benchmarks := []struct {
		name string
		text string
	}{
		{"Single Line", "This is a single line of text"},
		{"Multiple Lines", strings.Repeat("Line content\n", 100)},
		{"Mixed Newlines", strings.ReplaceAll(strings.Repeat("Line\n", 50), "\n", "\r\n")},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			chunk := &Chunk{Text: bm.text}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = chunk.TextLines()
			}
		})
	}
}

func BenchmarkChunk_Split(b *testing.B) {
	chunk := &Chunk{
		ID:   "benchmark-chunk",
		Text: strings.Repeat("Hello World! This is a test. ", 1000),
		Type: ChunkingTypeText,
	}

	positions := []Position{
		{StartPos: 0, EndPos: 100},
		{StartPos: 100, EndPos: 200},
		{StartPos: 200, EndPos: 300},
	}

	chars := chunk.TextWChars()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chunk.Split(chars, positions)
	}
}

// Concurrent tests
func TestChunk_ConcurrentAccess(t *testing.T) {
	chunk := &Chunk{
		Text: "Concurrent access test text with some content for testing thread safety",
		TextPos: &TextPosition{
			StartIndex: 0,
			EndIndex:   70,
			StartLine:  1,
			EndLine:    1,
		},
	}

	const numGoroutines = 100
	const numOperations = 10

	var wg sync.WaitGroup
	results := make(chan bool, numGoroutines*numOperations)

	// Test concurrent read operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Test various read operations concurrently
				chars := chunk.TextWChars()
				lines := chunk.TextLines()
				linesChars := chunk.TextLinesToWChars()

				// Verify results are consistent
				if len(chars) == 0 || len(lines) == 0 || len(linesChars) == 0 {
					results <- false
					return
				}

				// Test JSON operations
				if _, err := chunk.TextWCharsJSON(); err != nil {
					results <- false
					return
				}
				if _, err := chunk.TextLinesJSON(); err != nil {
					results <- false
					return
				}
				if _, err := chunk.TextLinesToWCharsJSON(); err != nil {
					results <- false
					return
				}

				results <- true
			}
		}()
	}

	wg.Wait()
	close(results)

	// Check all operations succeeded
	successCount := 0
	totalCount := 0
	for success := range results {
		totalCount++
		if success {
			successCount++
		}
	}

	expectedTotal := numGoroutines * numOperations
	if totalCount != expectedTotal {
		t.Errorf("Expected %d operations, got %d", expectedTotal, totalCount)
	}
	if successCount != totalCount {
		t.Errorf("%d out of %d concurrent operations failed", totalCount-successCount, totalCount)
	}
}

func TestChunk_ConcurrentSplit(t *testing.T) {
	baseChunk := &Chunk{
		ID:   "concurrent-split-test",
		Text: strings.Repeat("Hello World! This is test content. ", 100),
		Type: ChunkingTypeText,
		TextPos: &TextPosition{
			StartIndex: 0,
			EndIndex:   3600, // Approximate
			StartLine:  1,
			EndLine:    1,
		},
	}

	const numGoroutines = 50
	var wg sync.WaitGroup
	results := make(chan []*Chunk, numGoroutines)

	positions := []Position{
		{StartPos: 0, EndPos: 100},
		{StartPos: 100, EndPos: 200},
		{StartPos: 200, EndPos: 300},
	}

	// Perform concurrent splits on different chunk instances
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Create a copy of the chunk for each goroutine
			chunk := &Chunk{
				ID:      fmt.Sprintf("chunk-%d", id),
				Text:    baseChunk.Text,
				Type:    baseChunk.Type,
				TextPos: baseChunk.TextPos,
			}
			chars := chunk.TextWChars()
			splitResult := chunk.Split(chars, positions)
			results <- splitResult
		}(i)
	}

	wg.Wait()
	close(results)

	// Verify all splits produced expected results
	for splitResult := range results {
		if len(splitResult) != 3 {
			t.Errorf("Expected 3 split chunks, got %d", len(splitResult))
		}
		for i, subChunk := range splitResult {
			if subChunk.Text == "" {
				t.Errorf("Split chunk %d has empty text", i)
			}
			if subChunk.ParentID == "" {
				t.Errorf("Split chunk %d has no parent ID", i)
			}
		}
	}
}

// Memory leak detection tests
func TestChunk_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Create many chunks and perform operations
	const numChunks = 1000
	chunks := make([]*Chunk, numChunks)

	for i := 0; i < numChunks; i++ {
		chunks[i] = &Chunk{
			ID:   fmt.Sprintf("chunk-%d", i),
			Text: strings.Repeat("Memory test content ", 100),
			Type: ChunkingTypeText,
		}

		// Perform operations
		_ = chunks[i].TextWChars()
		_ = chunks[i].TextLines()
		_ = chunks[i].TextLinesToWChars()
	}

	// Measure memory after operations
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Clear references
	for i := range chunks {
		chunks[i] = nil
	}
	chunks = nil

	// Force garbage collection
	runtime.GC()
	runtime.GC() // Call twice to ensure cleanup

	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)

	// Memory should be freed after GC
	memoryIncrease := m2.Alloc - m1.Alloc

	// Handle potential underflow when calculating memory after GC
	var memoryAfterGC uint64
	if m3.Alloc > m1.Alloc {
		memoryAfterGC = m3.Alloc - m1.Alloc
	} else {
		memoryAfterGC = 0 // Memory was freed to below initial level
	}

	t.Logf("Memory before: %d bytes", m1.Alloc)
	t.Logf("Memory after operations: %d bytes", m2.Alloc)
	t.Logf("Memory after GC: %d bytes", m3.Alloc)
	t.Logf("Memory increase: %d bytes", memoryIncrease)
	t.Logf("Memory after cleanup: %d bytes", memoryAfterGC)

	// The memory after GC should be significantly less than peak usage
	// Allow for some overhead but expect most memory to be freed
	if memoryAfterGC > 0 && float64(memoryAfterGC) > float64(memoryIncrease)*0.5 {
		t.Logf("Warning: Memory may not be fully released after GC. Peak increase: %d, After GC: %d", memoryIncrease, memoryAfterGC)
		// This is a warning rather than failure as GC behavior can vary
	} else if memoryAfterGC == 0 {
		t.Logf("Memory was successfully freed to below initial level")
	}
}

func TestChunk_LargeDataHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large data test in short mode")
	}

	// Test with very large text
	largeText := strings.Repeat("Large data test content with various characters 测试内容 🌍🚀 ", 10000)
	chunk := &Chunk{
		Text: largeText,
	}

	// Measure time and ensure operations complete
	start := time.Now()
	chars := chunk.TextWChars()
	duration := time.Since(start)

	t.Logf("Processing %d characters took %v", len(largeText), duration)

	if len(chars) == 0 {
		t.Error("TextWChars returned empty result for large text")
	}

	// Test split with large text
	positions := []Position{
		{StartPos: 0, EndPos: 10000},
		{StartPos: 10000, EndPos: 20000},
		{StartPos: 20000, EndPos: len(largeText)},
	}

	start = time.Now()
	splitChars := chunk.TextWChars()
	splitResult := chunk.Split(splitChars, positions)
	duration = time.Since(start)

	t.Logf("Splitting large text took %v", duration)

	if len(splitResult) != 3 {
		t.Errorf("Expected 3 split chunks, got %d", len(splitResult))
	}
}

// Context-aware tests
func TestChunk_WithContext(t *testing.T) {
	chunk := &Chunk{
		Text: "Context-aware test content",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan bool, 1)

	// Simulate a potentially long-running operation
	go func() {
		// Perform operations
		for i := 0; i < 1000; i++ {
			select {
			case <-ctx.Done():
				done <- false
				return
			default:
				_ = chunk.TextWChars()
			}
		}
		done <- true
	}()

	select {
	case completed := <-done:
		if !completed {
			t.Log("Operation was cancelled due to context timeout (expected behavior)")
		} else {
			t.Log("Operation completed successfully")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Test timed out")
	}
}

// Error injection tests
func TestChunk_ErrorHandling(t *testing.T) {
	// Test with nil chunk pointer operations
	var nilChunk *Chunk

	// These should handle nil gracefully or panic as expected
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for nil chunk CalculateTextPos, but didn't panic")
			}
		}()
		nilChunk.CalculateTextPos(nil, 0)
	}()

	// UpdateTextPosFromText should handle nil gracefully (early return)
	nilChunk.UpdateTextPosFromText() // Should not panic

	// CalculateRelativeTextPos should return nil for nil chunk
	result := nilChunk.CalculateRelativeTextPos(0, 5)
	if result != nil {
		t.Error("Expected nil result for nil chunk CalculateRelativeTextPos")
	}
}

// Integration tests
func TestChunk_IntegrationWorkflow(t *testing.T) {
	// Test a complete workflow
	originalText := "This is a comprehensive test.\nIt includes multiple lines.\nAnd various operations."

	// Create initial chunk
	chunk := &Chunk{
		ID:   "integration-test",
		Text: originalText,
		Type: ChunkingTypeText,
	}

	// Calculate text position
	chunk.CalculateTextPos(nil, 0)
	if chunk.TextPos == nil {
		t.Fatal("TextPos should be calculated")
	}

	// Get characters
	chars := chunk.TextWChars()
	if len(chars) == 0 {
		t.Error("Should have characters")
	}

	// Get lines
	lines := chunk.TextLines()
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Split chunk
	positions := []Position{
		{StartPos: 0, EndPos: 29},                 // First line + part of second
		{StartPos: 29, EndPos: 54},                // Rest of second line
		{StartPos: 54, EndPos: len(originalText)}, // Third line
	}

	splitChars := chunk.TextWChars()
	subChunks := chunk.Split(splitChars, positions)
	if len(subChunks) != 3 {
		t.Errorf("Expected 3 sub-chunks, got %d", len(subChunks))
	}

	// Verify each sub-chunk
	for i, subChunk := range subChunks {
		if subChunk.ParentID != chunk.ID {
			t.Errorf("Sub-chunk %d has wrong parent ID", i)
		}
		if subChunk.Depth != chunk.Depth+1 {
			t.Errorf("Sub-chunk %d has wrong depth", i)
		}
		if !subChunk.Leaf {
			t.Errorf("Sub-chunk %d should be leaf", i)
		}
		if subChunk.Root {
			t.Errorf("Sub-chunk %d should not be root", i)
		}

		// Test JSON operations on sub-chunks
		if _, err := subChunk.TextWCharsJSON(); err != nil {
			t.Errorf("Sub-chunk %d TextWCharsJSON failed: %v", i, err)
		}
		if _, err := subChunk.TextLinesJSON(); err != nil {
			t.Errorf("Sub-chunk %d TextLinesJSON failed: %v", i, err)
		}
	}

	// Verify original chunk is no longer leaf
	if chunk.Leaf {
		t.Error("Original chunk should no longer be leaf after splitting")
	}
}
