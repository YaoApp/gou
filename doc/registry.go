package doc

import (
	"strings"
	"sync"
)

var (
	entries = map[string]*Entry{}
	mu      sync.RWMutex
)

func entryKey(t EntryType, name string) string {
	return string(t) + ":" + strings.ToLower(name)
}

// Register manually registers a single entry.
func Register(entry *Entry) {
	mu.Lock()
	defer mu.Unlock()
	entries[entryKey(entry.Type, entry.Name)] = entry
}

// Get retrieves a single entry by type and name (case-insensitive).
// For process entries it also resolves callable names like "models.user.find"
// to the handler key "models.find".
func Get(t EntryType, name string) (*Entry, bool) {
	mu.RLock()
	defer mu.RUnlock()
	if e, ok := entries[entryKey(t, name)]; ok {
		return e, ok
	}
	if t == TypeProcess {
		if hk := callableToHandler(name); hk != "" {
			if e, ok := entries[entryKey(t, hk)]; ok {
				return e, ok
			}
		}
	}
	return nil, false
}

// callableToHandler converts "models.user.find" → "models.find" for
// dynamic-ID groups. Returns "" if not applicable.
func callableToHandler(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) < 3 {
		return ""
	}
	group := strings.ToLower(parts[0])
	if !DynamicIDGroups[group] {
		return ""
	}
	method := parts[len(parts)-1]
	return group + "." + strings.ToLower(method)
}

// All returns a snapshot of every registered entry.
func All() []*Entry {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]*Entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, e)
	}
	return out
}

// Reset clears all registered entries (testing only).
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	entries = map[string]*Entry{}
}
