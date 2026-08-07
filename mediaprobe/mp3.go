package mediaprobe

import (
	"encoding/binary"
	"fmt"
	"io"
)

var (
	mpeg1Layer3Bitrates = [16]int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0}
	mpeg2Layer3Bitrates = [16]int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0}
	mpeg1SampleRates    = [3]int{44100, 48000, 32000}
	mpeg2SampleRates    = [3]int{22050, 24000, 16000}
	mpeg25SampleRates   = [3]int{11025, 12000, 8000}
)

func probeMP3(r io.ReadSeeker) (*MediaInfo, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("mediaprobe: mp3 seek: %w", err)
	}

	var prefix [3]byte
	if _, err := io.ReadFull(r, prefix[:]); err != nil {
		return nil, fmt.Errorf("mediaprobe: mp3 read: %w", err)
	}

	audioStart := int64(0)
	id3v2Size := int64(0)
	if string(prefix[:]) == "ID3" {
		var rest [7]byte
		if _, err := io.ReadFull(r, rest[:]); err != nil {
			return nil, fmt.Errorf("mediaprobe: mp3 id3 header: %w", err)
		}
		size := int64(rest[3])<<21 | int64(rest[4])<<14 | int64(rest[5])<<7 | int64(rest[6])
		id3v2Size = 10 + size
		audioStart = id3v2Size
		if _, err := r.Seek(audioStart, io.SeekStart); err != nil {
			return nil, fmt.Errorf("mediaprobe: mp3 seek past id3: %w", err)
		}
	} else if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("mediaprobe: mp3 seek: %w", err)
	}

	buf := make([]byte, 64*1024)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, fmt.Errorf("mediaprobe: mp3 read scan: %w", err)
	}
	if n < 4 {
		return nil, fmt.Errorf("mediaprobe: mp3: %w", ErrInvalidFile)
	}

	var (
		frameOffset     int64
		bitrate         int
		sampleRate      int
		samplesPerFrame int
		mpegVersion     int
		mono            bool
		found           bool
	)

	for i := 0; i <= n-4; i++ {
		if buf[i] != 0xFF || buf[i+1]&0xE0 != 0xE0 {
			continue
		}

		header := binary.BigEndian.Uint32(buf[i : i+4])
		version := int((header >> 19) & 0x3)
		layer := int((header >> 17) & 0x3)
		brIdx := int((header >> 12) & 0xF)
		srIdx := int((header >> 10) & 0x3)
		channelMode := int((header >> 6) & 0x3)

		if version == 1 || layer != 1 || srIdx == 3 {
			continue
		}

		var br int
		switch version {
		case 3:
			br = mpeg1Layer3Bitrates[brIdx]
		case 2, 0:
			br = mpeg2Layer3Bitrates[brIdx]
		default:
			continue
		}

		var sr int
		switch version {
		case 3:
			sr = mpeg1SampleRates[srIdx]
		case 2:
			sr = mpeg2SampleRates[srIdx]
		case 0:
			sr = mpeg25SampleRates[srIdx]
		}

		if br == 0 || sr == 0 {
			continue
		}

		samples := 1152
		if version != 3 {
			samples = 576
		}

		frameOffset = audioStart + int64(i)
		bitrate = br
		sampleRate = sr
		samplesPerFrame = samples
		mpegVersion = version
		mono = channelMode == 3
		found = true
		break
	}

	if !found {
		return nil, fmt.Errorf("mediaprobe: mp3: %w", ErrInvalidFile)
	}

	sideInfo := xingSideInfoOffset(mpegVersion, mono)
	vbrOffset := frameOffset + 4 + int64(sideInfo)

	if _, err := r.Seek(vbrOffset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("mediaprobe: mp3 seek vbr: %w", err)
	}

	var tag [4]byte
	if _, err := io.ReadFull(r, tag[:]); err != nil {
		return nil, fmt.Errorf("mediaprobe: mp3 read vbr tag: %w", err)
	}

	tagStr := string(tag[:])
	if tagStr == "Xing" || tagStr == "Info" {
		var flags [4]byte
		if _, err := io.ReadFull(r, flags[:]); err != nil {
			return nil, fmt.Errorf("mediaprobe: mp3 read xing flags: %w", err)
		}
		if flags[3]&0x01 != 0 {
			var frames [4]byte
			if _, err := io.ReadFull(r, frames[:]); err != nil {
				return nil, fmt.Errorf("mediaprobe: mp3 read xing frames: %w", err)
			}
			totalFrames := binary.BigEndian.Uint32(frames[:])
			duration := float64(totalFrames) * float64(samplesPerFrame) / float64(sampleRate)
			return &MediaInfo{Duration: duration, Format: "mp3"}, nil
		}
	} else if tagStr == "VBRI" {
		var vbriRest [14]byte
		if _, err := io.ReadFull(r, vbriRest[:]); err != nil {
			return nil, fmt.Errorf("mediaprobe: mp3 read vbri: %w", err)
		}
		totalFrames := binary.BigEndian.Uint32(vbriRest[10:14])
		duration := float64(totalFrames) * float64(samplesPerFrame) / float64(sampleRate)
		return &MediaInfo{Duration: duration, Format: "mp3"}, nil
	}

	fileSize, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("mediaprobe: mp3 seek end: %w", err)
	}

	audioBytes := fileSize - id3v2Size
	if fileSize >= 128 {
		if _, err := r.Seek(fileSize-128, io.SeekStart); err != nil {
			return nil, fmt.Errorf("mediaprobe: mp3 seek id3v1: %w", err)
		}
		var id3v1 [3]byte
		if _, err := io.ReadFull(r, id3v1[:]); err == nil && string(id3v1[:]) == "TAG" {
			audioBytes -= 128
		}
	}

	if audioBytes <= 0 || bitrate == 0 {
		return nil, fmt.Errorf("mediaprobe: mp3: %w", ErrInvalidFile)
	}

	duration := float64(audioBytes*8) / float64(bitrate*1000)
	return &MediaInfo{Duration: duration, Format: "mp3"}, nil
}

func xingSideInfoOffset(mpegVersion int, mono bool) int {
	if mpegVersion == 3 {
		if mono {
			return 17
		}
		return 32
	}
	if mono {
		return 9
	}
	return 17
}
