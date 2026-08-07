package mediaprobe

import (
	"bytes"
	"errors"
	"testing"
)

func TestProbeMP3SeekErrors(t *testing.T) {
	data := buildMP3WithXing(44100, 50)

	t.Run("seek start", func(t *testing.T) {
		_, err := probeMP3(newErrSeeker(bytes.NewReader(data), 0))
		if err == nil {
			t.Fatal("expected seek error")
		}
	})

	t.Run("seek vbr", func(t *testing.T) {
		_, err := probeMP3(newErrSeeker(bytes.NewReader(data), 36))
		if err == nil {
			t.Fatal("expected seek error")
		}
	})

	t.Run("seek end", func(t *testing.T) {
		// Xing without frames flag triggers CBR path needing seek end
		cbr := buildMP3WithXingNoFrames(44100)
		_, err := probeMP3(newErrSeeker(bytes.NewReader(cbr), int64(len(cbr))))
		if err == nil {
			t.Fatal("expected seek error")
		}
	})
	t.Run("seek past id3", func(t *testing.T) {
		mp3 := buildMP3WithXing(44100, 50)
		id3 := buildMP3WithID3v2(mp3, 10)
		_, err := probeMP3(newErrSeeker(bytes.NewReader(id3), 20))
		if err == nil {
			t.Fatal("expected seek error")
		}
	})

	t.Run("read vbr tag", func(t *testing.T) {
		// Truncate right before vbr tag read
		data := buildMP3WithXing(44100, 50)
		if len(data) > 36 {
			_, err := probeMP3(bytes.NewReader(data[:36]))
			if err == nil {
				t.Fatal("expected read error")
			}
		}
	})
}

func TestProbeFLACReadErrors(t *testing.T) {
	data := buildFLAC(44100, 441000)
	t.Run("truncated streaminfo", func(t *testing.T) {
		_, err := probeFLAC(bytes.NewReader(data[:10]))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestProbeWAVReadErrors(t *testing.T) {
	t.Run("truncated chunk header", func(t *testing.T) {
		data := buildWAV(44100, 1, 16, 10)
		_, err := probeWAV(bytes.NewReader(data[:14]))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestProbeMP4SeekErrors(t *testing.T) {
	data := buildMinimalMP4(1000, 5000)

	t.Run("seek end", func(t *testing.T) {
		_, err := probeMP4(newErrSeeker(bytes.NewReader(data), int64(len(data))))
		if err == nil {
			t.Fatal("expected seek error")
		}
	})
}

func TestProbeOGGSeekErrors(t *testing.T) {
	data := buildVorbisOGG(44100, 441000)

	t.Run("seek start", func(t *testing.T) {
		_, err := probeOGG(newErrSeeker(bytes.NewReader(data), 0))
		if err == nil {
			t.Fatal("expected seek error")
		}
	})

	t.Run("seek end", func(t *testing.T) {
		_, err := probeOGG(newErrSeeker(bytes.NewReader(data), int64(len(data))))
		if err == nil {
			t.Fatal("expected seek error")
		}
	})
}

func TestProbeReaderSeekError(t *testing.T) {
	data := buildWAV(44100, 1, 16, 1000)
	_, err := ProbeReader(newErrSeeker(bytes.NewReader(data), 0), "")
	if err == nil {
		t.Fatal("expected seek error")
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrUnsupportedFormat == nil || ErrInvalidFile == nil {
		t.Fatal("sentinel errors should not be nil")
	}
	if ErrUnsupportedFormat.Error() == "" || ErrInvalidFile.Error() == "" {
		t.Fatal("error strings should not be empty")
	}
	if !errors.Is(ErrUnsupportedFormat, ErrUnsupportedFormat) {
		t.Fatal("errors.Is should match ErrUnsupportedFormat")
	}
}
