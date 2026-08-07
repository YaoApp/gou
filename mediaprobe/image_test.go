package mediaprobe

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func encodePNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func encodeJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func encodeGIF(width, height int) []byte {
	palette := color.Palette{color.White, color.Black}
	img := image.NewPaletted(image.Rect(0, 0, width, height), palette)
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestProbeImage(t *testing.T) {
	t.Run("PNG", func(t *testing.T) {
		data := encodePNG(100, 200)
		info, err := probeImage(bytes.NewReader(data), "png")
		if err != nil {
			t.Fatalf("probeImage() error = %v", err)
		}
		if info.Format != "png" {
			t.Errorf("Format = %q, want png", info.Format)
		}
		if info.Width != 100 || info.Height != 200 {
			t.Errorf("dimensions = %dx%d, want 100x200", info.Width, info.Height)
		}
		if info.Duration != 0 {
			t.Errorf("Duration = %v, want 0", info.Duration)
		}
	})

	t.Run("JPEG", func(t *testing.T) {
		data := encodeJPEG(640, 480)
		info, err := probeImage(bytes.NewReader(data), "jpeg")
		if err != nil {
			t.Fatalf("probeImage() error = %v", err)
		}
		if info.Format != "jpeg" {
			t.Errorf("Format = %q, want jpeg", info.Format)
		}
		if info.Width != 640 || info.Height != 480 {
			t.Errorf("dimensions = %dx%d, want 640x480", info.Width, info.Height)
		}
	})

	t.Run("GIF", func(t *testing.T) {
		data := encodeGIF(32, 32)
		info, err := probeImage(bytes.NewReader(data), "gif")
		if err != nil {
			t.Fatalf("probeImage() error = %v", err)
		}
		if info.Format != "gif" {
			t.Errorf("Format = %q, want gif", info.Format)
		}
		if info.Width != 32 || info.Height != 32 {
			t.Errorf("dimensions = %dx%d, want 32x32", info.Width, info.Height)
		}
	})

	t.Run("jpg alias normalized", func(t *testing.T) {
		data := encodeJPEG(10, 10)
		info, err := probeImage(bytes.NewReader(data), "jpg")
		if err != nil {
			t.Fatalf("probeImage() error = %v", err)
		}
		if info.Format != "jpeg" {
			t.Errorf("Format = %q, want jpeg", info.Format)
		}
	})
}

func TestProbeImageEdgeCases(t *testing.T) {
	t.Run("invalid PNG data", func(t *testing.T) {
		_, err := probeImage(bytes.NewReader([]byte{0x89, 0x50, 0x4E, 0x47, 0x00}), "png")
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("invalid WebP data", func(t *testing.T) {
		data := append([]byte("RIFF\x00\x00\x00\x00"), []byte("WEBP")...)
		data = append(data, []byte("invalid")...)
		_, err := probeImage(bytes.NewReader(data), "webp")
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("invalid BMP data", func(t *testing.T) {
		_, err := probeImage(bytes.NewReader([]byte{0x42, 0x4D, 0x00, 0x00}), "bmp")
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})

	t.Run("invalid TIFF data", func(t *testing.T) {
		_, err := probeImage(bytes.NewReader([]byte{0x49, 0x49, 0x2A, 0x00, 0x00}), "tiff")
		if !errors.Is(err, ErrInvalidFile) {
			t.Errorf("error = %v, want ErrInvalidFile", err)
		}
	})
}

func TestProbeReaderImageDispatch(t *testing.T) {
	t.Run("PNG via ProbeReader", func(t *testing.T) {
		data := encodePNG(50, 75)
		info, err := ProbeReader(bytes.NewReader(data), "")
		if err != nil {
			t.Fatalf("ProbeReader() error = %v", err)
		}
		if info.Format != "png" {
			t.Errorf("Format = %q, want png", info.Format)
		}
		if info.Width != 50 || info.Height != 75 {
			t.Errorf("dimensions = %dx%d, want 50x75", info.Width, info.Height)
		}
	})
}
