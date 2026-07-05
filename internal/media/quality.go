package media

// Quality represents audio quality used by providers and writers.
type Quality uint8

const (
	// QualityUnknown means quality could not be determined.
	QualityUnknown Quality = iota
	// QualityMP3Mid is preferred standard MP3 quality (128 Kbps when available).
	QualityMP3Mid
	// QualityMP3High is MP3 320 Kbps.
	QualityMP3High
	// QualityFLAC is lossless FLAC.
	QualityFLAC
)

const (
	// QualityMP3MidString is API token for standard MP3.
	QualityMP3MidString = "mid"
	// QualityMP3HighString is API token for high MP3.
	QualityMP3HighString = "high"
	// QualityFLACString is API token for FLAC.
	QualityFLACString = "flac"
	// qualityFLACDescription is the human-readable label shared by FLAC quality descriptions.
	qualityFLACDescription = "FLAC lossless"
)

// String returns a stable short quality token for logs.
func (q Quality) String() string {
	switch q {
	case QualityMP3Mid:
		return "mp3_128"
	case QualityMP3High:
		return "mp3_320"
	case QualityFLAC:
		return QualityFLACString
	default:
		return "unknown"
	}
}

// Description returns a human-readable quality label.
func (q Quality) Description() string {
	switch q {
	case QualityMP3Mid:
		return "MP3, 128 Kbps"
	case QualityMP3High:
		return "MP3, 320 Kbps"
	case QualityFLAC:
		return qualityFLACDescription
	default:
		return "unknown quality"
	}
}

// Extension returns output filename extension for quality.
func (q Quality) Extension() string {
	switch q {
	case QualityMP3Mid, QualityMP3High:
		return ".mp3"
	case QualityFLAC:
		return ".flac"
	default:
		return ".bin"
	}
}

// ParseQuality parses quality string identifiers.
func ParseQuality(value string) Quality {
	switch value {
	case QualityMP3MidString, "med":
		return QualityMP3Mid
	case QualityMP3HighString:
		return QualityMP3High
	case QualityFLACString:
		return QualityFLAC
	default:
		return QualityUnknown
	}
}

// FromConfig maps config numeric quality to media quality.
func FromConfig(value uint8) Quality {
	switch Quality(value) {
	case QualityMP3Mid:
		return QualityMP3Mid
	case QualityMP3High:
		return QualityMP3High
	case QualityFLAC:
		return QualityFLAC
	default:
		return QualityUnknown
	}
}
