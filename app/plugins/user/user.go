package main

import (
	"encoding/json"
	"io"
	"os"
	"path"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/yaoapp/kun/grpc"
	"github.com/yaoapp/kun/maps"
)

// Model Here is a real implementation of KV that writes to a local file with
// the key name and the contents are the value of the key.
type Model struct {
	logger hclog.Logger
}

// Exec 读取
func (mod Model) Exec(name string, args ...interface{}) (*grpc.Response, error) {
	mod.logger.Debug("message from user.Exec")
	v := maps.MakeMap()
	v.Set("name", name)
	v.Set("args", args)
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &grpc.Response{Bytes: bytes}, nil
}

func main() {
	var output io.Writer = os.Stderr
	var logroot = os.Getenv("GOU_TEST_PLG_LOG")
	if logroot != "" {
		logfile, err := os.Create(path.Join(logroot, "user.log"))
		if err == nil {
			output = logfile
		}
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     output,
		JSONFormat: true,
	})
	model := &Model{logger: logger}
	pluginMap := map[string]plugin.Plugin{
		"model": &grpc.ModelGRPCPlugin{Impl: model},
	}

	logger.Debug("message from plugin", "foo", "bar")
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: grpc.Handshake,
		Plugins:         pluginMap,
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}
