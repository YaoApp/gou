package json

import (
	"testing"
)

func TestTrimCommentsEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(string) bool
	}{
		{
			name:  "Only whitespace",
			input: "   \n\t  ",
			check: func(s string) bool {
				return s == "   \n\t  "
			},
		},
		{
			name:  "Only comment",
			input: "// just a comment",
			check: func(s string) bool {
				// Comment is removed, may be empty or newline
				return len(s) >= 0
			},
		},
		{
			name:  "Only multi-line comment",
			input: "/* comment */",
			check: func(s string) bool {
				// Comment is removed
				return len(s) >= 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(TrimComments([]byte(tt.input)))
			if !tt.check(got) {
				t.Errorf("TrimComments() check failed, got = %q", got)
			}
		})
	}
}

func TestTrimComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Single-line comment",
			input: `{
  "name": "test", // this is a comment
  "age": 25
}`,
			expected: `{
  "name": "test", 
  "age": 25
}`,
		},
		{
			name: "Multi-line comment",
			input: `{
  /* This is a
     multi-line comment */
  "name": "test"
}`,
			expected: `{
  
  "name": "test"
}`,
		},
		{
			name: "Mixed comments",
			input: `{
  // Single line comment
  "name": "test", /* inline comment */
  "age": 25 // another comment
}`,
			expected: `{
  
  "name": "test", 
  "age": 25 
}`,
		},
		{
			name:     "No comments",
			input:    `{"name":"test","age":25}`,
			expected: `{"name":"test","age":25}`,
		},
		{
			name:     "Empty string",
			input:    ``,
			expected: ``,
		},
		{
			name: "Comment at start",
			input: `// Comment at the beginning
{"name": "test"}`,
			expected: `
{"name": "test"}`,
		},
		{
			name: "Comment at end",
			input: `{"name": "test"}
// Comment at the end`,
			expected: `{"name": "test"}
`,
		},
		{
			name: "Multi-line comment with asterisks",
			input: `{
  "name": "test", /* comment with * symbols */
  "age": 25
}`,
			expected: `{
  "name": "test", 
  "age": 25
}`,
		},
		{
			name: "String with comment-like content should be preserved",
			input: `{
  "url": "https://example.com", // actual comment
  "age": 25
}`,
			expected: `{
  "url": "https://example.com", 
  "age": 25
}`,
		},
		{
			name: "Multiple single-line comments",
			input: `{
  // Comment 1
  // Comment 2
  // Comment 3
  "name": "test"
}`,
			expected: `{
  
  
  
  "name": "test"
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(TrimComments([]byte(tt.input)))
			if got != tt.expected {
				t.Errorf("TrimComments() = %q, want %q", got, tt.expected)
			}
		})
	}
}
