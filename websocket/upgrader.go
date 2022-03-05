package websocket

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
)

// Upgraders have registered
var Upgraders = map[string]*Upgrader{}

// NewUpgrader create a new WebSocket upgrader
// {
// 		"name": "A Chat WebSocket server",
// 		"description": "A Chat WebSocket serverr",
// 		"version": "0.9.2",
// 		"protocols": ["yao-chat-01"],
// 		"guard": "bearer-jwt",
// 		"buffer": { "read": 1024, "write": 1024 },
// 		"limit": { "read-wait": 5, "pong-wait": 10, "max-message":512 },
// 		"process": "flows.websocket.chat",
// }
func NewUpgrader(name string, config ...[]byte) (*Upgrader, error) {

	// the default values
	var upgrader = &Upgrader{
		name:      name,
		Buffer:    BufferSize{Read: 1024, Write: 1024},
		Limit:     Limit{WriteWait: 10, PongWait: 60, MaxMessage: 1024},
		Protocols: []string{},
		Guard:     "-",
		handler:   func([]byte) ([]byte, error) { return nil, nil },
		Timeout:   5,
		interrupt: make(chan bool),
	}

	// load from config json
	if len(config) > 0 {
		err := jsoniter.Unmarshal(config[0], upgrader)
		if err != nil {
			return nil, err
		}
	}

	// create hub etc...
	upgrader.Limit.pingPeriod = (upgrader.Limit.PongWait * 9) / 10
	upgrader.hub = newHub()
	upgrader.up = &websocket.Upgrader{
		ReadBufferSize:   upgrader.Buffer.Read,
		WriteBufferSize:  upgrader.Buffer.Write,
		HandshakeTimeout: time.Duration(upgrader.Timeout) * time.Second,
		Subprotocols:     upgrader.Protocols,
		CheckOrigin:      func(r *http.Request) bool { return true },
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			log.Error("Upgrader: %s [%d]%s", name, status, reason.Error())
		},
		EnableCompression: true,
	}

	// register upgrader
	Upgraders[name] = upgrader

	return upgrader, nil
}

// SetHandler set the message handler
func (upgrader *Upgrader) SetHandler(handler func([]byte) ([]byte, error)) {
	upgrader.handler = handler
}

// SetRouter upgrades the Gin server connection to the WebSocket protocol.
func (upgrader *Upgrader) SetRouter(r *gin.Engine) {

	var path = fmt.Sprintf("/websocket/%s", upgrader.name)
	r.GET(path, func(c *gin.Context) {
		upgrader.UpgradeGin(c, nil)
	})
}

// Start the hub
func (upgrader *Upgrader) Start() {
	go upgrader.hub.run()
	<-upgrader.interrupt
}

// Stop the hub
func (upgrader *Upgrader) Stop() {
	upgrader.interrupt <- true
}

// UpgradeGin upgrades the Gin server connection to the WebSocket protocol.
func (upgrader *Upgrader) UpgradeGin(c *gin.Context, responseHeader http.Header) (*websocket.Conn, error) {
	return upgrader.Upgrade(c.Writer, c.Request, responseHeader)
}

// Upgrade upgrades the HTTP server connection to the WebSocket protocol.
//
// The responseHeader is included in the response to the client's upgrade
// request. Use the responseHeader to specify cookies (Set-Cookie). To specify
// subprotocols supported by the server, set Upgrader.Subprotocols directly.
//
// If the upgrade fails, then Upgrade replies to the client with an HTTP error
// response.
func (upgrader *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error) {
	conn, err := upgrader.up.Upgrade(w, r, responseHeader)
	if err != nil {
		log.Error("Upgrader: %s [500]%s", upgrader.name, err.Error())
		return nil, err
	}

	// Todo: add client id for direct message
	client := &Client{hub: upgrader.hub, upgrader: upgrader, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()

	return conn, nil
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {

	pingPeriod := time.Duration(c.upgrader.Limit.pingPeriod) * time.Second
	writeWait := time.Duration(c.upgrader.Limit.WriteWait) * time.Second

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			log.Trace("writePump: %s [200]%s", c.upgrader.name, message)

			// Add queued messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	pongWait := time.Duration(c.upgrader.Limit.PongWait) * time.Second
	c.conn.SetReadLimit(int64(c.upgrader.Limit.MaxMessage))
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("Upgrader: %s [500]%s", c.upgrader.name, err.Error())
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		response, err := c.upgrader.handler(message)
		if err != nil {
			log.Error("Upgrader: %s [500]%s", c.upgrader.name, err.Error())
			break
		}

		if response != nil {
			log.Trace("Upgrader Message: %s [200]%s", c.upgrader.name, message)
			log.Trace("Upgrader Response: %s [200]%s", c.upgrader.name, response)
			c.hub.broadcast <- response
		}
	}
}
