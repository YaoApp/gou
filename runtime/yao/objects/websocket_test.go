package objects

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/websocket"
	"rogchap.com/v8go"
)

func TestWebSocketPush(t *testing.T) {

	serve := serve(t)
	defer serve.Stop()

	iso := v8go.NewIsolate()
	defer iso.Dispose()

	ws := &WebSocket{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("WebSocket", ws.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	v, err := ctx.RunScript(`
		var ws = new WebSocket("ws://127.0.0.1:5056/websocket/test", "po")
		ws.push("Hello World!")
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, v.IsUndefined())
}

func serve(t *testing.T) *websocket.Upgrader {

	ws, err := websocket.NewUpgrader("test")
	if err != nil {
		t.Fatalf("%s", err)
	}

	router := gin.Default()
	ws.SetHandler(func(message []byte) ([]byte, error) { return message, nil })
	ws.SetRouter(router)

	go ws.Start()
	go router.Run(":5056")
	time.Sleep(200 * time.Millisecond)
	return ws
}
