package dsl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDSLWriteFile(t *testing.T) {
	data := []byte(`{"foo": "bar", "hello":{ "int": 1, "float": 0.618}}`)
	shoud := []byte(`{
  "foo": "bar",
  "hello": {
    "int": 1,
    "float": 0.618
  }
}`)

	file := "test.json"
	fs := New(filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data"))
	size, err := fs.WriteFile(file, data, 0644)
	assert.Nil(t, err)
	assert.Equal(t, 69, size)

	new, err := fs.ReadFile(file)
	assert.Nil(t, err)
	assert.Equal(t, shoud, new)

	data = []byte(`{"foo": "bar", "hello":{ "int": 1, "float": 0.618`)
	size, err = fs.WriteFile(file, data, 0644)
	assert.NotNil(t, err)
	assert.Equal(t, 0, size)
}
