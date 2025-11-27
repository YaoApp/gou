package json

import (
	"bytes"
)

// TrimComments removes comments from JSONC/Yao format data
// Supports both single-line (//) and multi-line (/* */) comments
func TrimComments(data []byte) []byte {
	var result bytes.Buffer
	var inString bool
	var inMultiComment bool
	var inSingleComment bool
	var escaped bool

	for i := 0; i < len(data); i++ {
		c := data[i]

		// Handle escape sequences in strings
		if escaped {
			result.WriteByte(c)
			escaped = false
			continue
		}

		// Handle string state
		if c == '"' && !inMultiComment && !inSingleComment {
			inString = !inString
			result.WriteByte(c)
			continue
		}

		if c == '\\' && inString {
			escaped = true
			result.WriteByte(c)
			continue
		}

		// Skip if in string
		if inString {
			result.WriteByte(c)
			continue
		}

		// Handle single-line comment start
		if !inMultiComment && !inSingleComment && c == '/' && i+1 < len(data) && data[i+1] == '/' {
			inSingleComment = true
			i++ // Skip the second '/'
			continue
		}

		// Handle single-line comment end
		if inSingleComment && (c == '\n' || c == '\r') {
			inSingleComment = false
			result.WriteByte(c) // Keep the newline
			continue
		}

		// Handle multi-line comment start
		if !inMultiComment && !inSingleComment && c == '/' && i+1 < len(data) && data[i+1] == '*' {
			inMultiComment = true
			i++ // Skip the '*'
			continue
		}

		// Handle multi-line comment end
		if inMultiComment && c == '*' && i+1 < len(data) && data[i+1] == '/' {
			inMultiComment = false
			i++ // Skip the '/'
			continue
		}

		// Skip content inside comments
		if inSingleComment || inMultiComment {
			continue
		}

		// Write regular content
		result.WriteByte(c)
	}

	return result.Bytes()
}
