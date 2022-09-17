package gou

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/connector"
)

func TestStoreProcess(t *testing.T) {
	loadConnectors(t)
	loadStores(t)
	testStoreProcess(t, "cache")
	testStoreProcess(t, "share")
	testStoreProcess(t, "data")
}

func testStoreProcess(t *testing.T, name string) {

	NewProcess(fmt.Sprintf("stores.%s.Clear", name)).Run()
	value := NewProcess(fmt.Sprintf("stores.%s.Len", name)).Run()
	assert.Equal(t, 0, value)

	assert.NotPanics(t, func() {
		NewProcess(fmt.Sprintf("stores.%s.Set", name), "key1", "foo").Run()
		NewProcess(fmt.Sprintf("stores.%s.Set", name), "key2", "bar").Run()
		NewProcess(fmt.Sprintf("stores.%s.Set", name), "key3", 1024).Run()
		NewProcess(fmt.Sprintf("stores.%s.Set", name), "key4", 0.618).Run()
	})

	value = NewProcess(fmt.Sprintf("stores.%s.Get", name), "key1").Run()
	assert.Equal(t, "foo", value)

	value = NewProcess(fmt.Sprintf("stores.%s.GetDel", name), "key2").Run()
	assert.Equal(t, "bar", value)

	value = NewProcess(fmt.Sprintf("stores.%s.Has", name), "key2").Run()
	assert.False(t, value.(bool))

	value = NewProcess(fmt.Sprintf("stores.%s.Len", name)).Run()
	assert.Equal(t, 3, value)

	value = NewProcess(fmt.Sprintf("stores.%s.Keys", name)).Run()

	assert.Contains(t, value, "key3")
	assert.Contains(t, value, "key4")
	assert.Contains(t, value, "key1")
	assert.NotContains(t, value, "key2")
}

func loadStores(t *testing.T) {
	_, err := LoadStore(source(t, "stores", "cache", "lru"), "cache")
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadStore(source(t, "stores", "data", "mongo"), "data")
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadStore(source(t, "stores", "share", "redis"), "share")
	if err != nil {
		t.Fatal(err)
	}
}

func loadConnectors(t *testing.T) {
	connector.Load(source(t, "connectors", "redis", "conn"), "redis")
	connector.Load(source(t, "connectors", "mongo", "conn"), "mongo")
}

func source(t *testing.T, dir, name, ext string) string {
	root := os.Getenv("GOU_TEST_APP_ROOT")
	path := filepath.Join(root, dir, fmt.Sprintf("%s.%s.json", name, ext))

	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
