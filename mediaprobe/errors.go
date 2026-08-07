package mediaprobe

import "errors"

var (
	ErrUnsupportedFormat = errors.New("mediaprobe: unsupported format")
	ErrInvalidFile       = errors.New("mediaprobe: invalid or corrupted file")
)
