package zvuk

const (
	// API Quality parameters.
	defaultStreamQuality = "hifi"
	defaultEncodeType    = "wv"
)

const (
	// zvukAPIGraphQLURI is the URI path for GraphQL API endpoint.
	zvukAPIGraphQLURI = "api/v1/graphql"
	// zvukAPILabelURI is the URI path for label metadata endpoint.
	zvukAPILabelURI = "api/tiny/labels"
	// zvukAPILyricsURI is the URI path for lyrics endpoint.
	zvukAPILyricsURI = "api/tiny/lyrics"
	// zvukAPIPlaylistURI is the URI path for playlist metadata endpoint.
	zvukAPIPlaylistURI = "api/tiny/playlists"
	// zvukAPIReleaseMetadataURI is the URI path for release metadata endpoint.
	zvukAPIReleaseMetadataURI = "api/tiny/releases"
	// zvukAPIReleaseURIPath is the URI path component for releases.
	zvukAPIReleaseURIPath = "releases"
	// zvukAPIStreamMetadataURI is the URI path for stream metadata endpoint.
	zvukAPIStreamMetadataURI = "api/tiny/track/stream"
	// zvukAPITrackURI is the URI path for track metadata endpoint.
	zvukAPITrackURI = "api/tiny/tracks"
	// zvukAPIUserProfileURI is the URI path for user profile endpoint.
	zvukAPIUserProfileURI = "api/v2/tiny/profile"
)

const (
	// labelsCacheSize defines the maximum number of label entries to cache.
	// Approximately 500 unique labels exist globally across all music.
	labelsCacheSize = 500
	// albumsCacheSize defines the maximum number of album entries to cache.
	// Sized to hold recent albums accessed during typical usage.
	albumsCacheSize = 5000
	// tracksCacheSize defines the maximum number of track entries to cache.
	// Sized to hold recently accessed tracks.
	tracksCacheSize = 10000
	// playlistsCacheSize defines the maximum number of playlist entries to cache.
	// Playlists don't change frequently, so we cache them.
	playlistsCacheSize = 2000
	// audiobooksCacheSize defines the maximum number of audiobook entries to cache.
	// Audiobooks don't change frequently, so we cache them.
	audiobooksCacheSize = 2000
	// podcastsCacheSize defines the maximum number of podcast entries to cache.
	// Podcasts don't change frequently, so we cache them.
	podcastsCacheSize = 2000
)
