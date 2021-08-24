package main

import (
	"encoding/json"

	"github.com/hashicorp/go-plugin"

	"github.com/yaoapp/kun/grpc"
	"github.com/yaoapp/kun/maps"
)

// Model Here is a real implementation of KV that writes to a local file with
// the key name and the contents are the value of the key.
type Model struct{}

// Put 写入
func (mod Model) Put(name string, payload []byte) error {
	v := maps.MakeMapStr()
	err := json.Unmarshal(payload, &v)
	if err != nil {
		return err
	}
	return nil
}

// Get 读取
func (mod Model) Get(name string, payload []byte) ([]byte, error) {
	v := maps.MakeMapStr()
	err := json.Unmarshal(payload, &v)
	if err != nil {
		return nil, err
	}
	v.Set("name", name)
	return json.Marshal(v)
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: grpc.Handshake,
		Plugins: map[string]plugin.Plugin{
			"model": &grpc.ModelGRPCPlugin{Impl: &Model{}},
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
