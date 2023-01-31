package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {
	router, option := prepare()
	server := New(router, option)
	var err error
	go func() { err = server.Start() }()
	defer server.Stop()

	<-server.Event()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, server.Ready())

	status, data := get(t, server, "/api/status")
	assert.Equal(t, 200, status)
	assert.Equal(t, []byte(`"ok"`), data)
}

func TestStop(t *testing.T) {
	router, option := prepare()
	server := New(router, option)
	var err error
	go func() { err = server.Start() }()

	<-server.Event()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, server.Ready())

	err = server.Stop()
	if err != nil {
		t.Fatal(err)
	}

	<-server.Event()
	assert.False(t, server.Ready())
}

func TestRestart(t *testing.T) {
	router, option := prepare()
	server := New(router, option)
	var err error
	go func() { err = server.Start() }()
	defer server.Stop()

	<-server.Event()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, server.Ready())

	err = server.Restart()
	if err != nil {
		t.Fatal(err)
	}

	<-server.Event()
	assert.True(t, server.Ready())
}

func TestWithStatic(t *testing.T) {
	router, option := prepare()
	server := New(router, option)
	server.With(testMiddlewares()...)

	var err error
	go func() { err = server.Start() }()
	defer server.Stop()

	<-server.Event()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, server.Ready())

	status, data := get(t, server, "/api/status")
	assert.Equal(t, 200, status)
	assert.Equal(t, []byte(`"ok"`), data)

	status, data = get(t, server, "/tests/http.html")
	assert.Equal(t, 200, status)
	assert.Equal(t, []byte("<div>It works!</div>\n"), data)

	status, data = get(t, server, "/admin/tests/http.html")
	assert.Equal(t, 200, status)
	assert.Equal(t, []byte("<div>It works! ADMIN</div>\n"), data)

}

func prepare() (*gin.Engine, Option) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/api/status", func(ctx *gin.Context) {
		ctx.JSON(200, "ok")
	})
	return router, Option{Port: 0, Root: "/", Timeout: 100 * time.Millisecond}
}

func testMiddlewares() []func(ctx *gin.Context) {

	root := os.Getenv("GOU_TEST_APPLICATION")
	public := http.FileServer(http.Dir(filepath.Join(root, "public")))
	admin := http.FileServer(http.Dir(filepath.Join(root, "static")))

	var handler = func(ctx *gin.Context) {
		length := len(ctx.Request.URL.Path)
		if (length >= 5 && ctx.Request.URL.Path[0:5] == "/api/") ||
			(length >= 11 && ctx.Request.URL.Path[0:11] == "/websocket/") { // API & websocket
			ctx.Next()
			return
		}

		if strings.HasPrefix(ctx.Request.URL.Path, "/admin/") {
			ctx.Request.URL.Path = strings.TrimPrefix(ctx.Request.URL.Path, "/admin")
			admin.ServeHTTP(ctx.Writer, ctx.Request)
			ctx.Abort()
			return
		}

		public.ServeHTTP(ctx.Writer, ctx.Request)
		ctx.Abort()
	}

	return []func(ctx *gin.Context){handler}
}

func get(t *testing.T, server *Server, path string) (int, []byte) {
	port, err := server.Port()
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, path))
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, body
}
