package mediaprobe

import (
	"encoding/binary"
	"fmt"
	"io"
)

func probeFLAC(r io.ReadSeeker) (*MediaInfo, error) {
	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return nil, fmt.Errorf("mediaprobe: read flac magic: %w", err)
	}
	if string(magic[:]) != "fLaC" {
		return nil, fmt.Errorf("mediaprobe: flac: %w", ErrInvalidFile)
	}

	var blockHeader [4]byte
	if _, err := io.ReadFull(r, blockHeader[:]); err != nil {
		return nil, fmt.Errorf("mediaprobe: read flac metadata header: %w", err)
	}

	blockType := blockHeader[0] & 0x7F
	blockLength := binary.BigEndian.Uint32([]byte{0, blockHeader[1], blockHeader[2], blockHeader[3]})
	if blockType != 0 || blockLength != 34 {
		return nil, fmt.Errorf("mediaprobe: flac streaminfo: %w", ErrInvalidFile)
	}

	var streamInfo [34]byte
	if _, err := io.ReadFull(r, streamInfo[:]); err != nil {
		return nil, fmt.Errorf("mediaprobe: read flac streaminfo: %w", err)
	}

	sampleRate := uint32(streamInfo[10])<<12 | uint32(streamInfo[11])<<4 | uint32(streamInfo[12]>>4)
	totalSamples := (uint64(streamInfo[13])&0x0F)<<32 | uint64(streamInfo[14])<<24 | uint64(streamInfo[15])<<16 | uint64(streamInfo[16])<<8 | uint64(streamInfo[17])
	if sampleRate == 0 || totalSamples == 0 {
		return nil, fmt.Errorf("mediaprobe: flac: %w", ErrInvalidFile)
	}

	duration := float64(totalSamples) / float64(sampleRate)
	return &MediaInfo{Duration: duration, Format: "flac"}, nil
}
