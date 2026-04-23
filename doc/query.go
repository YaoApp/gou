package doc

import (
	"sort"
	"strings"
)

// List returns entries of the given type, optionally filtered.
func List(entryType EntryType, opts ...ListOption) []*Entry {
	mu.RLock()
	defer mu.RUnlock()

	var opt ListOption
	if len(opts) > 0 {
		opt = opts[0]
	}

	out := make([]*Entry, 0)
	for _, e := range entries {
		if e.Type != entryType {
			continue
		}
		if opt.Group != "" && !strings.EqualFold(e.Group, opt.Group) {
			continue
		}
		if opt.Search != "" && !containsFold(e, opt.Search) {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// Validate checks whether a named entry of the given type is documented.
// For process entries it also resolves callable names like "models.user.find"
// to their handler key "models.find" (stripping the dynamic <id> segment).
func Validate(entryType EntryType, name string) *ValidationResult {
	mu.RLock()
	defer mu.RUnlock()

	key := entryKey(entryType, name)
	if e, ok := entries[key]; ok {
		return &ValidationResult{
			Valid:   true,
			Status:  "ok",
			Name:    name,
			Message: "Documented",
			Entry:   e,
		}
	}

	// For process type, try resolving callable name → handler key.
	if entryType == TypeProcess {
		if hk := callableToHandler(name); hk != "" {
			key2 := entryKey(entryType, hk)
			if e, ok := entries[key2]; ok {
				return &ValidationResult{
					Valid:   true,
					Status:  "ok",
					Name:    name,
					Message: "Documented",
					Entry:   e,
				}
			}
		}
	}

	// case-insensitive scan
	lower := strings.ToLower(name)
	for k, e := range entries {
		if e.Type != entryType {
			continue
		}
		parts := strings.SplitN(k, ":", 2)
		if len(parts) == 2 && parts[1] == lower {
			return &ValidationResult{
				Valid:   true,
				Status:  "ok",
				Name:    name,
				Message: "Documented",
				Entry:   e,
			}
		}
	}

	// fuzzy suggestions
	suggestions := fuzzyMatch(entryType, name)
	return &ValidationResult{
		Valid:      false,
		Status:     "not_found",
		Name:       name,
		Message:    "No documentation found",
		Suggestion: suggestions,
	}
}

// Search returns entries whose name or description contains the keyword.
func Search(keyword string) []*Entry {
	mu.RLock()
	defer mu.RUnlock()

	out := make([]*Entry, 0)
	for _, e := range entries {
		if containsFold(e, keyword) {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Groups returns a sorted list of distinct group names for the given type.
func Groups(entryType EntryType) []string {
	mu.RLock()
	defer mu.RUnlock()

	seen := map[string]bool{}
	for _, e := range entries {
		if e.Type == entryType && e.Group != "" {
			seen[e.Group] = true
		}
	}
	out := make([]string, 0, len(seen))
	for g := range seen {
		out = append(out, g)
	}
	sort.Strings(out)
	return out
}

// Stats returns documentation coverage statistics keyed by EntryType.
func Stats() map[EntryType]*StatInfo {
	mu.RLock()
	defer mu.RUnlock()

	m := map[EntryType]*StatInfo{}
	for _, e := range entries {
		si, ok := m[e.Type]
		if !ok {
			si = &StatInfo{}
			m[e.Type] = si
		}
		si.Total++
		si.Documented++
	}
	return m
}

func containsFold(e *Entry, kw string) bool {
	kw = strings.ToLower(kw)
	return strings.Contains(strings.ToLower(e.Name), kw) ||
		strings.Contains(strings.ToLower(e.Desc), kw) ||
		strings.Contains(strings.ToLower(e.Group), kw)
}

func fuzzyMatch(entryType EntryType, name string) []string {
	lower := strings.ToLower(name)
	var suggestions []string
	for _, e := range entries {
		if e.Type != entryType {
			continue
		}
		eName := strings.ToLower(e.Name)
		if strings.Contains(eName, lower) || strings.Contains(lower, eName) {
			suggestions = append(suggestions, e.Name)
		} else if levenshtein(lower, eName) <= 3 {
			suggestions = append(suggestions, e.Name)
		}
		if len(suggestions) >= 5 {
			break
		}
	}
	return suggestions
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
