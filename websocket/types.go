package websocket

import (
	"github.com/gorilla/websocket"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Upgrader the upgrader setting
// {
// 		"name": "A Chat WebSocket server",
// 		"description": "A Chat WebSocket serverr",
// 		"version": "0.9.2",
// 		"protocols": ["yao-chat-01"],
// 		"guard": "bearer-jwt",
// 		"buffer": { "read": 1024, "write": 1024 },
// 		"limit": { "read-wait": 5, "pong-wait": 10, "max-message":512 },
// 		"timeout": 10,
// 		"process": "flows.websocket.chat",
// }
type Upgrader struct {
	name        string
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Version     string     `json:"version,omitempty"`
	Protocols   []string   `json:"protocols,omitempty"`
	Guard       string     `json:"guard,omitempty"`
	Buffer      BufferSize `json:"buffer,omitempty"`
	Limit       Limit      `json:"limit,omitempty"`
	Timeout     int        `json:"timeout,omitempty"`
	Process     string     `json:"process,omitempty"` // serve handler
	handler     func([]byte) ([]byte, error)
	hub         *Hub
	up          *websocket.Upgrader
	interrupt   chan bool
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The upgrader.
	upgrader *Upgrader

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// Hub maintains the set of active clients and broadcasts messages to the
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

// BufferSize read and write buffer sizes
type BufferSize struct {
	Read  int `json:"read,omitempty"`
	Write int `json:"write,omitempty"`
}

// Limit the limit of session
type Limit struct {
	WriteWait  int `json:"write-wait,omitempty"`  // Time allowed to write a message to the peer.
	PongWait   int `json:"pong-wait,omitempty"`   // Time allowed to read the next pong message from the peer.
	MaxMessage int `json:"max-message,omitempty"` // Maximum message size allowed from peer. bytes
	pingPeriod int `json:"-"`                     // Send pings to peer with this period. Must be less than pongWait. (pongWait * 9) / 10
}
