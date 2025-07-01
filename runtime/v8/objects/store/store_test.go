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

func TestStoreListObjectLRU(t *testing.T) {
	lru := newStore(t, nil)
	testStoreListObject(t, lru)
}

func TestStoreListObjectRedis(t *testing.T) {
	redis := newStore(t, makeConnector(t, "redis"))
	testStoreListObject(t, redis)
}

func TestStoreListObjectMongo(t *testing.T) {
	mongo := newStore(t, makeConnector(t, "mongo"))
	testStoreListObject(t, mongo)
}

func testStoreObject(t *testing.T, c store.Store) {

	// Use unique name to avoid interference with list tests
	storeName := fmt.Sprintf("basic_%p", c)
	store.Pools[storeName] = c

	// Clear any existing data for persistent stores
	c.Clear()

	iso := v8go.NewIsolate()
	defer iso.Dispose()

	store := &Store{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("Store", store.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	v, err := ctx.RunScript(fmt.Sprintf(`
	function basic() {
		var basic = new Store("%s")`, storeName)+`
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

	value, err := bridge.GoValue(v, ctx)
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
	v, err = ctx.RunScript(fmt.Sprintf(`
	function del(){
		var basic = new Store("%s")`, storeName)+`
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

	value, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	res = any.Of(value).Map().MapStrAny
	flat = res.Dot()
	assert.False(t, flat.Get("has").(bool))
	assert.Equal(t, float64(1024), flat.Get("key2"))

	// GetDel
	v, err = ctx.RunScript(fmt.Sprintf(`
	function getDel(){
		var basic = new Store("%s")`, storeName)+`
		var value = basic.getDel("key2")
		return {
			"has": basic.Has("key2"),
			"key2": value,
		}
	}
	del()
	`, "")

	value, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	res = any.Of(value).Map().MapStrAny
	flat = res.Dot()
	assert.False(t, flat.Get("has").(bool))
	assert.Equal(t, float64(1024), flat.Get("key2"))

	// Keys
	v, err = ctx.RunScript(fmt.Sprintf(`
	function keys(){
		var basic = new Store("%s")`, storeName)+`
		return basic.Keys()
	}
	keys()
	`, "")

	value, err = bridge.GoValue(v, ctx)
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
	v, err = ctx.RunScript(fmt.Sprintf(`
		function len(){
			var basic = new Store("%s")`, storeName)+`
			return basic.Len()
		}
		len()
		`, "")

	value, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	len := any.Of(value).CInt()
	assert.Equal(t, 5, len)

	// Clear
	v, err = ctx.RunScript(fmt.Sprintf(`
		function clear(){
			var basic = new Store("%s")`, storeName)+`
			basic.Clear()
			return {
				"len": basic.Len(),
				"keys": basic.Keys()
			}
		}
		clear()
		`, "")

	value, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	value, err = bridge.GoValue(v, ctx)
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

func testStoreListObject(t *testing.T, c store.Store) {

	// Use unique name to avoid interference with basic tests
	storeName := fmt.Sprintf("listtest_%p", c)
	store.Pools[storeName] = c

	// Clear any existing data for persistent stores
	c.Clear()

	iso := v8go.NewIsolate()
	defer iso.Dispose()

	store := &Store{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("Store", store.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	v, err := ctx.RunScript(fmt.Sprintf(`
	function listOperations() {
		var store = new Store("%s")`, storeName)+`
		
		// Clear any existing data
		store.Clear()
		
		// Test Push
		store.Push("fruits", "apple", "banana", "cherry")
		store.Push("numbers", 1, 2, 3, 4, 5)
		
		// Test ArrayLen
		var fruitsLen = store.ArrayLen("fruits")
		var numbersLen = store.ArrayLen("numbers")
		
		// Test ArrayGet
		var firstFruit = store.ArrayGet("fruits", 0)
		var lastNumber = store.ArrayGet("numbers", 4)
		
		// Test ArrayAll
		var allFruits = store.ArrayAll("fruits")
		var allNumbers = store.ArrayAll("numbers")
		
		// Test ArraySet
		store.ArraySet("fruits", 1, "orange")
		var modifiedFruit = store.ArrayGet("fruits", 1)
		
		// Test Pop
		var poppedFruit = store.Pop("fruits", 1) // Pop from end
		var poppedNumber = store.Pop("numbers", -1) // Pop from beginning
		
		// Test ArraySlice
		store.Push("sequence", "a", "b", "c", "d", "e", "f")
		var slice = store.ArraySlice("sequence", 1, 3) // Skip 1, take 3
		
		// Test ArrayPage
		var page1 = store.ArrayPage("sequence", 1, 2) // Page 1, 2 items per page
		var page2 = store.ArrayPage("sequence", 2, 2) // Page 2, 2 items per page
		
		// Test AddToSet
		store.AddToSet("unique", "x", "y", "z", "x", "y") // Duplicates should be ignored
		var uniqueItems = store.ArrayAll("unique")
		
		// Test Pull
		store.Push("colors", "red", "blue", "red", "green", "red")
		store.Pull("colors", "red") // Remove all "red"
		var colorsAfterPull = store.ArrayAll("colors")
		
		// Test PullAll
		store.Push("mixed", "a", "b", "c", "d", "e", "a", "c")
		store.PullAll("mixed", "a", "c") // Remove all "a" and "c"
		var mixedAfterPullAll = store.ArrayAll("mixed")
		
		return {
			"fruitsLen": fruitsLen,
			"numbersLen": numbersLen,
			"firstFruit": firstFruit,
			"lastNumber": lastNumber,
			"allFruits": allFruits,
			"allNumbers": allNumbers,
			"modifiedFruit": modifiedFruit,
			"poppedFruit": poppedFruit,
			"poppedNumber": poppedNumber,
			"slice": slice,
			"page1": page1,
			"page2": page2,
			"uniqueItems": uniqueItems,
			"colorsAfterPull": colorsAfterPull,
			"mixedAfterPullAll": mixedAfterPullAll,
			"finalFruitsLen": store.ArrayLen("fruits"),
			"finalNumbersLen": store.ArrayLen("numbers")
		}
	}
	listOperations()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	value, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	res := any.Of(value).Map().MapStrAny
	flat := res.Dot()

	// Test basic operations results
	assert.Equal(t, float64(3), flat.Get("fruitsLen"))
	assert.Equal(t, float64(5), flat.Get("numbersLen"))
	assert.Equal(t, "apple", flat.Get("firstFruit"))
	assert.Equal(t, float64(5), flat.Get("lastNumber"))

	// Test ArrayAll results
	assert.Equal(t, []interface{}{"apple", "banana", "cherry"}, flat.Get("allFruits"))
	assert.Equal(t, []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)}, flat.Get("allNumbers"))

	// Test ArraySet result
	assert.Equal(t, "orange", flat.Get("modifiedFruit"))

	// Test Pop results
	assert.Equal(t, "cherry", flat.Get("poppedFruit"))    // Popped from end
	assert.Equal(t, float64(1), flat.Get("poppedNumber")) // Popped from beginning

	// Test ArraySlice result
	assert.Equal(t, []interface{}{"b", "c", "d"}, flat.Get("slice"))

	// Test ArrayPage results
	assert.Equal(t, []interface{}{"a", "b"}, flat.Get("page1"))
	assert.Equal(t, []interface{}{"c", "d"}, flat.Get("page2"))

	// Test AddToSet uniqueness
	uniqueItems := flat.Get("uniqueItems").([]interface{})
	assert.Equal(t, 3, len(uniqueItems))
	assert.Contains(t, uniqueItems, "x")
	assert.Contains(t, uniqueItems, "y")
	assert.Contains(t, uniqueItems, "z")

	// Test Pull result (all "red" removed)
	assert.Equal(t, []interface{}{"blue", "green"}, flat.Get("colorsAfterPull"))

	// Test PullAll result (all "a" and "c" removed)
	assert.Equal(t, []interface{}{"b", "d", "e"}, flat.Get("mixedAfterPullAll"))

	// Test final lengths after Pop operations
	assert.Equal(t, float64(2), flat.Get("finalFruitsLen"))  // 3 - 1 (popped)
	assert.Equal(t, float64(4), flat.Get("finalNumbersLen")) // 5 - 1 (popped)
}
