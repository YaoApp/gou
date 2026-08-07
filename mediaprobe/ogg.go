package mediaprobe

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var oggCapture = []byte("OggS")

type oggPageHeader struct {
	granulePosition int64
	headerType      byte
	segments        int
	pageSize        int
}

func readOggPageHeader(r io.Reader) (*oggPageHeader, error) {
	var buf [27]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, err
	}
	if !bytes.Equal(buf[:4], oggCapture) {
		return nil, fmt.Errorf("mediaprobe: ogg: missing capture pattern")
	}

	hdr := &oggPageHeader{
		granulePosition: int64(binary.LittleEndian.Uint64(buf[6:14])),
		headerType:      buf[5],
		segments:        int(buf[26]),
	}

	segTable := make([]byte, hdr.segments)
	if _, err := io.ReadFull(r, segTable); err != nil {
		return nil, err
	}
	for _, segLen := range segTable {
		hdr.pageSize += int(segLen)
	}

	return hdr, nil
}

func probeOGG(r io.ReadSeeker) (*MediaInfo, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("mediaprobe: ogg seek: %w", err)
	}

	hdr, err := readOggPageHeader(r)
	if err != nil {
		return nil, fmt.Errorf("mediaprobe: ogg page header: %w", err)
	}

	pageData := make([]byte, hdr.pageSize)
	if _, err := io.ReadFull(r, pageData); err != nil {
		return nil, fmt.Errorf("mediaprobe: ogg page data: %w", err)
	}

	sampleRate, err := oggSampleRate(pageData)
	if err != nil {
		return nil, err
	}

	fileSize, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("mediaprobe: ogg seek end: %w", err)
	}

	start := fileSize - 65536
	if start < 0 {
		start = 0
	}
	if _, err := r.Seek(start, io.SeekStart); err != nil {
		return nil, fmt.Errorf("mediaprobe: ogg seek tail: %w", err)
	}

	tailLen := fileSize - start
	tailData := make([]byte, tailLen)
	if _, err := io.ReadFull(r, tailData); err != nil {
		return nil, fmt.Errorf("mediaprobe: ogg read tail: %w", err)
	}

	lastGranule := int64(0)
	offset := 0
	for offset < len(tailData) {
		idx := bytes.Index(tailData[offset:], oggCapture)
		if idx < 0 {
			break
		}
		pos := offset + idx
		pageReader := bytes.NewReader(tailData[pos:])
		pageHdr, err := readOggPageHeader(pageReader)
		if err != nil {
			offset = pos + 1
			continue
		}
		if pageHdr.granulePosition > 0 && pageHdr.granulePosition > lastGranule {
			lastGranule = pageHdr.granulePosition
		}
		pageLen := 27 + pageHdr.segments + pageHdr.pageSize
		if pos+pageLen > len(tailData) {
			break
		}
		offset = pos + pageLen
	}

	if lastGranule <= 0 || sampleRate == 0 {
		return nil, fmt.Errorf("mediaprobe: ogg: %w", ErrInvalidFile)
	}

	duration := float64(lastGranule) / float64(sampleRate)
	return &MediaInfo{Duration: duration, Format: "ogg"}, nil
}

func oggSampleRate(pageData []byte) (uint32, error) {
	if len(pageData) >= 7 && pageData[0] == 0x01 && bytes.Equal(pageData[1:7], []byte("vorbis")) {
		if len(pageData) < 16 {
			return 0, fmt.Errorf("mediaprobe: ogg vorbis header: %w", ErrInvalidFile)
		}
		return binary.LittleEndian.Uint32(pageData[12:16]), nil
	}
	if len(pageData) >= 8 && bytes.Equal(pageData[:8], []byte("OpusHead")) {
		return 48000, nil
	}
	return 0, fmt.Errorf("mediaprobe: ogg codec: %w", ErrInvalidFile)
}
