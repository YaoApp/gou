package mediaprobe

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const (
	ebmlHeaderID  = 0x1A45DFA3
	ebmlSegment   = 0x18538067
	ebmlInfo      = 0x1549A966
	ebmlCluster   = 0x1F43B675
	ebmlTSScale   = 0x2AD7B1
	ebmlDuration  = 0x4489
	ebmlTimestamp = 0xE7
)

// readVINT reads an EBML variable-size integer.
// When raw is true the VINT_MARKER bit is kept (element IDs);
// when false the marker is stripped (data sizes).
func readVINT(r io.Reader, raw bool) (uint64, int, error) {
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, 0, err
	}
	first := b[0]
	if first == 0 {
		return 0, 0, fmt.Errorf("mediaprobe: webm: invalid VINT")
	}

	width := 1
	for mask := byte(0x80); mask > 0 && first&mask == 0; mask >>= 1 {
		width++
	}

	val := uint64(first)
	if !raw {
		val &^= uint64(1 << (8 - uint(width)))
	}

	if width > 1 {
		extra := make([]byte, width-1)
		if _, err := io.ReadFull(r, extra); err != nil {
			return 0, 0, err
		}
		for _, eb := range extra {
			val = val<<8 | uint64(eb)
		}
	}
	return val, width, nil
}

func vintIsUnknown(val uint64, width int) bool {
	return val == (1<<(7*uint(width)))-1
}

func ebmlReadUint(r io.Reader, n int) (uint64, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	var v uint64
	for _, b := range buf {
		v = v<<8 | uint64(b)
	}
	return v, nil
}

func probeWebM(r io.ReadSeeker) (*MediaInfo, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("mediaprobe: webm seek: %w", err)
	}

	id, _, err := readVINT(r, true)
	if err != nil || id != ebmlHeaderID {
		return nil, fmt.Errorf("mediaprobe: webm: %w", ErrInvalidFile)
	}
	hdrSize, _, err := readVINT(r, false)
	if err != nil {
		return nil, fmt.Errorf("mediaprobe: webm: %w", err)
	}
	if _, err := r.Seek(int64(hdrSize), io.SeekCurrent); err != nil {
		return nil, fmt.Errorf("mediaprobe: webm: %w", err)
	}

	segID, _, err := readVINT(r, true)
	if err != nil || segID != ebmlSegment {
		return nil, fmt.Errorf("mediaprobe: webm: %w", ErrInvalidFile)
	}
	segSize, segW, err := readVINT(r, false)
	if err != nil {
		return nil, fmt.Errorf("mediaprobe: webm: %w", err)
	}

	segStart, _ := r.Seek(0, io.SeekCurrent)
	segEnd := segStart + int64(segSize)
	if vintIsUnknown(segSize, segW) {
		end, err := r.Seek(0, io.SeekEnd)
		if err != nil {
			return nil, fmt.Errorf("mediaprobe: webm: %w", err)
		}
		segEnd = end
		if _, err := r.Seek(segStart, io.SeekStart); err != nil {
			return nil, fmt.Errorf("mediaprobe: webm: %w", err)
		}
	}

	var (
		tsScale       uint64 = 1000000 // default 1 ms
		infoDuration  float64
		hasDuration   bool
		lastClusterTS int64 = -1
	)

	for {
		pos, err := r.Seek(0, io.SeekCurrent)
		if err != nil || pos >= segEnd {
			break
		}
		elemID, _, err := readVINT(r, true)
		if err != nil {
			break
		}
		elemSize, elemW, err := readVINT(r, false)
		if err != nil {
			break
		}
		dataPos, _ := r.Seek(0, io.SeekCurrent)

		if vintIsUnknown(elemSize, elemW) {
			break
		}

		switch elemID {
		case ebmlInfo:
			webmParseInfo(r, dataPos, int64(elemSize), &tsScale, &infoDuration, &hasDuration)
		case ebmlCluster:
			if ts, err := webmClusterTS(r, dataPos, int64(elemSize)); err == nil {
				lastClusterTS = int64(ts)
			}
		}

		if _, err := r.Seek(dataPos+int64(elemSize), io.SeekStart); err != nil {
			break
		}
	}

	if hasDuration && infoDuration > 0 {
		return &MediaInfo{
			Duration: infoDuration * float64(tsScale) / 1e9,
			Format:   "webm",
		}, nil
	}
	if lastClusterTS >= 0 {
		return &MediaInfo{
			Duration: float64(lastClusterTS) * float64(tsScale) / 1e9,
			Format:   "webm",
		}, nil
	}
	return &MediaInfo{Format: "webm"}, nil
}

func webmParseInfo(r io.ReadSeeker, start, size int64, tsScale *uint64, dur *float64, hasDur *bool) {
	r.Seek(start, io.SeekStart)
	end := start + size

	for {
		pos, _ := r.Seek(0, io.SeekCurrent)
		if pos >= end {
			return
		}
		childID, _, err := readVINT(r, true)
		if err != nil {
			return
		}
		childSize, _, err := readVINT(r, false)
		if err != nil {
			return
		}

		switch childID {
		case ebmlTSScale:
			if childSize <= 8 {
				if v, err := ebmlReadUint(r, int(childSize)); err == nil {
					*tsScale = v
				}
			}
		case ebmlDuration:
			switch childSize {
			case 4:
				var buf [4]byte
				if _, err := io.ReadFull(r, buf[:]); err == nil {
					*dur = float64(math.Float32frombits(binary.BigEndian.Uint32(buf[:])))
					*hasDur = true
				}
			case 8:
				var buf [8]byte
				if _, err := io.ReadFull(r, buf[:]); err == nil {
					*dur = math.Float64frombits(binary.BigEndian.Uint64(buf[:]))
					*hasDur = true
				}
			default:
				r.Seek(int64(childSize), io.SeekCurrent)
			}
		default:
			r.Seek(int64(childSize), io.SeekCurrent)
		}
	}
}

// webmClusterTS reads the Timestamp child from a Cluster element.
func webmClusterTS(r io.ReadSeeker, start, size int64) (uint64, error) {
	r.Seek(start, io.SeekStart)
	end := start + size

	for i := 0; i < 8; i++ {
		pos, _ := r.Seek(0, io.SeekCurrent)
		if pos >= end {
			break
		}
		childID, _, err := readVINT(r, true)
		if err != nil {
			break
		}
		childSize, _, err := readVINT(r, false)
		if err != nil {
			break
		}
		if childID == ebmlTimestamp && childSize <= 8 {
			return ebmlReadUint(r, int(childSize))
		}
		r.Seek(int64(childSize), io.SeekCurrent)
	}
	return 0, fmt.Errorf("no timestamp")
}
