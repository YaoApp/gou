package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
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

	status, data := get(t, server, "/status")
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

func prepare() (*gin.Engine, Option) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/status", func(ctx *gin.Context) {
		ctx.JSON(200, "ok")
	})
	return router, Option{Port: 0, Root: "/", Timeout: 100 * time.Millisecond}
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
