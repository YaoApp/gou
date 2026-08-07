package mediaprobe

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

func buildMP3Frame(version, layer, bitrateIdx, srIdx int, padding bool) []byte {
	var header uint32
	header |= 0xFFE00000
	header |= uint32(version&0x3) << 19
	header |= uint32(layer&0x3) << 17
	header |= 1 << 16 // no CRC
	header |= uint32(bitrateIdx&0xF) << 12
	header |= uint32(srIdx&0x3) << 10
	if padding {
		header |= 1 << 9
	}
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, header)
	return buf
}

func mpeg1FrameSize(bitrateKbps, sampleRate int, padding bool) int {
	size := (144000 * bitrateKbps / sampleRate) + 4
	if padding {
		size++
	}
	return size
}

func buildMP3WithXing(sampleRate, totalFrames int) []byte {
	srIdx := 0
	switch sampleRate {
	case 48000:
		srIdx = 1
	case 32000:
		srIdx = 2
	}
	bitrateIdx := 9 // 128 kbps MPEG1

	frameSize := mpeg1FrameSize(128, sampleRate, false)
	frame := make([]byte, frameSize)
	copy(frame[0:4], buildMP3Frame(3, 1, bitrateIdx, srIdx, false))

	sideInfoOffset := 32 // MPEG1 stereo
	xingOffset := 4 + sideInfoOffset
	copy(frame[xingOffset:xingOffset+4], []byte("Xing"))
	frame[xingOffset+7] = 0x01 // frames flag in last flags byte

	binary.BigEndian.PutUint32(frame[xingOffset+8:xingOffset+12], uint32(totalFrames))
	return frame
}

func buildMP3WithVBRI(sampleRate, totalFrames int) []byte {
	srIdx := 0
	frameSize := mpeg1FrameSize(128, sampleRate, false)
	frame := make([]byte, frameSize)
	copy(frame[0:4], buildMP3Frame(3, 1, 9, srIdx, false))

	sideInfoOffset := 32
	vbriOffset := 4 + sideInfoOffset
	copy(frame[vbriOffset:vbriOffset+4], []byte("VBRI"))
	binary.BigEndian.PutUint32(frame[vbriOffset+14:vbriOffset+18], uint32(totalFrames))
	return frame
}

func buildMP3WithXingMono(sampleRate, totalFrames int) []byte {
	frameSize := mpeg1FrameSize(128, sampleRate, false)
	frame := make([]byte, frameSize)
	// MPEG1 mono: channel mode 3 (11)
	header := buildMP3Frame(3, 1, 9, 0, false)
	binary.BigEndian.PutUint32(header, binary.BigEndian.Uint32(header)|3<<6)
	copy(frame[0:4], header)
	sideInfoOffset := 17
	xingOffset := 4 + sideInfoOffset
	copy(frame[xingOffset:xingOffset+4], []byte("Xing"))
	frame[xingOffset+7] = 0x01
	binary.BigEndian.PutUint32(frame[xingOffset+8:xingOffset+12], uint32(totalFrames))
	return frame
}

func buildMP3WithInfo(sampleRate, totalFrames int) []byte {
	data := buildMP3WithXing(sampleRate, totalFrames)
	sideInfoOffset := 32
	xingOffset := 4 + sideInfoOffset
	copy(data[xingOffset:xingOffset+4], []byte("Info"))
	return data
}

func buildMP3WithXingNoFrames(sampleRate int) []byte {
	srIdx := 0
	frameSize := mpeg1FrameSize(128, sampleRate, false)
	frame := make([]byte, frameSize)
	copy(frame[0:4], buildMP3Frame(3, 1, 9, srIdx, false))
	sideInfoOffset := 32
	xingOffset := 4 + sideInfoOffset
	copy(frame[xingOffset:xingOffset+4], []byte("Xing"))
	frame[xingOffset+7] = 0x00 // no frames flag
	return frame
}

func buildMP3CBR(bitrateKbps, sampleRate, numFrames int) []byte {
	srIdx := 0
	switch sampleRate {
	case 48000:
		srIdx = 1
	case 32000:
		srIdx = 2
	}
	bitrateIdx := 9
	if bitrateKbps == 64 {
		bitrateIdx = 5
	}

	frameSize := mpeg1FrameSize(bitrateKbps, sampleRate, false)
	frame := buildMP3Frame(3, 1, bitrateIdx, srIdx, false)

	var buf []byte
	for i := 0; i < numFrames; i++ {
		f := make([]byte, frameSize)
		copy(f, frame)
		buf = append(buf, f...)
	}
	return buf
}

func buildMP3WithID3v2(mp3Data []byte, tagSize int64) []byte {
	id3 := make([]byte, 10+tagSize)
	copy(id3[0:3], "ID3")
	id3[3], id3[4] = 3, 0
	id3[6] = byte((tagSize >> 21) & 0x7F)
	id3[7] = byte((tagSize >> 14) & 0x7F)
	id3[8] = byte((tagSize >> 7) & 0x7F)
	id3[9] = byte(tagSize & 0x7F)
	return append(id3, mp3Data...)
}

