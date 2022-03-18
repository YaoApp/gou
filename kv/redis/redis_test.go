package redis

import (
	"testing"
	// "github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	// kv, err := New(&Option{})
	// if err != nil {
	// 	t.Fatalf("%s", err.Error())
	// }

	// kv.Set("key1", "bar", 0)
	// kv.Set("key2", 1024, 0)
	// kv.Set("key3", 0.618, 0)
	// value, ok := kv.Get("key1")
	// assert.True(t, ok)
	// assert.Equal(t, "bar", value)

	// value, ok = kv.Get("key2")
	// assert.True(t, ok)
	// assert.Equal(t, 1024, value)

	// value, ok = kv.Get("key3")
	// assert.True(t, ok)
	// assert.Equal(t, 0.618, value)

	// kv.Set("key1", "foo", 0)
	// value, ok = kv.Get("key1")
	// assert.True(t, ok)
	// assert.Equal(t, "foo", value)
	// assert.True(t, kv.Has("key1"))

	// kv.Del("key1")
	// _, ok = kv.Get("key1")
	// assert.False(t, ok)
	// assert.False(t, kv.Has("key1"))

	// assert.Equal(t, 2, kv.Len())
	// assert.False(t, kv.Has("key1"))
	// assert.True(t, kv.Has("key2"))
	// assert.True(t, kv.Has("key3"))

	// assert.Equal(t, []string{"key2", "key3"}, kv.Keys())
	// kv.Clear()
	// assert.Equal(t, 0, kv.Len())

	// value, err = kv.GetSet("key1", func(key string) (interface{}, error) {
	// 	return "bar", nil
	// })
	// assert.Nil(t, err)
	// value, ok = kv.Get("key1")
	// assert.Equal(t, "bar", value)

	// value, err = kv.GetSet("key1", func(key string) (interface{}, error) {
	// 	return nil, fmt.Errorf("error test")
	// })
	// assert.Nil(t, err)
	// assert.Equal(t, "bar", value)

	// value, err = kv.GetSet("key2", func(key string) (interface{}, error) {
	// 	return nil, fmt.Errorf("error test")
	// })
	// assert.Equal(t, "error test", err.Error())
	// assert.Nil(t, value)

	// value, ok = kv.GetDel("key1")
	// assert.True(t, ok)
	// assert.Equal(t, "bar", value)
	// assert.Equal(t, 0, kv.Len())

}
