package websocket

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPush(t *testing.T) {
	ws := serve(t)
	conn, err := NewWebSocket("ws://127.0.0.1:5056/websocket/test", []string{"po"})
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer ws.Stop()

	err = Push(conn, "Hello World!")
	if err != nil {
		t.Fatalf("%s", err)
	}
	assert.Nil(t, err)
}

func serve(t *testing.T) *Upgrader {

	ws, err := NewUpgrader("test")
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
