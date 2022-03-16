package lru

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	kv, err := New(100)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	kv.Set("key1", "bar")
	kv.Set("key2", 1024)
	kv.Set("key3", 0.618)
	value, ok := kv.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "bar", value)

	value, ok = kv.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, 1024, value)

	value, ok = kv.Get("key3")
	assert.True(t, ok)
	assert.Equal(t, 0.618, value)

	kv.Set("key1", "foo")
	value, ok = kv.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "foo", value)
	assert.True(t, kv.Has("key1"))

	kv.Del("key1")
	_, ok = kv.Get("key1")
	assert.False(t, ok)
	assert.False(t, kv.Has("key1"))

	assert.Equal(t, 2, kv.Len())
	assert.False(t, kv.Has("key1"))
	assert.True(t, kv.Has("key2"))
	assert.True(t, kv.Has("key3"))

	assert.Equal(t, []string{"key2", "key3"}, kv.Keys())
	kv.Clear()
	assert.Equal(t, 0, kv.Len())

	value, err = kv.GetSet("key1", func(key string) (interface{}, error) {
		return "bar", nil
	})
	assert.Nil(t, err)
	value, ok = kv.Get("key1")
	assert.Equal(t, "bar", value)

	value, err = kv.GetSet("key1", func(key string) (interface{}, error) {
		return nil, fmt.Errorf("error test")
	})
	assert.Nil(t, err)
	assert.Equal(t, "bar", value)

	value, err = kv.GetSet("key2", func(key string) (interface{}, error) {
		return nil, fmt.Errorf("error test")
	})
	assert.Equal(t, "error test", err.Error())
	assert.Nil(t, value)

	value, ok = kv.GetDel("key1")
	assert.True(t, ok)
	assert.Equal(t, "bar", value)
	assert.Equal(t, 0, kv.Len())

}

func TestMulti(t *testing.T) {
	kv, err := New(100)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	kv.SetMulti(map[string]interface{}{"key1": "foo", "key2": 1024, "key3": 0.618})
	assert.Equal(t, 3, kv.Len())

	values := kv.GetMulti([]string{"key1", "key2", "key3", "key4"})
	assert.Equal(t, "foo", values["key1"])
	assert.Equal(t, 1024, values["key2"])
	assert.Equal(t, 0.618, values["key3"])
	assert.Equal(t, nil, values["key4"])

	kv.DelMulti([]string{"key1", "key2", "key3"})
	assert.Equal(t, 0, kv.Len())

	values = kv.GetSetMulti([]string{"key1", "key2", "key3", "key4"}, func(key string) (interface{}, error) {
		return key, nil
	})
	assert.Equal(t, "key1", values["key1"])
	assert.Equal(t, "key2", values["key2"])
	assert.Equal(t, "key3", values["key3"])
	assert.Equal(t, "key4", values["key4"])
	kv.Clear()

	values = kv.GetSetMulti([]string{"key1", "key2", "key3", "key4"}, func(key string) (interface{}, error) {
		switch key {
		case "key1", "key2":
			return key, nil
		default:
			return nil, fmt.Errorf("error test")
		}
	})

	assert.Equal(t, "key1", values["key1"])
	assert.Equal(t, "key2", values["key2"])
	assert.Equal(t, nil, values["key3"])
	assert.Equal(t, nil, values["key4"])

	kv.DelMulti([]string{"key1", "key2"})
	assert.Equal(t, 0, kv.Len())
}
