package api

import (
	"testing"

	"github.com/yaoapp/gou/dsl"
)

func TestDSLCompilePing(t *testing.T) {
	// yao := newREST(t, "ping.flow.yao")
	// err := yao.Compile()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// flow, has := gou.Flows["ping"]
	// assert.Equal(t, true, has)
	// assert.Equal(t, "ping", flow.Name)
	// assert.Equal(t, "PONG", flow.Output)
}

func newREST(t *testing.T, name string) *dsl.YAO {
	return nil
}
