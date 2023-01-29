package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/any"
)

func TestLoad(t *testing.T) {

}

func TestLRU(t *testing.T) {
	lru := newStore(t, nil)
	testBasic(t, lru)
	testMulti(t, lru)
}

func TestRedis(t *testing.T) {
	redis := newStore(t, getConnector(t, "redis"))
	testBasic(t, redis)
	testMulti(t, redis)
}

func TestMongo(t *testing.T) {
	mongo := newStore(t, getConnector(t, "mongo"))
	testBasic(t, mongo)
	testMulti(t, mongo)
}

func testBasic(t *testing.T, kv Store) {

	var err error
	kv.Clear()
	kv.Set("key1", "bar", 0)
	kv.Set("key2", 1024, 0)
	kv.Set("key3", 0.618, 0)
	value, ok := kv.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "bar", value)

	value, ok = kv.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, 1024, any.Of(value).CInt())

	value, ok = kv.Get("key3")
	assert.True(t, ok)
	assert.Equal(t, 0.618, value)

	kv.Set("key1", "foo", 0)
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

	assert.Contains(t, kv.Keys(), "key2")
	assert.Contains(t, kv.Keys(), "key3")
	assert.Equal(t, 2, len(kv.Keys()))

	kv.Clear()
	assert.Equal(t, 0, kv.Len())

	value, err = kv.GetSet("key1", 0, func(key string) (interface{}, error) {
		return "bar", nil
	})
	assert.Nil(t, err)
	value, ok = kv.Get("key1")
	assert.Equal(t, "bar", value)

	value, err = kv.GetSet("key1", 0, func(key string) (interface{}, error) {
		return nil, fmt.Errorf("error test")
	})
	assert.Nil(t, err)
	assert.Equal(t, "bar", value)

	value, err = kv.GetSet("key2", 0, func(key string) (interface{}, error) {
		return nil, fmt.Errorf("error test")
	})
	assert.Equal(t, "error test", err.Error())
	assert.Nil(t, value)

	value, ok = kv.GetDel("key1")
	assert.True(t, ok)
	assert.Equal(t, "bar", value)
	assert.Equal(t, 0, kv.Len())

}

func testMulti(t *testing.T, kv Store) {

	kv.SetMulti(map[string]interface{}{"key1": "foo", "key2": 1024, "key3": 0.618}, 0)
	assert.Equal(t, 3, kv.Len())

	values := kv.GetMulti([]string{"key1", "key2", "key3", "key4"})
	assert.Equal(t, "foo", values["key1"])
	assert.Equal(t, 1024, any.Of(values["key2"]).CInt())
	assert.Equal(t, 0.618, values["key3"])
	assert.Equal(t, nil, values["key4"])

	kv.DelMulti([]string{"key1", "key2", "key3"})
	assert.Equal(t, 0, kv.Len())

	values = kv.GetSetMulti([]string{"key1", "key2", "key3", "key4"}, 0, func(key string) (interface{}, error) {
		return key, nil
	})
	assert.Equal(t, "key1", values["key1"])
	assert.Equal(t, "key2", values["key2"])
	assert.Equal(t, "key3", values["key3"])
	assert.Equal(t, "key4", values["key4"])
	kv.Clear()

	values = kv.GetSetMulti([]string{"key1", "key2", "key3", "key4"}, 0, func(key string) (interface{}, error) {
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

func newStore(t *testing.T, c connector.Connector) Store {
	store, err := New(c, Option{"size": 20480})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func getConnector(t *testing.T, name string) connector.Connector {
	return connector.Connectors[name]
}

func prepareStores(t *testing.T) {
	stores := map[string]string{
		"cache": filepath.Join("stores", "cache.lru.yao"),
		"share": filepath.Join("stores", "share.redis.yao"),
		"data":  filepath.Join("stores", "data.mongo.yao"),
	}
	for id, file := range stores {
		_, err := Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func prepare(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := application.OpenFromDisk(root) // Load app
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	connectors := map[string]string{
		"mysql":  filepath.Join("connectors", "mysql.conn.yao"),
		"mongo":  filepath.Join("connectors", "mongo.conn.yao"),
		"redis":  filepath.Join("connectors", "redis.conn.yao"),
		"sqlite": filepath.Join("connectors", "sqlite.conn.yao"),
	}

	for id, file := range connectors {
		_, err = connector.Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}
}
