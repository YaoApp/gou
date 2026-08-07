package mediaprobe

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func appendVINTID(buf *bytes.Buffer, id uint64) {
	switch {
	case id <= 0xFF:
		buf.WriteByte(byte(id))
	case id <= 0xFFFF:
		buf.WriteByte(byte(id >> 8))
		buf.WriteByte(byte(id))
	case id <= 0xFFFFFF:
		buf.WriteByte(byte(id >> 16))
		buf.WriteByte(byte(id >> 8))
		buf.WriteByte(byte(id))
	default:
		buf.WriteByte(byte(id >> 24))
		buf.WriteByte(byte(id >> 16))
		buf.WriteByte(byte(id >> 8))
		buf.WriteByte(byte(id))
	}
}

func appendVINTSize(buf *bytes.Buffer, size int) {
	if size < 0x80 {
		buf.WriteByte(byte(0x80 | size))
	} else if size < 0x4000 {
		buf.WriteByte(byte(0x40 | (size >> 8)))
		buf.WriteByte(byte(size))
	} else {
		buf.WriteByte(byte(0x20 | (size >> 16)))
		buf.WriteByte(byte(size >> 8))
		buf.WriteByte(byte(size))
	}
}

func buildWebMWithDuration(durationMS float64) []byte {
	var b bytes.Buffer

	// EBML Header
	appendVINTID(&b, ebmlHeaderID)
	appendVINTSize(&b, 5)
	b.Write(make([]byte, 5))

	// Segment (unknown size)
	appendVINTID(&b, ebmlSegment)
	b.Write([]byte{0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})

	// Info element
	var info bytes.Buffer
	appendVINTID(&info, ebmlTSScale)
	appendVINTSize(&info, 3)
	info.Write([]byte{0x0F, 0x42, 0x40}) // 1000000

	appendVINTID(&info, ebmlDuration)
	appendVINTSize(&info, 8)
	var dur [8]byte
	binary.BigEndian.PutUint64(dur[:], math.Float64bits(durationMS))
	info.Write(dur[:])

	appendVINTID(&b, ebmlInfo)
	appendVINTSize(&b, info.Len())
	b.Write(info.Bytes())

	return b.Bytes()
}

func buildWebMWithClusters(timestampsMS []uint64) []byte {
	var b bytes.Buffer

	appendVINTID(&b, ebmlHeaderID)
	appendVINTSize(&b, 5)
	b.Write(make([]byte, 5))

	appendVINTID(&b, ebmlSegment)
	b.Write([]byte{0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})

	// Info (no Duration)
	var info bytes.Buffer
	appendVINTID(&info, ebmlTSScale)
	appendVINTSize(&info, 3)
	info.Write([]byte{0x0F, 0x42, 0x40})
	appendVINTID(&b, ebmlInfo)
	appendVINTSize(&b, info.Len())
	b.Write(info.Bytes())

	for _, ts := range timestampsMS {
		var cluster bytes.Buffer
		appendVINTID(&cluster, ebmlTimestamp)
		if ts <= 0xFF {
			appendVINTSize(&cluster, 1)
			cluster.WriteByte(byte(ts))
		} else if ts <= 0xFFFF {
			appendVINTSize(&cluster, 2)
			cluster.WriteByte(byte(ts >> 8))
			cluster.WriteByte(byte(ts))
		} else {
			appendVINTSize(&cluster, 3)
			cluster.WriteByte(byte(ts >> 16))
			cluster.WriteByte(byte(ts >> 8))
			cluster.WriteByte(byte(ts))
		}

		appendVINTID(&b, ebmlCluster)
		appendVINTSize(&b, cluster.Len())
		b.Write(cluster.Bytes())
	}

	return b.Bytes()
}

func TestProbeWebM_Duration(t *testing.T) {
	data := buildWebMWithDuration(5123.0) // 5123 ms
	r := bytes.NewReader(data)

	info, err := probeWebM(r)
	if err != nil {
		t.Fatal(err)
	}

	if info.Format != "webm" {
		t.Errorf("format = %q, want webm", info.Format)
	}
	// 5123ms * 1000000ns/ms / 1e9 = 5.123s
	want := 5.123
	if math.Abs(info.Duration-want) > 0.001 {
		t.Errorf("duration = %v, want %v", info.Duration, want)
	}
}

func TestProbeWebM_Float32Duration(t *testing.T) {
	var b bytes.Buffer

	appendVINTID(&b, ebmlHeaderID)
	appendVINTSize(&b, 5)
	b.Write(make([]byte, 5))

	appendVINTID(&b, ebmlSegment)
	b.Write([]byte{0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})

	var info bytes.Buffer
	appendVINTID(&info, ebmlTSScale)
	appendVINTSize(&info, 3)
	info.Write([]byte{0x0F, 0x42, 0x40})

	appendVINTID(&info, ebmlDuration)
	appendVINTSize(&info, 4)
	var dur [4]byte
	binary.BigEndian.PutUint32(dur[:], math.Float32bits(3000.0))
	info.Write(dur[:])

	appendVINTID(&b, ebmlInfo)
	appendVINTSize(&b, info.Len())
	b.Write(info.Bytes())

	mi, err := probeWebM(bytes.NewReader(b.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	want := 3.0
	if math.Abs(mi.Duration-want) > 0.001 {
		t.Errorf("duration = %v, want %v", mi.Duration, want)
	}
}

func TestProbeWebM_ClusterFallback(t *testing.T) {
	data := buildWebMWithClusters([]uint64{0, 500, 1000, 3500})
	r := bytes.NewReader(data)

	info, err := probeWebM(r)
	if err != nil {
		t.Fatal(err)
	}
	// Last cluster at 3500ms → 3.5s
	want := 3.5
	if math.Abs(info.Duration-want) > 0.001 {
		t.Errorf("duration = %v, want %v", info.Duration, want)
	}
}

func TestProbeWebM_EmptyFile(t *testing.T) {
	_, err := probeWebM(bytes.NewReader(nil))
	if err == nil {
		t.Error("expected error for empty file")
	}
}

func TestProbeWebM_BadMagic(t *testing.T) {
	_, err := probeWebM(bytes.NewReader([]byte{0x00, 0x00, 0x00, 0x00}))
	if err == nil {
		t.Error("expected error for bad magic")
	}
}

func TestDetectFormat_WebM(t *testing.T) {
	data := buildWebMWithDuration(1000)
	r := bytes.NewReader(data)
	fmt := detectFormat(r, "")
	if fmt != "webm" {
		t.Errorf("detectFormat = %q, want webm", fmt)
	}
}

func TestNormalizeHint_WebM(t *testing.T) {
	cases := map[string]string{
		"webm":       "webm",
		"audio/webm": "webm",
		"video/webm": "webm",
	}
	for hint, want := range cases {
		if got := normalizeHint(hint); got != want {
			t.Errorf("normalizeHint(%q) = %q, want %q", hint, got, want)
		}
	}
}

func TestProbeReader_WebM(t *testing.T) {
	data := buildWebMWithDuration(2500)
	info, err := ProbeReader(bytes.NewReader(data), "webm")
	if err != nil {
		t.Fatal(err)
	}
	if info.Format != "webm" {
		t.Errorf("format = %q, want webm", info.Format)
	}
	want := 2.5
	if math.Abs(info.Duration-want) > 0.001 {
		t.Errorf("duration = %v, want %v", info.Duration, want)
	}
}
