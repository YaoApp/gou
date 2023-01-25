package wasm

import (
	"github.com/yaoapp/gou/runtime/wasm/wamr"
)

// Instance the wamr instance
type Instance struct {
	File         string
	wamrInstance *wamr.Instance
}
