package mediaprobe

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"testing"
)

func buildAtom(typ string, payload []byte) []byte {
	size := 8 + len(payload)
	buf := make([]byte, size)
	binary.BigEndian.PutUint32(buf[0:4], uint32(size))
	copy(buf[4:8], typ)
	copy(buf[8:], payload)
	return buf
}

func buildAtomExtended(typ string, payload []byte) []byte {
	totalSize := 16 + len(payload)
	buf := make([]byte, totalSize)
	binary.BigEndian.PutUint32(buf[0:4], 1)
	copy(buf[4:8], typ)
	binary.BigEndian.PutUint64(buf[8:16], uint64(totalSize))
	copy(buf[16:], payload)
	return buf
}

func buildMVHD(version byte, timescale uint32, duration uint64) []byte {
	if version == 0 {
		payload := make([]byte, 24)
		payload[0] = version
		// payload[1:4] = flags (zero)
		binary.BigEndian.PutUint32(payload[12:16], timescale)
		binary.BigEndian.PutUint32(payload[16:20], uint32(duration))
		return buildAtom("mvhd", payload)
	}
	payload := make([]byte, 32)
	payload[0] = version
	// payload[1:4] = flags (zero)
	binary.BigEndian.PutUint32(payload[20:24], timescale)
	binary.BigEndian.PutUint64(payload[24:32], duration)
	return buildAtom("mvhd", payload)
}

func buildTKHD(version byte, width, height uint32) []byte {
	skip := 68
	if version == 1 {
		skip = 80
	}
	payload := make([]byte, 4+skip+8)
	payload[0] = version
	// payload[1:4] = flags (zero)
	w := make([]byte, 4)
	binary.BigEndian.PutUint32(w, width<<16)
	h := make([]byte, 4)
	binary.BigEndian.PutUint32(h, height<<16)
	copy(payload[4+skip:], w)
	copy(payload[4+skip+4:], h)
	return buildAtom("tkhd", payload)
}

func buildContainerAtom(typ string, children ...[]byte) []byte {
	var payload []byte
	for _, c := range children {
		payload = append(payload, c...)
	}
	return buildAtom(typ, payload)
}

func buildMinimalMP4(timescale uint32, duration uint64) []byte {
	ftyp := buildAtom("ftyp", append([]byte("isom"), 0, 0, 0, 0, 'i', 's', 'o', 'm', 0, 0, 0, 0))
	mvhd := buildMVHD(0, timescale, duration)
	moov := buildContainerAtom("moov", mvhd)
	return append(ftyp, moov...)
}

func buildMP4WithVideo(timescale uint32, duration uint64, width, height uint32) []byte {
	ftyp := buildAtom("ftyp", append([]byte("isom"), 0, 0, 0, 0))
	mvhd := buildMVHD(0, timescale, duration)
	tkhd := buildTKHD(0, width, height)
	trak := buildContainerAtom("trak", tkhd)
	moov := buildContainerAtom("moov", mvhd, trak)
	return append(ftyp, moov...)
}

func buildMP4Version1(timescale uint32, duration uint64) []byte {
	ftyp := buildAtom("ftyp", []byte("isom"))
	mvhd := buildMVHD(1, timescale, duration)
	moov := buildContainerAtom("moov", mvhd)
	return append(ftyp, moov...)
}

