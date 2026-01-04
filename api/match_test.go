package api_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/capsule"
)

func TestFindHandlerExactMatch(t *testing.T) {
	prepareTest(t)
	defer cleanTest()

	// Build route table
	api.BuildRouteTable()

	// Test exact match
	apiDef, pathDef, handler, params, err := api.FindHandler("GET", "/user/hello")
	assert.NoError(t, err)
	assert.NotNil(t, apiDef)
	assert.NotNil(t, pathDef)
	assert.NotNil(t, handler)
	assert.Empty(t, params)
	assert.Equal(t, "/hello", pathDef.Path)
}

func TestFindHandlerWithParams(t *testing.T) {
	prepareTest(t)
	defer cleanTest()

	// Build route table
	api.BuildRouteTable()

	// Test path with parameter :id
	apiDef, pathDef, handler, params, err := api.FindHandler("GET", "/user/info/123")
	assert.NoError(t, err)
	assert.NotNil(t, apiDef)
	assert.NotNil(t, pathDef)
	assert.NotNil(t, handler)
	assert.Equal(t, "123", params["id"])
	assert.Equal(t, "/info/:id", pathDef.Path)
}

func TestFindHandlerWithNamedParam(t *testing.T) {
	prepareTest(t)
	defer cleanTest()

	// Build route table
	api.BuildRouteTable()

	// Test path with named parameter :name
	apiDef, pathDef, handler, params, err := api.FindHandler("POST", "/user/auth/john")
	assert.NoError(t, err)
	assert.NotNil(t, apiDef)
	assert.NotNil(t, pathDef)
	assert.NotNil(t, handler)
	assert.Equal(t, "john", params["name"])
	assert.Equal(t, "/auth/:name", pathDef.Path)
}

func TestFindHandlerNotFound(t *testing.T) {
	prepareTest(t)
	defer cleanTest()

	// Build route table
	api.BuildRouteTable()

	// Test non-existent route
	apiDef, pathDef, handler, params, err := api.FindHandler("GET", "/nonexistent/path")
	assert.Error(t, err)
	assert.Nil(t, apiDef)
	assert.Nil(t, pathDef)
	assert.Nil(t, handler)
	assert.Nil(t, params)
}

func TestFindHandlerWrongMethod(t *testing.T) {
	prepareTest(t)
	defer cleanTest()

	// Build route table
	api.BuildRouteTable()

	// Test wrong method - /user/hello is GET, not POST
	apiDef, pathDef, handler, params, err := api.FindHandler("POST", "/user/hello")
	assert.Error(t, err)
	assert.Nil(t, apiDef)
	assert.Nil(t, pathDef)
	assert.Nil(t, handler)
	assert.Nil(t, params)
}

func TestBuildRouteTableMultipleTimes(t *testing.T) {
	prepareTest(t)
	defer cleanTest()

	// Build route table multiple times should not cause issues
	api.BuildRouteTable()
	api.BuildRouteTable()
	api.BuildRouteTable()

	// Should still work correctly
	apiDef, pathDef, handler, params, err := api.FindHandler("GET", "/user/hello")
	assert.NoError(t, err)
	assert.NotNil(t, apiDef)
	assert.NotNil(t, pathDef)
	assert.NotNil(t, handler)
	assert.Empty(t, params)
}

func TestFindHandlerConcurrency(t *testing.T) {
	prepareTest(t)
	defer cleanTest()

	// Build route table
	api.BuildRouteTable()

	// Test concurrent access
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			apiDef, pathDef, handler, _, err := api.FindHandler("GET", "/user/hello")
			assert.NoError(t, err)
			assert.NotNil(t, apiDef)
			assert.NotNil(t, pathDef)
			assert.NotNil(t, handler)
		}()
	}

	wg.Wait()
}

func TestReloadAPIs(t *testing.T) {
	prepareTest(t)
	defer cleanTest()

	// Build initial route table
	api.BuildRouteTable()

	// Verify initial state
	_, _, _, _, err := api.FindHandler("GET", "/user/hello")
	assert.NoError(t, err)

	// Reload APIs
	err = api.ReloadAPIs("apis")
	assert.NoError(t, err)

	// Should still work after reload
	apiDef, pathDef, handler, _, err := api.FindHandler("GET", "/user/hello")
	assert.NoError(t, err)
	assert.NotNil(t, apiDef)
	assert.NotNil(t, pathDef)
	assert.NotNil(t, handler)
}

