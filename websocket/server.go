package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/yaoapp/kun/log"
)

// NewServer Create a websocket server
func NewServer() *WebSocket {
	return &WebSocket{
		clients: []*websocket.Conn{},
	}
}

// Start create a websocket server
func (ws *WebSocket) Start(w http.ResponseWriter, r *http.Request, responseHeader http.Header) error {
	c, err := ws.Upgrade(w, r, responseHeader)
	if err != nil {
		log.Error("WebSocket Serve:%s", err.Error())
		return err
	}
	defer c.Close()
	ws.handleRequest(c)
	return nil
}

func (ws *WebSocket) handleRequest(c *websocket.Conn) {
	for {
		log.Trace("handleRequest--")
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Error("read: %s", err)
			break
		}
		log.Trace("recv: %s", message)
		err = c.WriteMessage(mt, message)

	}
}