func TestProbeMP4(t *testing.T) {
	t.Run("audio only", func(t *testing.T) {
		data := buildMinimalMP4(1000, 5000)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if info.Format != "mp4" {
			t.Errorf("Format = %q, want mp4", info.Format)
		}
		if math.Abs(info.Duration-5.0) > 0.01 {
			t.Errorf("Duration = %v, want 5.0", info.Duration)
		}
		if info.Width != 0 || info.Height != 0 {
			t.Errorf("dimensions = %dx%d, want 0x0", info.Width, info.Height)
		}
	})

	t.Run("video with dimensions", func(t *testing.T) {
		data := buildMP4WithVideo(1000, 10000, 1920, 1080)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if info.Width != 1920 || info.Height != 1080 {
			t.Errorf("dimensions = %dx%d, want 1920x1080", info.Width, info.Height)
		}
		if math.Abs(info.Duration-10.0) > 0.01 {
			t.Errorf("Duration = %v, want 10.0", info.Duration)
		}
	})

	t.Run("mvhd version 0", func(t *testing.T) {
		data := buildMinimalMP4(44100, 88200)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if math.Abs(info.Duration-2.0) > 0.01 {
			t.Errorf("Duration = %v, want 2.0", info.Duration)
		}
	})

	t.Run("mvhd version 1", func(t *testing.T) {
		data := buildMP4Version1(1000, 3000)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if math.Abs(info.Duration-3.0) > 0.01 {
			t.Errorf("Duration = %v, want 3.0", info.Duration)
		}
	})

	t.Run("tkhd version 1", func(t *testing.T) {
		mvhd := buildMVHD(0, 1000, 1000)
		tkhd := buildTKHD(1, 640, 480)
		trak := buildContainerAtom("trak", tkhd)
		moov := buildContainerAtom("moov", mvhd, trak)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if info.Width != 640 || info.Height != 480 {
			t.Errorf("dimensions = %dx%d, want 640x480", info.Width, info.Height)
		}
	})

	t.Run("extended atom size", func(t *testing.T) {
		mvhd := buildMVHD(0, 1000, 2000)
		moov := buildAtomExtended("moov", mvhd)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if math.Abs(info.Duration-2.0) > 0.01 {
			t.Errorf("Duration = %v, want 2.0", info.Duration)
		}
	})

	t.Run("size zero atom", func(t *testing.T) {
		mvhd := buildMVHD(0, 1000, 3000)
		moovPayload := mvhd
		moovSize := uint32(8 + len(moovPayload))
		moov := make([]byte, moovSize)
		binary.BigEndian.PutUint32(moov[0:4], moovSize)
		copy(moov[4:8], "moov")
		copy(moov[8:], moovPayload)
		free := make([]byte, 8)
		copy(free[4:8], "free")
		// size 0 extends to end of file
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		data = append(data, free...)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if math.Abs(info.Duration-3.0) > 0.01 {
			t.Errorf("Duration = %v, want 3.0", info.Duration)
		}
	})

	t.Run("mvhd v1 timescale zero", func(t *testing.T) {
		payload := make([]byte, 32)
		payload[0] = 1
		// payload[1:4] = flags (zero)
		binary.BigEndian.PutUint32(payload[20:24], 0)
		binary.BigEndian.PutUint64(payload[24:32], 1000)
		mvhd := buildAtom("mvhd", payload)
		moov := buildContainerAtom("moov", mvhd)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		_, err := probeMP4(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("truncated mvhd v1", func(t *testing.T) {
		mvhd := buildAtom("mvhd", []byte{1, 0, 0, 0, 0, 0, 0})
		moov := buildContainerAtom("moov", mvhd)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		_, err := probeMP4(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("truncated mvhd v0", func(t *testing.T) {
		mvhd := buildAtom("mvhd", []byte{0, 0, 0, 0, 0})
		moov := buildContainerAtom("moov", mvhd)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		_, err := probeMP4(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("tkhd unknown version", func(t *testing.T) {
		tkhd := buildAtom("tkhd", []byte{99, 0, 0, 0})
		trak := buildContainerAtom("trak", tkhd)
		mvhd := buildMVHD(0, 1000, 1000)
		moov := buildContainerAtom("moov", mvhd, trak)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if info.Width != 0 || info.Height != 0 {
			t.Errorf("dimensions = %dx%d, want 0x0", info.Width, info.Height)
		}
	})

	t.Run("nested mdia container walk", func(t *testing.T) {
		tkhd := buildTKHD(0, 320, 240)
		mdia := buildContainerAtom("mdia", tkhd)
		trak := buildContainerAtom("trak", mdia)
		mvhd := buildMVHD(0, 1000, 2000)
		moov := buildContainerAtom("moov", mvhd, trak)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if info.Width != 320 || info.Height != 240 {
			t.Errorf("dimensions = %dx%d, want 320x240", info.Width, info.Height)
		}
	})
}

func TestProbeMP4EdgeCases(t *testing.T) {
	t.Run("no moov", func(t *testing.T) {
		ftyp := buildAtom("ftyp", []byte("isom"))
		_, err := probeMP4(bytes.NewReader(ftyp))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("timescale zero", func(t *testing.T) {
		data := buildMinimalMP4(0, 5000)
		_, err := probeMP4(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("truncated", func(t *testing.T) {
		data := buildMinimalMP4(1000, 5000)
		_, err := probeMP4(bytes.NewReader(data[:12]))
		if err == nil {
			t.Fatal("expected error for truncated mp4")
		}
	})

	t.Run("invalid mvhd version", func(t *testing.T) {
		mvhd := buildAtom("mvhd", []byte{99, 0, 0, 0})
		moov := buildContainerAtom("moov", mvhd)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		_, err := probeMP4(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("atom size too small", func(t *testing.T) {
		data := []byte{0, 0, 0, 4, 'f', 't', 'y', 'p'}
		_, err := probeMP4(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("invalid atom inside moov", func(t *testing.T) {
		mvhd := buildMVHD(0, 1000, 1000)
		bad := []byte{0, 0, 0, 4, 'b', 'a', 'd', '!'}
		moov := buildContainerAtom("moov", mvhd, bad)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		_, err := probeMP4(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("tkhd too short", func(t *testing.T) {
		tkhd := buildAtom("tkhd", []byte{0})
		trak := buildContainerAtom("trak", tkhd)
		mvhd := buildMVHD(0, 1000, 1000)
		moov := buildContainerAtom("moov", mvhd, trak)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if info.Width != 0 {
			t.Errorf("Width = %d, want 0 for short tkhd", info.Width)
		}
	})
}

func TestWalkAtoms(t *testing.T) {
	t.Run("size zero atom", func(t *testing.T) {
		// Atom with size 0 extends to end of container
		mvhdPayload := buildMVHD(0, 1000, 1000)[8:]
		mvhdSize := 8 + len(mvhdPayload)
		buf := make([]byte, 8+mvhdSize)
		binary.BigEndian.PutUint32(buf[0:4], 0) // size 0
		copy(buf[4:8], "free")
		copy(buf[8:], buildMVHD(0, 1000, 1000))
		mvhd := buildMVHD(0, 1000, 1000)
		moov := buildContainerAtom("moov", mvhd)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if info.Duration <= 0 {
			t.Errorf("Duration = %v, want > 0", info.Duration)
		}
		_ = buf
	})
	t.Run("nested udta container", func(t *testing.T) {
		meta := buildAtom("meta", []byte{0, 0, 0, 0})
		udta := buildContainerAtom("udta", meta)
		mvhd := buildMVHD(0, 1000, 4000)
		moov := buildContainerAtom("moov", mvhd, udta)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if math.Abs(info.Duration-4.0) > 0.01 {
			t.Errorf("Duration = %v, want 4.0", info.Duration)
		}
	})

	t.Run("deep stbl edts nesting", func(t *testing.T) {
		elst := buildAtom("elst", []byte{0, 0, 0, 0})
		edts := buildContainerAtom("edts", elst)
		stsd := buildAtom("stsd", []byte{0, 0, 0, 0})
		stbl := buildContainerAtom("stbl", stsd)
		dinf := buildAtom("dinf", []byte{0})
		minf := buildContainerAtom("minf", stbl, dinf)
		mdia := buildContainerAtom("mdia", minf, edts)
		tkhd := buildTKHD(0, 1280, 720)
		trak := buildContainerAtom("trak", tkhd, mdia)
		mvhd := buildMVHD(0, 1000, 6000)
		moov := buildContainerAtom("moov", mvhd, trak)
		ftyp := buildAtom("ftyp", []byte("isom"))
		data := append(ftyp, moov...)
		info, err := probeMP4(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP4() error = %v", err)
		}
		if info.Width != 1280 || info.Height != 720 {
			t.Errorf("dimensions = %dx%d, want 1280x720", info.Width, info.Height)
		}
		if math.Abs(info.Duration-6.0) > 0.01 {
			t.Errorf("Duration = %v, want 6.0", info.Duration)
		}
	})
}

func TestParseMP4MVHD(t *testing.T) {
	t.Run("version 0 success", func(t *testing.T) {
		payload := buildMVHD(0, 1000, 5000)[8:]
		d, err := parseMP4MVHD(bytes.NewReader(payload), int64(len(payload)))
		if err != nil {
			t.Fatalf("parseMP4MVHD() error = %v", err)
		}
		if math.Abs(d-5.0) > 0.01 {
			t.Errorf("duration = %v, want 5.0", d)
		}
	})

	t.Run("version 1 success", func(t *testing.T) {
		payload := buildMVHD(1, 1000, 8000)[8:]
		d, err := parseMP4MVHD(bytes.NewReader(payload), int64(len(payload)))
		if err != nil {
			t.Fatalf("parseMP4MVHD() error = %v", err)
		}
		if math.Abs(d-8.0) > 0.01 {
			t.Errorf("duration = %v, want 8.0", d)
		}
	})

	t.Run("version 0 timescale zero", func(t *testing.T) {
		payload := make([]byte, 21)
		_, err := parseMP4MVHD(bytes.NewReader(payload), int64(len(payload)))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("size too small v0", func(t *testing.T) {
		_, err := parseMP4MVHD(bytes.NewReader([]byte{0}), 0)
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("size too small v1", func(t *testing.T) {
		_, err := parseMP4MVHD(bytes.NewReader([]byte{1, 0}), 1)
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})
}

func TestParseMP4TKHD(t *testing.T) {
	t.Run("version 0 dimensions", func(t *testing.T) {
		payload := buildTKHD(0, 800, 600)[8:]
		w, h, err := parseMP4TKHD(bytes.NewReader(payload), int64(len(payload)))
		if err != nil {
			t.Fatalf("parseMP4TKHD() error = %v", err)
		}
		if w != 800 || h != 600 {
			t.Errorf("dimensions = %dx%d, want 800x600", w, h)
		}
	})

	t.Run("version 1 dimensions", func(t *testing.T) {
		payload := buildTKHD(1, 1024, 768)[8:]
		w, h, err := parseMP4TKHD(bytes.NewReader(payload), int64(len(payload)))
		if err != nil {
			t.Fatalf("parseMP4TKHD() error = %v", err)
		}
		if w != 1024 || h != 768 {
			t.Errorf("dimensions = %dx%d, want 1024x768", w, h)
		}
	})

	t.Run("too short", func(t *testing.T) {
		w, h, err := parseMP4TKHD(bytes.NewReader([]byte{0}), 1)
		if err != nil {
			t.Fatalf("parseMP4TKHD() error = %v", err)
		}
		if w != 0 || h != 0 {
			t.Errorf("dimensions = %dx%d, want 0x0", w, h)
		}
	})

	t.Run("v0 insufficient for dimensions", func(t *testing.T) {
		payload := make([]byte, 72)
		w, h, err := parseMP4TKHD(bytes.NewReader(payload), 72)
		if err != nil {
			t.Fatalf("parseMP4TKHD() error = %v", err)
		}
		if w != 0 || h != 0 {
			t.Errorf("dimensions = %dx%d, want 0x0", w, h)
		}
	})

	t.Run("unknown version", func(t *testing.T) {
		w, h, err := parseMP4TKHD(bytes.NewReader([]byte{2, 0, 0, 0}), 4)
		if err != nil {
			t.Fatalf("parseMP4TKHD() error = %v", err)
		}
		if w != 0 || h != 0 {
			t.Errorf("dimensions = %dx%d, want 0x0", w, h)
		}
	})
}

func TestWalkAtomsExtended(t *testing.T) {
	t.Run("extended size header", func(t *testing.T) {
		mvhd := buildMVHD(0, 1000, 1000)
		moov := buildAtomExtended("moov", mvhd)
		err := walkAtoms(bytes.NewReader(moov), int64(len(moov)), "", func(path string, r io.ReadSeeker, size int64) error {
			return nil
		})
		if err != nil {
			t.Fatalf("walkAtoms() error = %v", err)
		}
	})

	t.Run("invalid extended size", func(t *testing.T) {
		buf := []byte{0, 0, 0, 1, 'm', 'o', 'o', 'v'}
		err := walkAtoms(bytes.NewReader(buf), int64(len(buf)), "", func(path string, r io.ReadSeeker, size int64) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected error for truncated extended size")
		}
	})
}

func TestMP4AtomName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"moov", "moov"},
		{"moov/mvhd", "mvhd"},
		{"moov/trak/tkhd", "tkhd"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := mp4AtomName(tt.path); got != tt.want {
				t.Errorf("mp4AtomName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
