package json

import (
	"testing"
)

func TestDetectFormatEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected string
	}{
		{
			name:     "Only whitespace",
			data:     "   \n\t  ",
			expected: "",
		},
		{
			name:     "JSON with leading spaces",
			data:     "   {\"name\": \"test\"}",
			expected: "json",
		},
		{
			name:     "Array with leading spaces",
			data:     "\n\n[1, 2, 3]",
			expected: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.data)
			if got != tt.expected {
				t.Errorf("DetectFormat() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected string
	}{
		{
			name:     "JSON object",
			data:     `{"name": "test"}`,
			expected: "json",
		},
		{
			name:     "JSON array",
			data:     `[1, 2, 3]`,
			expected: "json",
		},
		{
			name:     "JSONC with single-line comment",
			data:     `{"name": "test"} // comment`,
			expected: "jsonc",
		},
		{
			name: "JSONC with multi-line comment",
			data: `/* comment */
{"name": "test"}`,
			expected: "jsonc",
		},
		{
			name: "YAML simple",
			data: `name: test
age: 25`,
			expected: "yaml",
		},
		{
			name: "YAML with comment",
			data: `# comment
name: test`,
			expected: "yaml",
		},
		{
			name:     "Empty string",
			data:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.data)
			if got != tt.expected {
				t.Errorf("DetectFormat() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		hint    []string
		wantErr bool
	}{
		{
			name:    "Invalid YAML",
			data:    "name: test\nage: [invalid",
			hint:    []string{".yaml"},
			wantErr: true,
		},
		{
			name:    "Invalid YAML auto-detect",
			data:    "name: test\n  invalid indentation\n age: 25",
			hint:    nil,
			wantErr: true,
		},
		{
			name:    "Completely invalid JSON (unrepairable)",
			data:    "this is not json at all {{[[",
			hint:    []string{".json"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.data, tt.hint...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		hint    []string
		wantErr bool
	}{
		{
			name:    "JSON auto-detect",
			data:    `{"name": "test", "age": 25}`,
			hint:    nil,
			wantErr: false,
		},
		{
			name:    "JSONC auto-detect",
			data:    `{"name": "test"} // comment`,
			hint:    nil,
			wantErr: false,
		},
		{
			name: "YAML auto-detect",
			data: `name: test
age: 25`,
			hint:    nil,
			wantErr: false,
		},
		{
			name: "YAML with hint",
			data: `name: test
age: 25`,
			hint:    []string{".yaml"},
			wantErr: false,
		},
		{
			name:    "Broken JSON auto-repair",
			data:    `{"name": "test", "age": 25`,
			hint:    nil,
			wantErr: false,
		},
		{
			name:    "JSON with hint",
			data:    `{"name": "test"}`,
			hint:    []string{".json"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.data, tt.hint...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRepair(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "Missing closing brace",
			data:    `{"name": "test"`,
			wantErr: false,
		},
		{
			name:    "Missing comma",
			data:    `{"name": "test" "age": 25}`,
			wantErr: false,
		},
		{
			name:    "Trailing comma",
			data:    `{"name": "test",}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repaired, err := Repair(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Repair() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && repaired == "" {
				t.Errorf("Repair() returned empty string")
			}
		})
	}
}

func TestParseTyped(t *testing.T) {
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		data    string
		hint    []string
		target  interface{}
		wantErr bool
		check   func(interface{}) bool
	}{
		{
			name:    "JSON to struct",
			data:    `{"name":"test","age":25}`,
			hint:    nil,
			target:  &User{},
			wantErr: false,
			check: func(v interface{}) bool {
				u, ok := v.(*User)
				return ok && u.Name == "test" && u.Age == 25
			},
		},
		{
			name:    "JSON to struct with hint",
			data:    `{"name":"test","age":25}`,
			hint:    []string{".json"},
			target:  &User{},
			wantErr: false,
			check: func(v interface{}) bool {
				u, ok := v.(*User)
				return ok && u.Name == "test" && u.Age == 25
			},
		},
		{
			name: "JSONC to struct",
			data: `{
				"name": "test", // comment
				"age": 25
			}`,
			hint:    nil,
			target:  &User{},
			wantErr: false,
			check: func(v interface{}) bool {
				u, ok := v.(*User)
				return ok && u.Name == "test" && u.Age == 25
			},
		},
		{
			name: "YAML to struct",
			data: `name: test
age: 25`,
			hint:    []string{".yaml"},
			target:  &User{},
			wantErr: false,
			check: func(v interface{}) bool {
				u, ok := v.(*User)
				return ok && u.Name == "test" && u.Age == 25
			},
		},
		{
			name: "YAML to struct with auto-detect",
			data: `name: test
age: 25`,
			hint:    nil,
			target:  &User{},
			wantErr: false,
			check: func(v interface{}) bool {
				u, ok := v.(*User)
				return ok && u.Name == "test" && u.Age == 25
			},
		},
		{
			name:    "Broken JSON auto-repair",
			data:    `{"name":"test","age":25`,
			hint:    nil,
			target:  &User{},
			wantErr: false,
			check: func(v interface{}) bool {
				u, ok := v.(*User)
				return ok && u.Name == "test" && u.Age == 25
			},
		},
		{
			name:    "Invalid YAML",
			data:    `name: test\nage: [invalid yaml`,
			hint:    []string{".yaml"},
			target:  &User{},
			wantErr: true,
			check:   nil,
		},
		{
			name:    "JSON to map",
			data:    `{"name":"test","age":25}`,
			hint:    nil,
			target:  &map[string]interface{}{},
			wantErr: false,
			check: func(v interface{}) bool {
				m, ok := v.(*map[string]interface{})
				return ok && (*m)["name"] == "test"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseTyped(tt.data, tt.target, tt.hint...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTyped() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(tt.target) {
				t.Errorf("ParseTyped() result check failed")
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	type Config struct {
		Name string `json:"name" yaml:"name"`
		Port int    `json:"port" yaml:"port"`
	}

	tests := []struct {
		name     string
		filename string
		data     []byte
		wantErr  bool
		check    func(interface{}) bool
	}{
		{
			name:     "JSON file",
			filename: "config.json",
			data:     []byte(`{"name":"test","port":8080}`),
			wantErr:  false,
			check: func(v interface{}) bool {
				c, ok := v.(*Config)
				return ok && c.Name == "test" && c.Port == 8080
			},
		},
		{
			name:     "JSONC file",
			filename: "config.jsonc",
			data: []byte(`{
				"name": "test", // service name
				"port": 8080
			}`),
			wantErr: false,
			check: func(v interface{}) bool {
				c, ok := v.(*Config)
				return ok && c.Name == "test" && c.Port == 8080
			},
		},
		{
			name:     "Yao file",
			filename: "config.yao",
			data: []byte(`{
				// Yao config
				"name": "test",
				"port": 8080
			}`),
			wantErr: false,
			check: func(v interface{}) bool {
				c, ok := v.(*Config)
				return ok && c.Name == "test" && c.Port == 8080
			},
		},
		{
			name:     "YAML file",
			filename: "config.yaml",
			data: []byte(`name: test
port: 8080`),
			wantErr: false,
			check: func(v interface{}) bool {
				c, ok := v.(*Config)
				return ok && c.Name == "test" && c.Port == 8080
			},
		},
		{
			name:     "YML file",
			filename: "config.yml",
			data: []byte(`name: test
port: 8080`),
			wantErr: false,
			check: func(v interface{}) bool {
				c, ok := v.(*Config)
				return ok && c.Name == "test" && c.Port == 8080
			},
		},
		{
			name:     "Invalid JSON file",
			filename: "config.json",
			data:     []byte(`{invalid}`),
			wantErr:  true,
			check:    nil,
		},
		{
			name:     "Invalid YAML file",
			filename: "config.yaml",
			data:     []byte(`name: test\nport: [invalid`),
			wantErr:  true,
			check:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := ParseFile(tt.filename, tt.data, &config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(&config) {
				t.Errorf("ParseFile() result check failed")
			}
		})
	}
}
