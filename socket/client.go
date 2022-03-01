package socket

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/yaoapp/kun/log"
)

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
