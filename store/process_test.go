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

func TestStoreListProcess(t *testing.T) {
	prepare(t)
	prepareStores(t)
	testStoreListProcess(t, "cache")
	testStoreListProcess(t, "share")
	testStoreListProcess(t, "data")
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

func testStoreListProcess(t *testing.T, name string) {

	// Clear and test empty list
	process.New(fmt.Sprintf("stores.%s.Clear", name)).Run()

	// Test Push
	assert.NotPanics(t, func() {
		process.New(fmt.Sprintf("stores.%s.Push", name), "mylist", "item1", "item2", "item3").Run()
	})

	// Test ArrayLen
	length := process.New(fmt.Sprintf("stores.%s.ArrayLen", name), "mylist").Run()
	assert.Equal(t, 3, length)

	// Test ArrayGet
	value := process.New(fmt.Sprintf("stores.%s.ArrayGet", name), "mylist", 0).Run()
	assert.Equal(t, "item1", value)

	value = process.New(fmt.Sprintf("stores.%s.ArrayGet", name), "mylist", 2).Run()
	assert.Equal(t, "item3", value)

	// Test ArrayAll
	allItems := process.New(fmt.Sprintf("stores.%s.ArrayAll", name), "mylist").Run()
	assert.Equal(t, []interface{}{"item1", "item2", "item3"}, allItems)

	// Test ArraySet
	assert.NotPanics(t, func() {
		process.New(fmt.Sprintf("stores.%s.ArraySet", name), "mylist", 1, "modified").Run()
	})

	value = process.New(fmt.Sprintf("stores.%s.ArrayGet", name), "mylist", 1).Run()
	assert.Equal(t, "modified", value)

	// Test Pop
	value = process.New(fmt.Sprintf("stores.%s.Pop", name), "mylist", 1).Run()
	assert.Equal(t, "item3", value) // Pop from end

	length = process.New(fmt.Sprintf("stores.%s.ArrayLen", name), "mylist").Run()
	assert.Equal(t, 2, length)

	// Test Pull
	assert.NotPanics(t, func() {
		process.New(fmt.Sprintf("stores.%s.Pull", name), "mylist", "modified").Run()
	})

	length = process.New(fmt.Sprintf("stores.%s.ArrayLen", name), "mylist").Run()
	assert.Equal(t, 1, length)

	// Test AddToSet
	assert.NotPanics(t, func() {
		process.New(fmt.Sprintf("stores.%s.AddToSet", name), "uniquelist", "apple", "banana", "apple").Run()
	})

	length = process.New(fmt.Sprintf("stores.%s.ArrayLen", name), "uniquelist").Run()
	assert.Equal(t, 2, length) // Only unique items

	// Test ArraySlice
	assert.NotPanics(t, func() {
		process.New(fmt.Sprintf("stores.%s.Push", name), "biglist", "a", "b", "c", "d", "e").Run()
	})

	slice := process.New(fmt.Sprintf("stores.%s.ArraySlice", name), "biglist", 1, 3).Run()
	assert.Equal(t, []interface{}{"b", "c", "d"}, slice)

	// Test ArrayPage
	page1 := process.New(fmt.Sprintf("stores.%s.ArrayPage", name), "biglist", 1, 2).Run()
	assert.Equal(t, []interface{}{"a", "b"}, page1)

	page2 := process.New(fmt.Sprintf("stores.%s.ArrayPage", name), "biglist", 2, 2).Run()
	assert.Equal(t, []interface{}{"c", "d"}, page2)

	// Test PullAll
	assert.NotPanics(t, func() {
		process.New(fmt.Sprintf("stores.%s.PullAll", name), "biglist", "a", "c", "e").Run()
	})

	remaining := process.New(fmt.Sprintf("stores.%s.ArrayAll", name), "biglist").Run()
	assert.Equal(t, []interface{}{"b", "d"}, remaining)
}
