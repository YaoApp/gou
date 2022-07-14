package socket

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yaoapp/kun/log"
)

// NewClient Create a socket client
func NewClient(option Option, handlers Handlers) *Client {
	return &Client{
		Status:   WAITING,
		Option:   option,
		Handlers: handlers,
	}
}

// Open Connect the socket server
func (client *Client) Open() error {
	if client.Option.Protocol == "tcp" {
		return client.tcpOpen()
	}
	err := fmt.Errorf("Socket Open: protocol %s does not support", client.Option.Protocol)
	log.With(log.F{"option": client.Option}).Error(err.Error())
	return err
}

// tcpOpen connect to the server using TCP/IP protocol
func (client *Client) tcpOpen() error {

	option := client.Option
	if client.Status == CONNECTING {
		err := fmt.Errorf("Socket Open: %s:%s is connecting", option.Host, option.Port)
		log.With(log.F{"option": client.Option}).Error(err.Error())
		client.emitError(err)
		return err
	}

	if client.Status == CONNECTED {
		err := fmt.Errorf("Socket Open: %s:%s was connected", option.Host, option.Port)
		log.With(log.F{"option": client.Option}).Error(err.Error())
		client.emitError(err)
		return err
	}

	client.Status = CONNECTING
	log.With(log.F{"option": option}).Trace("Connecting")
	dial := net.Dialer{Timeout: option.Timeout}
	if option.KeepAlive > 0 {
		dial.KeepAlive = option.KeepAlive
	}

	conn, err := dial.Dial("tcp", fmt.Sprintf("%s:%s", option.Host, option.Port))
	if err != nil {
		log.With(log.F{"option": option}).Error("Socket Open: %s", err)
		client.emitError(err)
		return err
	}

	defer conn.Close()
	client.Status = CONNECTED
	client.Conn = conn
	log.With(log.F{"option": option}).Trace("Connected")
	err = client.emitConnected()
	if err != nil {
		log.With(log.F{"option": option}).Error("Socket Open trigger connected event: %s", err.Error())
		client.emitError(err)
	}

	ch := make(chan uint)
	// read and write
	go func() {
		for {
			buffer := make([]byte, option.BufferSize)
			recvLen, err := conn.Read(buffer)
			if err != nil && err != io.EOF {
				log.With(log.F{"option": option}).Error("Socket Open read: %s", err)
				ch <- MREAD
				return
			}

			if err == io.EOF {
				log.With(log.F{"option": option}).Error("Socket Open Connection lost")
				ch <- MBREAK
				return
			}

			log.With(log.F{"option": option, "recvLen": recvLen, "data": fmt.Sprintf("%x", buffer)}).Trace("Receive")
			data := []byte{}
			data = append(data, buffer[:recvLen]...)
			resp, err := client.emitData(data, recvLen)
			if err != nil {
				log.With(log.F{"option": option}).Error("Socket Open trigger connected event: %s", err.Error())
				client.emitError(err)
			}

			// Send Response to Server
			if resp != nil {
				recvLen, err := conn.Write(resp)
				if err != nil {
					log.With(log.F{"option": option, "data": fmt.Sprintf("%x", resp)}).Error("Socket Open Send Response: %s", err)
					client.emitError(err)
				}
				log.Trace("Send Response to server: %d", recvLen)
			}

			// KeepLive
			if option.KeepAlive == -1 {
				ch <- MCLOSE
				return
			}
		}
	}()

	// System os signal
	go func() {
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-done
		ch <- MCLOSE
	}()

	switch <-ch {
	case MREAD, MBREAK:
		client.Status = CLOSED
		client.emitClose(nil, fmt.Errorf("BREAK"))
		return client.tcpOpen()
	case MCLOSE:
		resp := client.emitClose([]byte("CLOSE"), nil)
		if resp != nil {
			recvLen, err := conn.Write(resp)
			if err != nil {
				log.With(log.F{"option": option, "data": fmt.Sprintf("%x", resp)}).Error("Socket Open Send Response: %s", err)
				// client.emitError(err)
			}
			log.Trace("Send Response to server: %d", recvLen)
		}
		client.Status = CLOSED
		return nil
	}

	return nil
}

