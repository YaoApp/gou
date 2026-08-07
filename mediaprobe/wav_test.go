package mediaprobe

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

func buildWAV(sampleRate, channels, bitsPerSample, numSamples int) []byte {
	blockAlign := channels * bitsPerSample / 8
	byteRate := sampleRate * blockAlign
	dataSize := numSamples * blockAlign

	fmtData := make([]byte, 16)
	binary.LittleEndian.PutUint16(fmtData[0:2], 1)
	binary.LittleEndian.PutUint16(fmtData[2:4], uint16(channels))
	binary.LittleEndian.PutUint32(fmtData[4:8], uint32(sampleRate))
	binary.LittleEndian.PutUint32(fmtData[8:12], uint32(byteRate))
	binary.LittleEndian.PutUint16(fmtData[12:14], uint16(blockAlign))
	binary.LittleEndian.PutUint16(fmtData[14:16], uint16(bitsPerSample))

	data := make([]byte, dataSize)
	riffSize := 4 + 8 + 16 + 8 + dataSize

	buf := make([]byte, 0, 12+8+16+8+dataSize)
	buf = append(buf, "RIFF"...)
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, uint32(riffSize))
	buf = append(buf, sizeBytes...)
	buf = append(buf, "WAVE"...)
	buf = append(buf, "fmt "...)
	chunkSize := make([]byte, 4)
	binary.LittleEndian.PutUint32(chunkSize, 16)
	buf = append(buf, chunkSize...)
	buf = append(buf, fmtData...)
	buf = append(buf, "data"...)
	dataSizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(dataSizeBytes, uint32(dataSize))
	buf = append(buf, dataSizeBytes...)
	buf = append(buf, data...)

	return buf
}

func buildWAVWithByteRate(sampleRate, channels, bitsPerSample, numSamples int, byteRateOverride uint32) []byte {
	data := buildWAV(sampleRate, channels, bitsPerSample, numSamples)
	binary.LittleEndian.PutUint32(data[28:32], byteRateOverride)
	return data
}

func assertDuration(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.01 {
		t.Errorf("Duration = %v, want %v (±0.01)", got, want)
	}
}

func TestProbeWAV(t *testing.T) {
	tests := []struct {
		name       string
		sampleRate int
		channels   int
		numSamples int
		wantDur    float64
	}{
		{"16kHz mono 1s", 16000, 1, 16000, 1.0},
		{"44100Hz stereo 2s", 44100, 2, 88200, 2.0},
		{"48000Hz stereo 0.5s", 48000, 2, 24000, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := buildWAV(tt.sampleRate, tt.channels, 16, tt.numSamples)
			info, err := probeWAV(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("probeWAV() error = %v", err)
			}
			if info.Format != "wav" {
				t.Errorf("Format = %q, want wav", info.Format)
			}
			assertDuration(t, info.Duration, tt.wantDur)
		})
	}
}