func TestProbeMP3(t *testing.T) {
	t.Run("VBR Xing", func(t *testing.T) {
		totalFrames := 100
		data := buildMP3WithXing(44100, totalFrames)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		if info.Format != "mp3" {
			t.Errorf("Format = %q, want mp3", info.Format)
		}
		want := float64(totalFrames) * 1152.0 / 44100.0
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("VBR VBRI", func(t *testing.T) {
		totalFrames := 50
		data := buildMP3WithVBRI(44100, totalFrames)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := float64(totalFrames) * 1152.0 / 44100.0
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("CBR estimate", func(t *testing.T) {
		numFrames := 100
		bitrate := 128
		sampleRate := 44100
		data := buildMP3CBR(bitrate, sampleRate, numFrames)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := float64(len(data)*8) / float64(bitrate*1000)
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("with ID3v2 prefix", func(t *testing.T) {
		mp3 := buildMP3WithXing(44100, 100)
		data := buildMP3WithID3v2(mp3, 0)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := 100.0 * 1152.0 / 44100.0
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("CBR with ID3v1 suffix", func(t *testing.T) {
		data := buildMP3CBR(128, 44100, 50)
		id3v1 := make([]byte, 128)
		copy(id3v1, "TAG")
		data = append(data, id3v1...)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		audioBytes := len(data) - 128
		want := float64(audioBytes*8) / float64(128*1000)
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("mono Xing VBR", func(t *testing.T) {
		data := buildMP3WithXingMono(44100, 100)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := 100.0 * 1152.0 / 44100.0
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("Info tag VBR", func(t *testing.T) {
		data := buildMP3WithInfo(44100, 100)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := 100.0 * 1152.0 / 44100.0
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("Xing without frames flag falls back to CBR", func(t *testing.T) {
		data := buildMP3WithXingNoFrames(44100)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := float64(len(data)*8) / float64(128*1000)
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("MPEG2.5 frame", func(t *testing.T) {
		frameSize := 200
		frame := buildMP3Frame(0, 1, 4, 0, false) // MPEG2.5, 32kbps, 11025Hz
		data := make([]byte, frameSize)
		copy(data, frame)
		for j := 4; j < frameSize; j++ {
			data[j] = 0xCD
		}
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := float64(len(data)*8) / float64(32*1000)
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("MPEG2 CBR estimate", func(t *testing.T) {
		numFrames := 100
		frameSize := 200
		frame := buildMP3Frame(2, 1, 4, 0, false) // MPEG2, 32kbps (idx 4), 22050Hz
		var data []byte
		for i := 0; i < numFrames; i++ {
			f := make([]byte, frameSize)
			copy(f, frame)
			for j := 4; j < frameSize; j++ {
				f[j] = 0xAB // avoid false Xing/VBRI/ID3 matches
			}
			data = append(data, f...)
		}
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := float64(len(data)*8) / float64(32*1000)
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})
	t.Run("with large ID3v2 prefix", func(t *testing.T) {
		mp3 := buildMP3WithXing(44100, 100)
		data := buildMP3WithID3v2(mp3, 256)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := 100.0 * 1152.0 / 44100.0
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("frame after garbage prefix", func(t *testing.T) {
		garbage := bytes.Repeat([]byte{0x00}, 100)
		frame := buildMP3WithXing(44100, 50)
		data := append(garbage, frame...)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := 50.0 * 1152.0 / 44100.0
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("skip invalid frame headers", func(t *testing.T) {
		prefix := []byte{0xFF, 0x00, 0x00, 0x00} // false sync
		frame := buildMP3WithXing(44100, 50)
		data := append(prefix, frame...)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := 50.0 * 1152.0 / 44100.0
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("skip reserved version and free bitrate frames", func(t *testing.T) {
		bad := buildMP3Frame(1, 1, 0, 0, false) // reserved version
		good := buildMP3WithXing(44100, 50)
		data := append(bad, good...)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := 50.0 * 1152.0 / 44100.0
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("skip layer II and invalid sample rate index", func(t *testing.T) {
		layerII := buildMP3Frame(3, 2, 9, 0, false)
		badSR := buildMP3Frame(3, 1, 9, 3, false)
		good := buildMP3WithXing(44100, 50)
		data := append(layerII, badSR...)
		data = append(data, good...)
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		want := 50.0 * 1152.0 / 44100.0
		if math.Abs(info.Duration-want) > 0.01 {
			t.Errorf("Duration = %v, want %v", info.Duration, want)
		}
	})

	t.Run("small file skips id3v1 check", func(t *testing.T) {
		data := buildMP3WithXing(44100, 10)
		if len(data) >= 128 {
			data = data[:100]
		}
		info, err := probeMP3(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeMP3() error = %v", err)
		}
		if info.Duration <= 0 {
			t.Errorf("Duration = %v, want > 0", info.Duration)
		}
	})
}

func TestProbeMP3EdgeCases(t *testing.T) {
	t.Run("audioBytes zero after tags", func(t *testing.T) {
		// ID3v2 tag with no audio data following
		id3 := make([]byte, 20)
		copy(id3, "ID3")
		_, err := probeMP3(bytes.NewReader(id3))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("no valid frame", func(t *testing.T) {
		_, err := probeMP3(bytes.NewReader([]byte("garbage data here")))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("ID3v2 only", func(t *testing.T) {
		id3 := make([]byte, 20)
		copy(id3, "ID3")
		_, err := probeMP3(bytes.NewReader(id3))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("too short", func(t *testing.T) {
		_, err := probeMP3(bytes.NewReader([]byte{0xFF}))
		if err == nil {
			t.Fatal("expected error for too-short input")
		}
	})
}

func TestXingSideInfoOffset(t *testing.T) {
	tests := []struct {
		name    string
		version int
		mono    bool
		want    int
	}{
		{"MPEG1 stereo", 3, false, 32},
		{"MPEG1 mono", 3, true, 17},
		{"MPEG2 stereo", 2, false, 17},
		{"MPEG2 mono", 2, true, 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xingSideInfoOffset(tt.version, tt.mono)
			if got != tt.want {
				t.Errorf("xingSideInfoOffset() = %d, want %d", got, tt.want)
			}
		})
	}
}
