package gou

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yaoapp/kun/log"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/socket"
	"github.com/yaoapp/kun/exception"
)

// Servers 已加载的Socket服务器 (Alpha)
var Servers = map[string]*SocketServer{}

// LoadServer 加载服务器配置
func LoadServer(source string, name string) (*SocketServer, error) {
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
	srv := SocketServer{}
	err := helper.UnmarshalFile(input, &srv)
	if err != nil {
		return nil, err
	}
	Servers[name] = &srv
	return Servers[name], nil
}

// SelectServer 读取已加载Socket 服务器
func SelectServer(name string) *SocketServer {
	srv, has := Servers[name]
	if !has {
		exception.New(
			fmt.Sprintf("Socket Server:%s; 尚未加载", name),
			500,
		).Throw()
	}
	return srv
}

// Start 启动服务
func (srv SocketServer) Start() {
	socket.Start(srv.Protocol, srv.Host, srv.Port, func(data []byte) []byte {
		// fmt.Printf("%#v\n", data)
		res, err := NewProcess(srv.Process, hex.EncodeToString(data)).Exec()
		if err != nil {
			log.Error(err.Error())
			return nil
		}
		switch res.(type) {
		case []byte:
			return res.([]byte)
		case string:
			return []byte(res.(string))
		case interface{}:
			v := fmt.Sprintf("%v", res)
			return []byte(v)
		}
		return nil
	})
}
