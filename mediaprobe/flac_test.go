package mediaprobe

import (
	"bytes"
	"errors"
	"math"
	"testing"
)

func buildFLAC(sampleRate uint32, totalSamples uint64) []byte {
	var streamInfo [34]byte
	streamInfo[10] = byte(sampleRate >> 12)
	streamInfo[11] = byte(sampleRate >> 4)
	streamInfo[12] = byte(sampleRate&0xF) << 4
	streamInfo[13] = byte(totalSamples>>32) & 0x0F
	streamInfo[14] = byte(totalSamples >> 24)
	streamInfo[15] = byte(totalSamples >> 16)
	streamInfo[16] = byte(totalSamples >> 8)
	streamInfo[17] = byte(totalSamples)

	buf := make([]byte, 0, 42)
	buf = append(buf, "fLaC"...)
	buf = append(buf, 0x80) // last block + STREAMINFO
	buf = append(buf, 0, 0, 34)
	buf = append(buf, streamInfo[:]...)
	return buf
}

func TestProbeFLAC(t *testing.T) {
	tests := []struct {
		name         string
		sampleRate   uint32
		totalSamples uint64
		wantDur      float64
	}{
		{"44100Hz 10s", 44100, 441000, 10.0},
		{"96000Hz 1s", 96000, 96000, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := buildFLAC(tt.sampleRate, tt.totalSamples)
			info, err := probeFLAC(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("probeFLAC() error = %v", err)
			}
			if info.Format != "flac" {
				t.Errorf("Format = %q, want flac", info.Format)
			}
			if math.Abs(info.Duration-tt.wantDur) > 0.01 {
				t.Errorf("Duration = %v, want %v", info.Duration, tt.wantDur)
			}
		})
	}
}

func TestProbeFLACEdgeCases(t *testing.T) {
	t.Run("sampleRate zero", func(t *testing.T) {
		data := buildFLAC(0, 441000)
		_, err := probeFLAC(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("totalSamples zero", func(t *testing.T) {
		data := buildFLAC(44100, 0)
		_, err := probeFLAC(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("truncated", func(t *testing.T) {
		data := buildFLAC(44100, 441000)
		_, err := probeFLAC(bytes.NewReader(data[:20]))
		if err == nil {
			t.Fatal("expected error for truncated flac")
		}
	})

	t.Run("invalid magic", func(t *testing.T) {
		data := buildFLAC(44100, 441000)
		copy(data[0:4], "XXXX")
		_, err := probeFLAC(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("wrong block type", func(t *testing.T) {
		data := buildFLAC(44100, 441000)
		data[4] = 0x81 // PADDING block instead of STREAMINFO
		_, err := probeFLAC(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("wrong block length", func(t *testing.T) {
		data := buildFLAC(44100, 441000)
		data[6] = 33
		_, err := probeFLAC(bytes.NewReader(data))
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})
}
