package v8

import (
	"testing"
	"time"

	"github.com/yaoapp/kun/log"
)

func TestSelectIsoStandard(t *testing.T) {
	option := option()
	option.Mode = "standard"
	option.HeapSizeLimit = 4294967296

	prepare(t, option)
	defer Stop()

	iso, err := SelectIsoStandard(time.Millisecond * 100)
	if err != nil {
		t.Fatal(err)
	}
	defer iso.Dispose()
}

// go test -bench=BenchmarkSelectIsoStandard
// go test -bench=BenchmarkSelectIsoStandard -benchmem -benchtime=5s
// go test -bench=BenchmarkSelectIsoStandard -benchtime=5s
func BenchmarkSelectIsoStandard(b *testing.B) {
	option := option()
	option.Mode = "standard"
	option.HeapSizeLimit = 4294967296

	b.ResetTimer()
	var t *testing.T
	prepare(t, option)
	defer Stop()
	log.SetLevel(log.FatalLevel)

	// run the Call function b.N times
	for n := 0; n < b.N; n++ {
		iso, err := SelectIsoStandard(500 * time.Millisecond)
		if err != nil {
			b.Fatal(err)
		}
		iso.Dispose()
	}
	b.StopTimer()
}

func BenchmarkSelectIsoStandardPB(b *testing.B) {
	option := option()
	option.Mode = "standard"
	option.HeapSizeLimit = 4294967296

	b.ResetTimer()
	var t *testing.T
	prepare(t, option)
	defer Stop()
	log.SetLevel(log.FatalLevel)

	// run the Call function b.N times
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			iso, err := SelectIsoStandard(500 * time.Millisecond)
			if err != nil {
				b.Fatal(err)
			}
			iso.Dispose()
		}
	})
	b.StopTimer()
}
