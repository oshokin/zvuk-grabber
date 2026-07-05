package media

// Tag keys used across:
// - filename/folder templates (see README placeholders)
// - audio tag writing (ID3/Vorbis)
//
// Keeping these in one place prevents typos and makes refactors safer.
const (
	// TagType is a shared/general tag key.
	TagType = "type"
	// TagCollectionTitle is the collection title tag key.
	TagCollectionTitle = "collectionTitle"

	// TagAlbumArtist is the album artist tag key.
	TagAlbumArtist = "albumArtist"
	// TagAlbumID is the album identifier tag key.
	TagAlbumID = "albumID"
	// TagAlbumTitle is the album title tag key.
	TagAlbumTitle = "albumTitle"
	// TagAlbumTrackCount is the album track count tag key.
	TagAlbumTrackCount = "albumTrackCount"
	// TagRecordLabel is the record label tag key.
	TagRecordLabel = "recordLabel"
	// TagReleaseDate is the release date tag key.
	TagReleaseDate = "releaseDate"
	// TagReleaseTimestamp is the release timestamp tag key.
	TagReleaseTimestamp = "releaseTimestamp"
	// TagReleaseYear is the release year tag key.
	TagReleaseYear = "releaseYear"

	// TagPlaylistID is the playlist identifier tag key.
	TagPlaylistID = "playlistID"
	// TagPlaylistTitle is the playlist title tag key.
	TagPlaylistTitle = "playlistTitle"
	// TagPlaylistTrackCount is the playlist track count tag key.
	TagPlaylistTrackCount = "playlistTrackCount"

	// TagTrackArtist is the track artist tag key.
	TagTrackArtist = "trackArtist"
	// TagTrackCount is the track count tag key.
	TagTrackCount = "trackCount"
	// TagTrackDuration is the track duration tag key.
	TagTrackDuration = "trackDuration"
	// TagTrackGenre is the track genre tag key.
	TagTrackGenre = "trackGenre"
	// TagTrackID is the track identifier tag key.
	TagTrackID = "trackID"
	// TagTrackNumber is the track number tag key.
	TagTrackNumber = "trackNumber"
	// TagTrackNumberPad is the zero-padded track number tag key.
	TagTrackNumberPad = "trackNumberPad"
	// TagTrackTitle is the track title tag key.
	TagTrackTitle = "trackTitle"

	// TagAudiobookID is the audiobook identifier tag key.
	TagAudiobookID = "audiobookID"
	// TagAudiobookTitle is the audiobook title tag key.
	TagAudiobookTitle = "audiobookTitle"
	// TagAudiobookAuthors is the audiobook authors tag key.
	TagAudiobookAuthors = "audiobookAuthors"
	// TagAudiobookTrackCount is the audiobook chapter count tag key.
	TagAudiobookTrackCount = "audiobookTrackCount"
	// TagAudiobookPublisher is the audiobook publisher tag key.
	TagAudiobookPublisher = "audiobookPublisher"
	// TagAudiobookPublisherName is the audiobook publisher name tag key.
	TagAudiobookPublisherName = "audiobookPublisherName"
	// TagAudiobookCopyright is the audiobook copyright tag key.
	TagAudiobookCopyright = "audiobookCopyright"
	// TagAudiobookDescription is the audiobook description tag key.
	TagAudiobookDescription = "audiobookDescription"
	// TagAudiobookPerformers is the audiobook performers tag key.
	TagAudiobookPerformers = "audiobookPerformers"
	// TagAudiobookGenres is the audiobook genres tag key.
	TagAudiobookGenres = "audiobookGenres"
	// TagAudiobookAgeLimit is the audiobook age limit tag key.
	TagAudiobookAgeLimit = "audiobookAgeLimit"
	// TagAudiobookDuration is the audiobook duration tag key.
	TagAudiobookDuration = "audiobookDuration"
	// TagAudiobookPublicationDate is the audiobook publication date tag key.
	TagAudiobookPublicationDate = "audiobookPublicationDate"
	// TagPublishYear is the publication year tag key.
	TagPublishYear = "publishYear"

	// TagPodcastID is the podcast identifier tag key.
	TagPodcastID = "podcastID"
	// TagPodcastTitle is the podcast title tag key.
	TagPodcastTitle = "podcastTitle"
	// TagPodcastAuthors is the podcast authors tag key.
	TagPodcastAuthors = "podcastAuthors"
	// TagPodcastTrackCount is the podcast episode count tag key.
	TagPodcastTrackCount = "podcastTrackCount"
	// TagPodcastDescription is the podcast description tag key.
	TagPodcastDescription = "podcastDescription"
	// TagPodcastCategory is the podcast category tag key.
	TagPodcastCategory = "podcastCategory"
	// TagPodcastExplicit is the podcast explicit-content flag tag key.
	TagPodcastExplicit = "podcastExplicit"

	// TagEpisodePublicationDate is the episode publication date tag key.
	TagEpisodePublicationDate = "episodePublicationDate"
	// TagEpisodeID is the episode identifier tag key.
	TagEpisodeID = "episodeID"
	// TagEpisodeTitle is the episode title tag key.
	TagEpisodeTitle = "episodeTitle"
	// TagEpisodeNumber is the episode number tag key.
	TagEpisodeNumber = "episodeNumber"
	// TagEpisodeNumberPad is the zero-padded episode number tag key.
	TagEpisodeNumberPad = "episodeNumberPad"
	// TagEpisodeDuration is the episode duration tag key.
	TagEpisodeDuration = "episodeDuration"
)
