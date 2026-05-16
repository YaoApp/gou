package utils

import (
	"path/filepath"
)

// JoinPath joins Unix-style relative path segments into an OS-native path.
// Input segments may use forward slashes (developer convention).
// Output is an OS-native path suitable for os.Open, os.Stat, etc.
// For absolute root + relative path, use AbsJoinPath instead.
func JoinPath(segments ...string) string {
	for i, s := range segments {
		segments[i] = filepath.FromSlash(s)
	}
	return filepath.Join(segments...)
}

// AbsJoinPath joins an OS absolute root with Unix-style relative path segments.
// root is kept as-is (typically from App.Root()), rel segments are FromSlash-converted.
func AbsJoinPath(root string, rel ...string) string {
	parts := make([]string, 0, 1+len(rel))
	parts = append(parts, root)
	for _, r := range rel {
		parts = append(parts, filepath.FromSlash(r))
	}
	return filepath.Join(parts...)
}

// SlashPath converts an OS-native path to Unix-style forward-slash path.
// Use when returning paths to developers (JS bridge, Dict name, API routes, etc.).
func SlashPath(osPath string) string {
	return filepath.ToSlash(osPath)
}
