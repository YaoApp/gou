package api

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/flow"
	httpTest "github.com/yaoapp/gou/http"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/capsule"
)

func TestSelect(t *testing.T) {
	prepare(t)
	defer clean()
	var user *API
	assert.NotPanics(t, func() {
		user = Select("user")
	})

	user.Reload()
	assert.Equal(t, user.ID, "user")
}

func TestAPIUserHello(t *testing.T) {
	prepare(t)
	defer clean()
	router := testRouter(t)
	response := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/user/hello", nil)
	router.ServeHTTP(response, req)
	assert.Equal(t, `"hello:world"`, response.Body.String())
}

func TestAPIUserAuth(t *testing.T) {
	prepare(t)
	defer clean()
	router := testRouter(t)
	response := httptest.NewRecorder()
	body := []byte(`{"response":"success"}`)
	req, _ := http.NewRequest("POST", "/user/auth/hi?foo=bar&hello=world", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer Token:123456")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, req)
	assert.Equal(t, `"hello:world"`, response.Body.String())
}

func TestAPIUserAuthSid(t *testing.T) {
	prepare(t)
	defer clean()
	router := testRouter(t, func(c *gin.Context) {
		c.Set("__sid", c.Query("sid"))
		c.Set("__global", map[string]interface{}{"hello": "world"})
	})
	response := httptest.NewRecorder()
	id := session.ID()
	ss := session.Global().ID(id)
	ss.Set("id", 1)

	body := []byte(`{"response":"success"}`)
	req, _ := http.NewRequest("POST", "/user/auth/hi?foo=bar&hello=world&sid="+id, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer Token:123456")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, req)
	assert.Equal(t, `"hello:world"`, response.Body.String())
}

func TestAPIUserAuthFail(t *testing.T) {
	prepare(t)
	defer clean()
	router := testRouter(t)
	response := httptest.NewRecorder()
	body := []byte(`{"response":"failure"}`)
	req, _ := http.NewRequest("POST", "/user/auth/hi?foo=bar&hello=world", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer Token:123456")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, req)
	assert.Equal(t, `{"code":403,"message":"failure"}`, response.Body.String())
}

func TestAPIUserSessionFlow(t *testing.T) {
	prepare(t)
	defer clean()
	router := testRouter(t, func(c *gin.Context) {
		c.Set("__sid", c.Query("sid"))
		c.Set("__global", map[string]interface{}{"hello": "world"})
	})
	response := httptest.NewRecorder()
	id := session.ID()
	ss := session.Global().ID(id)
	ss.Set("id", 1)
	req, _ := http.NewRequest("GET", "/user/session/flow?sid="+id, nil)
	router.ServeHTTP(response, req)
	res := responseMap(response).Dot()
	assert.Equal(t, float64(1), res.Get("ID"))
	assert.Equal(t, float64(1), res.Get("SessionData.id"))
	assert.Equal(t, "admin", res.Get("SessionData.type"))
	assert.Equal(t, "world", res.Get("Global.hello"))
	assert.Equal(t, float64(1), res.Get("User.id"))
	assert.Equal(t, "U1", res.Get("User.name"))
	assert.Equal(t, "admin", res.Get("User.type"))
	assert.Equal(t, "application/json", response.Header()["Content-Type"][0])
	assert.Equal(t, "1", response.Header()["User-Agent"][0])
}

func TestAPIUserSessionIn(t *testing.T) {
	prepare(t)
	defer clean()
	router := testRouter(t, func(c *gin.Context) {
		c.Set("__sid", c.Query("sid"))
		c.Set("__global", map[string]interface{}{"hello": "world"})
	})
	response := httptest.NewRecorder()
	id := session.ID()
	ss := session.Global().ID(id)
	ss.Set("id", 1)
	req, _ := http.NewRequest("GET", "/user/session/in?sid="+id, nil)
	router.ServeHTTP(response, req)
	res := responseMap(response).Dot()
	assert.Equal(t, float64(1), res.Get("id"))
}

func TestAPIStreamUnitTest(t *testing.T) {
	prepare(t)
	defer clean()
	router := testRouter(t)

	// Listen
	l, err := net.Listen("tcp4", ":0")
	if err != nil {
		t.Fatal(err)
	}

	srv := &http.Server{Addr: ":0", Handler: router}
	defer func() {
		srv.Close()
		l.Close()
	}()

	// start serve
	go func() {
		fmt.Println("[TestServer] Starting")
		if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
			fmt.Println("[TestServer] Error:", err)
			return
		}
	}()

	addr := strings.Split(l.Addr().String(), ":")
	if len(addr) != 2 {
		t.Fatal("invalid address")
	}

	host := fmt.Sprintf("http://127.0.0.1:%s", addr[1])
	time.Sleep(50 * time.Millisecond)

	res := []byte{}
	req := httpTest.New(fmt.Sprintf("%s/stream/unit/test", host)).
		WithHeader(http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req.Stream(ctx, "POST", map[string]interface{}{"foo": "bar"}, func(data []byte) int {
		res = append(res, data...)
		return 1
	})

	assert.Equal(t, `event:messagedata:{"foo":"bar0"}event:messagedata:{"foo":"bar1"}event:messagedata:{"foo":"bar2"}event:messagedata:{"foo":"bar3"}event:messagedata:{"foo":"bar4"}`, string(res))
}

func testRouter(t *testing.T, middlewares ...gin.HandlerFunc) *gin.Engine {
	prepare(t)
	router := gin.New()
	gin.SetMode(gin.ReleaseMode)
	router.Use(middlewares...)
	SetRoutes(router, "/", "a.com", "b.com")
	return router
}

func responseMap(resp *httptest.ResponseRecorder) maps.MapStrAny {
	body := resp.Body.Bytes()
	res := map[string]interface{}{}
	jsoniter.Unmarshal(body, &res)
	return maps.Of(res)
}

func prepare(t *testing.T) {

	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := application.OpenFromDisk(root) // Load app
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	loadModel(t)
	loadQuery(t)
	loadScripts(t)
	loadFlows(t)

	_, err = Load("/apis/user.http.yao", "user")
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load("/apis/stream.http.yao", "stream")
	if err != nil {
		t.Fatal(err)
	}

	SetGuards(map[string]gin.HandlerFunc{"bearer-jwt": func(ctx *gin.Context) {}})
}

func clean() {
	dbclose()
	v8.Stop()
}

func dbclose() {
	if capsule.Global != nil {
		capsule.Global.Connections.Range(func(key, value any) bool {
			if conn, ok := value.(*capsule.Connection); ok {
				conn.Close()
			}
			return true
		})
	}
}

func dbconnect(t *testing.T) {

	TestDriver := os.Getenv("GOU_TEST_DB_DRIVER")
	TestDSN := os.Getenv("GOU_TEST_DSN")

	// connect db
	switch TestDriver {
	case "sqlite3":
		capsule.AddConn("primary", "sqlite3", TestDSN).SetAsGlobal()
		break
	default:
		capsule.AddConn("primary", "mysql", TestDSN).SetAsGlobal()
		break
	}

}

func loadModel(t *testing.T) {
	dbconnect(t)
	TestAESKey := os.Getenv("GOU_TEST_AES_KEY")
	_, err := model.WithCrypt([]byte(fmt.Sprintf(`{"key":"%s"}`, TestAESKey)), "AES")
	if err != nil {
		t.Fatal(err)
	}

	mods := map[string]string{
		"user": filepath.Join("models", "user.mod.yao"),
	}

	// load mods
	for id, file := range mods {
		_, err := model.Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Migrate
	for id := range mods {
		mod := model.Select(id)
		err := mod.Migrate(true)
		if err != nil {
			t.Fatal(err)
		}
	}

	model.Select("user").Save(map[string]interface{}{"name": "U1", "type": "admin"})
}

func loadQuery(t *testing.T) {

	TestAESKey := os.Getenv("GOU_TEST_AES_KEY")

	// query engine
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

func loadScripts(t *testing.T) {

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

func loadFlows(t *testing.T) {

	flows := map[string]string{
		"tests.session": filepath.Join("flows", "tests", "session.flow.yao"),
	}

	for id, file := range flows {
		_, err := flow.Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}

}
