package v8

import (
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/gou/process"
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
	i := 0
	b.RunParallel(func(pb *testing.PB) {
		i++
		for pb.Next() {
			var _ string = fmt.Sprint(i)
		}
	})
	b.StopTimer()
}

func BenchmarkSelect(b *testing.B) {
	b.ResetTimer()
	var t *testing.T
	prepare(t)
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
	log.SetLevel(log.FatalLevel)

	// run the Call function b.N times
	for n := 0; n < b.N; n++ {
		iso, err := SelectIso(500 * time.Millisecond)
		if err != nil {
			b.Fatal(err)
		}
		iso.Unlock()
	}
	b.StopTimer()
}

func BenchmarkSelectIsoPB(b *testing.B) {
	b.ResetTimer()
	var t *testing.T
	prepare(t)
	log.SetLevel(log.FatalLevel)

	// run the Call function b.N times
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			iso, err := SelectIso(500 * time.Millisecond)
			if err != nil {
				b.Fatal(err)
			}
			iso.Unlock()
		}
	})
	b.StopTimer()
}

func BenchmarkNewContext(b *testing.B) {
	b.ResetTimer()
	var t *testing.T
	prepare(t)
	log.SetLevel(log.FatalLevel)

	basic, err := Select("runtime.basic")
	if err != nil {
		b.Fatal(err)
	}

	basic.Timeout = time.Minute * 5

	// run the Call function b.N times
	for n := 0; n < b.N; n++ {
		ctx, err := basic.NewContext("SID_1010", map[string]interface{}{"name": "testing"})
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
	isolates.Resize(100, 100)
	log.SetLevel(log.FatalLevel)

	basic, err := Select("runtime.basic")
	if err != nil {
		b.Fatal(err)
	}

	basic.Timeout = time.Millisecond * 500
	// run the Call function b.N times
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, err := basic.NewContext("SID_1010", map[string]interface{}{"name": "testing"})
			if err != nil {
				b.Fatal(err)
			}
			ctx.Close()
		}
	})

	b.StopTimer()
}

func BenchmarkNewContentPBRelease(b *testing.B) {
	b.ResetTimer()
	var t *testing.T
	prepare(t)
	isolates.Resize(100, 100)
	log.SetLevel(log.FatalLevel)

	SetHeapAvailableSize(2018051350)
	defer SetHeapAvailableSize(524288000)

	DisablePrecompile()
	defer EnablePrecompile()

	basic, err := Select("runtime.basic")
	if err != nil {
		b.Fatal(err)
	}

	basic.Timeout = time.Millisecond * 500
	// run the Call function b.N times
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, err := basic.NewContext("SID_1010", map[string]interface{}{"name": "testing"})
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
	log.SetLevel(log.FatalLevel)

	basic, err := Select("runtime.basic")
	if err != nil {
		b.Fatal(err)
	}

	basic.Timeout = time.Minute * 5
	ctx, err := basic.NewContext("SID_1010", map[string]interface{}{"name": "testing"})
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

//
// func BenchmarkCallPB(b *testing.B) {
// 	b.ResetTimer()
// 	var t *testing.T
// 	prepare(t)
// 	isolates.Resize(100, 100)
// 	log.SetLevel(log.FatalLevel)

// 	basic, err := Select("runtime.basic")
// 	if err != nil {
// 		b.Fatal(err)
// 	}

// 	basic.Timeout = time.Minute * 5
// 	ctx, err := basic.NewContext("SID_1010", map[string]interface{}{"name": "testing"})
// 	if err != nil {
// 		b.Fatal(err)
// 	}
// 	defer ctx.Close()

// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			_, err = ctx.Call("Hello", "world")
// 			if err != nil {
// 				b.Fatal(err)
// 			}
// 		}
// 	})

// 	b.StopTimer()
// }

func BenchmarkProcessScripts(b *testing.B) {
	b.ResetTimer()
	var t *testing.T
	prepare(t)

	p, err := process.Of("scripts.runtime.basic.Hello", "world")
	if err != nil {
		t.Fatal(err)
	}

	// run the Call function b.N times
	for n := 0; n < b.N; n++ {
		_, err := p.Exec()
		if err != nil {
			t.Fatal(err)
		}
	}
	b.StopTimer()
}

func BenchmarkProcessScriptsPB(b *testing.B) {
	b.ResetTimer()
	var t *testing.T
	prepare(t)
	isolates.Resize(100, 100)

	p, err := process.Of("scripts.runtime.basic.Hello", "world")
	if err != nil {
		t.Fatal(err)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := p.Exec()
			if err != nil {
				t.Fatal(err)
			}
		}
	})

	b.StopTimer()
}