// emitConnect trigger the connected event
func (client *Client) emitConnected() error {
	if client.Handlers.Connected == nil {
		return nil
	}
	return client.Handlers.Connected(client.Option)
}

// emitData trigger the data event and get the response
func (client *Client) emitData(data []byte, length int) ([]byte, error) {
	if client.Handlers.Data == nil {
		return nil, nil
	}
	return client.Handlers.Data(data, length)
}

// emitError trigger the error event and get the response
func (client *Client) emitError(err error) {
	if client.Handlers.Error == nil {
		return
	}
	client.Handlers.Error(err)
}

// emitError trigger the error event and get the response
func (client *Client) emitClose(data []byte, err error) []byte {
	if client.Handlers.Close == nil {
		return nil
	}
	return client.Handlers.Close(data, err)
}

// Connect Connect socket server  (alpha -> will be refactored at a beta version...)
func Connect(proto string, host string, port string, timeout time.Duration, bufferSize int, KeepAlive time.Duration, handler func([]byte, int, error) ([]byte, error)) error {
	if proto == "tcp" {
		return tcpConnect(host, port, timeout, bufferSize, KeepAlive, handler)
	}
	err := fmt.Errorf("Protocol: %s does not support", proto)
	log.With(log.F{"host": host, "port": port, "timeout": timeout}).Error("Protocol: %s", proto)
	return err
}

// tcpConnect connect to the server using TCP/IP protocol
func tcpConnect(host string, port string, timeout time.Duration, bufferSize int, KeepAlive time.Duration, handler func([]byte, int, error) ([]byte, error)) error {

	dial := net.Dialer{Timeout: timeout}
	if KeepAlive > 0 {
		dial.KeepAlive = KeepAlive
	}

	log.With(log.F{"host": host, "port": port, "bufferSize": bufferSize, "KeepAlive": KeepAlive}).Trace("Connecting")
	conn, err := dial.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		log.With(log.F{"host": host, "port": port}).Error("Connection: %s", err)
		return err
	}

	defer conn.Close()
	log.With(log.F{"host": host, "port": port}).Trace("Connected")

	for {
		buffer := make([]byte, bufferSize)
		recvLen, err := conn.Read(buffer)
		if err != nil && err != io.EOF {
			log.With(log.F{"host": host, "port": port}).Error("Server Error: %s", err)
			defer tcpConnect(host, port, timeout, bufferSize, KeepAlive, handler) // try reconnect
			break
		}

		if err == io.EOF {
			log.With(log.F{"host": host, "port": port}).Error("Connection lost")
			if KeepAlive == 0 { // try reconnect
				defer tcpConnect(host, port, timeout, bufferSize, KeepAlive, handler)
			}
			break
		}

		// receve data
		data := []byte{}
		data = append(data, buffer[:recvLen]...)
		log.With(log.F{"recvLen": recvLen, "data": fmt.Sprintf("%x", data)}).Trace("Receive")

		// handle data
		respMsg, err := handler(data, recvLen, err)
		if err != nil {
			log.With(log.F{"host": host, "port": port}).Error("Handler: %s", err)
		}
		log.With(log.F{"respMsg": fmt.Sprintf("%x", respMsg)}).Trace("Handler Response")

		// Send Response to Server
		if respMsg != nil {
			recvLen, err := conn.Write(respMsg)
			if err != nil {
				log.With(log.F{"data": fmt.Sprintf("%x", data)}).Error("Send Response: %s", err)
			}
			log.Trace("Send Response to server: %d", recvLen)
		}

		if KeepAlive == -1 {
			break
		}
	}

	return nil
}
