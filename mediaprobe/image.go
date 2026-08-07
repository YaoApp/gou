package mediaprobe

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

func probeImage(r io.ReadSeeker, format string) (*MediaInfo, error) {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFile, err)
	}

	f := format
	if f == "jpg" {
		f = "jpeg"
	}

	return &MediaInfo{
		Width:  cfg.Width,
		Height: cfg.Height,
		Format: f,
	}, nil
}
