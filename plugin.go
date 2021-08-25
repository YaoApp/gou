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
var pluginLogger = hclog.New(&hclog.LoggerOptions{
	Name:   "plugin",
	Output: os.Stdout,
	Level:  hclog.Error,
})

// LoadPlugin 加载插件
func LoadPlugin(cmd string, name string) *Plugin {

	// 已载入，如果进程存在杀掉重载
	plug, has := Plugins[name]
	if has {
		if !plug.Client.Exited() {
			plug.Client.Kill()
		}
	}

	// We're a host. Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  grpc.Handshake,
		Plugins:          grpc.PluginMap,
		Cmd:              exec.Command("sh", "-c", cmd),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           pluginLogger,
	})

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

// KillPlugins 关闭插件进程
func KillPlugins() {
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

// SelectPlugin 选择插件
func SelectPlugin(name string) *Plugin {
	plug, has := Plugins[name]
	if !has {
		exception.New(
			fmt.Sprintf("Plugin:%s; 尚未加载", name),
			400,
		).Throw()
	}

	// 如果进程已退出，重载
	if plug.Client.Exited() {
		plug = LoadPlugin(plug.Cmd, plug.Name)
	}

	return plug
}

// SelectPluginModel 选择插件
func SelectPluginModel(name string) grpc.Model {
	plug := SelectPlugin(name)
	return plug.Model
}
