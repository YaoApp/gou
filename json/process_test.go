package json

import (
	"testing"

	"github.com/yaoapp/gou/process"
)

func TestProcessEncodeError(t *testing.T) {
	// Test encoding an un-serializable value
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("ProcessEncode() expected panic for channel, got none")
		}
	}()

	ch := make(chan int)
	p := process.New("json.encode", ch)
	ProcessEncode(p)
}

func TestProcessEncode(t *testing.T) {
	tests := []struct {
		name      string
		args      []interface{}
		want      string
		wantPanic bool
	}{
		{
			name: "Encode object",
			args: []interface{}{
				map[string]interface{}{"name": "test", "age": 25},
			},
			want:      "", // Map order not guaranteed, just check it's not empty
			wantPanic: false,
		},
		{
			name: "Encode array",
			args: []interface{}{
				[]int{1, 2, 3},
			},
			want:      `[1,2,3]`,
			wantPanic: false,
		},
		{
			name: "Encode string",
			args: []interface{}{
				"hello",
			},
			want:      `"hello"`,
			wantPanic: false,
		},
		{
			name:      "No arguments",
			args:      []interface{}{},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("ProcessEncode() panic = %v, wantPanic %v", r, tt.wantPanic)
				}
			}()

			p := process.New("json.encode", tt.args...)
			result := ProcessEncode(p)
			if !tt.wantPanic {
				resultStr, ok := result.(string)
				if !ok {
					t.Errorf("ProcessEncode() result is not a string")
					return
				}
				if tt.want == "" {
					// For map encoding, just check it's not empty
					if len(resultStr) == 0 {
						t.Errorf("ProcessEncode() returned empty string")
					}
				} else if resultStr != tt.want {
					t.Errorf("ProcessEncode() = %v, want %v", resultStr, tt.want)
				}
			}
		})
	}
}

func TestProcessDecode(t *testing.T) {
	tests := []struct {
		name      string
		args      []interface{}
		wantPanic bool
		check     func(interface{}) bool
	}{
		{
			name: "Decode object",
			args: []interface{}{`{"name":"test","age":25}`},
			check: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				return ok && m["name"] == "test"
			},
			wantPanic: false,
		},
		{
			name: "Decode array",
			args: []interface{}{`[1,2,3]`},
			check: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				return ok && len(arr) == 3
			},
			wantPanic: false,
		},
		{
			name:      "Invalid JSON",
			args:      []interface{}{`{invalid}`},
			wantPanic: true,
			check:     nil,
		},
		{
			name:      "No arguments",
			args:      []interface{}{},
			wantPanic: true,
			check:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("ProcessDecode() panic = %v, wantPanic %v", r, tt.wantPanic)
				}
			}()

			p := process.New("json.decode", tt.args...)
			result := ProcessDecode(p)
			if !tt.wantPanic && tt.check != nil && !tt.check(result) {
				t.Errorf("ProcessDecode() result check failed")
			}
		})
	}
}

func TestProcessParse(t *testing.T) {
	tests := []struct {
		name      string
		args      []interface{}
		wantPanic bool
		check     func(interface{}) bool
	}{
		{
			name: "Parse JSON",
			args: []interface{}{`{"name":"test"}`},
			check: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				return ok && m["name"] == "test"
			},
			wantPanic: false,
		},
		{
			name: "Parse JSONC",
			args: []interface{}{`{"name":"test"} // comment`},
			check: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				return ok && m["name"] == "test"
			},
			wantPanic: false,
		},
		{
			name: "Parse YAML with hint",
			args: []interface{}{`name: test
age: 25`, ".yaml"},
			check: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				return ok && m["name"] == "test"
			},
			wantPanic: false,
		},
		{
			name: "Parse YAML with auto-detect",
			args: []interface{}{`name: test
age: 25`},
			check: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				return ok && m["name"] == "test"
			},
			wantPanic: false,
		},
		{
			name:      "Parse broken JSON (auto-repair)",
			args:      []interface{}{`{"name":"test"`},
			wantPanic: false,
			check: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				return ok && m["name"] == "test"
			},
		},
		{
			name: "Parse Yao file with hint",
			args: []interface{}{`{
  // comment
  "name": "test"
}`, "config.yao"},
			check: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				return ok && m["name"] == "test"
			},
			wantPanic: false,
		},
		{
			name:      "No arguments",
			args:      []interface{}{},
			wantPanic: true,
			check:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("ProcessParse() panic = %v, wantPanic %v", r, tt.wantPanic)
				}
			}()

			p := process.New("json.parse", tt.args...)
			result := ProcessParse(p)
			if !tt.wantPanic && tt.check != nil && !tt.check(result) {
				t.Errorf("ProcessParse() result check failed, got = %v", result)
			}
		})
	}
}

