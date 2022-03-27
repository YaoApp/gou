package gou

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadStore(t *testing.T) {
	global, err := LoadStore("file://"+path.Join(TestStoreRoot, "global.lru.json"), "global")
	if err != nil {
		t.Fatal(err)
	}
	global.Clear()
	assert.Equal(t, 0, global.Len())
	global.Set("key1", "foo", 0)

	lru := SelectStore("global")
	assert.Equal(t, 1, lru.Len())
	v, ok := lru.Get("key1")
	assert.Equal(t, "foo", v)
	assert.True(t, ok)
}

func TestStoreProcess(t *testing.T) {
	LoadStore("file://"+path.Join(TestStoreRoot, "global.lru.json"), "global")

	NewProcess("stores.global.Clear").Run()
	value := NewProcess("stores.global.Len").Run()
	assert.Equal(t, 0, value)

	assert.NotPanics(t, func() {
		NewProcess("stores.global.Set", "key1", "foo").Run()
		NewProcess("stores.global.Set", "key2", "bar").Run()
		NewProcess("stores.global.Set", "key3", 1024).Run()
		NewProcess("stores.global.Set", "key4", 0.618).Run()
	})

	value = NewProcess("stores.global.Get", "key1").Run()
	assert.Equal(t, "foo", value)

	value = NewProcess("stores.global.GetDel", "key2").Run()
	assert.Equal(t, "bar", value)

	value = NewProcess("stores.global.Has", "key2").Run()
	assert.False(t, value.(bool))

	value = NewProcess("stores.global.Len").Run()
	assert.Equal(t, 3, value)

	value = NewProcess("stores.global.Keys").Run()
	assert.Equal(t, []string{"key3", "key4", "key1"}, value)

}
