package query

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/schema"
	"rogchap.com/v8go"
)

func TestQueryObject(t *testing.T) {

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	query := &Object{}
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
			"from": "queryobj_test"
		})
		return data
	}
	Get()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
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
			"from": "queryobj_test"
		})
		return data
	}
	Paginate()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
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
			"from": "queryobj_test"
		})
		return data
	}
	First()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, float64(1), res.(map[string]interface{})["id"])

	// ===== run
	v, err = ctx.RunScript(`
	function Run() {
		var query = new Query("query-test")
		var data = query.Run({
			"select": ["id", "name"],
			"from": "queryobj_test",
			"limit": 1
		})
		return data
	}
	Run()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(res.([]interface{})))
}

func TestQueryLint(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	queryObj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("Query", queryObj.ExportFunction(iso))

	// Register a dummy query engine for Lint test
	if _, has := query.Engines["lint-test"]; !has {
		query.Register("lint-test", &gou.Query{
			Query: nil,
			GetTableName: func(s string) string {
				return s
			},
		})
	}

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// ===== Test valid DSL
	v, err := ctx.RunScript(`
	function LintValid() {
		var q = new Query("lint-test")
		var result = q.Lint('{"select":["id","name"],"from":"users"}')
		return result
	}
	LintValid()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result := res.(map[string]interface{})
	assert.Equal(t, true, result["valid"])
	diagnostics := result["diagnostics"].([]interface{})
	assert.Equal(t, 0, len(diagnostics))
	assert.NotNil(t, result["dsl"])

	// ===== Test invalid DSL - missing select
	v, err = ctx.RunScript(`
	function LintMissingSelect() {
		var q = new Query("lint-test")
		var result = q.Lint('{"from":"users"}')
		return result
	}
	LintMissingSelect()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result = res.(map[string]interface{})
	assert.Equal(t, false, result["valid"])
	diagnostics = result["diagnostics"].([]interface{})
	assert.Greater(t, len(diagnostics), 0)

	// Check first diagnostic
	firstDiag := diagnostics[0].(map[string]interface{})
	assert.Equal(t, "error", firstDiag["severity"])
	assert.Contains(t, firstDiag["message"], "select")

	// ===== Test invalid JSON syntax
	v, err = ctx.RunScript(`
	function LintInvalidJSON() {
		var q = new Query("lint-test")
		var result = q.Lint('{"select":["id",}')
		return result
	}
	LintInvalidJSON()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result = res.(map[string]interface{})
	assert.Equal(t, false, result["valid"])
	diagnostics = result["diagnostics"].([]interface{})
	assert.Greater(t, len(diagnostics), 0)

	// ===== Test DSL with wheres
	v, err = ctx.RunScript(`
	function LintWithWheres() {
		var q = new Query("lint-test")
		var result = q.Lint('{"select":["id","name"],"from":"users","wheres":[{"field":"status","op":"=","value":"active"}]}')
		return result
	}
	LintWithWheres()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result = res.(map[string]interface{})
	assert.Equal(t, true, result["valid"])

	// ===== Test DSL with invalid operator
	v, err = ctx.RunScript(`
	function LintInvalidOp() {
		var q = new Query("lint-test")
		var result = q.Lint('{"select":["id"],"from":"users","wheres":[{"field":"id","op":"invalid_op","value":1}]}')
		return result
	}
	LintInvalidOp()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result = res.(map[string]interface{})
	assert.Equal(t, false, result["valid"])
	diagnostics = result["diagnostics"].([]interface{})
	assert.Greater(t, len(diagnostics), 0)

	firstDiag = diagnostics[0].(map[string]interface{})
	assert.Contains(t, firstDiag["message"], "invalid_op")

	// ===== Test diagnostic position info
	v, err = ctx.RunScript(`
	function LintCheckPosition() {
		var q = new Query("lint-test")
		var result = q.Lint('{"from":"users"}')
		return result.diagnostics[0].position
	}
	LintCheckPosition()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	position := res.(map[string]interface{})
	assert.NotNil(t, position["line"])
	assert.NotNil(t, position["column"])
}

func TestQuerySchema(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	queryObj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("Query", queryObj.ExportFunction(iso))

	// Register a dummy query engine for Schema test
	if _, has := query.Engines["schema-test"]; !has {
		query.Register("schema-test", &gou.Query{
			Query: nil,
			GetTableName: func(s string) string {
				return s
			},
		})
	}

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// ===== Test Schema() returns object
	v, err := ctx.RunScript(`
	function SchemaObject() {
		var q = new Query("schema-test")
		var schema = q.Schema()
		return schema
	}
	SchemaObject()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	schema := res.(map[string]interface{})
	assert.Equal(t, "QueryDSL", schema["title"])
	assert.Equal(t, "http://json-schema.org/draft-07/schema#", schema["$schema"])
	assert.NotNil(t, schema["definitions"])
	assert.NotNil(t, schema["properties"])

	// ===== Test Schema("json") returns string
	v, err = ctx.RunScript(`
	function SchemaJSON() {
		var q = new Query("schema-test")
		var schema = q.Schema("json")
		return typeof schema
	}
	SchemaJSON()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "string", res)

	// ===== Test Schema("json") is valid JSON
	v, err = ctx.RunScript(`
	function SchemaJSONParse() {
		var q = new Query("schema-test")
		var schemaStr = q.Schema("json")
		var parsed = JSON.parse(schemaStr)
		return parsed.title
	}
	SchemaJSONParse()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "QueryDSL", res)
}

func TestQueryValidate(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	queryObj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("Query", queryObj.ExportFunction(iso))

	// Register a dummy query engine for Validate test
	if _, has := query.Engines["validate-test"]; !has {
		query.Register("validate-test", &gou.Query{
			Query: nil,
			GetTableName: func(s string) string {
				return s
			},
		})
	}

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// ===== Test valid data
	v, err := ctx.RunScript(`
	function ValidateValid() {
		var q = new Query("validate-test")
		var result = q.Validate({"select": ["id", "name"], "from": "users"})
		return result
	}
	ValidateValid()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result := res.(map[string]interface{})
	assert.Equal(t, true, result["valid"])
	assert.Nil(t, result["error"])

	// ===== Test invalid data - select should be array
	v, err = ctx.RunScript(`
	function ValidateInvalidSelect() {
		var q = new Query("validate-test")
		var result = q.Validate({"select": "id,name", "from": "users"})
		return result
	}
	ValidateInvalidSelect()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result = res.(map[string]interface{})
	assert.Equal(t, false, result["valid"])
	assert.NotNil(t, result["error"])

	// ===== Test valid data with wheres
	v, err = ctx.RunScript(`
	function ValidateWithWheres() {
		var q = new Query("validate-test")
		var result = q.Validate({
			"select": ["id", "name"],
			"from": "users",
			"wheres": [{"field": "status", "op": "=", "value": "active"}]
		})
		return result
	}
	ValidateWithWheres()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result = res.(map[string]interface{})
	assert.Equal(t, true, result["valid"])

	// ===== Test valid data with condition shorthand
	v, err = ctx.RunScript(`
	function ValidateConditionShorthand() {
		var q = new Query("validate-test")
		var result = q.Validate({
			"select": ["id"],
			"from": "users",
			"wheres": [{"field": "status", "=": "active"}]
		})
		return result
	}
	ValidateConditionShorthand()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result = res.(map[string]interface{})
	assert.Equal(t, true, result["valid"])

	// ===== Test valid data with joins
	v, err = ctx.RunScript(`
	function ValidateWithJoins() {
		var q = new Query("validate-test")
		var result = q.Validate({
			"select": ["id", "name"],
			"from": "users",
			"joins": [{"from": "orders", "key": "user_id", "foreign": "users.id", "left": true}]
		})
		return result
	}
	ValidateWithJoins()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result = res.(map[string]interface{})
	assert.Equal(t, true, result["valid"])

	// ===== Test valid data with sql
	v, err = ctx.RunScript(`
	function ValidateWithSQL() {
		var q = new Query("validate-test")
		var result = q.Validate({
			"sql": {"stmt": "SELECT * FROM users WHERE id = ?", "args": [1]}
		})
		return result
	}
	ValidateWithSQL()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result = res.(map[string]interface{})
	assert.Equal(t, true, result["valid"])

	// ===== Test invalid data - wheres should be array
	v, err = ctx.RunScript(`
	function ValidateInvalidWheres() {
		var q = new Query("validate-test")
		var result = q.Validate({
			"select": ["id"],
			"from": "users",
			"wheres": "invalid"
		})
		return result
	}
	ValidateInvalidWheres()
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result = res.(map[string]interface{})
	assert.Equal(t, false, result["valid"])
	assert.NotNil(t, result["error"])
}

func initTestEngine() {

	if capsule.Global == nil {

		var TestDriver = os.Getenv("GOU_TEST_DB_DRIVER")
		var TestDSN = os.Getenv("GOU_TEST_DSN")

		// Connect DB
		switch TestDriver {
		case "sqlite3":
			capsule.AddConn("primary", "sqlite3", TestDSN).SetAsGlobal()
		default:
			capsule.AddConn("primary", "mysql", TestDSN).SetAsGlobal()
		}
	}

	sch := capsule.Schema()
	sch.MustDropTableIfExists("queryobj_test")
	sch.MustCreateTable("queryobj_test", func(table schema.Blueprint) {
		table.ID("id")
		table.String("name", 20)
	})

	qb := capsule.Query()
	qb.Table("queryobj_test").MustInsert([][]interface{}{
		{1, "Lucy"},
		{2, "Join"},
		{3, "Lily"},
	}, []string{"id", "name"})

	query.Register("query-test", &gou.Query{
		Query: capsule.Query(),
		GetTableName: func(s string) string {
			return s
		},
	})
}
