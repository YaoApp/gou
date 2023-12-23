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

func TestSelectSelectIsoPerformance(t *testing.T) {
	option := option()
	option.Mode = "performance"
	option.HeapSizeLimit = 4294967296

	prepare(t, option)
	defer Stop()

	runtimeOption.Mode = "performance"
	runtimeOption.HeapSizeLimit = 4294967296
	iso, err := SelectIsoPerformance(time.Millisecond * 100)
	if err != nil {
		t.Fatal(err)
	}
	defer iso.Dispose()
}

// go test -bench=BenchmarkSelectIsoPerformance
// go test -bench=BenchmarkSelectIsoPerformance -benchmem -benchtime=5s
// go test -bench=BenchmarkSelectIsoPerformance -benchtime=5s
func BenchmarkSelectIsoPerformance(b *testing.B) {
	option := option()
	option.MinSize = 10
	option.MaxSize = 100
	option.Mode = "performance"
	option.HeapSizeLimit = 4294967296

	b.StartTimer()
	var t *testing.T
	prepare(t, option)
	defer Stop()
	log.SetLevel(log.FatalLevel)

	// Report memory allocations
	b.ReportAllocs()

	// run the Call function b.N times
	for n := 0; n < b.N; n++ {
		iso, err := SelectIsoPerformance(500 * time.Millisecond)
		if err != nil {
			b.Fatal(err)
		}
		Unlock(iso)
	}
	b.StopTimer()
}

func BenchmarkSelectIsoPerformancePB(b *testing.B) {
	option := option()
	option.MinSize = 60
	option.MaxSize = 100
	option.Mode = "performance"
	option.HeapSizeLimit = 4294967296

	b.ResetTimer()
	var t *testing.T
	prepare(t, option)
	defer Stop()
	log.SetLevel(log.FatalLevel)

	// run the Call function b.N times
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			iso, err := SelectIsoPerformance(500 * time.Millisecond)
			if err != nil {
				b.Fatal(err)
			}
			Unlock(iso)
		}
	})
	b.StopTimer()
}

func BenchmarkSelectIsoPerformanceUnhealth(b *testing.B) {

	log.SetLevel(log.FatalLevel)
	option := option()
	option.MinSize = 10
	option.MaxSize = 100
	option.Mode = "performance"
	option.HeapAvailableSize = 1024 * 1024 * 5000
	option.HeapSizeLimit = 4294967296

	b.StartTimer()
	var t *testing.T
	prepare(t, option)
	defer Stop()

	// Report memory allocations
	b.ReportAllocs()

	// run the Call function b.N times
	for n := 0; n < b.N; n++ {
		iso, err := SelectIsoPerformance(500 * time.Millisecond)
		if err != nil {
			b.Fatal(err)
		}
		Unlock(iso)
	}
	b.StopTimer()
}

func BenchmarkSelectIsoPerformanceUnhealthPB(b *testing.B) {

	log.SetLevel(log.FatalLevel)
	option := option()
	option.MinSize = 10
	option.MaxSize = 100
	option.Mode = "performance"
	option.HeapAvailableSize = 1024 * 1024 * 5000
	option.HeapSizeLimit = 4294967296

	b.StartTimer()
	var t *testing.T
	prepare(t, option)
	defer Stop()

	// Report memory allocations
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			iso, err := SelectIsoPerformance(500 * time.Millisecond)
			if err != nil {
				b.Fatal(err)
			}
			Unlock(iso)
		}
	})
	b.StopTimer()
}
