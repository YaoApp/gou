package doc

import (
	"testing"
)

const testYAML = `
group: test
type: process
entries:
  - name: test.hello
    desc: Say hello
    args:
      - name: name
        type: string
        required: true
        desc: Name to greet
    return:
      type: string
      desc: Greeting message

  - name: test.add
    desc: Add two numbers
    args:
      - name: a
        type: number
        required: true
      - name: b
        type: number
        required: true
    return:
      type: number
      desc: Sum of a and b

  - name: test.info
    desc: Get info
    return:
      type: object
      fields:
        - name: version
          type: string
        - name: count
          type: number
`

const testRuntimeYAML = `
group: objects
type: js_object
entries:
  - name: log
    desc: Logging utility
    methods:
      - name: Info
        desc: Log at info level
        args:
          - name: message
            type: string
            required: true
        return:
          type: void
      - name: Error
        desc: Log at error level
        args:
          - name: message
            type: string
            required: true
        return:
          type: void
`

const testClassYAML = `
group: objects
type: js_class
entries:
  - name: FS
    desc: File system operations
    args:
      - name: name
        type: string
        desc: FS engine name
    methods:
      - name: ReadFile
        desc: Read file content
        args:
          - name: path
            type: string
            required: true
        return:
          type: string
`

const testFunctionYAML = `
group: functions
type: js_function
entries:
  - name: Process
    desc: Execute a Yao process
    args:
      - name: name
        type: string
        required: true
      - name: args
        type: any
    return:
      type: any
`

const testWildcardYAML = `
group: models
type: process
entries:
  - name: find
    desc: Find a record by ID
    args:
      - name: id
        type: any
        required: true
    return:
      type: object
`

const testUnionYAML = `
group: stores
type: process
entries:
  - name: get
    desc: Get value by key
    args:
      - name: key
        type: string
        required: true
    return:
      type: union
      desc: Stored value or null
      variants:
        - type: any
          desc: The stored value
        - type: "null"
          desc: Key does not exist
`

func setup(t *testing.T) {
	t.Helper()
	Reset()
	if err := LoadYAML([]byte(testYAML)); err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}
}

func TestLoadYAML(t *testing.T) {
	setup(t)

	all := All()
	if len(all) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(all))
	}

	e, ok := Get(TypeProcess, "test.hello")
	if !ok {
		t.Fatal("test.hello not found")
	}
	if e.Desc != "Say hello" {
		t.Errorf("desc = %q, want %q", e.Desc, "Say hello")
	}
	if e.Group != "test" {
		t.Errorf("group = %q, want %q", e.Group, "test")
	}
	if len(e.Args) != 1 {
		t.Fatalf("args len = %d, want 1", len(e.Args))
	}
	if e.Args[0].Type != "string" {
		t.Errorf("arg type = %q, want %q", e.Args[0].Type, "string")
	}
	if e.Return == nil || e.Return.Type != "string" {
		t.Error("return type should be string")
	}
}

func TestLoadYAML_GroupTypeInheritance(t *testing.T) {
	Reset()
	yaml := `
group: mygroup
type: process
entries:
  - name: mygroup.action
    desc: Some action
`
	if err := LoadYAML([]byte(yaml)); err != nil {
		t.Fatal(err)
	}
	e, ok := Get(TypeProcess, "mygroup.action")
	if !ok {
		t.Fatal("entry not found")
	}
	if e.Group != "mygroup" {
		t.Errorf("group = %q, want %q", e.Group, "mygroup")
	}
	if e.Type != TypeProcess {
		t.Errorf("type = %q, want %q", e.Type, TypeProcess)
	}
}

