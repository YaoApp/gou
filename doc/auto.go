package doc

import (
	"sort"

	"github.com/yaoapp/gou/process"
)

// AutoDiscover compares process.Handlers with registered doc entries and
// returns a sorted list of handler names that have no documentation.
func AutoDiscover() []string {
	mu.RLock()
	defer mu.RUnlock()

	var undocumented []string
	for name := range process.Handlers {
		key := entryKey(TypeProcess, name)
		if _, ok := entries[key]; ok {
			continue
		}
		undocumented = append(undocumented, name)
	}
	sort.Strings(undocumented)
	return undocumented
}

// CallableName returns the user-facing callable pattern for a process entry.
// For dynamic-ID groups (models, schemas, stores, fs, tasks, schedules) the
// handler key is "group.method" but users call "group.<id>.method", so this
// function returns "models.<id>.find". For all other groups it returns the
// entry name as-is.
func CallableName(e *Entry) string {
	if e.Type != TypeProcess {
		return e.Name
	}
	if DynamicIDGroups[e.Group] {
		parts := splitDot(e.Name)
		if len(parts) == 2 {
			return parts[0] + ".<id>." + parts[1]
		}
	}
	return e.Name
}

func splitDot(s string) []string {
	out := make([]string, 0, 4)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
