package gou

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/websocket"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

func init() {
	RegisterProcessHandler("websocket.Broadcast", processBroadcast)
}

// LoadWebSocketServer load websocket servers
func LoadWebSocketServer(source string, name string) (*websocket.Upgrader, error) {
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

	config, err := helper.ReadFile(input)
	if err != nil {
		return nil, err
	}
	ws, err := websocket.NewUpgrader(name, config)
	if err != nil {
		return nil, err
	}

	// SetHandler
	ws.SetHandler(func(message []byte) ([]byte, error) {
		response, err := NewProcess(ws.Process, message).Exec()
		if err != nil {
			log.Error("Websocket: %s %s", name, err.Error())
			return nil, err
		}
		switch response.(type) {
		case string:
			return []byte(response.(string)), nil
		case []byte:
			return response.([]byte), nil
		default:
			message := fmt.Sprintf("Websocket: %s response message dose not support", name)
			log.Error(message)
			return nil, fmt.Errorf(message)
		}
	})

	return ws, nil
}

// SelectWebSocketServer Get WebSocket server by name
func SelectWebSocketServer(name string) *websocket.Upgrader {
	ws, has := websocket.Upgraders[name]
	if !has {
		exception.New("WebSocket:%s does not load", 500, name).Throw()
	}
	return ws
}

// processBroadcast WebSocket Server broadcast the message
func processBroadcast(process *Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	message := process.ArgsString(1)
	ws := SelectWebSocketServer(name)
	ws.Broadcast([]byte(message))
	return nil
}