func TestProbeWAVEdgeCases(t *testing.T) {
	t.Run("empty data zero duration", func(t *testing.T) {
		data := buildWAV(44100, 1, 16, 0)
		info, err := probeWAV(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeWAV() error = %v", err)
		}
		assertDuration(t, info.Duration, 0)
	})

	t.Run("truncated header", func(t *testing.T) {
		_, err := probeWAV(bytes.NewReader([]byte("RIFF")))
		if err == nil {
			t.Fatal("expected error for truncated header")
		}
	})

	t.Run("invalid magic", func(t *testing.T) {
		data := buildWAV(44100, 1, 16, 1000)
		copy(data[0:4], "XXXX")
		_, err := probeWAV(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("byteRate zero", func(t *testing.T) {
		data := buildWAVWithByteRate(44100, 1, 16, 44100, 0)
		_, err := probeWAV(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("missing fmt chunk", func(t *testing.T) {
		data := buildWAV(44100, 1, 16, 1000)
		// Replace fmt chunk id with junk so fmt is not found before data ends
		copy(data[12:16], "junk")
		_, err := probeWAV(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("skip unknown chunk", func(t *testing.T) {
		wave := buildWAV(44100, 1, 16, 100)
		list := make([]byte, 13)
		copy(list[0:4], "LIST")
		binary.LittleEndian.PutUint32(list[4:8], 5)
		list[8], list[9], list[10], list[11], list[12] = 1, 2, 3, 4, 5
		rebuilt := append(wave, list...)
		binary.LittleEndian.PutUint32(rebuilt[4:8], uint32(len(rebuilt)-8))
		info, err := probeWAV(bytes.NewReader(rebuilt))
		if err != nil {
			t.Fatalf("probeWAV() error = %v", err)
		}
		assertDuration(t, info.Duration, 100.0/44100.0)
	})

	t.Run("data chunk before fmt", func(t *testing.T) {
		data := buildWAVDataBeforeFmt(44100, 1, 16, 4410)
		info, err := probeWAV(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeWAV() error = %v", err)
		}
		assertDuration(t, info.Duration, 0.1)
	})

	t.Run("unknown chunk before fmt", func(t *testing.T) {
		data := buildWAVWithPrefixChunk(44100, 1, 16, 100)
		info, err := probeWAV(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeWAV() error = %v", err)
		}
		assertDuration(t, info.Duration, 100.0/44100.0)
	})

	t.Run("fmt chunk too small", func(t *testing.T) {
		data := buildWAV(44100, 1, 16, 100)
		binary.LittleEndian.PutUint32(data[16:20], 8)
		_, err := probeWAV(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("fmt chunk odd size padding", func(t *testing.T) {
		data := buildWAVWithOddFmtChunk(44100, 1, 16, 100)
		info, err := probeWAV(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeWAV() error = %v", err)
		}
		assertDuration(t, info.Duration, 100.0/44100.0)
	})

	t.Run("data before fmt with odd data size", func(t *testing.T) {
		data := buildWAVDataBeforeFmtOdd(44100, 1, 8, 101)
		info, err := probeWAV(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("probeWAV() error = %v", err)
		}
		assertDuration(t, info.Duration, 101.0/44100.0)
	})

	t.Run("EOF during chunk iteration", func(t *testing.T) {
		data := buildWAV(44100, 1, 16, 100)
		// Truncate inside fmt chunk data
		_, err := probeWAV(bytes.NewReader(data[:30]))
		if err == nil {
			t.Fatal("expected error for EOF during fmt read")
		}
	})

	t.Run("truncated after fmt chunk", func(t *testing.T) {
		data := buildWAV(44100, 1, 16, 100)
		_, err := probeWAV(bytes.NewReader(data[:40]))
		if err == nil {
			t.Fatal("expected error for truncated wav")
		}
	})
}

func buildWAVDataBeforeFmtOdd(sampleRate, channels, bitsPerSample, numSamples int) []byte {
	blockAlign := channels * bitsPerSample / 8
	byteRate := sampleRate * blockAlign
	dataSize := numSamples * blockAlign

	fmtData := make([]byte, 16)
	binary.LittleEndian.PutUint16(fmtData[0:2], 1)
	binary.LittleEndian.PutUint16(fmtData[2:4], uint16(channels))
	binary.LittleEndian.PutUint32(fmtData[4:8], uint32(sampleRate))
	binary.LittleEndian.PutUint32(fmtData[8:12], uint32(byteRate))
	binary.LittleEndian.PutUint16(fmtData[12:14], uint16(blockAlign))
	binary.LittleEndian.PutUint16(fmtData[14:16], uint16(bitsPerSample))

	audio := make([]byte, dataSize)
	header := make([]byte, 12)
	copy(header[0:4], "RIFF")
	copy(header[8:12], "WAVE")

	dataChunk := make([]byte, 8+dataSize+1)
	copy(dataChunk[0:4], "data")
	binary.LittleEndian.PutUint32(dataChunk[4:8], uint32(dataSize))
	copy(dataChunk[8:], audio)
	// pad byte for odd data chunk size

	fmtChunk := make([]byte, 8+16)
	copy(fmtChunk[0:4], "fmt ")
	binary.LittleEndian.PutUint32(fmtChunk[4:8], 16)
	copy(fmtChunk[8:], fmtData)

	out := append(header, dataChunk...)
	out = append(out, fmtChunk...)
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	return out
}

func buildWAVDataBeforeFmt(sampleRate, channels, bitsPerSample, numSamples int) []byte {
	wave := buildWAV(sampleRate, channels, bitsPerSample, numSamples)
	header := make([]byte, 12)
	copy(header, wave[:12])
	dataChunk := wave[36:]
	fmtChunk := wave[12:36]
	body := append(dataChunk, fmtChunk...)
	out := append(header, body...)
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	return out
}

func buildWAVWithPrefixChunk(sampleRate, channels, bitsPerSample, numSamples int) []byte {
	wave := buildWAV(sampleRate, channels, bitsPerSample, numSamples)
	prefix := make([]byte, 12)
	copy(prefix[0:4], "JUNK")
	binary.LittleEndian.PutUint32(prefix[4:8], 4)
	prefix[8], prefix[9], prefix[10], prefix[11] = 9, 8, 7, 6
	header := make([]byte, 12)
	copy(header, wave[:12])
	out := append(header, prefix...)
	out = append(out, wave[12:]...)
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	return out
}

func buildWAVWithOddFmtChunk(sampleRate, channels, bitsPerSample, numSamples int) []byte {
	blockAlign := channels * bitsPerSample / 8
	byteRate := sampleRate * blockAlign
	dataSize := numSamples * blockAlign

	fmtData := make([]byte, 16)
	binary.LittleEndian.PutUint16(fmtData[0:2], 1)
	binary.LittleEndian.PutUint16(fmtData[2:4], uint16(channels))
	binary.LittleEndian.PutUint32(fmtData[4:8], uint32(sampleRate))
	binary.LittleEndian.PutUint32(fmtData[8:12], uint32(byteRate))
	binary.LittleEndian.PutUint16(fmtData[12:14], uint16(blockAlign))
	binary.LittleEndian.PutUint16(fmtData[14:16], uint16(bitsPerSample))

	data := make([]byte, dataSize)
	// fmt chunk size 17 (odd): 16 bytes PCM fmt + 1 byte, then 1 pad byte after chunk
	riffSize := 4 + 8 + 17 + 1 + 8 + dataSize
	buf := make([]byte, 0, 12+8+17+1+8+dataSize)
	buf = append(buf, "RIFF"...)
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, uint32(riffSize))
	buf = append(buf, sizeBytes...)
	buf = append(buf, "WAVE"...)
	buf = append(buf, "fmt "...)
	chunkSize := make([]byte, 4)
	binary.LittleEndian.PutUint32(chunkSize, 17)
	buf = append(buf, chunkSize...)
	buf = append(buf, fmtData...)
	buf = append(buf, 0) // last byte of odd-sized fmt chunk payload
	buf = append(buf, 0) // WAV pad byte after odd-sized chunk
	buf = append(buf, "data"...)
	dataSizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(dataSizeBytes, uint32(dataSize))
	buf = append(buf, dataSizeBytes...)
	buf = append(buf, data...)
	return buf
}
