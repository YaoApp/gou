package v8go

import (
	"fmt"
	"testing"
)

// go test -v -bench=Test  -benchmem
func BenchmarkTest(b *testing.B) {
	fmt.Println(b.N)
	for i := 0; i < 10000; i++ {
		Test()
	}
}

// 4416723311
// 1068377489
