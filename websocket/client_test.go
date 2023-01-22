package websocket

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPush(t *testing.T) {
	srv, url := serve(t)
	defer srv.Stop()

	conn, err := NewWebSocket(url, []string{"po"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = Push(conn, "Hello World!")
	if err != nil {
		t.Fatalf("%s", err)
	}
	assert.Nil(t, err)
}

func TestOpen(t *testing.T) {
	srv, url := serve(t)
	defer srv.Stop()

	var ws *WSClient = nil
	var handlers = Handlers{
		Connected: func(option WSClientOption) error {
			fmt.Println("onConnected", option)
			fmt.Println("Online", srv.Online())
			fmt.Println("Clients", srv.Clients())
			srv.Broadcast([]byte("Hello world"))
			time.Sleep(500 * time.Millisecond)
			ws.Write([]byte("1|I'm Here, The connection will be closed"))
			return nil
		},
		Closed: func(data []byte, err error) []byte {
			fmt.Printf("onClosed: %s %v\n", data, err)
			return nil
		},
		Data: func(data []byte, length int) ([]byte, error) {
			fmt.Printf("onData: %s %d\n", data, length)
			if data[0] == 0x31 {
				err := ws.Close()
				fmt.Println("Close connection", err)
			}
			return nil, nil
		},
		Error: func(err error) {
			fmt.Printf("Error: %s\n", err)
		},
	}
	ws = NewWSClient(WSClientOption{URL: url, Protocols: []string{"po"}}, handlers)
	err := ws.Open()
	if err != nil {
		t.Fatal(err)
	}
}

func serve(t *testing.T) (*Upgrader, string) {

	ws, err := NewUpgrader("test")
	if err != nil {
		t.Fatalf("%s", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	ws.SetHandler(func(message []byte, client int) ([]byte, error) { return message, nil })
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
