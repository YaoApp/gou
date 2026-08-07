package mediaprobe

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

func buildOggPage(granule int64, headerType byte, serial uint32, seqNum uint32, data []byte) []byte {
	segments := splitOggSegments(data)
	headerSize := 27 + len(segments)

	page := make([]byte, headerSize+len(data))
	copy(page[0:4], "OggS")
	page[4] = 0
	page[5] = headerType
	binary.LittleEndian.PutUint64(page[6:14], uint64(granule))
	binary.LittleEndian.PutUint32(page[14:18], serial)
	binary.LittleEndian.PutUint32(page[18:22], seqNum)
	page[26] = byte(len(segments))
	copy(page[27:headerSize], segments)
	copy(page[headerSize:], data)
	return page
}

func splitOggSegments(data []byte) []byte {
	if len(data) == 0 {
		return []byte{0}
	}
	var segs []byte
	remaining := len(data)
	for remaining > 0 {
		if remaining >= 255 {
			segs = append(segs, 255)
			remaining -= 255
		} else {
			segs = append(segs, byte(remaining))
			remaining = 0
		}
	}
	return segs
}

func buildVorbisIDHeader(sampleRate uint32) []byte {
	buf := make([]byte, 30)
	buf[0] = 0x01
	copy(buf[1:7], "vorbis")
	binary.LittleEndian.PutUint32(buf[7:11], 0)
	buf[11] = 2 // stereo
	binary.LittleEndian.PutUint32(buf[12:16], sampleRate)
	return buf
}

func buildOpusHead(sampleRate uint32) []byte {
	buf := make([]byte, 19)
	copy(buf[0:8], "OpusHead")
	buf[8] = 1
	buf[9] = 2
	binary.LittleEndian.PutUint32(buf[10:14], sampleRate)
	return buf
}

func buildVorbisOGG(sampleRate uint32, lastGranule int64) []byte {
	idHeader := buildVorbisIDHeader(sampleRate)
	page0 := buildOggPage(0, 0x02, 1, 0, idHeader)
	page1 := buildOggPage(lastGranule, 0x04, 1, 1, []byte{0x05})
	return append(page0, page1...)
}

func buildOpusOGG(lastGranule int64) []byte {
	head := buildOpusHead(48000)
	page0 := buildOggPage(0, 0x02, 1, 0, head)
	page1 := buildOggPage(lastGranule, 0x04, 1, 1, []byte{0x01})
	return append(page0, page1...)
}

func TestProbeOGG(t *testing.T) {
	t.Run("Vorbis 10s", func(t *testing.T) {
		data := buildVorbisOGG(44100, 441000)
		info, err := probeOGG(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeOGG() error = %v", err)
		}
		if info.Format != "ogg" {
			t.Errorf("Format = %q, want ogg", info.Format)
		}
		if math.Abs(info.Duration-10.0) > 0.01 {
			t.Errorf("Duration = %v, want 10.0", info.Duration)
		}
	})

	t.Run("Opus 10s", func(t *testing.T) {
		data := buildOpusOGG(480000)
		info, err := probeOGG(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeOGG() error = %v", err)
		}
		if math.Abs(info.Duration-10.0) > 0.01 {
			t.Errorf("Duration = %v, want 10.0", info.Duration)
		}
	})

	t.Run("large file tail scan", func(t *testing.T) {
		data := buildVorbisOGG(44100, 441000)
		padding := make([]byte, 70000)
		for i := range padding {
			padding[i] = 0x00
		}
		// Place last page at very end
		lastPage := buildOggPage(882000, 0x04, 1, 2, []byte{0x05})
		data = append(data, padding...)
		data = append(data, lastPage...)
		info, err := probeOGG(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeOGG() error = %v", err)
		}
		if math.Abs(info.Duration-20.0) > 0.01 {
			t.Errorf("Duration = %v, want 20.0", info.Duration)
		}
	})

	t.Run("multi segment page", func(t *testing.T) {
		payload := bytes.Repeat([]byte{0xAB}, 300)
		page := buildOggPage(441000, 0x04, 1, 1, payload)
		idHeader := buildVorbisIDHeader(44100)
		page0 := buildOggPage(0, 0x02, 1, 0, idHeader)
		data := append(page0, page...)
		info, err := probeOGG(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeOGG() error = %v", err)
		}
		if math.Abs(info.Duration-10.0) > 0.01 {
			t.Errorf("Duration = %v, want 10.0", info.Duration)
		}
	})

	t.Run("corrupt capture skipped in tail scan", func(t *testing.T) {
		idHeader := buildVorbisIDHeader(44100)
		page0 := buildOggPage(0, 0x02, 1, 0, idHeader)
		// Corrupt page body after valid capture in tail region
		corrupt := append([]byte("OggS"), byte(0), byte(0))
		corrupt = append(corrupt, make([]byte, 20)...)
		corrupt = append(corrupt, byte(255)) // many segments -> read error -> continue
		lastPage := buildOggPage(441000, 0x04, 1, 2, []byte{0x05})
		data := append(page0, corrupt...)
		data = append(data, lastPage...)
		info, err := probeOGG(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeOGG() error = %v", err)
		}
		if math.Abs(info.Duration-10.0) > 0.01 {
			t.Errorf("Duration = %v, want 10.0", info.Duration)
		}
	})
}

