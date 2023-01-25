package wasm

import (
	"testing"
)

func BenchmarkSelect(b *testing.B) {
	var t *testing.T
	prepare(t)

	// run the Call function b.N times
	for n := 0; n < b.N; n++ {
		_, err := Select("test")
		if err != nil {
			b.Fatal(err)
		}

	}
}
