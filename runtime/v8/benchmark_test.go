package v8

import (
	"testing"
	"time"

	"github.com/yaoapp/kun/log"
)

func BenchmarkSelect(b *testing.B) {
	var t *testing.T
	prepare(t)
	// Setup(10, 100)
	log.SetLevel(log.FatalLevel)

	// run the Call function b.N times
	for n := 0; n < b.N; n++ {
		_, err := Select("runtime.basic")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSelectIso(b *testing.B) {
	var t *testing.T
	prepare(t)
	// Setup(10, 100)
	log.SetLevel(log.FatalLevel)

	// run the Call function b.N times
	for n := 0; n < b.N; n++ {
		iso, err := SelectIso(100 * time.Millisecond)
		if err != nil {
			b.Fatal(err)
		}
		iso.Unlock()
	}
}

func BenchmarkNewContent(b *testing.B) {
	var t *testing.T
	prepare(t)
	// Setup(50, 100)
	log.SetLevel(log.FatalLevel)

	basic, err := Select("runtime.basic")
	if err != nil {
		b.Fatal(err)
	}

	basic.Timeout = time.Minute * 5

	// run the Call function b.N times
	for n := 0; n < b.N; n++ {
		ctx, err := basic.NewContent("SID_1010", map[string]interface{}{"name": "testing"})
		if err != nil {
			b.Fatal(err)
		}
		ctx.Close()
	}
}

func BenchmarkCall(b *testing.B) {
	var t *testing.T
	prepare(t)
	// Setup(100, 300)
	log.SetLevel(log.FatalLevel)

	basic, err := Select("runtime.basic")
	if err != nil {
		b.Fatal(err)
	}

	basic.Timeout = time.Minute * 5
	ctx, err := basic.NewContent("SID_1010", map[string]interface{}{"name": "testing"})
	if err != nil {
		b.Fatal(err)
	}
	defer ctx.Close()

	// run the Call function b.N times
	for n := 0; n < b.N; n++ {
		_, err = ctx.Call("Hello", "world")
		if err != nil {
			b.Fatal(err)
		}
	}
}