func TestReloadAPIsNonExistentDir(t *testing.T) {
	prepareTest(t)
	defer cleanTest()

	// Reload from non-existent directory should not error
	err := api.ReloadAPIs("nonexistent_dir")
	assert.NoError(t, err)
}

func TestReloadAPIsConcurrency(t *testing.T) {
	prepareTest(t)
	defer cleanTest()

	// Build initial route table
	api.BuildRouteTable()

	// Test concurrent reload and find
	var wg sync.WaitGroup

	// Readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				api.FindHandler("GET", "/user/hello")
			}
		}()
	}

	// Writers (reload)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			api.ReloadAPIs("apis")
		}()
	}

	wg.Wait()

	// Should still work after concurrent operations
	apiDef, pathDef, handler, _, err := api.FindHandler("GET", "/user/hello")
	assert.NoError(t, err)
	assert.NotNil(t, apiDef)
	assert.NotNil(t, pathDef)
	assert.NotNil(t, handler)
}

// Test helper functions

func prepareTest(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	if root == "" {
		t.Skip("GOU_TEST_APPLICATION not set")
	}

	app, err := application.OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	loadTestModel(t)
	loadTestQuery(t)
	loadTestScripts(t)

	// Load test APIs
	_, err = api.Load("/apis/user.http.yao", "user")
	if err != nil {
		t.Fatal(err)
	}

	_, err = api.Load("/apis/stream.http.yao", "stream")
	if err != nil {
		t.Fatal(err)
	}

	api.SetGuards(map[string]gin.HandlerFunc{"bearer-jwt": func(ctx *gin.Context) {}})
}

func cleanTest() {
	dbClose()
	v8.Stop()
}

func dbClose() {
	if capsule.Global != nil {
		capsule.Global.Connections.Range(func(key, value any) bool {
			if conn, ok := value.(*capsule.Connection); ok {
				conn.Close()
			}
			return true
		})
	}
}

func dbConnect(t *testing.T) {
	TestDriver := os.Getenv("GOU_TEST_DB_DRIVER")
	TestDSN := os.Getenv("GOU_TEST_DSN")

	switch TestDriver {
	case "sqlite3":
		capsule.AddConn("primary", "sqlite3", TestDSN).SetAsGlobal()
	default:
		capsule.AddConn("primary", "mysql", TestDSN).SetAsGlobal()
	}
}

func loadTestModel(t *testing.T) {
	dbConnect(t)
	TestAESKey := os.Getenv("GOU_TEST_AES_KEY")
	_, err := model.WithCrypt([]byte(`{"key":"`+TestAESKey+`"}`), "AES")
	if err != nil {
		t.Fatal(err)
	}

	mods := map[string]string{
		"user": filepath.Join("models", "user.mod.yao"),
	}

	for id, file := range mods {
		_, err := model.Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}

	for id := range mods {
		mod := model.Select(id)
		err := mod.Migrate(true)
		if err != nil {
			t.Fatal(err)
		}
	}

	model.Select("user").Save(map[string]interface{}{"name": "U1", "type": "admin"})
}

func loadTestQuery(t *testing.T) {
	TestAESKey := os.Getenv("GOU_TEST_AES_KEY")

	query.Register("query-test", &gou.Query{
		Query: capsule.Query(),
		GetTableName: func(s string) string {
			if mod, has := model.Models[s]; has {
				return mod.MetaData.Table.Name
			}
			exception.New("[query] %s not found", 404).Throw()
			return s
		},
		AESKey: TestAESKey,
	})
}

func loadTestScripts(t *testing.T) {
	scripts := map[string]string{
		"test.api": filepath.Join("scripts", "tests", "api.js"),
	}

	for id, file := range scripts {
		_, err := v8.Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}

	err := v8.Start(&v8.Option{})
	if err != nil {
		t.Fatal(err)
	}
}
