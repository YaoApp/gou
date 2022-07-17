package websocket

import (
	"github.com/gorilla/websocket"
	"github.com/yaoapp/kun/log"
)

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		interrupt:  make(chan int),
	}
}

func (h *Hub) run() {
LOOP:
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}

		case exit := <-h.interrupt:
			if exit == 1 {
				for client := range h.clients {
					err := client.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseServiceRestart, "Repair"))
					log.Trace("Close Client Connection, %v", err)
					// close(client.send)
					// delete(h.clients, client)
				}
				break LOOP
			}
		}
	}
}
