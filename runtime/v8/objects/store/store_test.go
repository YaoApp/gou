package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/any"
	"rogchap.com/v8go"
)

func TestStoreObjectLRU(t *testing.T) {
	lru := newStore(t, nil)
	testStoreObject(t, lru)
}

func TestStoreObjectRedis(t *testing.T) {
	redis := newStore(t, makeConnector(t, "redis"))
	testStoreObject(t, redis)
}

func TestStoreObjectMongo(t *testing.T) {
	mongo := newStore(t, makeConnector(t, "mongo"))
	testStoreObject(t, mongo)
}

func testStoreObject(t *testing.T, c store.Store) {

	store.Pools["basic"] = c
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	store := &Store{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("Store", store.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	v, err := ctx.RunScript(`
	function basic() {
		var basic = new Store("basic")
		basic.Set("key1", "bar")
		basic.Set("key2", 1024)
		basic.Set("key3", 0.618)
		basic.Set("key4", {"foo":"bar", "int":1024, "float":0.618})
		basic.Set("key5", [0,1,{"foo":"bar"}, [11,12,13],"hello"])
		basic.GetSet("key6", function(key){
			return key + " value"
		})
		return {
			"has": {
				"key1":basic.Has("key1"),
				"key2":basic.Has("key2"),
				"key3":basic.Has("key3"),
				"key4":basic.Has("key4"),
				"key5":basic.Has("key5"),
				"key6":basic.Has("key6"),
				"key7":basic.Has("key7"),
			},
			"key1": basic.Get("key1"),
			"key2": basic.Get("key2"),
			"key3": basic.Get("key3"),
			"key4": basic.Get("key4"),
			"key5": basic.Get("key5"),
			"key6": basic.Get("key6"),
			"key7": basic.Get("key7"),
		}
	}
	basic()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	value, err := bridge.GoValue(v)
	if err != nil {
		t.Fatal(err)
	}

	res := any.Of(value).Map().MapStrAny
	flat := res.Dot()
	assert.Equal(t, "bar", flat.Get("key1"))
	assert.Equal(t, float64(1024), flat.Get("key2"))
	assert.Equal(t, float64(0.618), flat.Get("key3"))
	assert.Equal(t, "bar", flat.Get("key4.foo"))
	assert.Equal(t, float64(1024), flat.Get("key4.int"))
	assert.Equal(t, float64(0.618), flat.Get("key4.float"))
	assert.Equal(t, float64(0), flat.Get("key5.0"))
	assert.Equal(t, float64(1), flat.Get("key5.1"))
	assert.Equal(t, "bar", flat.Get("key5.2.foo"))
	assert.Equal(t, float64(11), flat.Get("key5.3.0"))
	assert.Equal(t, float64(12), flat.Get("key5.3.1"))
	assert.Equal(t, float64(13), flat.Get("key5.3.2"))
	assert.Equal(t, "hello", flat.Get("key5.4"))
	assert.Equal(t, "key6 value", flat.Get("key6"))
	assert.True(t, flat.Get("has.key1").(bool))
	assert.True(t, flat.Get("has.key2").(bool))
	assert.True(t, flat.Get("has.key3").(bool))
	assert.True(t, flat.Get("has.key4").(bool))
	assert.True(t, flat.Get("has.key5").(bool))
	assert.True(t, flat.Get("has.key6").(bool))
	assert.False(t, flat.Get("has.key7").(bool))

	// Del
	v, err = ctx.RunScript(`
	function del(){
		var basic = new Store("basic")
		basic.Del("key1")
		return {
			"has": basic.Has("key1"),
			"key2": basic.Get("key2"),
		}
	}
	del()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	value, err = bridge.GoValue(v)
	if err != nil {
		t.Fatal(err)
	}

	res = any.Of(value).Map().MapStrAny
	flat = res.Dot()
	assert.False(t, flat.Get("has").(bool))
	assert.Equal(t, float64(1024), flat.Get("key2"))

	// GetDel
	v, err = ctx.RunScript(`
	function getDel(){
		var basic = new Store("basic")
		var value = basic.getDel("key2")
		return {
			"has": basic.Has("key2"),
			"key2": value,
		}
	}
	del()
	`, "")

	value, err = bridge.GoValue(v)
	if err != nil {
		t.Fatal(err)
	}

	res = any.Of(value).Map().MapStrAny
	flat = res.Dot()
	assert.False(t, flat.Get("has").(bool))
	assert.Equal(t, float64(1024), flat.Get("key2"))

	// Keys
	v, err = ctx.RunScript(`
	function keys(){
		var basic = new Store("basic")
		return basic.Keys()
	}
	keys()
	`, "")

	value, err = bridge.GoValue(v)
	if err != nil {
		t.Fatal(err)
	}
	keys := any.Of(value).CStrings()

	assert.Contains(t, keys, "key3")
	assert.Contains(t, keys, "key4")
	assert.Contains(t, keys, "key5")
	assert.Contains(t, keys, "key6")
	assert.Contains(t, keys, "key2")
	assert.Equal(t, 5, len(keys))

	// Len
	v, err = ctx.RunScript(`
		function len(){
			var basic = new Store("basic")
			return basic.Len()
		}
		len()
		`, "")

	value, err = bridge.GoValue(v)
	if err != nil {
		t.Fatal(err)
	}
	len := any.Of(value).CInt()
	assert.Equal(t, 5, len)

	// Clear
	v, err = ctx.RunScript(`
		function clear(){
			var basic = new Store("basic")
			basic.Clear()
			return {
				"len": basic.Len(),
				"keys": basic.Keys()
			}
		}
		clear()
		`, "")

	value, err = bridge.GoValue(v)
	if err != nil {
		t.Fatal(err)
	}

	value, err = bridge.GoValue(v)
	if err != nil {
		t.Fatal(err)
	}

	res = any.Of(value).Map().MapStrAny
	flat = res.Dot()
	assert.Equal(t, float64(0), res.Get("len"))
	assert.Equal(t, []interface{}{}, res.Get("keys"))
}

func newStore(t *testing.T, c connector.Connector) store.Store {
	s, err := store.New(c, store.Option{"size": 20480})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func makeConnector(t *testing.T, id string) connector.Connector {

	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := application.OpenFromDisk(root) // Load app
	if err != nil {
		t.Fatal(err)
	}

	application.Load(app)
	file := filepath.Join("connectors", fmt.Sprintf("%s.conn.yao", id))
	_, err = connector.Load(file, id)
	if err != nil {
		t.Fatal(err)
	}
	return connector.Connectors[id]
}
