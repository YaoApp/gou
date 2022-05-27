package yao

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/runtime/yao/bridge"
	v8 "rogchap.com/v8go"
)

func TestMustAnyToValue(t *testing.T) {
	ctx := v8.NewContext()
	v := bridge.MustAnyToValue(ctx, 0.618)
	assert.True(t, v.IsNumber())
}
