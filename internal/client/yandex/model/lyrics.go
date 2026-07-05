package model

import "strings"

// TrackLyricsResponse is the common Yandex Music API wrapper for track lyrics.
type TrackLyricsResponse struct {
	// Result contains lyrics metadata and text fields.
	Result *TrackLyrics `json:"result"`
}

// TrackLyrics represents lyrics returned by Yandex Music.
type TrackLyrics struct {
	// ID is the lyrics identifier returned by older API variants.
	ID int64 `json:"id,omitempty"`
	// LyricID is the lyrics identifier returned by newer track lyrics responses.
	LyricID int64 `json:"lyricId,omitempty"`
	// LyricIDSnake keeps compatibility with snake_case API responses.
	LyricIDSnake int64 `json:"lyric_id,omitempty"`
	// Lyrics stores the preview or full text in older API variants.
	Lyrics string `json:"lyrics,omitempty"`
	// FullLyrics stores the full plain-text lyrics in camelCase responses.
	FullLyrics string `json:"fullLyrics,omitempty"`
	// FullLyricsSnake stores the full plain-text lyrics in snake_case responses.
	FullLyricsSnake string `json:"full_lyrics,omitempty"`
	// DownloadURL points to downloadable text in camelCase responses.
	DownloadURL string `json:"downloadUrl,omitempty"`
	// DownloadURLSnake points to downloadable text in snake_case responses.
	DownloadURLSnake string `json:"download_url,omitempty"`
	// HasRights reports whether lyrics can be shown by the provider.
	HasRights bool `json:"hasRights,omitempty"`
	// HasRightsSnake keeps compatibility with snake_case API responses.
	HasRightsSnake bool `json:"has_rights,omitempty"`
	// TextLanguage is the provider-reported lyrics language.
	TextLanguage string `json:"textLanguage,omitempty"`
	// TextLanguageSnake is the provider-reported lyrics language in snake_case responses.
	TextLanguageSnake string `json:"text_language,omitempty"`
	// URL is an optional source URL, often a lyrics provider page.
	URL string `json:"url,omitempty"`
}

// Text returns the best embedded text available in the response.
func (l *TrackLyrics) Text() string {
	if l == nil {
		return ""
	}

	if fullLyrics := strings.TrimSpace(l.FullLyrics); fullLyrics != "" {
		return fullLyrics
	}

	if fullLyrics := strings.TrimSpace(l.FullLyricsSnake); fullLyrics != "" {
		return fullLyrics
	}

	return strings.TrimSpace(l.Lyrics)
}

// DownloadLink returns the best text download URL available in the response.
func (l *TrackLyrics) DownloadLink() string {
	if l == nil {
		return ""
	}

	if downloadURL := strings.TrimSpace(l.DownloadURL); downloadURL != "" {
		return downloadURL
	}

	return strings.TrimSpace(l.DownloadURLSnake)
}
