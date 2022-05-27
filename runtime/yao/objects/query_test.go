package objects

import (
	"os"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/gou/runtime/yao/bridge"
	"github.com/yaoapp/xun/capsule"
	"rogchap.com/v8go"
)

func TestQueryObject(t *testing.T) {

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	query := &QueryOBJ{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("Query", query.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// ===== Get
	v, err := ctx.RunScript(`
	function Get() {
		var query = new Query("query-test")
		var data = query.Get({
			"select": ["id", "name"],
			"from": "user"
		})
		return data
	}
	Get()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.ToInterface(v)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 3, len(res.([]interface{})))

	// ===== Paginate
	v, err = ctx.RunScript(`
	function Paginate() {
		var query = new Query("query-test")
		var data = query.Paginate({
			"select": ["id", "name"],
			"from": "user"
		})
		return data
	}
	Paginate()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.ToInterface(v)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, float64(3), res.(map[string]interface{})["total"])

	// ===== first
	v, err = ctx.RunScript(`
	function First() {
		var query = new Query("query-test")
		var data = query.First({
			"select": ["id", "name"],
			"from": "user"
		})
		return data
	}
	First()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.ToInterface(v)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, float64(2), res.(map[string]interface{})["id"])

	// ===== run
	v, err = ctx.RunScript(`
	function Run() {
		var query = new Query("query-test")
		var data = query.Run({
			"select": ["id", "name"],
			"from": "user",
			"limit": 1
		})
		return data
	}
	Run()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.ToInterface(v)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(res.([]interface{})))
}

func initTestEngine() {

	var TestDriver = os.Getenv("GOU_TEST_DB_DRIVER")
	var TestDSN = os.Getenv("GOU_TEST_DSN")
	var TestAESKey = os.Getenv("GOU_TEST_AES_KEY")

	// Connect DB
	switch TestDriver {
	case "sqlite3":
		capsule.AddConn("primary", "sqlite3", TestDSN).SetAsGlobal()
		break
	default:
		capsule.AddConn("primary", "mysql", TestDSN).SetAsGlobal()
		break
	}

	query.Register("query-test", &gou.Query{
		Query: capsule.Query(),
		GetTableName: func(s string) string {
			return s
		},
		AESKey: TestAESKey,
	})
}
