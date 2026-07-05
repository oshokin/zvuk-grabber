package media

import "strings"

// Codec represents an audio codec or codec/container pair returned by providers.
type Codec string

const (
	// CodecUnknown indicates an unknown codec token.
	CodecUnknown Codec = "unknown"
	// CodecMP3 indicates MP3 audio.
	CodecMP3 Codec = "mp3"
	// CodecFLAC indicates plain FLAC audio.
	CodecFLAC Codec = "flac"
	// CodecFLACMP4 indicates FLAC audio in an MP4 container.
	CodecFLACMP4 Codec = "flac-mp4"
	// CodecAAC indicates AAC audio.
	CodecAAC Codec = "aac"
	// CodecHEAAC indicates HE-AAC audio.
	CodecHEAAC Codec = "he-aac"
	// CodecAACMP4 indicates AAC audio in an MP4 container.
	CodecAACMP4 Codec = "aac-mp4"
	// CodecHEAACMP4 indicates HE-AAC audio in an MP4 container.
	CodecHEAACMP4 Codec = "he-aac-mp4"
)

// ParseCodec normalizes a raw provider codec value.
func ParseCodec(raw string) Codec {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "mp3":
		return CodecMP3
	case string(CodecFLAC):
		return CodecFLAC
	case "flac-mp4":
		return CodecFLACMP4
	case "aac":
		return CodecAAC
	case "he-aac":
		return CodecHEAAC
	case "aac-mp4":
		return CodecAACMP4
	case "he-aac-mp4":
		return CodecHEAACMP4
	default:
		return CodecUnknown
	}
}

// String returns a stable technical codec token.
func (c Codec) String() string {
	if c == "" {
		return string(CodecUnknown)
	}

	return string(c)
}

// Description returns a human-readable codec or container label.
func (c Codec) Description() string {
	switch c {
	case CodecMP3:
		return "MP3"
	case CodecFLAC:
		return qualityFLACDescription
	case CodecFLACMP4:
		return "FLAC in MP4 container"
	case CodecAAC:
		return "AAC"
	case CodecHEAAC:
		return "HE-AAC"
	case CodecAACMP4:
		return "AAC in MP4 container"
	case CodecHEAACMP4:
		return "HE-AAC in MP4 container"
	default:
		return "unknown codec"
	}
}

// IsPlainFLAC reports whether the codec is directly writable as a regular FLAC file.
func (c Codec) IsPlainFLAC() bool {
	return c == CodecFLAC
}

// IsFLACFamily reports whether the codec is FLAC-based.
func (c Codec) IsFLACFamily() bool {
	return c == CodecFLAC || c == CodecFLACMP4
}
