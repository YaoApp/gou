package gou

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
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
	socket.Start(sock.Protocol, sock.Host, sock.Port, sock.BufferSize, sock.KeepAlive, func(data []byte, _ int, err error) ([]byte, error) {
		res, err := NewProcess(sock.Event.Data, hex.EncodeToString(data)).Exec()
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
		return nil, fmt.Errorf("%s response data type error", sock.Event.Data)
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
		func(data []byte, _ int, err error) ([]byte, error) {
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

			return nil, fmt.Errorf("%s response data type error", sock.Event.Data)
		})
}

// Open Connect the socket server
func (sock Socket) Open(args ...interface{}) error {
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

	client := socket.NewClient(
		socket.Option{
			Host:         host,
			Port:         port,
			Timeout:      time.Duration(sock.Timeout) * time.Second,
			KeepAlive:    time.Duration(sock.KeepAlive) * time.Second,
			BufferSize:   sock.BufferSize,
			Protocol:     sock.Protocol,
			Attempts:     sock.Attempts,
			AttemptAfter: time.Duration(sock.AttemptAfter) * time.Second,
		},
		socket.Handlers{
			Connected: sock.onConnected,
			Closed:    sock.onClosed,
			Data:      sock.onData,
			Error:     sock.onError,
		})

	return client.Open()
}

func (sock Socket) onClosed(data []byte, err error) []byte {
	if sock.Event.Closed == "" {
		return nil
	}

	var msg = ""
	if data != nil {
		msg = string(data)
	}

	errstr := ""
	if err != nil {
		errstr = err.Error()
	}

	p, err := ProcessOf(sock.Event.Closed, msg, errstr)
	if err != nil {
		log.Error("sock.Event.Closed Error: %s", err)
		return nil
	}

	res, err := p.Exec()
	if err != nil {
		log.Error("sock.Event.Closed Error: %s", err)
		return nil
	}

	return sock.toBytes(res, "sock.Event.Closed")
}

func (sock Socket) onData(data []byte, recvLen int) ([]byte, error) {

	if sock.Event.Data == "" {
		return nil, nil
	}

	p, err := ProcessOf(sock.Event.Data, hex.EncodeToString(data), recvLen)
	if err != nil {
		return nil, err
	}

	res, err := p.Exec()
	if err != nil {
		return nil, err
	}

	return sock.toBytes(res, "sock.Event.Data"), nil
}

func (sock Socket) onError(err error) {
	if sock.Event.Error == "" {
		return
	}

	p, err := ProcessOf(sock.Event.Error, err.Error())
	if err != nil {
		log.Error("sock.Event.Error Error: %s", err.Error())
	}

	_, err = p.Exec()
	if err != nil {
		log.Error("sock.Event.Error Error: %s", err.Error())
	}
}

func (sock Socket) onConnected(option socket.Option) error {
	if sock.Event.Connected == "" {
		return nil
	}

	p, err := ProcessOf(sock.Event.Connected, option)
	if err != nil {
		return err
	}

	_, err = p.Exec()
	return err
}

func (sock Socket) toBytes(value interface{}, name string) []byte {
	if value == nil {
		return nil
	}

	switch value.(type) {
	case []byte:
		return value.([]byte)

	case string:
		if value.(string) == "" {
			return nil
		}

		bytes, err := hex.DecodeString(value.(string))
		if err != nil {
			log.Error("%s Error: %s", name, err.Error())
			return nil
		}

		return bytes

	default:
		v, err := jsoniter.Marshal(value)
		if err != nil {
			log.Error("%s Error: %s", name, err.Error())
		}
		return v
	}
}
