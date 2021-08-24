package gou

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/grpc"
)

// Plugins 已加载插件
var Plugins = map[string]*Plugin{}

// LoadPlugin 加载插件
func LoadPlugin(cmd string, name string) *Plugin {

	// We're a host. Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  grpc.Handshake,
		Plugins:          grpc.PluginMap,
		Cmd:              exec.Command("sh", "-c", cmd),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
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
	}

	Plugins[name] = p
	return p
}

// SelectPlugin 选择插件
func SelectPlugin(name string) grpc.Model {
	plugin, has := Plugins[name]
	if !has {
		exception.New(
			fmt.Sprintf("Plugin:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return plugin.Model
}
