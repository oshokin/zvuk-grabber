package zvuk

// Tag keys used across:
// - filename/folder templates (see README placeholders)
// - audio tag writing (ID3/Vorbis)
//
// Keeping these in one place prevents typos and makes refactors safer.
const (
	// TagType is a shared/general tag key.
	TagType            = "type"
	TagCollectionTitle = "collectionTitle"

	// TagAlbumArtist is an album (release) tag key.
	TagAlbumArtist      = "albumArtist"
	TagAlbumID          = "albumID"
	TagAlbumTitle       = "albumTitle"
	TagAlbumTrackCount  = "albumTrackCount"
	TagRecordLabel      = "recordLabel"
	TagReleaseDate      = "releaseDate"
	TagReleaseTimestamp = "releaseTimestamp"
	TagReleaseYear      = "releaseYear"

	// TagPlaylistID is a playlist tag key.
	TagPlaylistID         = "playlistID"
	TagPlaylistTitle      = "playlistTitle"
	TagPlaylistTrackCount = "playlistTrackCount"

	// TagTrackArtist is a track (common) tag key.
	TagTrackArtist    = "trackArtist"
	TagTrackCount     = "trackCount"
	TagTrackDuration  = "trackDuration"
	TagTrackGenre     = "trackGenre"
	TagTrackID        = "trackID"
	TagTrackNumber    = "trackNumber"
	TagTrackNumberPad = "trackNumberPad"
	TagTrackTitle     = "trackTitle"

	// TagAudiobookID is an audiobook tag key.
	TagAudiobookID              = "audiobookID"
	TagAudiobookTitle           = "audiobookTitle"
	TagAudiobookAuthors         = "audiobookAuthors"
	TagAudiobookTrackCount      = "audiobookTrackCount"
	TagAudiobookPublisher       = "audiobookPublisher"
	TagAudiobookPublisherName   = "audiobookPublisherName"
	TagAudiobookCopyright       = "audiobookCopyright"
	TagAudiobookDescription     = "audiobookDescription"
	TagAudiobookPerformers      = "audiobookPerformers"
	TagAudiobookGenres          = "audiobookGenres"
	TagAudiobookAgeLimit        = "audiobookAgeLimit"
	TagAudiobookDuration        = "audiobookDuration"
	TagAudiobookPublicationDate = "audiobookPublicationDate"
	TagPublishYear              = "publishYear"

	// TagPodcastID is a podcast tag key.
	TagPodcastID          = "podcastID"
	TagPodcastTitle       = "podcastTitle"
	TagPodcastAuthors     = "podcastAuthors"
	TagPodcastTrackCount  = "podcastTrackCount"
	TagPodcastDescription = "podcastDescription"
	TagPodcastCategory    = "podcastCategory"
	TagPodcastExplicit    = "podcastExplicit"

	// TagEpisodePublicationDate is a podcast episode tag key (and aliases to track keys).
	TagEpisodePublicationDate = "episodePublicationDate"
	TagEpisodeID              = "episodeID"
	TagEpisodeTitle           = "episodeTitle"
	TagEpisodeNumber          = "episodeNumber"
	TagEpisodeNumberPad       = "episodeNumberPad"
	TagEpisodeDuration        = "episodeDuration"
)