func TestProbeOGGEdgeCases(t *testing.T) {
	t.Run("no valid granule", func(t *testing.T) {
		idHeader := buildVorbisIDHeader(44100)
		page := buildOggPage(0, 0x02, 1, 0, idHeader)
		_, err := probeOGG(bytes.NewReader(page))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("sampleRate zero", func(t *testing.T) {
		idHeader := buildVorbisIDHeader(0)
		page0 := buildOggPage(0, 0x02, 1, 0, idHeader)
		page1 := buildOggPage(1000, 0x04, 1, 1, []byte{0x05})
		data := append(page0, page1...)
		_, err := probeOGG(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("unknown codec", func(t *testing.T) {
		page := buildOggPage(1000, 0x02, 1, 0, []byte("unknown codec data"))
		_, err := probeOGG(bytes.NewReader(page))
		if err == nil {
			t.Fatal("expected error for unknown codec")
		}
	})

	t.Run("truncated page header", func(t *testing.T) {
		_, err := probeOGG(bytes.NewReader([]byte("OggS\x00")))
		if err == nil {
			t.Fatal("expected error for truncated ogg page")
		}
	})

	t.Run("invalid capture pattern in tail", func(t *testing.T) {
		idHeader := buildVorbisIDHeader(44100)
		page0 := buildOggPage(0, 0x02, 1, 0, idHeader)
		// Tail with fake OggS that has invalid segment data
		fake := append([]byte("OggS\x00\x00"), make([]byte, 22)...)
		fake = append(fake, 255) // 255 segments - will fail read
		data := append(page0, fake...)
		_, err := probeOGG(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("vorbis header too short", func(t *testing.T) {
		short := []byte{0x01, 'v', 'o', 'r', 'b', 'i', 's'}
		page := buildOggPage(0, 0x02, 1, 0, short)
		_, err := probeOGG(bytes.NewReader(page))
		if err == nil {
			t.Fatal("expected error for short vorbis header")
		}
	})
}

func TestOggSampleRate(t *testing.T) {
	t.Run("vorbis", func(t *testing.T) {
		hdr := buildVorbisIDHeader(48000)
		sr, err := oggSampleRate(hdr)
		if err != nil {
			t.Fatalf("oggSampleRate() error = %v", err)
		}
		if sr != 48000 {
			t.Errorf("sampleRate = %d, want 48000", sr)
		}
	})

	t.Run("opus", func(t *testing.T) {
		hdr := buildOpusHead(0)
		sr, err := oggSampleRate(hdr)
		if err != nil {
			t.Fatalf("oggSampleRate() error = %v", err)
		}
		if sr != 48000 {
			t.Errorf("sampleRate = %d, want 48000", sr)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		_, err := oggSampleRate([]byte("unknown"))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})
}

func TestReadOggPageHeader(t *testing.T) {
	t.Run("valid page", func(t *testing.T) {
		page := buildOggPage(12345, 0x04, 42, 7, []byte("testdata"))
		hdr, err := readOggPageHeader(bytes.NewReader(page))
		if err != nil {
			t.Fatalf("readOggPageHeader() error = %v", err)
		}
		if hdr.granulePosition != 12345 {
			t.Errorf("granulePosition = %d, want 12345", hdr.granulePosition)
		}
		if hdr.pageSize != len("testdata") {
			t.Errorf("pageSize = %d, want %d", hdr.pageSize, len("testdata"))
		}
	})

	t.Run("segment table read error", func(t *testing.T) {
		page := buildOggPage(0, 0x02, 1, 0, bytes.Repeat([]byte{0xAB}, 300))
		_, err := readOggPageHeader(bytes.NewReader(page[:27]))
		if err == nil {
			t.Fatal("expected error for truncated segment table")
		}
	})

	t.Run("missing capture", func(t *testing.T) {
		_, err := readOggPageHeader(bytes.NewReader([]byte("BAD!")))
		if err == nil {
			t.Fatal("expected error for bad capture pattern")
		}
	})
}
