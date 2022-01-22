package script

import (
	"testing"

	"github.com/yaoapp/gou/script/otto"
	"github.com/yaoapp/gou/script/v8go"
)

// go test -cpu 1,2,4,8,16 -bench=Test  -benchmem
func BenchmarkTestOtto(b *testing.B) {
	for i := 0; i < b.N; i++ {
		otto.Test()
	}
}

func BenchmarkTestV8go(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v8go.Test()
	}
}

func TestTestV8go(t *testing.T) {
	v8go.Test()
}

func TestTestOtto(t *testing.T) {
	otto.Test()
}
