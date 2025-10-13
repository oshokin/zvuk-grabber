package zvuk

import "errors"

var (
	// ErrUnexpectedHTTPStatus indicates an unexpected HTTP status code was received.
	ErrUnexpectedHTTPStatus = errors.New("unexpected HTTP status")
	// ErrArtistNotFound indicates that the requested artist was not found.
	ErrArtistNotFound = errors.New("artist not found")
	// ErrUnexpectedArtistResponseFormat indicates an unexpected artist API response format.
	ErrUnexpectedArtistResponseFormat = errors.New("unexpected artist response format")
	// ErrUnexpectedReleasesResponseFormat indicates an unexpected releases API response format.
	ErrUnexpectedReleasesResponseFormat = errors.New("unexpected releases response format")
	// ErrFailedToFetchStreamMetadata indicates failure to fetch stream metadata after all retry attempts.
	ErrFailedToFetchStreamMetadata = errors.New("failed to fetch stream metadata after retries")
	// ErrAudiobookNotFound is returned when audiobook is not found in GraphQL response.
	ErrAudiobookNotFound = errors.New("audiobook not found or unexpected response format")
	// ErrUnexpectedAudiobookFormat is returned when audiobook response has unexpected format.
	ErrUnexpectedAudiobookFormat = errors.New("unexpected audiobook response format")
	// ErrUnexpectedMediaContentsFormat is returned when mediaContents response has unexpected format.
	ErrUnexpectedMediaContentsFormat = errors.New("unexpected mediaContents response format")
	// ErrPodcastNotFound is returned when podcast is not found in GraphQL response.
	ErrPodcastNotFound = errors.New("podcast not found or unexpected response format")
	// ErrUnexpectedPodcastFormat is returned when podcast response has unexpected format.
	ErrUnexpectedPodcastFormat = errors.New("unexpected podcast response format")
)
