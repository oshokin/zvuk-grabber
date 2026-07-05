package model

// PlaylistResponse wraps a single playlist API response.
type PlaylistResponse struct {
	// Result contains the playlist payload.
	Result Playlist `json:"result"`
}

// PlaylistsResponse wraps a list of playlists from API responses.
type PlaylistsResponse struct {
	// Result contains the playlist list payload.
	Result []Playlist `json:"result"`
}

// ChartResponse wraps a chart playlist API response.
type ChartResponse struct {
	// Result contains the chart playlist wrapper.
	Result struct {
		// Chart is the chart playlist payload.
		Chart *Playlist `json:"chart"`
	} `json:"result"`
}

// AlbumResponse wraps a single album API response.
type AlbumResponse struct {
	// Result contains the album payload.
	Result Album `json:"result"`
}

// AccountStatusResponse wraps the authenticated account status API response.
type AccountStatusResponse struct {
	// Result contains the account status payload.
	Result Status `json:"result"`
}

// DownloadInfoResponse wraps track download options from the API.
type DownloadInfoResponse struct {
	// Result contains available download options.
	Result []DownloadInfo `json:"result"`
}

// TracksResponse wraps one or more tracks from the API.
type TracksResponse struct {
	// Result contains the track list payload.
	Result []Track `json:"result"`
}

// Status represents account status metadata from the API.
type Status struct {
	// Account contains the authenticated account profile.
	Account Account `json:"account"`
}
