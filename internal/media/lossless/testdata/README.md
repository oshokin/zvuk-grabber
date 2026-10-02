# Synthetic audio fixture

`tone.mp4` contains 0.1 seconds of a generated 440 Hz sine wave, mono,
44.1 kHz, encoded as FLAC inside MP4. It contains no service/user audio.

Reproduce with FFmpeg:

```sh
ffmpeg -f lavfi -i sine=frequency=440:duration=0.1 -c:a flac -strict experimental -y tone.mp4
```

Tests use the committed fixture and do not require FFmpeg.
