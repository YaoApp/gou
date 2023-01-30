package plugin

import (
	"github.com/hashicorp/go-plugin"
	"github.com/yaoapp/kun/grpc"
)

// Plugin 插件
type Plugin struct {
	Client *plugin.Client
	Model  grpc.Model
	ID     string
	File   string
}
