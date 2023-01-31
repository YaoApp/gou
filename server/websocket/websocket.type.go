package api

import "github.com/yaoapp/gou/websocket"

// WebSocket the websocket struct
type WebSocket struct {
	websocket.WSClientOption
	Event  WebSocketEvent `json:"event,omitempty"`
	Client *websocket.WSClient
}

// WebSocketEvent the websocket  struct
type WebSocketEvent struct {
	Data      string `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
	Closed    string `json:"closed,omitempty"`
	Connected string `json:"connected,omitempty"`
}
