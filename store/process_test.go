package store

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestStoreProcess(t *testing.T) {
	prepare(t)
	prepareStores(t)
	testStoreProcess(t, "cache")
	testStoreProcess(t, "share")
	testStoreProcess(t, "data")
}

func testStoreProcess(t *testing.T, name string) {

	process.New(fmt.Sprintf("stores.%s.Clear", name)).Run()
	value := process.New(fmt.Sprintf("stores.%s.Len", name)).Run()
	assert.Equal(t, 0, value)

	assert.NotPanics(t, func() {
		process.New(fmt.Sprintf("stores.%s.Set", name), "key1", "foo").Run()
		process.New(fmt.Sprintf("stores.%s.Set", name), "key2", "bar").Run()
		process.New(fmt.Sprintf("stores.%s.Set", name), "key3", 1024).Run()
		process.New(fmt.Sprintf("stores.%s.Set", name), "key4", 0.618).Run()
	})

	value = process.New(fmt.Sprintf("stores.%s.Get", name), "key1").Run()
	assert.Equal(t, "foo", value)

	value = process.New(fmt.Sprintf("stores.%s.GetDel", name), "key2").Run()
	assert.Equal(t, "bar", value)

	value = process.New(fmt.Sprintf("stores.%s.Has", name), "key2").Run()
	assert.False(t, value.(bool))

	value = process.New(fmt.Sprintf("stores.%s.Len", name)).Run()
	assert.Equal(t, 3, value)

	value = process.New(fmt.Sprintf("stores.%s.Keys", name)).Run()

	assert.Contains(t, value, "key3")
	assert.Contains(t, value, "key4")
	assert.Contains(t, value, "key1")
	assert.NotContains(t, value, "key2")
}
