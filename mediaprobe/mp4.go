package mediaprobe

import (
	"encoding/binary"
	"fmt"
	"io"
)

var mp4ContainerAtoms = map[string]bool{
	"moov": true,
	"trak": true,
	"mdia": true,
	"minf": true,
	"stbl": true,
	"udta": true,
	"edts": true,
}

type atomVisitor func(path string, r io.ReadSeeker, size int64) error

func walkAtoms(r io.ReadSeeker, end int64, pathPrefix string, visitor atomVisitor) error {
	for {
		pos, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("mediaprobe: mp4 seek: %w", err)
		}
		if pos >= end {
			break
		}

		var header [8]byte
		if _, err := io.ReadFull(r, header[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return fmt.Errorf("mediaprobe: mp4 read atom header: %w", err)
		}

		size := int64(binary.BigEndian.Uint32(header[0:4]))
		atomType := string(header[4:8])
		headerSize := int64(8)

		switch size {
		case 1:
			var extSize [8]byte
			if _, err := io.ReadFull(r, extSize[:]); err != nil {
				return fmt.Errorf("mediaprobe: mp4 read extended size: %w", err)
			}
			size = int64(binary.BigEndian.Uint64(extSize[:]))
			headerSize = 16
		case 0:
			size = end - pos
		}

		if size < headerSize {
			return fmt.Errorf("mediaprobe: mp4: %w", ErrInvalidFile)
		}

		payloadSize := size - headerSize
		path := atomType
		if pathPrefix != "" {
			path = pathPrefix + "/" + atomType
		}

		payloadStart, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("mediaprobe: mp4 seek: %w", err)
		}

		if err := visitor(path, r, payloadSize); err != nil {
			return err
		}

		if mp4ContainerAtoms[atomType] {
			if _, err := r.Seek(payloadStart, io.SeekStart); err != nil {
				return fmt.Errorf("mediaprobe: mp4 seek: %w", err)
			}
			if err := walkAtoms(r, payloadStart+payloadSize, path, visitor); err != nil {
				return err
			}
		}

		if _, err := r.Seek(payloadStart+payloadSize, io.SeekStart); err != nil {
			return fmt.Errorf("mediaprobe: mp4 seek: %w", err)
		}
	}
	return nil
}

func probeMP4(r io.ReadSeeker) (*MediaInfo, error) {
	fileEnd, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("mediaprobe: mp4 seek end: %w", err)
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("mediaprobe: mp4 seek start: %w", err)
	}

	var (
		foundMoov bool
		duration  float64
		width     int
		height    int
	)

	visitor := func(path string, payload io.ReadSeeker, size int64) error {
		if mp4AtomName(path) == "moov" {
			foundMoov = true
		}

		switch mp4AtomName(path) {
		case "mvhd":
			d, err := parseMP4MVHD(payload, size)
			if err != nil {
				return err
			}
			duration = d
		case "tkhd":
			if width == 0 && height == 0 {
				w, h, err := parseMP4TKHD(payload, size)
				if err != nil {
					return err
				}
				if w > 0 && h > 0 {
					width = w
					height = h
				}
			}
		}
		return nil
	}

	if err := walkAtoms(r, fileEnd, "", visitor); err != nil {
		return nil, err
	}

	if !foundMoov {
		return nil, fmt.Errorf("mediaprobe: mp4: %w", ErrInvalidFile)
	}

	return &MediaInfo{
		Duration: duration,
		Width:    width,
		Height:   height,
		Format:   "mp4",
	}, nil
}

func mp4AtomName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

func parseMP4MVHD(r io.ReadSeeker, size int64) (float64, error) {
	if size < 4 {
		return 0, fmt.Errorf("mediaprobe: mp4: %w", ErrInvalidFile)
	}

	var vf [4]byte
	if _, err := io.ReadFull(r, vf[:]); err != nil {
		return 0, fmt.Errorf("mediaprobe: mp4 read mvhd: %w", err)
	}
	version := vf[0]

	switch version {
	case 0:
		if size < 20 {
			return 0, fmt.Errorf("mediaprobe: mp4: %w", ErrInvalidFile)
		}
		var buf [16]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, fmt.Errorf("mediaprobe: mp4 read mvhd: %w", err)
		}
		timescale := binary.BigEndian.Uint32(buf[8:12])
		dur := binary.BigEndian.Uint32(buf[12:16])
		if timescale == 0 {
			return 0, fmt.Errorf("mediaprobe: mp4: %w", ErrInvalidFile)
		}
		return float64(dur) / float64(timescale), nil
	case 1:
		if size < 32 {
			return 0, fmt.Errorf("mediaprobe: mp4: %w", ErrInvalidFile)
		}
		var buf [28]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, fmt.Errorf("mediaprobe: mp4 read mvhd: %w", err)
		}
		timescale := binary.BigEndian.Uint32(buf[16:20])
		dur := binary.BigEndian.Uint64(buf[20:28])
		if timescale == 0 {
			return 0, fmt.Errorf("mediaprobe: mp4: %w", ErrInvalidFile)
		}
		return float64(dur) / float64(timescale), nil
	default:
		return 0, fmt.Errorf("mediaprobe: mp4: %w", ErrInvalidFile)
	}
}

func parseMP4TKHD(r io.ReadSeeker, size int64) (int, int, error) {
	if size < 4 {
		return 0, 0, nil
	}

	var vf [4]byte
	if _, err := io.ReadFull(r, vf[:]); err != nil {
		return 0, 0, fmt.Errorf("mediaprobe: mp4 read tkhd: %w", err)
	}
	version := vf[0]

	var skip int64
	switch version {
	case 0:
		skip = 68
	case 1:
		skip = 80
	default:
		return 0, 0, nil
	}

	if size < 4+skip+8 {
		return 0, 0, nil
	}

	if _, err := io.CopyN(io.Discard, r, skip); err != nil {
		return 0, 0, fmt.Errorf("mediaprobe: mp4 read tkhd: %w", err)
	}

	var dims [8]byte
	if _, err := io.ReadFull(r, dims[:]); err != nil {
		return 0, 0, fmt.Errorf("mediaprobe: mp4 read tkhd: %w", err)
	}

	w := int(binary.BigEndian.Uint32(dims[0:4]) >> 16)
	h := int(binary.BigEndian.Uint32(dims[4:8]) >> 16)
	return w, h, nil
}
