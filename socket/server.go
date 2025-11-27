package socket

import (
	"io"
	"net"

	"github.com/yaoapp/kun/log"
)

// Socket server (alpha -> will be refactored at a beta version...)

// Start start socket server
func Start(proto string, host string, port string, bufferSize int, KeepAlive int, handler func([]byte, int, error) ([]byte, error)) {
	if proto == "tcp" {
		tcpStart(host, port, bufferSize, KeepAlive, handler)
	}
}

// tcpStart start socket server with TCP/IP using TCP/IP protocol
func tcpStart(host string, port string, bufferSize int, KeepAlive int, handler func([]byte, int, error) ([]byte, error)) error {
	listen, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		log.Error("Start error: %s", err)
		return err
	}
	defer listen.Close()
	log.With(log.F{"bufferSize": bufferSize, "KeepAlive": KeepAlive}).Info("Listening ON: %s:%s", host, port)

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Error("Received error: %s", err)
			continue
		}
		log.With(log.F{"Remote": conn.RemoteAddr()}).Info("Connected")
		go handleRequest(conn, bufferSize, KeepAlive, handler)
	}
}

// handleRequest handel the client request
func handleRequest(conn net.Conn, bufferSize int, KeepAlive int, handler func([]byte, int, error) ([]byte, error)) {
	clientAddr := conn.RemoteAddr().String()
	defer conn.Close()

	log.Trace("Connection success. Client address: %s", clientAddr)
	for {
		buffer := make([]byte, bufferSize)
		recvLen, err := conn.Read(buffer)
		if err != nil && err != io.EOF {
			log.Error("Read error: %s %s", err.Error(), clientAddr)
			break
		}

		if err == io.EOF {
			log.With(log.F{"Remote": conn.RemoteAddr()}).Info("Connection closed")
			break
		}

		log.Trace("Message received from Client %s %x", clientAddr, buffer)

		res, err := handler(buffer[:recvLen], recvLen, err)
		if err != nil {
			log.Error("Handler error: %s %s", err, clientAddr)
			break
		}

		log.Trace("Handler response %s %x", clientAddr, res)

		// Send Response to Server
		if res != nil {
			if _, err := conn.Write(res); err != nil {
				log.Error("Send error: %s %s", err, clientAddr)
				break
			}
		}

		if KeepAlive == -1 {
			log.With(log.F{"Remote": conn.RemoteAddr()}).Info("Close connection")
			break
		}
	}
}