func TestLoadYAML_InvalidYAML(t *testing.T) {
	Reset()
	err := LoadYAML([]byte("{{invalid yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestRegister(t *testing.T) {
	Reset()
	Register(&Entry{
		Name:  "manual.entry",
		Type:  TypeProcess,
		Group: "manual",
		Desc:  "Manual registration",
	})

	e, ok := Get(TypeProcess, "manual.entry")
	if !ok {
		t.Fatal("manual.entry not found")
	}
	if e.Desc != "Manual registration" {
		t.Errorf("desc = %q", e.Desc)
	}
}

func TestGet_CaseInsensitive(t *testing.T) {
	setup(t)

	e, ok := Get(TypeProcess, "TEST.HELLO")
	if !ok {
		t.Fatal("case-insensitive Get failed")
	}
	if e.Name != "test.hello" {
		t.Errorf("name = %q", e.Name)
	}
}

func TestList(t *testing.T) {
	setup(t)

	entries := List(TypeProcess)
	if len(entries) != 3 {
		t.Fatalf("List returned %d entries, want 3", len(entries))
	}

	filtered := List(TypeProcess, ListOption{Group: "test"})
	if len(filtered) != 3 {
		t.Errorf("group filter returned %d, want 3", len(filtered))
	}

	filtered = List(TypeProcess, ListOption{Group: "nonexistent"})
	if len(filtered) != 0 {
		t.Errorf("nonexistent group returned %d, want 0", len(filtered))
	}
}

func TestList_Search(t *testing.T) {
	setup(t)

	results := List(TypeProcess, ListOption{Search: "hello"})
	if len(results) != 1 {
		t.Fatalf("search 'hello' returned %d, want 1", len(results))
	}
	if results[0].Name != "test.hello" {
		t.Errorf("name = %q", results[0].Name)
	}
}

func TestList_Sorted(t *testing.T) {
	setup(t)

	entries := List(TypeProcess)
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Name > entries[i].Name {
			t.Errorf("not sorted: %s > %s", entries[i-1].Name, entries[i].Name)
		}
	}
}

func TestValidate_Found(t *testing.T) {
	setup(t)

	r := Validate(TypeProcess, "test.hello")
	if !r.Valid {
		t.Fatal("expected valid")
	}
	if r.Status != "ok" {
		t.Errorf("status = %q", r.Status)
	}
	if r.Entry == nil {
		t.Fatal("expected entry")
	}
}

func TestValidate_NotFound(t *testing.T) {
	setup(t)

	r := Validate(TypeProcess, "nonexistent.process")
	if r.Valid {
		t.Fatal("expected not valid")
	}
	if r.Status != "not_found" {
		t.Errorf("status = %q", r.Status)
	}
}

func TestValidate_FuzzySuggestions(t *testing.T) {
	setup(t)

	r := Validate(TypeProcess, "test.hell")
	if r.Valid {
		t.Fatal("expected not valid")
	}
	if len(r.Suggestion) == 0 {
		t.Error("expected suggestions")
	}
	found := false
	for _, s := range r.Suggestion {
		if s == "test.hello" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected test.hello in suggestions, got %v", r.Suggestion)
	}
}

func TestSearch(t *testing.T) {
	setup(t)

	results := Search("add")
	if len(results) != 1 {
		t.Fatalf("search 'add' returned %d, want 1", len(results))
	}
	if results[0].Name != "test.add" {
		t.Errorf("name = %q", results[0].Name)
	}

	results = Search("zzz")
	if len(results) != 0 {
		t.Errorf("search 'zzz' returned %d, want 0", len(results))
	}
}

func TestGroups(t *testing.T) {
	setup(t)

	groups := Groups(TypeProcess)
	if len(groups) != 1 || groups[0] != "test" {
		t.Errorf("groups = %v, want [test]", groups)
	}
}

func TestStats(t *testing.T) {
	setup(t)

	stats := Stats()
	si, ok := stats[TypeProcess]
	if !ok {
		t.Fatal("no stats for process type")
	}
	if si.Total != 3 {
		t.Errorf("total = %d, want 3", si.Total)
	}
	if si.Documented != 3 {
		t.Errorf("documented = %d, want 3", si.Documented)
	}
}

func TestRuntimeTypes(t *testing.T) {
	Reset()
	if err := LoadYAML([]byte(testRuntimeYAML)); err != nil {
		t.Fatal(err)
	}

	entries := List(TypeJSObject)
	if len(entries) != 1 {
		t.Fatalf("expected 1 js_object, got %d", len(entries))
	}
	if entries[0].Name != "log" {
		t.Errorf("name = %q", entries[0].Name)
	}
	if len(entries[0].Methods) != 2 {
		t.Errorf("methods = %d, want 2", len(entries[0].Methods))
	}
}

func TestClassType(t *testing.T) {
	Reset()
	if err := LoadYAML([]byte(testClassYAML)); err != nil {
		t.Fatal(err)
	}

	e, ok := Get(TypeJSClass, "FS")
	if !ok {
		t.Fatal("FS not found")
	}
	if len(e.Args) != 1 {
		t.Errorf("constructor args = %d", len(e.Args))
	}
	if len(e.Methods) != 1 {
		t.Errorf("methods = %d", len(e.Methods))
	}
	if e.Methods[0].Name != "ReadFile" {
		t.Errorf("method name = %q", e.Methods[0].Name)
	}
}

func TestFunctionType(t *testing.T) {
	Reset()
	if err := LoadYAML([]byte(testFunctionYAML)); err != nil {
		t.Fatal(err)
	}

	entries := List(TypeJSFunction)
	if len(entries) != 1 {
		t.Fatalf("expected 1, got %d", len(entries))
	}
	if entries[0].Name != "Process" {
		t.Errorf("name = %q", entries[0].Name)
	}
}

func TestUnionReturn(t *testing.T) {
	Reset()
	if err := LoadYAML([]byte(testUnionYAML)); err != nil {
		t.Fatal(err)
	}

	e, ok := Get(TypeProcess, "stores.get")
	if !ok {
		t.Fatal("stores.get not found")
	}
	if e.Return == nil {
		t.Fatal("expected return")
	}
	if e.Return.Type != "union" {
		t.Errorf("return type = %q, want union", e.Return.Type)
	}
	if len(e.Return.Variants) != 2 {
		t.Errorf("variants = %d, want 2", len(e.Return.Variants))
	}
}

func TestObjectFields(t *testing.T) {
	setup(t)

	e, ok := Get(TypeProcess, "test.info")
	if !ok {
		t.Fatal("test.info not found")
	}
	if e.Return == nil {
		t.Fatal("expected return")
	}
	if e.Return.Type != "object" {
		t.Errorf("return type = %q", e.Return.Type)
	}
	if len(e.Return.Fields) != 2 {
		t.Errorf("fields = %d, want 2", len(e.Return.Fields))
	}
}

func TestMultipleLoadYAML(t *testing.T) {
	Reset()
	if err := LoadYAML([]byte(testYAML)); err != nil {
		t.Fatal(err)
	}
	if err := LoadYAML([]byte(testRuntimeYAML)); err != nil {
		t.Fatal(err)
	}
	if err := LoadYAML([]byte(testFunctionYAML)); err != nil {
		t.Fatal(err)
	}

	all := All()
	if len(all) != 5 {
		t.Errorf("expected 5 entries total, got %d", len(all))
	}

	stats := Stats()
	if _, ok := stats[TypeProcess]; !ok {
		t.Error("missing process stats")
	}
	if _, ok := stats[TypeJSObject]; !ok {
		t.Error("missing js_object stats")
	}
	if _, ok := stats[TypeJSFunction]; !ok {
		t.Error("missing js_function stats")
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"kitten", "sitting", 3},
		{"hello", "hello", 0},
	}
	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestCallableName(t *testing.T) {
	tests := []struct {
		entry *Entry
		want  string
	}{
		{&Entry{Name: "models.find", Type: TypeProcess, Group: "models"}, "models.<id>.find"},
		{&Entry{Name: "stores.get", Type: TypeProcess, Group: "stores"}, "stores.<id>.get"},
		{&Entry{Name: "http.get", Type: TypeProcess, Group: "http"}, "http.get"},
		{&Entry{Name: "encoding.base64.Encode", Type: TypeProcess, Group: "encoding"}, "encoding.base64.Encode"},
		{&Entry{Name: "console", Type: TypeJSObject, Group: "global"}, "console"},
	}
	for _, tt := range tests {
		got := CallableName(tt.entry)
		if got != tt.want {
			t.Errorf("CallableName(%q, group=%q) = %q, want %q", tt.entry.Name, tt.entry.Group, got, tt.want)
		}
	}
}

func TestCallableToHandler(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"models.user.find", "models.find"},
		{"models.user.pet.Find", "models.find"},
		{"stores.cache.get", "stores.get"},
		{"http.get", ""},
		{"encoding.base64.Encode", ""},
	}
	for _, tt := range tests {
		got := callableToHandler(tt.name)
		if got != tt.want {
			t.Errorf("callableToHandler(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestGetByCallableName(t *testing.T) {
	setup(t)
	Register(&Entry{Name: "models.find", Type: TypeProcess, Group: "models", Desc: "Find a record"})
	e, ok := Get(TypeProcess, "models.user.find")
	if !ok {
		t.Fatal("expected Get to resolve models.user.find → models.find")
	}
	if e.Name != "models.find" {
		t.Errorf("expected entry name models.find, got %s", e.Name)
	}
}

func TestReset(t *testing.T) {
	setup(t)
	if len(All()) == 0 {
		t.Fatal("expected entries before reset")
	}
	Reset()
	if len(All()) != 0 {
		t.Fatal("expected 0 entries after reset")
	}
}
