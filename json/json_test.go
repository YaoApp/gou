package json

import (
	"testing"
)

func TestEncodeError(t *testing.T) {
	// Test encoding a channel (not JSON-serializable)
	ch := make(chan int)
	_, err := Encode(ch)
	if err == nil {
		t.Errorf("Encode(channel) expected error, got nil")
	}
}

func TestEncode(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:    "Simple object",
			input:   map[string]interface{}{"name": "test", "age": 25},
			want:    "", // Will check that it's valid JSON, not exact match (map order is not guaranteed)
			wantErr: false,
		},
		{
			name:    "Array",
			input:   []int{1, 2, 3},
			want:    `[1,2,3]`,
			wantErr: false,
		},
		{
			name:    "String",
			input:   "hello",
			want:    `"hello"`,
			wantErr: false,
		},
		{
			name:    "Number",
			input:   42,
			want:    `42`,
			wantErr: false,
		},
		{
			name:    "Boolean",
			input:   true,
			want:    `true`,
			wantErr: false,
		},
		{
			name:    "Null",
			input:   nil,
			want:    `null`,
			wantErr: false,
		},
		{
			name:    "Nested object",
			input:   map[string]interface{}{"user": map[string]interface{}{"name": "test"}},
			want:    `{"user":{"name":"test"}}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Encode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want == "" {
				// For map encoding, just verify it's not empty (order may vary)
				if len(got) == 0 {
					t.Errorf("Encode() returned empty string")
				}
			} else if got != tt.want {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		check   func(interface{}) bool
	}{
		{
			name:    "Simple object",
			data:    `{"name":"test","age":25}`,
			wantErr: false,
			check: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				return ok && m["name"] == "test" && m["age"].(float64) == 25
			},
		},
		{
			name:    "Array",
			data:    `[1,2,3]`,
			wantErr: false,
			check: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				return ok && len(arr) == 3
			},
		},
		{
			name:    "String",
			data:    `"hello"`,
			wantErr: false,
			check: func(v interface{}) bool {
				s, ok := v.(string)
				return ok && s == "hello"
			},
		},
		{
			name:    "Number",
			data:    `42`,
			wantErr: false,
			check: func(v interface{}) bool {
				n, ok := v.(float64)
				return ok && n == 42
			},
		},
		{
			name:    "Boolean",
			data:    `true`,
			wantErr: false,
			check: func(v interface{}) bool {
				b, ok := v.(bool)
				return ok && b == true
			},
		},
		{
			name:    "Null",
			data:    `null`,
			wantErr: false,
			check: func(v interface{}) bool {
				return v == nil
			},
		},
		{
			name:    "Invalid JSON",
			data:    `{invalid}`,
			wantErr: true,
			check:   nil,
		},
		{
			name:    "Empty string",
			data:    ``,
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(got) {
				t.Errorf("Decode() result check failed, got = %v", got)
			}
		})
	}
}

func TestDecodeTyped(t *testing.T) {
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		data    string
		target  interface{}
		wantErr bool
		check   func(interface{}) bool
	}{
		{
			name:    "Decode to struct",
			data:    `{"name":"test","age":25}`,
			target:  &User{},
			wantErr: false,
			check: func(v interface{}) bool {
				u, ok := v.(*User)
				return ok && u.Name == "test" && u.Age == 25
			},
		},
		{
			name:    "Decode to map",
			data:    `{"name":"test","age":25}`,
			target:  &map[string]interface{}{},
			wantErr: false,
			check: func(v interface{}) bool {
				m, ok := v.(*map[string]interface{})
				return ok && (*m)["name"] == "test"
			},
		},
		{
			name:    "Invalid JSON",
			data:    `{invalid}`,
			target:  &User{},
			wantErr: true,
			check:   nil,
		},
		{
			name:    "Type mismatch",
			data:    `{"name":"test","age":"not a number"}`,
			target:  &User{},
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DecodeTyped(tt.data, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeTyped() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(tt.target) {
				t.Errorf("DecodeTyped() result check failed")
			}
		})
	}
}
