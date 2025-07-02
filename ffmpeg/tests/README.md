# FFmpeg Test Media Files

This directory contains various test media files for testing the FFmpeg wrapper functionality.

## Audio Files

### Chinese Speech Files

- **chinese_speech.wav** (2.7MB) - Chinese speech sample from Wikimedia Commons (Edmund Yeo introducing himself in Mandarin)
- **chinese_short.ogg** (21KB) - Short Chinese phrase in OGG format
- **chinese_speech_test.wav** (38KB) - Generated Chinese speech: "你好，这是一个中文测试音频文件，用于 FFmpeg 音频转换测试。我们正在测试不同格式的音频转换功能。"
- **chinese_speech_test.mp3** (4KB) - MP3 version of the Chinese speech test

### English Speech Files

- **english_sample.wav** (1.8MB) - English speech sample from ICAO language testing (2 minutes)
- **english_speech_test.wav** (303KB) - Generated English speech: "Hello, this is an English test audio file for FFmpeg conversion testing. We are testing audio conversion from different formats."
- **english_speech_test.mp3** (28KB) - MP3 version of the English speech test

### Music Files

- **sample_1mb.mp3** (4.3MB) - Music MP3 file for audio conversion testing
- **sample_small.mp3** (3.1MB) - Smaller music MP3 file for quick testing

## Video Files

### Pure Video Files (No Audio)

- **sample_1mb.mp4** (1.6MB) - 720p test pattern video (2 minutes, 30fps)
- **sample_small.mp4** (941KB) - 480p test pattern video (2 minutes, 25fps)

### Speech Video Files

- **english_speech_video.mp4** (3.5MB) - 720p video with real English speech (2 minutes)
- **chinese_speech_video.mp4** (499KB) - 480p video with real Chinese speech (29 seconds)

## Test Coverage

These files provide comprehensive test coverage for:

1. **Audio Format Conversion**: WAV ↔ MP3 ↔ OGG
2. **Video Format Conversion**: MP4 with different codecs
3. **Audio Extraction**: Extract audio from video files
4. **Audio Replacement**: Replace audio tracks in video files
5. **Language Support**: Both Chinese and English speech content
6. **File Size Variety**: Different file sizes for performance testing
7. **Quality Testing**: Different bitrates and sample rates

## Usage in Tests

These files can be used to test:

- Basic audio/video conversion functionality
- Speech recognition and processing
- Batch processing with multiple files
- Error handling with different file formats
- Performance testing with various file sizes
