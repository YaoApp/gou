package otto

import (
	"fmt"

	"github.com/robertkrimen/otto"
)

var vm = otto.New()

// Test 性能测试
func Test() interface{} {
	v, err := vm.Run("function add(a, b){ return a + b; }; add(1,2);")
	if err != nil {
		fmt.Printf("ERROR: %s", err.Error())
	}
	return v
}
