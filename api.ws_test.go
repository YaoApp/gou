package gou

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/log"
)

func TestLoadWebSocket(t *testing.T) {

	ws, err := LoadWebSocket("file://"+path.Join(TestAPIRoot, "chat.ws.json"), "chat")
	if err != nil {
		t.Fatalf("%s", err)
	}

	assert.Equal(t, 1024, ws.Buffer.Read)
	assert.Equal(t, 1024, ws.Buffer.Write)
	assert.Equal(t, 1024, ws.Limit.MaxMessage)
	assert.Equal(t, 20, ws.Limit.PongWait)
	assert.Equal(t, 10, ws.Limit.WriteWait)
	assert.Equal(t, 5, ws.Timeout)
	assert.Equal(t, []string{"yao-chat-01"}, ws.Protocols)
	assert.Equal(t, "bearer-jwt", ws.Guard)
	assert.Equal(t, "A Chat WebSocket serverr", ws.Description)
	assert.Equal(t, "A Chat WebSocket server", ws.Name)
	assert.Equal(t, "0.9.2", ws.Version)

}

func TestStartWebSocket(t *testing.T) {

	ws, err := LoadWebSocket("file://"+path.Join(TestAPIRoot, "chat.ws.json"), "chat")
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer ws.Stop()

	router := gin.Default()
	ws.SetHandler(func(message []byte) ([]byte, error) { return message, nil })
	ws.SetRouter(router)

	go ws.Start()
	go router.Run(":5055")

	send(t)
}

func TestWebSocketRunProcess(t *testing.T) {
	LoadFlow("file://"+path.Join(TestFLWRoot, "websocket", "chat.flow.json"), "websocket.chat")
	ws, err := LoadWebSocket("file://"+path.Join(TestAPIRoot, "chat.ws.json"), "chat")
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer ws.Stop()

	router := gin.Default()
	ws.SetRouter(router)

	go ws.Start()
	go router.Run(":5055")

	send(t)
}

func send(t *testing.T) {

	fmt.Println("Waiting for the WebSocket server to start")
	time.Sleep(200 * time.Millisecond)
	var cstDialer = websocket.Dialer{
		Subprotocols:     []string{"yao-chat-01"},
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: 5 * time.Second,
	}

	ws, _, err := cstDialer.Dial("ws://127.0.0.1:5055/websocket/chat", nil)
	if err != nil {
		log.Error("Dial: %v", err)
	}

	echo(t, ws)
}

func echo(t *testing.T, ws *websocket.Conn) {

	const message = "Hello World!"
	if err := ws.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetWriteDeadline: %v", err)
	}
	if err := ws.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	if err := ws.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	_, p, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if string(p) != message {
		t.Fatalf("message=%s, want %s", p, message)
	}

	log.Trace("Message:%s", message)
}
