package wasm

import (
	"fmt"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/runtime/wasm/wamr"
	"github.com/yaoapp/kun/log"
)

// ********************************************************
// WARNING: EXPERIMENTAL DO NOT USE IN PRODUCTION
// ********************************************************

// Instances the wams instance
var Instances = map[string]*Instance{}
var heap = []byte{}

func init() {
	err := wamr.Runtime().FullInit(false, nil, 1)
	if err != nil {
		log.Error("[wasm] runtime init error :%s", err)
	}
}

// NewInstance create a new instance
func NewInstance(file string, wamrInstance *wamr.Instance) *Instance {
	return &Instance{File: file, wamrInstance: wamrInstance}
}

// Load the warm instance
func Load(file string, id string) (*Instance, error) {
	wasmBytes, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	module, err := wamr.NewModule(wasmBytes)
	if err != nil {
		return nil, err
	}

	wasmInstance, err := wamr.NewInstance(module, 10240, 10240)
	if err != nil {
		return nil, err
	}

	instance := NewInstance(file, wasmInstance)
	Instances[id] = instance
	return Instances[id], nil
}

// Select a warm intance
func Select(id string) (*Instance, error) {
	instance, has := Instances[id]
	if !has {
		return nil, fmt.Errorf("[Wasm] %s does not exist", id)
	}
	return instance, nil
}

// func main() {

// 	var module *wamr.Module
// 	var instance *wamr.Instance
// 	var wasmBytes []byte

// 	// Code
// 	wasmFile := filepath.Join(os.Getenv("GOU_TEST_APPLICATION"), "scripts", "test.wasm")
// 	wasmBytes, err := os.ReadFile(wasmFile)
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}

// 	/* Runtime initialization */
// 	err = wamr.Runtime().FullInit(false, nil, 1)
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}

// 	/* Load WASM/AOT module from the memory buffer */
// 	module, err = wamr.NewModule(wasmBytes)

// 	/* Create WASM/AOT instance from the module */
// 	instance, err = wamr.NewInstance(module, 16384, 16384)
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}

// 	fmt.Println("-add----")
// 	err = instance.CallFunc("add", 2, []uint32{2, 3})
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	fmt.Println("-----")

// 	fmt.Println()
// 	fmt.Println("-add return----")
// 	results := []interface{}{nil}
// 	err = instance.CallFuncV("add", 1, results, int32(2), int32(3))
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	var value int32 = results[0].(int32)
// 	fmt.Println("value=", value)
// 	fmt.Println("-----")

// 	/* Destroy runtime */
// 	wamr.Runtime().Destroy()
// }
