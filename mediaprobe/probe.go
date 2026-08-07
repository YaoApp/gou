package mediaprobe

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MediaInfo holds metadata extracted from a media file.
type MediaInfo struct {
	Duration float64 // seconds (audio/video), 0 for images
	Width    int     // pixels (video/image), 0 for audio-only
	Height   int     // pixels (video/image), 0 for audio-only
	Format   string  // detected: "wav", "mp3", "mp4", "flac", "ogg", "jpeg", "png", etc.
}

// ProbeFile probes a media file at the given path and returns its metadata.
func ProbeFile(path string) (*MediaInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("mediaprobe: open %s: %w", path, err)
	}
	defer f.Close()

	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	return ProbeReader(f, ext)
}

// ProbeReader probes media from an io.ReadSeeker.
// hint is a format hint (extension or MIME type), may be empty for auto-detection.
func ProbeReader(r io.ReadSeeker, hint string) (*MediaInfo, error) {
	format := detectFormat(r, hint)
	if format == "" {
		return nil, ErrUnsupportedFormat
	}

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("mediaprobe: seek: %w", err)
	}

	switch format {
	case "wav":
		return probeWAV(r)
	case "mp3":
		return probeMP3(r)
	case "mp4", "m4a":
		return probeMP4(r)
	case "flac":
		return probeFLAC(r)
	case "ogg":
		return probeOGG(r)
	case "jpeg", "jpg", "png", "gif", "webp", "bmp", "tiff":
		return probeImage(r, format)
	default:
		return nil, ErrUnsupportedFormat
	}
}

// detectFormat reads the first 12 bytes to identify the file format via magic bytes.
// Falls back to hint (extension) if magic bytes don't match.
func detectFormat(r io.ReadSeeker, hint string) string {
	var buf [12]byte
	n, _ := io.ReadFull(r, buf[:])
	r.Seek(0, io.SeekStart)

	if n < 4 {
		return normalizeHint(hint)
	}

	switch {
	case n >= 12 && bytes.Equal(buf[:4], []byte("RIFF")) && bytes.Equal(buf[8:12], []byte("WAVE")):
		return "wav"
	case n >= 12 && bytes.Equal(buf[:4], []byte("RIFF")) && bytes.Equal(buf[8:12], []byte("WEBP")):
		return "webp"
	case bytes.Equal(buf[:4], []byte("fLaC")):
		return "flac"
	case bytes.Equal(buf[:4], []byte("OggS")):
		return "ogg"
	case n >= 8 && bytes.Equal(buf[4:8], []byte("ftyp")):
		return "mp4"
	case buf[0] == 0xFF && (buf[1]&0xE0) == 0xE0:
		return "mp3"
	case n >= 3 && bytes.Equal(buf[:3], []byte("ID3")):
		return "mp3"
	case n >= 3 && buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF:
		return "jpeg"
	case bytes.Equal(buf[:4], []byte{0x89, 0x50, 0x4E, 0x47}):
		return "png"
	case bytes.Equal(buf[:4], []byte("GIF8")):
		return "gif"
	case n >= 2 && buf[0] == 0x42 && buf[1] == 0x4D:
		return "bmp"
	case n >= 4 && buf[0] == 0x49 && buf[1] == 0x49 && buf[2] == 0x2A && buf[3] == 0x00:
		return "tiff"
	case n >= 4 && buf[0] == 0x4D && buf[1] == 0x4D && buf[2] == 0x00 && buf[3] == 0x2A:
		return "tiff"
	}

	return normalizeHint(hint)
}

func normalizeHint(hint string) string {
	h := strings.ToLower(strings.TrimPrefix(hint, "."))
	switch h {
	case "wav", "mp3", "flac", "ogg", "opus":
		return h
	case "mp4", "m4a", "m4v", "mov":
		return "mp4"
	case "jpg", "jpeg":
		return "jpeg"
	case "png", "gif", "webp", "bmp", "tiff", "tif":
		if h == "tif" {
			return "tiff"
		}
		return h
	case "audio/wav", "audio/wave", "audio/x-wav":
		return "wav"
	case "audio/mpeg", "audio/mp3":
		return "mp3"
	case "audio/mp4", "audio/m4a", "audio/x-m4a", "video/mp4":
		return "mp4"
	case "audio/flac":
		return "flac"
	case "audio/ogg", "audio/opus":
		return "ogg"
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	}
	return ""
}
