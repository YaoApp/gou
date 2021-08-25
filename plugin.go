package gou

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/grpc"
)

// Plugins 已加载插件
var Plugins = map[string]*Plugin{}

// Create an hclog.Logger
var logger = hclog.New(&hclog.LoggerOptions{
	Name:   "plugin",
	Output: os.Stdout,
	Level:  hclog.Error,
})

// LoadPlugin 加载插件
func LoadPlugin(cmd string, name string) *Plugin {

	var client *plugin.Client = nil

	// 已载入
	plug, has := Plugins[name]
	if has {
		reattachConfig := plug.Client.ReattachConfig()
		client = plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig:  grpc.Handshake,
			Plugins:          grpc.PluginMap,
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			Logger:           logger,
			Reattach:         reattachConfig,
		})

	} else {

		// We're a host. Start by launching the plugin process.
		client = plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig:  grpc.Handshake,
			Plugins:          grpc.PluginMap,
			Cmd:              exec.Command("sh", "-c", cmd),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			Logger:           logger,
		})
	}

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("model")
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	mod := raw.(grpc.Model)
	p := &Plugin{
		Client: client,
		Model:  mod,
		Name:   name,
		Cmd:    cmd,
	}

	Plugins[name] = p
	return p
}

// SetPluginLogger 设置日志
func SetPluginLogger(name string, output io.Writer, level hclog.Level) {
	logger = hclog.New(&hclog.LoggerOptions{
		Name:   name,
		Output: output,
		Level:  level,
	})
}

// SelectPlugin 选择插件
func SelectPlugin(name string) *Plugin {
	plugin, has := Plugins[name]
	if !has {
		exception.New(
			fmt.Sprintf("Plugin:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return plugin
}

// SelectPluginModel 选择插件
func SelectPluginModel(name string) grpc.Model {
	plugin, has := Plugins[name]
	if !has {
		exception.New(
			fmt.Sprintf("Plugin:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return plugin.Model
}
