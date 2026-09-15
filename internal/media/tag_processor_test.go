package media

import (
	"testing"

	"github.com/oshokin/id3v2/v2"
	"github.com/stretchr/testify/assert"
)

// TestAddMP3Tags_WritesUnsynchronisedLyricsByDefault verifies plain lyrics use USLT frames.
func TestAddMP3Tags_WritesUnsynchronisedLyricsByDefault(t *testing.T) {
	t.Parallel()

	processor := new(TagProcessorImpl)
	tag := id3v2.NewEmptyTag()
	request := &WriteTagsRequest{
		Lyrics: "Plain lyrics without synchronization",
	}

	processor.addMP3Tags(t.Context(), tag, request)

	assert.Len(t, tag.GetFrames("USLT"), 1)
	assert.Empty(t, tag.GetFrames("SYLT"))
}

// TestAddMP3Tags_WritesSynchronisedLyricsForSubtitleType verifies subtitle lyrics use SYLT frames.
func TestAddMP3Tags_WritesSynchronisedLyricsForSubtitleType(t *testing.T) {
	t.Parallel()

	processor := new(TagProcessorImpl)
	tag := id3v2.NewEmptyTag()
	request := &WriteTagsRequest{
		Lyrics:     "[00:00.00]Hello\n[00:01.00]World",
		LyricsType: LyricsTypeSubtitle,
	}

	processor.addMP3Tags(t.Context(), tag, request)

	assert.Len(t, tag.GetFrames("SYLT"), 1)
	assert.Empty(t, tag.GetFrames("USLT"))
}

// TestAddMP3Tags_FallsBackToUnsynchronisedLyricsWhenLRCParseFails verifies invalid LRC falls back to USLT.
func TestAddMP3Tags_FallsBackToUnsynchronisedLyricsWhenLRCParseFails(t *testing.T) {
	t.Parallel()

	processor := new(TagProcessorImpl)
	tag := id3v2.NewEmptyTag()
	request := &WriteTagsRequest{
		Lyrics:     "not an lrc payload",
		LyricsType: LyricsTypeLRC,
	}

	processor.addMP3Tags(t.Context(), tag, request)

	assert.Empty(t, tag.GetFrames("SYLT"))
	assert.Len(t, tag.GetFrames("USLT"), 1)
}
