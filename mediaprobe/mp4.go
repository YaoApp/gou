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
	"mvex": true,
	"moof": true,
	"traf": true,
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
		foundMoov        bool
		duration         float64
		width            int
		height           int
		mdhdTimescale    uint32
		defaultSampleDur uint32 // from trex
		currentFragDur   uint32 // per-fragment default from tfhd
		fragDurationSum  uint64
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
		case "mdhd":
			if ts, err := parseMP4MDHD(payload, size); err == nil && ts > 0 {
				mdhdTimescale = ts
			}
		case "trex":
			if defDur, err := parseMP4TREX(payload, size); err == nil {
				defaultSampleDur = defDur
			}
		case "traf":
			currentFragDur = 0
		case "tfhd":
			if defDur, err := parseMP4TFHD(payload, size); err == nil && defDur > 0 {
				currentFragDur = defDur
			}
		case "trun":
			defDur := currentFragDur
			if defDur == 0 {
				defDur = defaultSampleDur
			}
			if sum, err := parseMP4TRUN(payload, size, defDur); err == nil {
				fragDurationSum += sum
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

	if duration == 0 && fragDurationSum > 0 && mdhdTimescale > 0 {
		duration = float64(fragDurationSum) / float64(mdhdTimescale)
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

// parseMP4MDHD extracts timescale from Media Header Box.
func parseMP4MDHD(r io.ReadSeeker, size int64) (uint32, error) {
	if size < 4 {
		return 0, fmt.Errorf("mediaprobe: mp4 mdhd: %w", ErrInvalidFile)
	}
	var vf [4]byte
	if _, err := io.ReadFull(r, vf[:]); err != nil {
		return 0, err
	}
	switch vf[0] {
	case 0:
		if size < 20 {
			return 0, fmt.Errorf("mediaprobe: mp4 mdhd: %w", ErrInvalidFile)
		}
		var buf [12]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, err
		}
		return binary.BigEndian.Uint32(buf[8:12]), nil
	case 1:
		if size < 32 {
			return 0, fmt.Errorf("mediaprobe: mp4 mdhd: %w", ErrInvalidFile)
		}
		var buf [20]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, err
		}
		return binary.BigEndian.Uint32(buf[16:20]), nil
	}
	return 0, fmt.Errorf("mediaprobe: mp4 mdhd: %w", ErrInvalidFile)
}

// parseMP4TREX extracts default_sample_duration from Track Extends Box.
func parseMP4TREX(r io.ReadSeeker, size int64) (uint32, error) {
	if size < 24 {
		return 0, fmt.Errorf("mediaprobe: mp4 trex: %w", ErrInvalidFile)
	}
	var buf [24]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	// [0:4] version+flags, [4:8] trackID, [8:12] defSampleDescIdx,
	// [12:16] defSampleDuration, [16:20] defSampleSize, [20:24] defSampleFlags
	return binary.BigEndian.Uint32(buf[12:16]), nil
}

// parseMP4TFHD extracts default_sample_duration from Track Fragment Header.
func parseMP4TFHD(r io.ReadSeeker, size int64) (uint32, error) {
	if size < 8 {
		return 0, nil
	}
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	flags := uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
	offset := int64(8) // past version+flags + trackID

	if flags&0x01 != 0 { // base-data-offset-present
		offset += 8
	}
	if flags&0x02 != 0 { // sample-description-index-present
		offset += 4
	}
	if flags&0x08 == 0 { // default-sample-duration NOT present
		return 0, nil
	}
	toSkip := offset - 8
	if toSkip > 0 {
		if _, err := r.Seek(toSkip, io.SeekCurrent); err != nil {
			return 0, err
		}
	}
	var dur [4]byte
	if _, err := io.ReadFull(r, dur[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(dur[:]), nil
}

// parseMP4TRUN sums sample durations from a Track Run Box.
// defDur is used when individual sample durations are absent.
func parseMP4TRUN(r io.ReadSeeker, size int64, defDur uint32) (uint64, error) {
	if size < 8 {
		return 0, fmt.Errorf("mediaprobe: mp4 trun: %w", ErrInvalidFile)
	}
	var hdr [8]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, err
	}
	flags := uint32(hdr[1])<<16 | uint32(hdr[2])<<8 | uint32(hdr[3])
	sampleCount := binary.BigEndian.Uint32(hdr[4:8])

	// Skip optional header fields
	if flags&0x001 != 0 { // data-offset-present
		r.Seek(4, io.SeekCurrent)
	}
	if flags&0x004 != 0 { // first-sample-flags-present
		r.Seek(4, io.SeekCurrent)
	}

	hasDuration := flags&0x100 != 0
	hasSize := flags&0x200 != 0
	hasFlags := flags&0x400 != 0
	hasCTO := flags&0x800 != 0

	if !hasDuration {
		return uint64(sampleCount) * uint64(defDur), nil
	}

	var total uint64
	for i := uint32(0); i < sampleCount; i++ {
		var dur [4]byte
		if _, err := io.ReadFull(r, dur[:]); err != nil {
			return total, err
		}
		total += uint64(binary.BigEndian.Uint32(dur[:]))
		if hasSize {
			r.Seek(4, io.SeekCurrent)
		}
		if hasFlags {
			r.Seek(4, io.SeekCurrent)
		}
		if hasCTO {
			r.Seek(4, io.SeekCurrent)
		}
	}
	return total, nil
}
