package helper

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToString(t *testing.T) {
	t.Run("Basic types", func(t *testing.T) {
		// Test integers
		result := ToString(42)
		assert.Equal(t, "42", result)

		// Test float
		result = ToString(3.14)
		assert.Equal(t, "3.14", result)

		// Test boolean
		result = ToString(true)
		assert.Equal(t, "true", result)
	})

	t.Run("String and bytes", func(t *testing.T) {
		// Test string
		result := ToString("hello")
		assert.Equal(t, "hello", result)

		// Test []byte
		result = ToString([]byte("world"))
		assert.Equal(t, "world", result)
	})

	t.Run("Error type", func(t *testing.T) {
		// Test error
		err := errors.New("test error")
		result := ToString(err)
		assert.Equal(t, "test error", result)
	})

	t.Run("JSON object", func(t *testing.T) {
		// Test JSON object
		obj := map[string]interface{}{
			"foo": "bar",
			"num": 123,
			"nested": map[string]interface{}{
				"key": "value",
			},
		}
		result := ToString(obj)
		assert.Contains(t, result, `"foo": "bar"`)
		assert.Contains(t, result, `"num": 123`)
		assert.Contains(t, result, `"nested": {`)
		assert.Contains(t, result, `"key": "value"`)
	})

	t.Run("Multiple values", func(t *testing.T) {
		// Test multiple values
		result := ToString("first", 123, true)
		lines := strings.Split(result, "\n")
		assert.Equal(t, 3, len(lines))
		assert.Equal(t, "first", lines[0])
		assert.Equal(t, "123", lines[1])
		assert.Equal(t, "true", lines[2])
	})

	t.Run("Array", func(t *testing.T) {
		// Test array
		arr := []interface{}{"one", 2, true}
		result := ToString(arr)
		assert.Contains(t, result, `"one"`)
		assert.Contains(t, result, `2`)
		assert.Contains(t, result, `true`)
	})
}
