package zvuk

//go:generate $MOCKGEN -source=tag_processor.go -destination=mocks/tag_processor_mock.go

import "github.com/oshokin/zvuk-grabber/internal/media"

// TODO: Remove this compatibility shim after migrating remaining mocks/tests to internal/media.

// TagProcessor is a compatibility alias to the shared media tag processor.
type TagProcessor = media.TagProcessor

// WriteTagsRequest is a compatibility alias to the shared media write request.
type WriteTagsRequest = media.WriteTagsRequest

// NewTagProcessor creates a shared tag processor.
func NewTagProcessor() TagProcessor {
	return media.NewTagProcessor()
}
