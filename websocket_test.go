package gou

import (
	"fmt"
	"net"
	"net/http"
	"path"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/websocket"
)

func TestLoadWebSocket(t *testing.T) {
	ws, err := LoadWebSocket("file://"+path.Join(TestServerRoot, "message.ws.json"), "message")
	assert.Nil(t, err)
	assert.Equal(t, ws.Name, "message")
	assert.Equal(t, ws.URL, "ws://127.0.0.1:5011/websocket/message")
	assert.Equal(t, ws.Protocols, []string{"yao-message-01"})
	assert.Equal(t, ws.Event.Data, "scripts.websocket.onData")
	assert.Equal(t, ws.Event.Closed, "scripts.websocket.onClosed")
	assert.Equal(t, ws.Event.Error, "scripts.websocket.onError")
	assert.Equal(t, ws.Event.Connected, "scripts.websocket.onConnected")
}

func TestWebSocketOpen(t *testing.T) {
	err := Yao.Load(path.Join(TestScriptRoot, "websocket.js"), "websocket")
	if err != nil {
		t.Fatal(err)
	}

	srv, url := serve(t)
	defer srv.Stop()

	LoadWebSocket("file://"+path.Join(TestServerRoot, "message.ws.json"), "message")
	ws := SelectWebSocket("message")
	err = ws.Open(url, "messageV2", "chatV3")
	if err != nil {
		t.Fatal(err)
	}

}

func serve(t *testing.T) (*websocket.Upgrader, string) {

	ws, err := websocket.NewUpgrader("test")
	if err != nil {
		t.Fatalf("%s", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	ws.SetHandler(func(message []byte, id int) ([]byte, error) { return message, nil })
	ws.SetRouter(router)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go ws.Start()
	go func() {
		http.Serve(listener, router)
	}()
	time.Sleep(200 * time.Millisecond)

	return ws, fmt.Sprintf("ws://127.0.0.1:%d/websocket/test", listener.Addr().(*net.TCPAddr).Port)
}
