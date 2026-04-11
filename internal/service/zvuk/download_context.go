package zvuk

// // trackDownloadData encapsulates all data needed for downloading a single track.
// type trackDownloadData struct {
// 	// Metadata.
// 	Metadata *downloadTracksMetadata

// 	// Track metadata.
// 	TrackID       string
// 	Track         *zvuk.Track
// 	TrackIndex    int64
// 	TrackPosition int64
// 	TrackFilename string
// 	TrackPath     string

// 	// Quality and streaming.
// 	Quality   TrackQuality
// 	StreamURL string

// 	// Collection context.
// 	AlbumTags map[string]string
// 	Album     *zvuk.Release

// 	// Error reporting context.
// 	ParentID    string
// 	ParentTitle string
// }

// // NewTrackDownloadData creates a download context with basic information.
// func NewTrackDownloadData(
// 	metadata *downloadTracksMetadata,
// 	trackID string,
// 	track *zvuk.Track,
// 	trackIndex int64,
// ) *trackDownloadData {
// 	return &trackDownloadData{
// 		Metadata:   metadata,
// 		TrackID:    trackID,
// 		Track:      track,
// 		TrackIndex: trackIndex,
// 	}
// }
