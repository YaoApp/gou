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

// LoadWebSocket load websockets api
func LoadWebSocket(source string, name string) (*websocket.Upgrader, error) {
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

// SelectWebSocket Get WebSocket by name
func SelectWebSocket(name string) *websocket.Upgrader {
	ws, has := websocket.Upgraders[name]
	if !has {
		exception.New("WebSocket:%s does not load", 500, name).Throw()
	}
	return ws
}
