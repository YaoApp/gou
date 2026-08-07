package mediaprobe

import (
	"encoding/binary"
	"fmt"
	"io"
)

func probeWAV(r io.ReadSeeker) (*MediaInfo, error) {
	var header [12]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("mediaprobe: read wav header: %w", err)
	}
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return nil, fmt.Errorf("mediaprobe: wav: %w", ErrInvalidFile)
	}

	var (
		byteRate uint32
		dataSize uint32
		hasFmt   bool
		hasData  bool
	)

	for {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(r, chunkHeader[:]); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("mediaprobe: read wav chunk header: %w", err)
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			if chunkSize < 12 {
				return nil, fmt.Errorf("mediaprobe: wav fmt chunk: %w", ErrInvalidFile)
			}
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, fmtData); err != nil {
				return nil, fmt.Errorf("mediaprobe: read wav fmt chunk: %w", err)
			}
			byteRate = binary.LittleEndian.Uint32(fmtData[8:12])
			hasFmt = true
			if chunkSize%2 == 1 {
				if _, err := r.Seek(1, io.SeekCurrent); err != nil {
					return nil, fmt.Errorf("mediaprobe: skip wav chunk padding: %w", err)
				}
			}
		case "data":
			dataSize = chunkSize
			hasData = true
			if !hasFmt {
				if _, err := r.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
					return nil, fmt.Errorf("mediaprobe: skip wav data chunk: %w", err)
				}
				if chunkSize%2 == 1 {
					if _, err := r.Seek(1, io.SeekCurrent); err != nil {
						return nil, fmt.Errorf("mediaprobe: skip wav chunk padding: %w", err)
					}
				}
			}
		default:
			if _, err := r.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return nil, fmt.Errorf("mediaprobe: skip wav chunk: %w", err)
			}
			if chunkSize%2 == 1 {
				if _, err := r.Seek(1, io.SeekCurrent); err != nil {
					return nil, fmt.Errorf("mediaprobe: skip wav chunk padding: %w", err)
				}
			}
		}

		if hasFmt && hasData {
			break
		}
	}

	if !hasFmt || !hasData {
		return nil, fmt.Errorf("mediaprobe: wav: %w", ErrInvalidFile)
	}
	if byteRate == 0 {
		return nil, fmt.Errorf("mediaprobe: wav: %w", ErrInvalidFile)
	}

	duration := float64(dataSize) / float64(byteRate)
	return &MediaInfo{Duration: duration, Format: "wav"}, nil
}
