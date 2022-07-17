package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

const (
	// WAITING waiting for connecting
	WAITING uint = iota

	// CONNECTING connecting the host
	CONNECTING

	// CONNECTED  the socket is connected
	CONNECTED

	// CLOSED the socket is closed
	CLOSED

	// LISTENING the websocket server is listening
	LISTENING
)

const (

	// MREAD socket read error ( the local peer closed )
	MREAD uint = iota + 1

	// MBREAK the remote peer closed
	MBREAK

	// MCLOSE user send the CLOSE signal
	MCLOSE
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
	handler     func([]byte, int) ([]byte, error)
	hub         *Hub
	up          *websocket.Upgrader
	interrupt   chan int
	status      uint
}

// WSClient the websocket client
type WSClient struct {
	name         string
	status       uint
	keepAlive    time.Duration
	timeout      time.Duration
	attemptAfter time.Duration
	ping         time.Duration
	conn         *websocket.Conn
	option       WSClientOption
	attemptTimes int
	interrupt    uint
	handlers     Handlers `json:"-"`
}

// WSClientOption the webocket client option
type WSClientOption struct {
	Name         string     `json:"name,omitempty"`
	Description  string     `json:"description,omitempty"`
	Version      string     `json:"version,omitempty"`
	URL          string     `json:"url,omitempty"`
	Protocols    []string   `json:"protocols,omitempty"`
	Guard        string     `json:"guard,omitempty"`
	Buffer       BufferSize `json:"buffer,omitempty"`
	Timeout      int        `json:"timeout,omitempty"`
	Ping         int        `json:"ping,omitempty"`
	KeepAlive    int        `json:"keep,omitempty"`          // -1 not keep alive, 0 keep alive always, keep alive n seconds.
	AttemptAfter int        `json:"attempt_after,omitempty"` // Attempt attempt_after
	Attempts     int        `json:"attempts,omitempty"`      // max times try to reconnect server when connection break (client mode only)
	Timestamp    int        `json:"timestamp,omitempty"`
	IP           string     `json:"ip,omitempty"`
	Port         int        `json:"port,omitempty"`
}

// Handlers the websocket hanlders
type Handlers struct {
	Data      DataHandler
	Error     ErrorHandler
	Closed    ClosedHandler
	Connected ConnectedHandler
}

// DataHandler Handler
type DataHandler func([]byte, int) ([]byte, error)

// ErrorHandler Handler
type ErrorHandler func(error)

// ClosedHandler Handler
type ClosedHandler func([]byte, error) []byte

// ConnectedHandler Handler
type ConnectedHandler func(option WSClientOption) error

// Client is a middleman between the websocket connection and the hub.
type Client struct {

	// The upgrader.
	upgrader *Upgrader

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// The websocket client ID
	id uint32
}

// Hub maintains the set of active clients and broadcasts messages to the
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// The client indexes
	indexes map[uint32]*Client

	// Inbound messages from the clients.
	broadcast chan []byte

	// Direct message, 0-4 id, 4~N message eg: [0 0 0 1 49 124...]
	direct chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Interrupt from the server
	interrupt chan int
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
