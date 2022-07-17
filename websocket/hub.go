package websocket

import (
	"encoding/binary"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/yaoapp/kun/log"
)

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		direct:     make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		interrupt:  make(chan int),
		indexes:    map[uint32]*Client{},
	}
}

// NextID return the next client ID
func (h *Hub) NextID() uint32 {
	var mutex sync.Mutex
	mutex.Lock()
	id := len(h.clients)
	mutex.Unlock()
	return uint32(id) + 1
}

// Clients return the online clients
func (h *Hub) Clients() []uint32 {
	ids := []uint32{}
	for client := range h.clients {
		ids = append(ids, client.id)
	}
	return ids
}

// Nums count the online client's nums
func (h *Hub) Nums() int {
	return len(h.clients)
}

// AddID add a id to the message
// 0-4 id, 4~N message eg: [0 0 0 1 49 124...]
func (h *Hub) AddID(id uint32, message []byte) []byte {
	head := make([]byte, 4)
	binary.BigEndian.PutUint32(head, id)
	res := append([]byte{}, head...)
	res = append(res, message...)
	return res
}

func (h *Hub) run() {
LOOP:
	for {
		select {

		case client := <-h.register:
			h.clients[client] = true
			h.indexes[client.id] = client

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.indexes, client.id)
				delete(h.clients, client)
				close(client.send)
			}

		case message := <-h.direct:
			if len(message) > 4 {
				//  0-4 id, 4~N message eg: [0 0 0 1 49 124...]
				id := binary.BigEndian.Uint32(message[0:4])
				msg := message[4:]
				if client, ok := h.indexes[id]; ok {
					select {
					case client.send <- msg:
					default:
						close(client.send)
						delete(h.indexes, client.id)
						delete(h.clients, client)
					}
				}
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.indexes, client.id)
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