func TestProcessRepair(t *testing.T) {
	tests := []struct {
		name      string
		args      []interface{}
		wantPanic bool
		check     func(string) bool
	}{
		{
			name: "Repair missing brace",
			args: []interface{}{`{"name":"test"`},
			check: func(s string) bool {
				return len(s) > 0 && s != `{"name":"test"`
			},
			wantPanic: false,
		},
		{
			name: "Repair trailing comma",
			args: []interface{}{`{"name":"test",}`},
			check: func(s string) bool {
				return len(s) > 0
			},
			wantPanic: false,
		},
		{
			name:      "No arguments",
			args:      []interface{}{},
			wantPanic: true,
			check:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("ProcessRepair() panic = %v, wantPanic %v", r, tt.wantPanic)
				}
			}()

			p := process.New("json.repair", tt.args...)
			result := ProcessRepair(p)
			if !tt.wantPanic {
				resultStr, ok := result.(string)
				if !ok {
					t.Errorf("ProcessRepair() result is not a string")
					return
				}
				if tt.check != nil && !tt.check(resultStr) {
					t.Errorf("ProcessRepair() result check failed, got = %v", resultStr)
				}
			}
		})
	}
}

func TestProcessValidate(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "number"},
		},
		"required": []interface{}{"name"},
	}

	tests := []struct {
		name      string
		args      []interface{}
		wantError bool
		wantPanic bool
	}{
		{
			name: "Valid data",
			args: []interface{}{
				map[string]interface{}{"name": "test", "age": 25},
				schema,
			},
			wantError: false,
			wantPanic: false,
		},
		{
			name: "Invalid data - missing required field",
			args: []interface{}{
				map[string]interface{}{"age": 25},
				schema,
			},
			wantError: true,
			wantPanic: false,
		},
		{
			name: "Invalid data - wrong type",
			args: []interface{}{
				map[string]interface{}{"name": 123, "age": 25},
				schema,
			},
			wantError: true,
			wantPanic: false,
		},
		{
			name: "Valid data - only required fields",
			args: []interface{}{
				map[string]interface{}{"name": "test"},
				schema,
			},
			wantError: false,
			wantPanic: false,
		},
		{
			name: "Invalid schema",
			args: []interface{}{
				map[string]interface{}{"name": "test"},
				map[string]interface{}{"type": "invalidtype"},
			},
			wantError: true,
			wantPanic: false,
		},
		{
			name:      "Not enough arguments",
			args:      []interface{}{map[string]interface{}{"name": "test"}},
			wantPanic: true,
		},
		{
			name:      "No arguments",
			args:      []interface{}{},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("ProcessValidate() panic = %v, wantPanic %v", r, tt.wantPanic)
				}
			}()

			p := process.New("json.validate", tt.args...)
			result := ProcessValidate(p)
			if !tt.wantPanic {
				if tt.wantError && result == nil {
					t.Errorf("ProcessValidate() expected error, got nil")
				}
				if !tt.wantError && result != nil {
					t.Errorf("ProcessValidate() expected nil, got %v", result)
				}
			}
		})
	}
}

func TestProcessValidateSchema(t *testing.T) {
	tests := []struct {
		name      string
		args      []interface{}
		wantError bool
		wantPanic bool
	}{
		{
			name: "Valid schema",
			args: []interface{}{
				map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string"},
					},
				},
			},
			wantError: false,
			wantPanic: false,
		},
		{
			name: "Invalid schema - properties not object",
			args: []interface{}{
				map[string]interface{}{
					"type":       "object",
					"properties": "invalid",
				},
			},
			wantError: true,
			wantPanic: false,
		},
		{
			name: "Valid schema - array",
			args: []interface{}{
				map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "number",
					},
				},
			},
			wantError: false,
			wantPanic: false,
		},
		{
			name:      "No arguments",
			args:      []interface{}{},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("ProcessValidateSchema() panic = %v, wantPanic %v", r, tt.wantPanic)
				}
			}()

			p := process.New("json.validateschema", tt.args...)
			result := ProcessValidateSchema(p)
			if !tt.wantPanic {
				if tt.wantError && result == nil {
					t.Errorf("ProcessValidateSchema() expected error, got nil")
				}
				if !tt.wantError && result != nil {
					t.Errorf("ProcessValidateSchema() expected nil, got %v", result)
				}
			}
		})
	}
}
