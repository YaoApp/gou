package mediaprobe

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		hint string
		want string
	}{
		{"WAV", append([]byte("RIFF\x00\x00\x00\x00"), []byte("WAVE")...), "", "wav"},
		{"FLAC", []byte("fLaC"), "", "flac"},
		{"OGG", []byte("OggS"), "", "ogg"},
		{"MP4", append(make([]byte, 4), []byte("ftyp")...), "", "mp4"},
		{"MP3 sync", []byte{0xFF, 0xFB, 0x90, 0x00}, "", "mp3"},
		{"MP3 ID3", []byte("ID3\x04"), "", "mp3"},
		{"JPEG", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "", "jpeg"},
		{"PNG", []byte{0x89, 0x50, 0x4E, 0x47}, "", "png"},
		{"GIF", []byte("GIF8"), "", "gif"},
		{"BMP", []byte{0x42, 0x4D, 0x00, 0x00}, "", "bmp"},
		{"TIFF LE", []byte{0x49, 0x49, 0x2A, 0x00}, "", "tiff"},
		{"TIFF BE", []byte{0x4D, 0x4D, 0x00, 0x2A}, "", "tiff"},
		{"WebP", append([]byte("RIFF\x00\x00\x00\x00"), []byte("WEBP")...), "", "webp"},
		{"too short with hint", []byte{0x01}, "wav", "wav"},
		{"unknown with hint", []byte("XXXX"), "flac", "flac"},
		{"unknown no hint", []byte("XXXX"), "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectFormat(bytes.NewReader(tt.data), tt.hint)
			if got != tt.want {
				t.Errorf("detectFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeHint(t *testing.T) {
	tests := []struct {
		hint string
		want string
	}{
		{"wav", "wav"},
		{".wav", "wav"},
		{"mp3", "mp3"},
		{"flac", "flac"},
		{"ogg", "ogg"},
		{"opus", "opus"},
		{"mp4", "mp4"},
		{"m4a", "mp4"},
		{"m4v", "mp4"},
		{"mov", "mp4"},
		{"jpg", "jpeg"},
		{"jpeg", "jpeg"},
		{"png", "png"},
		{"gif", "gif"},
		{"webp", "webp"},
		{"bmp", "bmp"},
		{"tiff", "tiff"},
		{"tif", "tiff"},
		{"audio/wav", "wav"},
		{"audio/wave", "wav"},
		{"audio/x-wav", "wav"},
		{"audio/mpeg", "mp3"},
		{"audio/mp3", "mp3"},
		{"audio/mp4", "mp4"},
		{"audio/m4a", "mp4"},
		{"audio/x-m4a", "mp4"},
		{"video/mp4", "mp4"},
		{"audio/flac", "flac"},
		{"audio/ogg", "ogg"},
		{"audio/opus", "ogg"},
		{"image/jpeg", "jpeg"},
		{"image/png", "png"},
		{"image/gif", "gif"},
		{"image/webp", "webp"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.hint, func(t *testing.T) {
			got := normalizeHint(tt.hint)
			if got != tt.want {
				t.Errorf("normalizeHint(%q) = %q, want %q", tt.hint, got, tt.want)
			}
		})
	}
}

func TestProbeReader(t *testing.T) {
	t.Run("JPEG dispatch", func(t *testing.T) {
		data := encodeJPEG(100, 100)
		info, err := ProbeReader(bytes.NewReader(data), "")
		if err != nil {
			t.Fatalf("ProbeReader() error = %v", err)
		}
		if info.Format != "jpeg" {
			t.Errorf("Format = %q, want jpeg", info.Format)
		}
	})

	t.Run("GIF dispatch", func(t *testing.T) {
		data := encodeGIF(16, 16)
		info, err := ProbeReader(bytes.NewReader(data), "")
		if err != nil {
			t.Fatalf("ProbeReader() error = %v", err)
		}
		if info.Format != "gif" {
			t.Errorf("Format = %q, want gif", info.Format)
		}
	})

	t.Run("m4a hint dispatch", func(t *testing.T) {
		data := buildMinimalMP4(1000, 5000)
		info, err := ProbeReader(bytes.NewReader(data), "m4a")
		if err != nil {
			t.Fatalf("ProbeReader() error = %v", err)
		}
		if info.Format != "mp4" {
			t.Errorf("Format = %q, want mp4", info.Format)
		}
	})

	t.Run("BMP dispatch via magic", func(t *testing.T) {
		data := []byte{0x42, 0x4D, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		_, err := ProbeReader(bytes.NewReader(data), "")
		if err == nil {
			t.Fatal("expected error for invalid bmp payload")
		}
	})

	t.Run("TIFF dispatch via magic", func(t *testing.T) {
		data := []byte{0x49, 0x49, 0x2A, 0x00, 0x00, 0x00, 0x00, 0x00}
		_, err := ProbeReader(bytes.NewReader(data), "")
		if err == nil {
			t.Fatal("expected error for invalid tiff payload")
		}
	})

	t.Run("WebP dispatch via magic", func(t *testing.T) {
		data := append([]byte("RIFF\x00\x00\x00\x00"), []byte("WEBP")...)
		data = append(data, 0, 0, 0, 0)
		_, err := ProbeReader(bytes.NewReader(data), "")
		if err == nil {
			t.Fatal("expected error for invalid webp payload")
		}
	})

	t.Run("flac hint only", func(t *testing.T) {
		data := buildFLAC(44100, 44100)
		info, err := ProbeReader(bytes.NewReader(data), "flac")
		if err != nil {
			t.Fatalf("ProbeReader() error = %v", err)
		}
		if info.Format != "flac" {
			t.Errorf("Format = %q, want flac", info.Format)
		}
	})

	t.Run("opus hint maps to ogg", func(t *testing.T) {
		data := buildOpusOGG(480000)
		info, err := ProbeReader(bytes.NewReader(data), "opus")
		if err != nil {
			t.Fatalf("ProbeReader() error = %v", err)
		}
		if info.Format != "ogg" {
			t.Errorf("Format = %q, want ogg", info.Format)
		}
	})

	t.Run("WAV dispatch", func(t *testing.T) {
		data := buildWAV(44100, 1, 16, 44100)
		info, err := ProbeReader(bytes.NewReader(data), "")
		if err != nil {
			t.Fatalf("ProbeReader() error = %v", err)
		}
		if info.Format != "wav" {
			t.Errorf("Format = %q, want wav", info.Format)
		}
	})

	t.Run("FLAC dispatch", func(t *testing.T) {
		data := buildFLAC(44100, 44100)
		info, err := ProbeReader(bytes.NewReader(data), "")
		if err != nil {
			t.Fatalf("ProbeReader() error = %v", err)
		}
		if info.Format != "flac" {
			t.Errorf("Format = %q, want flac", info.Format)
		}
	})

	t.Run("MP3 dispatch", func(t *testing.T) {
		data := buildMP3WithXing(44100, 100)
		info, err := ProbeReader(bytes.NewReader(data), "")
		if err != nil {
			t.Fatalf("ProbeReader() error = %v", err)
		}
		if info.Format != "mp3" {
			t.Errorf("Format = %q, want mp3", info.Format)
		}
	})

	t.Run("MP4 dispatch", func(t *testing.T) {
		data := buildMinimalMP4(1000, 5000)
		info, err := ProbeReader(bytes.NewReader(data), "")
		if err != nil {
			t.Fatalf("ProbeReader() error = %v", err)
		}
		if info.Format != "mp4" {
			t.Errorf("Format = %q, want mp4", info.Format)
		}
	})

	t.Run("OGG dispatch", func(t *testing.T) {
		data := buildVorbisOGG(44100, 44100)
		info, err := ProbeReader(bytes.NewReader(data), "")
		if err != nil {
			t.Fatalf("ProbeReader() error = %v", err)
		}
		if info.Format != "ogg" {
			t.Errorf("Format = %q, want ogg", info.Format)
		}
	})

	t.Run("empty data", func(t *testing.T) {
		_, err := ProbeReader(bytes.NewReader(nil), "")
		if !errors.Is(err, ErrUnsupportedFormat) {
			t.Errorf("error = %v, want ErrUnsupportedFormat", err)
		}
	})

	t.Run("invalid data with hint", func(t *testing.T) {
		_, err := ProbeReader(bytes.NewReader([]byte("garbage")), "wav")
		if err == nil {
			t.Fatal("expected error for invalid wav data")
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		_, err := ProbeReader(bytes.NewReader([]byte("garbage")), "xyz")
		if !errors.Is(err, ErrUnsupportedFormat) {
			t.Errorf("error = %v, want ErrUnsupportedFormat", err)
		}
	})
}

func TestProbeFile(t *testing.T) {
	t.Run("nonexistent path", func(t *testing.T) {
		_, err := ProbeFile("/nonexistent/path/to/file.wav")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("valid temp file", func(t *testing.T) {
		data := buildWAV(44100, 1, 16, 44100)
		tmp, err := os.CreateTemp("", "mediaprobe-*.wav")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.Write(data); err != nil {
			t.Fatal(err)
		}
		tmp.Close()

		info, err := ProbeFile(tmp.Name())
		if err != nil {
			t.Fatalf("ProbeFile() error = %v", err)
		}
		if info.Format != "wav" {
			t.Errorf("Format = %q, want wav", info.Format)
		}
	})
}
