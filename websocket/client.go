package websocket

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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

// NewWSClient create a new webocket client connection
func NewWSClient(option WSClientOption, handlers Handlers) *WSClient {
	if option.Timeout == 0 {
		option.Timeout = 5
	}

	if option.Buffer.Read == 0 {
		option.Buffer.Read = 1024
	}

	if option.Buffer.Write == 0 {
		option.Buffer.Read = 1024
	}

	if option.Attempts > 0 && option.AttemptAfter == 0 {
		option.AttemptAfter = 50
	}

	cli := &WSClient{option: option, handlers: handlers}
	cli.timeout = time.Duration(option.Timeout) * time.Second
	cli.attemptAfter = time.Duration(option.AttemptAfter) * time.Millisecond
	cli.keepAlive = time.Duration(option.KeepAlive) * time.Second
	cli.interrupt = make(chan uint)
	return cli
}

// Open the websockt connetion
func (ws *WSClient) Open() error {

	if ws.status == CONNECTING {
		err := fmt.Errorf("WebSocket Open: %s:%v is connecting", ws.option.URL, ws.option.Protocols)
		log.With(log.F{"option": ws.option}).Error(err.Error())
		ws.emitError(err)
		return err
	}

	if ws.status == CONNECTED {
		err := fmt.Errorf("WebSocket Open: %s:%v was connected", ws.option.URL, ws.option.Protocols)
		log.With(log.F{"option": ws.option}).Error(err.Error())
		ws.emitError(err)
		return err
	}

	ws.status = CONNECTING
	log.With(log.F{"option": ws.option}).Trace("Connecting")

	var dialer = websocket.Dialer{
		Subprotocols:     ws.option.Protocols,
		ReadBufferSize:   ws.option.Buffer.Read,
		WriteBufferSize:  ws.option.Buffer.Write,
		HandshakeTimeout: ws.timeout,
	}

	// System os signal
	go func() {
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-done
		ws.interrupt <- MCLOSE
	}()

	conn, _, err := dialer.Dial(ws.option.URL, nil)
	if err != nil {
		log.With(log.F{"option": ws.option}).Error("WebSocket Open: %s", err)
		ws.emitError(err)
		if ws.attempts > ws.attemptTimes {
			ws.attemptTimes = ws.attemptTimes + 1
			if ws.attemptAfter > 0 {
				var after = time.Duration(int(ws.attemptAfter) * ws.attemptTimes)
				log.With(log.F{"option": ws.option}).Trace("Try to reconnect after %v", after)
				time.Sleep(after)
			}
			log.With(log.F{"option": ws.option}).Trace("Connecting ... %d/%d", ws.attemptTimes, ws.attempts)
			return ws.Open()
		}
		return err
	}

	log.With(log.F{"option": ws.option}).Trace("Connected")
	defer conn.Close()
	ws.conn = conn
	ws.status = CONNECTED
	ws.conn = conn
	ws.attemptTimes = 0
	err = ws.emitConnected(ws.option)

	go ws.readPump()

	select {
	case exit := <-ws.interrupt:
		if exit == MBREAK || exit == MREAD {
			ws.emitClosed(nil, fmt.Errorf("BREAK"))
			return ws.Open()
		}
		if exit == MCLOSE {
			ws.emitClosed([]byte("CLOSE"), nil)
			ws.status = CLOSED
			return nil
		}
		break
	}

	return nil
}

// Close the connection
func (ws *WSClient) Close() error {
	err := ws.conn.Close()
	if err != nil {
		return err
	}
	ws.interrupt <- MCLOSE
	return nil
}

// Write messge
func (ws *WSClient) Write(message []byte) error {
	return ws.conn.WriteMessage(websocket.TextMessage, message)
}

// readPump pumps messages from the websocket connection.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (ws *WSClient) readPump() {

	if err := ws.conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		log.Error("Websocket SetReadDeadline: %v", err)
		ws.interrupt <- MCLOSE
		return
	}

	for {
		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Trace("WebSocket Read: %s [500]%s", ws.name, err.Error())
				ws.emitError(err)
				ws.interrupt <- MBREAK
				return
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		response, err := ws.emitData(message, len(message))
		if err != nil {
			log.Error("WebSocket Read: %s [500]%s", ws.name, err.Error())
			ws.emitError(err)
			break
		}

		if response != nil {
			if err := ws.conn.WriteMessage(websocket.TextMessage, response); err != nil {
				log.Error("Websocket WriteMessage: %v", err)
			}
		}

		// KeepLive
		if ws.option.KeepAlive == -1 {
			ws.interrupt <- MCLOSE
			return
		}
	}
}

// emitConnect trigger the connected event
func (ws *WSClient) emitConnected(option WSClientOption) error {
	if ws.handlers.Connected != nil {
		return ws.handlers.Connected(option)
	}
	return nil
}

// emitError trigger the error event
func (ws *WSClient) emitError(err error) {
	if ws.handlers.Error != nil {
		ws.handlers.Error(err)
	}
}

// emitClosed trigger the closed event
func (ws *WSClient) emitClosed(data []byte, err error) []byte {
	if ws.handlers.Closed != nil {
		return ws.handlers.Closed(data, err)
	}
	return nil
}

// emitData trigger the data event
func (ws *WSClient) emitData(data []byte, length int) ([]byte, error) {
	if ws.handlers.Data != nil {
		return ws.handlers.Data(data, length)
	}
	return nil, nil
}

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
