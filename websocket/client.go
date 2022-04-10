package websocket

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/yaoapp/kun/log"
)

const (
	maxMessage      = int64(10485760) // 10 M
	readBufferSize  = 1024
	writeBufferSize = 1024
	timeout         = 5 * time.Second
)

// NewWebSocket create a new websocket connection
func NewWebSocket(url string, protocals []string) (*websocket.Conn, error) {

	var dialer = websocket.Dialer{
		Subprotocols:     protocals,
		ReadBufferSize:   readBufferSize,
		WriteBufferSize:  writeBufferSize,
		HandshakeTimeout: timeout,
	}

	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		log.Error("NewWebSocket Dial: %v", err)
		return nil, err
	}

	return conn, nil
}

// Push a message to websocket connection and get the response message
func Push(conn *websocket.Conn, message string) error {
	defer conn.Close()
	if err := conn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		log.Error("Websocket SetWriteDeadline: %v", err)
		return err
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		log.Error("Websocket WriteMessage: %v", err)
		return err
	}
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		log.Error("Websocket SetReadDeadline: %v", err)
		return err
	}

	return nil

	// conn.SetReadLimit(maxMessage)

	// _, response, err := conn.ReadMessage()
	// if err != nil {
	// 	log.Error("Websocket ReadMessage: %v", err)
	// 	return "", nil // Ignore error
	// }
	// return string(response), nil
}
