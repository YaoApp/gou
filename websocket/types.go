package websocket

import "github.com/gorilla/websocket"

// WebSocket the websocket struct
type WebSocket struct {
	clients []*websocket.Conn
	websocket.Upgrader
}
