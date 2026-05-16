package utils

import (
	"bytes"
	"strings"
)

// BytesLF normalizes \r\n and isolated \r to \n.
func BytesLF(data []byte) []byte {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	return data
}

// StringLF normalizes \r\n and isolated \r to \n (string variant).
func StringLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}
