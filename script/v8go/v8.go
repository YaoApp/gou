package v8go

import (
	"fmt"

	v8 "rogchap.com/v8go"
)

var iso = v8.NewIsolate()    // creates a new JavaScript VM
var ctx = v8.NewContext(iso) // new context within the VM

// Test 性能测试
func Test() interface{} {
	v, err := ctx.RunScript("function add(a, b){ return a + b; }; add(1,2);", "math.js")
	if err != nil {
		fmt.Printf("ERROR: %s", err.Error())
	}
	return v
}
