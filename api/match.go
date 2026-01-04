package api

import (
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// RouteTable is a thread-safe route table for dynamic route lookup
type RouteTable struct {
	mu     sync.RWMutex
	routes map[string][]*RouteEntry // method -> entries
}

// RouteEntry represents a single route entry
type RouteEntry struct {
	Pattern string         // Original route pattern, e.g., /user/:id
	Regex   *regexp.Regexp // Compiled regex for matching
	Params  []string       // Parameter names, e.g., ["id"]
	API     *API           // Reference to the API definition
	Path    *Path          // Reference to the Path definition
}

// Global route table instance
var routeTable = &RouteTable{
	routes: make(map[string][]*RouteEntry),
}

// compilePattern compiles a route pattern into a regex and extracts parameter names
// Supports:
//   - Exact match: /user/list
//   - Path parameters: /user/:id -> matches /user/123
//   - Wildcard: /file/*path -> matches /file/a/b/c
func compilePattern(pattern string) (*regexp.Regexp, []string) {
	var params []string
	var regexParts []string

	// Split pattern into segments
	segments := strings.Split(pattern, "/")
	for _, segment := range segments {
		if segment == "" {
			continue
		}

		if strings.HasPrefix(segment, ":") {
			// Path parameter :id
			paramName := segment[1:]
			params = append(params, paramName)
			regexParts = append(regexParts, "([^/]+)")
		} else if strings.HasPrefix(segment, "*") {
			// Wildcard *path
			paramName := segment[1:]
			params = append(params, paramName)
			regexParts = append(regexParts, "(.*)")
		} else {
			// Exact match segment
			regexParts = append(regexParts, regexp.QuoteMeta(segment))
		}
	}

	// Build the full regex pattern
	regexStr := "^/" + strings.Join(regexParts, "/") + "$"
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, nil
	}

	return regex, params
}

// matchPath matches a path against a route entry and extracts parameters
// Returns the extracted parameters and whether the match was successful
func matchPath(entry *RouteEntry, path string) (map[string]string, bool) {
	// If no regex, do exact match
	if entry.Regex == nil {
		if entry.Pattern == path {
			return nil, true
		}
		return nil, false
	}

	// Regex match
	matches := entry.Regex.FindStringSubmatch(path)
	if matches == nil {
		return nil, false
	}

	// Extract parameters
	params := make(map[string]string)
	for i, name := range entry.Params {
		if i+1 < len(matches) {
			params[name] = matches[i+1]
		}
	}

	return params, true
}

// hasPathParams checks if a pattern contains path parameters or wildcards
func hasPathParams(pattern string) bool {
	return strings.Contains(pattern, ":") || strings.Contains(pattern, "*")
}

// buildFullPath builds the full path from API group and path
func buildFullPath(apiGroup, pathPattern string) string {
	if apiGroup == "" {
		return pathPattern
	}
	return filepath.Join("/", apiGroup, pathPattern)
}

// addEntry adds a route entry to the route table
func (rt *RouteTable) addEntry(method string, entry *RouteEntry) {
	method = strings.ToUpper(method)
	rt.routes[method] = append(rt.routes[method], entry)
}

// clear clears all routes from the table
func (rt *RouteTable) clear() {
	rt.routes = make(map[string][]*RouteEntry)
}

// find finds a matching route entry for the given method and path
func (rt *RouteTable) find(method, path string) (*RouteEntry, map[string]string) {
	method = strings.ToUpper(method)
	entries, ok := rt.routes[method]
	if !ok {
		return nil, nil
	}

	// First try exact matches (entries without regex)
	for _, entry := range entries {
		if entry.Regex == nil && entry.Pattern == path {
			return entry, nil
		}
	}

	// Then try regex matches
	for _, entry := range entries {
		if entry.Regex != nil {
			if params, matched := matchPath(entry, path); matched {
				return entry, params
			}
		}
	}

	return nil, nil
}
