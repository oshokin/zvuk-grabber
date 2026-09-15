package zvuk

import "github.com/oshokin/zvuk-grabber/internal/media"

// Compatibility aliases for the shared media tag processor.

// TagProcessor is a compatibility alias to the shared media tag processor.
type TagProcessor = media.TagProcessor

// WriteTagsRequest is a compatibility alias to the shared media write request.
type WriteTagsRequest = media.WriteTagsRequest

// NewTagProcessor creates a shared tag processor.
func NewTagProcessor() TagProcessor {
	return media.NewTagProcessor()
}
