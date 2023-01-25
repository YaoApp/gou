package v8

import (
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/kun/log"
)

func BenchmarkStd(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var _ string = fmt.Sprint(i)
	}
	b.StopTimer()
}

func BenchmarkStdPB(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var _ string = fmt.Sprint(i)
	}
	b.StopTimer()
}

func BenchmarkSelect(b *testing.B) {
	b.ResetTimer()
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
	b.StopTimer()
}

func BenchmarkSelectIso(b *testing.B) {
	b.ResetTimer()
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
	b.StopTimer()
}

func BenchmarkNewContent(b *testing.B) {
	b.ResetTimer()
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
	b.StopTimer()
}

func BenchmarkNewContentPB(b *testing.B) {
	b.ResetTimer()
	var t *testing.T
	prepare(t)
	Setup(50, 50)
	log.SetLevel(log.FatalLevel)

	basic, err := Select("runtime.basic")
	if err != nil {
		b.Fatal(err)
	}

	basic.Timeout = time.Millisecond * 500
	// run the Call function b.N times
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, err := basic.NewContent("SID_1010", map[string]interface{}{"name": "testing"})
			if err != nil {
				b.Fatal(err)
			}
			ctx.Close()
		}
	})

	b.StopTimer()
}

func BenchmarkCall(b *testing.B) {
	b.ResetTimer()
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
	b.StopTimer()
}

func BenchmarkCallPB(b *testing.B) {
	b.ResetTimer()
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

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err = ctx.Call("Hello", "world")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.StopTimer()
}
