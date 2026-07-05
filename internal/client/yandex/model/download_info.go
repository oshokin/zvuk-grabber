package model

// DownloadInfo describes one available MP3 download option for a track.
type DownloadInfo struct {
	// BitrateInKbps is the MP3 bitrate in kilobits per second.
	BitrateInKbps int `json:"bitrateInKbps"`
	// Codec is the audio codec name, typically mp3.
	Codec string `json:"codec"`
	// Direct reports whether the stream can be downloaded directly.
	Direct bool `json:"direct"`
	// DownloadInfoURL is the URL that resolves to a signed direct download link.
	DownloadInfoURL string `json:"downloadInfoUrl"`
	// Gain reports whether loudness normalization is enabled.
	Gain bool `json:"gain"`
	// Preview reports whether the option is a preview stream.
	Preview bool `json:"preview"`
}
