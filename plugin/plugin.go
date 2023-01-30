package plugin

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/yaoapp/kun/grpc"
)

// Plugins 已加载插件
var Plugins = map[string]*Plugin{}

// Create an hclog.Logger
var pluginLogger = hclog.New(&hclog.LoggerOptions{
	Name:   "plugin",
	Output: os.Stdout,
	Level:  hclog.Error,
})

// Load a plugin
func Load(file string, id string) (*Plugin, error) {

	// 已载入，如果进程存在杀掉重载
	plug, has := Plugins[id]
	if has {
		if !plug.Client.Exited() {
			plug.Client.Kill()
		}
	}

	// We're a host. Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  grpc.Handshake,
		Plugins:          grpc.PluginMap,
		Cmd:              exec.Command(file),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           pluginLogger,
	})

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		return nil, fmt.Errorf("%s(%s) %s", id, file, err.Error())
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("model")
	if err != nil {
		return nil, fmt.Errorf("%s(%s) %s", id, file, err.Error())
	}

	mod := raw.(grpc.Model)
	p := &Plugin{
		Client: client,
		Model:  mod,
		ID:     id,
		File:   file,
	}

	Plugins[id] = p
	return p, nil
}

// KillAll kill all loaded plugins
func KillAll() {
	for _, plug := range Plugins {
		if !plug.Client.Exited() {
			plug.Client.Kill()
		}
	}
}

// SetPluginLogger 设置日志
func SetPluginLogger(name string, output io.Writer, level hclog.Level) {
	pluginLogger = hclog.New(&hclog.LoggerOptions{
		Name:   name,
		Output: output,
		Level:  level,
	})
}

// Select a plugin
func Select(id string) (grpc.Model, error) {
	var err error
	plug, has := Plugins[id]
	if !has {
		return nil, fmt.Errorf("plugin %s not loaded", id)

	}

	// 如果进程已退出，重载
	if plug.Client.Exited() {
		plug, err = Load(plug.File, plug.ID)
		if err != nil {
			return nil, fmt.Errorf("%s %s", id, err)
		}
	}
	return plug.Model, nil
}

// Kill a plugin process
func (plugin *Plugin) Kill() {
	plugin.Client.Kill()
}
