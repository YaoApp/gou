package api

import (
	"io"
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/websocket"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// WebSockets websockets loaded (Alpha)
var WebSockets = map[string]*WebSocket{}

// LoadWebSocket load websockets client
func LoadWebSocket(source string, name string) (*WebSocket, error) {
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

	ws := WebSocket{}
	err = jsoniter.Unmarshal(config, &ws)
	if err != nil {
		return nil, err
	}

	handers := websocket.Handlers{
		Connected: ws.onConnected,
		Closed:    ws.onClosed,
		Data:      ws.onData,
		Error:     ws.onError,
	}

	ws.WSClientOption.Name = name
	ws.Client = websocket.NewWSClient(ws.WSClientOption, handers)
	WebSockets[name] = &ws
	return WebSockets[name], nil
}

// SelectWebSocket Get WebSocket client by name
func SelectWebSocket(name string) *WebSocket {
	ws, has := WebSockets[name]
	if !has {
		exception.New("WebSocket Client:%s does not load", 500, name).Throw()
	}
	return ws
}

// Open open the websocket connection
func (ws *WebSocket) Open(args ...string) error {
	if len(args) > 0 {
		ws.Client.SetURL(args[0])
	}
	if len(args) > 1 {
		ws.Client.SetProtocols(args[1:]...)
	}

	return ws.Client.Open()
}

// processWrite WebSocket client send message to server
func processWrite(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	message := process.ArgsString(1)
	ws := SelectWebSocket(name)
	err := ws.Client.Write([]byte(message))
	if err != nil {
		return map[string]interface{}{"code": 500, "message": err.Error()}
	}
	return nil
}

// processClose WebSocket client close the connection
func processClose(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	ws := SelectWebSocket(name)
	err := ws.Client.Close()
	if err != nil {
		return map[string]interface{}{"code": 500, "message": err.Error()}
	}
	return nil
}

func (ws WebSocket) onConnected(option websocket.WSClientOption) error {
	if ws.Event.Connected == "" {
		return nil
	}

	p, err := process.Of(ws.Event.Connected, option)
	if err != nil {
		return err
	}

	_, err = p.Exec()
	return err
}

func (ws WebSocket) onClosed(data []byte, err error) []byte {
	if ws.Event.Closed == "" {
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

	p, err := process.Of(ws.Event.Closed, msg, errstr)
	if err != nil {
		log.Error("ws.Event.Closed Error: %s", err)
		return nil
	}

	res, err := p.Exec()
	if err != nil {
		log.Error("ws.Event.Closed Error: %s", err)
		return nil
	}

	return ws.toBytes(res, "ws.Event.Closed")
}

func (ws WebSocket) onData(data []byte, recvLen int) ([]byte, error) {
	if ws.Event.Data == "" {
		return nil, nil
	}
	p, err := process.Of(ws.Event.Data, string(data), recvLen)
	if err != nil {
		return nil, err
	}
	res, err := p.Exec()
	if err != nil {
		return nil, err
	}
	return ws.toBytes(res, "ws.Event.Data"), nil
}

func (ws WebSocket) onError(err error) {
	if ws.Event.Error == "" {
		return
	}

	p, err := process.Of(ws.Event.Error, err.Error())
	if err != nil {
		log.Error("ws.Event.Error Error: %s", err.Error())
	}

	_, err = p.Exec()
	if err != nil {
		log.Error("ws.Event.Error Error: %s", err.Error())
	}
}

func (ws WebSocket) toBytes(value interface{}, name string) []byte {
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
		return []byte(value.(string))

	default:
		v, err := jsoniter.Marshal(value)
		if err != nil {
			log.Error("%s Error: %s", name, err.Error())
		}
		return v
	}
}
