package doc

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// DynamicIDGroups lists groups whose process.make() uses group.<id>.method
// addressing. The handler key in process.Handlers is "group.method" (without
// <id>), but users call them as "group.<id>.method".
//
// Exported so that CLI layers can build the callable pattern for display.
var DynamicIDGroups = map[string]bool{
	"models": true, "schemas": true, "stores": true,
	"fs": true, "tasks": true, "schedules": true,
}

// LoadYAML unmarshals embedded YAML data and registers all entries.
// Process entry names are normalised to match the handler key stored in
// process.Handlers (gou/process/process.go RegisterGroup):
//
//	group="models", name="find"   → "models.find"  (handler key)
//	group="http",   name="get"    → "http.get"     (handler key)
//	group="encoding", name="encoding.base64.Encode" → unchanged
func LoadYAML(data []byte) error {
	var file DocFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	for i := range file.Entries {
		entry := &file.Entries[i]
		if entry.Type == "" {
			entry.Type = file.Type
		}
		if entry.Group == "" {
			entry.Group = file.Group
		}
		if entry.Type == TypeProcess {
			entry.Name = normaliseProcessName(entry.Group, entry.Name)
		}
		entries[entryKey(entry.Type, entry.Name)] = entry
	}
	return nil
}

// normaliseProcessName ensures the entry name matches the handler key
// stored in process.Handlers after RegisterGroup.
//
//	group="models", name="find"                      → "models.find"
//	group="http",   name="get"                       → "http.get"
//	group="utils",  name="throw.Forbidden"           → "utils.throw.Forbidden"
//	group="encoding", name="encoding.base64.Encode"  → unchanged (already prefixed)
func normaliseProcessName(group, name string) string {
	prefix := strings.ToLower(group) + "."
	if strings.HasPrefix(strings.ToLower(name), prefix) {
		return name
	}
	return group + "." + name
}
