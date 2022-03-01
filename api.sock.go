package gou

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/yaoapp/kun/log"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/socket"
	"github.com/yaoapp/kun/exception"
)

// Sockets sockets loaded (Alpha)
var Sockets = map[string]*Socket{}

// LoadSocket load socket server/client
func LoadSocket(source string, name string) (*Socket, error) {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}
	sock := Socket{}
	err := helper.UnmarshalFile(input, &sock)
	if err != nil {
		return nil, err
	}
	Sockets[name] = &sock
	return Sockets[name], nil
}

// SelectSocket Get socket by name
func SelectSocket(name string) *Socket {
	sock, has := Sockets[name]
	if !has {
		exception.New("Socket:%s does not load", 500, name).Throw()
	}
	return sock
}

// Start Start server
func (sock Socket) Start(args ...interface{}) {
	socket.Start(sock.Protocol, sock.Host, sock.Port, sock.BufferSize, sock.KeepAlive, func(data []byte, recvLen int, err error) ([]byte, error) {
		res, err := NewProcess(sock.Process, hex.EncodeToString(data)).Exec()
		if err != nil {
			log.Error(err.Error())
			return nil, err
		}
		switch res.(type) {
		case []byte:
			return res.([]byte), nil
		case string:
			return []byte(res.(string)), nil
		case interface{}:
			v := fmt.Sprintf("%v", res)
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s response data type error", sock.Process)
	})
}

// Connect Connect to server
func (sock Socket) Connect(args ...interface{}) error {
	host := sock.Host
	port := sock.Port
	argsLen := len(args)
	if argsLen > 0 {
		if inputHost, ok := args[0].(string); ok {
			host = inputHost
		}
	}

	if argsLen > 1 {
		if inputPort, ok := args[1].(string); ok {
			port = inputPort
		}
	}

	return socket.Connect(
		sock.Protocol, host, port,
		time.Duration(sock.Timeout)*time.Second,
		sock.BufferSize,
		time.Duration(sock.KeepAlive)*time.Second,
		func(data []byte, recvLen int, err error) ([]byte, error) {
			res, err := NewProcess(sock.Process, hex.EncodeToString(data)).Exec()
			if err != nil {
				return nil, err
			}

			switch res.(type) {
			case []byte:
				return res.([]byte), nil
			case string:
				return []byte(res.(string)), nil
			case interface{}:
				v := fmt.Sprintf("%v", res)
				return []byte(v), nil
			}

			return nil, fmt.Errorf("%s response data type error", sock.Process)
		})
}
