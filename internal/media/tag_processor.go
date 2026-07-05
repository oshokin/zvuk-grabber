package media

//go:generate $MOCKGEN -source=tag_processor.go -destination=mocks/tag_processor_mock.go

import (
	"context"
	"errors"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
	"github.com/oshokin/id3v2/v2"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// TagProcessor defines the interface for writing metadata tags to audio files.
type TagProcessor interface {
	// WriteTags writes metadata tags to the audio file described by the request.
	WriteTags(ctx context.Context, req *WriteTagsRequest) error
}

// WriteTagsRequest contains parameters for writing metadata to audio files.
type WriteTagsRequest struct {
	// TrackPath is the file path of the audio track.
	TrackPath string
	// CoverPath is the file path of the cover art image.
	CoverPath string
	// Quality specifies the audio quality level.
	Quality Quality
	// Tags contains metadata key-value pairs to write.
	Tags map[string]string
	// Lyrics contains plain lyrics to write into tags.
	Lyrics string
	// LyricsType indicates the provider lyrics format (for example, subtitle/lrc).
	LyricsType string
	// EmbedCover indicates whether cover art is embedded in the audio file.
	EmbedCover bool
}

// TagProcessorImpl provides the default implementation of TagProcessor.
type TagProcessorImpl struct{}

// imageMetadata contains image data and its MIME type.
type imageMetadata struct {
	// data contains the raw image bytes.
	data []byte
	// mimeType specifies the image format (e.g., "image/jpeg").
	mimeType string
}

// extractFLACCommentResult contains the result of extracting FLAC comment metadata.
type extractFLACCommentResult struct {
	// Comment is the FLAC Vorbis comment metadata block.
	Comment *flacvorbis.MetaDataBlockVorbisComment
	// Index is the index of the comment block in the FLAC file metadata (-1 if not found).
	Index int
}

const (
	// LyricsTypeSubtitle identifies synchronized subtitle-style lyrics payloads.
	LyricsTypeSubtitle = "subtitle"
	// LyricsTypeLRC identifies synchronized LRC lyrics payloads.
	LyricsTypeLRC = "lrc"
)

// Static error definitions for better error handling.
var (
	// ErrEmptyTrackPath indicates that the track file path is empty.
	ErrEmptyTrackPath = errors.New("track path cannot be empty")
)

// NewTagProcessor creates a new TagProcessor instance.
func NewTagProcessor() TagProcessor {
	return new(TagProcessorImpl)
}

// WriteTags writes metadata to audio files based on the provided request.
func (tp *TagProcessorImpl) WriteTags(ctx context.Context, req *WriteTagsRequest) error {
	if req.TrackPath == "" {
		return ErrEmptyTrackPath
	}

	var image *imageMetadata

	// If a cover path is provided and embedding is enabled, read the cover art.
	if req.CoverPath != "" && req.EmbedCover {
		imageData, err := os.ReadFile(filepath.Clean(req.CoverPath))
		if err != nil {
			logger.Errorf(ctx, "Failed to read cover file '%s': %v", req.CoverPath, err)
			return err
		}

		// Determine the MIME type of the cover art based on its file extension.
		imageMIMEType := mime.TypeByExtension(filepath.Ext(req.CoverPath))

		logger.Debugf(ctx, "Cover image loaded from disk: %s, MIME: %s, path: %s",
			humanize.IBytes(uint64(len(imageData))), imageMIMEType, req.CoverPath)

		image = &imageMetadata{
			data:     imageData,
			mimeType: imageMIMEType,
		}
	}

	// Write tags based on the track quality (FLAC or MP3).
	if req.Quality == QualityFLAC {
		return tp.writeFLACTags(ctx, req, image)
	}

	return tp.writeMP3Tags(ctx, req, image)
}

// writeFLACTags writes metadata and optional cover art to a FLAC file.
func (tp *TagProcessorImpl) writeFLACTags(ctx context.Context, req *WriteTagsRequest, image *imageMetadata) error {
	// Parse the FLAC file.
	f, err := flac.ParseFile(filepath.Clean(req.TrackPath))
	if err != nil {
		return err
	}

	// Extract existing FLAC comments (metadata) from the parsed file.
	commentResult := tp.extractFLACComment(f)
	tag := commentResult.Comment

	// If no existing comments are found, create a new metadata block.
	if tag == nil {
		tag = flacvorbis.New()
	}

	// Add tags to the FLAC metadata block.
	err = tp.addFLACTags(tag, req)
	if err != nil {
		return err
	}

	// Marshal the updated metadata and update the FLAC file's metadata blocks.
	tagMeta := tag.Marshal()
	if commentResult.Index >= 0 {
		f.Meta[commentResult.Index] = &tagMeta
	} else {
		f.Meta = append(f.Meta, &tagMeta)
	}

	// Embed the cover art into the FLAC file if provided.
	tp.embedFLACCover(ctx, f, image)

	// Save the updated FLAC file.
	return f.Save(req.TrackPath)
}

// extractFLACComment finds and parses the Vorbis comment block from a FLAC file.
func (tp *TagProcessorImpl) extractFLACComment(f *flac.File) *extractFLACCommentResult {
	// Iterate through the metadata blocks to find the Vorbis comment block.
	for idx, meta := range f.Meta {
		if meta.Type != flac.VorbisComment {
			continue
		}

		comment, err := flacvorbis.ParseFromMetaDataBlock(*meta)
		if err == nil {
			return &extractFLACCommentResult{Comment: comment, Index: idx}
		}
	}

	// Return nil comment if no Vorbis comment block is found.
	return &extractFLACCommentResult{Index: -1}
}

// addFLACTags maps track tags to Vorbis comment fields and adds them to the block.
func (tp *TagProcessorImpl) addFLACTags(tag *flacvorbis.MetaDataBlockVorbisComment, req *WriteTagsRequest) error {
	// Map of FLAC tag keys to their corresponding values in req.Tags.
	flacTags := map[string]string{
		"ALBUM":       req.Tags[TagCollectionTitle],
		"ALBUMARTIST": req.Tags[TagAlbumArtist],
		"ARTIST":      req.Tags[TagTrackArtist],
		"COPYRIGHT":   req.Tags[TagRecordLabel],
		"DATE":        req.Tags[TagReleaseDate],
		"GENRE":       req.Tags[TagTrackGenre],
		"PLAYLIST_ID": req.Tags[TagPlaylistID],
		"RELEASE_ID":  req.Tags[TagAlbumID],
		"TITLE":       req.Tags[TagTrackTitle],
		"TOTALTRACKS": req.Tags[TagTrackCount],
		"TRACK_ID":    req.Tags[TagTrackID],
		"TRACKNUMBER": req.Tags[TagTrackNumber],
		"YEAR":        req.Tags[TagReleaseYear],
		"DESCRIPTION": req.Tags[TagAudiobookDescription],
		"PERFORMER":   req.Tags[TagAudiobookPerformers],
	}

	if strings.TrimSpace(req.Lyrics) != "" {
		flacTags["LYRICS"] = req.Lyrics
	}

	// Add each tag to the Vorbis comment block.
	for k, v := range flacTags {
		if v == "" {
			continue
		}

		err := tag.Add(k, v)
		if err != nil {
			return err
		}
	}

	return nil
}

// embedFLACCover appends a picture metadata block to the FLAC file when cover art is provided.
func (tp *TagProcessorImpl) embedFLACCover(ctx context.Context, f *flac.File, image *imageMetadata) {
	if image == nil {
		return
	}

	// Create a new FLAC picture block from the image data.
	picture, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "", image.data, image.mimeType)
	if err != nil {
		logger.Errorf(ctx, "Failed to embed image to FLAC: %v", err)

		return
	}

	// Add the picture block to the FLAC file's metadata.
	pictureMeta := picture.Marshal()
	f.Meta = append(f.Meta, &pictureMeta)
}

// writeMP3Tags writes metadata and optional cover art to an MP3 file.
func (tp *TagProcessorImpl) writeMP3Tags(ctx context.Context, req *WriteTagsRequest, image *imageMetadata) error {
	// Open the MP3 file for writing metadata.
	//nolint:exhaustruct // ParseFrames intentionally omitted when Parse=false (parsing disabled).
	tag, err := id3v2.Open(req.TrackPath, id3v2.Options{Parse: false})
	if err != nil {
		return err
	}

	defer tag.Close()

	// Add metadata tags to the MP3 file.
	tp.addMP3Tags(ctx, tag, req)

	// Embed the cover art into the MP3 file if provided.
	if image != nil {
		logger.Debugf(
			ctx,
			"Embedding cover to MP3: %s, MIME: %s",
			humanize.IBytes(uint64(len(image.data))),
			image.mimeType,
		)

		//nolint:exhaustruct // Description field intentionally empty for cover images.
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    image.mimeType,
			PictureType: id3v2.PTFrontCover,
			Picture:     image.data,
		})

		logger.Debugf(ctx, "Cover embedded successfully to MP3")
	}

	// Save the updated MP3 file.
	return tag.Save()
}

// addMP3Tags adds metadata tags to an MP3 file.
func (tp *TagProcessorImpl) addMP3Tags(ctx context.Context, tag *id3v2.Tag, req *WriteTagsRequest) {
	// Set default encoding for the tags.
	tag.SetDefaultEncoding(id3v2.EncodingUTF8)

	// Add basic metadata tags.
	tag.SetAlbum(req.Tags[TagCollectionTitle])
	tag.SetArtist(req.Tags[TagTrackArtist])
	tag.SetGenre(req.Tags[TagTrackGenre])
	tag.SetTitle(req.Tags[TagTrackTitle])
	tag.SetYear(req.Tags[TagReleaseYear])

	// Add track number and total tracks (e.g., "1/10").
	var (
		trackNumber = req.Tags[TagTrackNumber]
		trackCount  = req.Tags[TagTrackCount]
	)

	if trackNumber != "" && trackCount != "" {
		tag.AddTextFrame(
			tag.CommonID("Track number/Position in set"),
			tag.DefaultEncoding(),
			trackNumber+"/"+trackCount,
		)
	}

	// Add additional metadata tags.
	tag.AddTextFrame(tag.CommonID("Band/Orchestra/Accompaniment"), tag.DefaultEncoding(), req.Tags[TagAlbumArtist])
	tag.AddTextFrame(tag.CommonID("Publisher"), tag.DefaultEncoding(), req.Tags[TagRecordLabel])

	// Add audiobook-specific metadata.
	if req.Tags[TagAudiobookPerformers] != "" {
		tag.AddTextFrame(
			tag.CommonID("Conductor/Performer refinement"),
			tag.DefaultEncoding(),
			req.Tags[TagAudiobookPerformers],
		)
	}

	if req.Tags[TagAudiobookDescription] != "" {
		tag.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "Description",
			Text:        req.Tags[TagAudiobookDescription],
		})
	}

	lyrics := strings.TrimSpace(req.Lyrics)
	if lyrics == "" {
		return
	}

	if isSynchronizedLyricsType(req.LyricsType) {
		parsed, err := id3v2.ParseLRCFile(strings.NewReader(lyrics))
		if err == nil && len(parsed.SynchronizedTexts) > 0 {
			tag.AddSynchronisedLyricsFrame(id3v2.SynchronisedLyricsFrame{
				Encoding: id3v2.EncodingUTF8,
				// Field is required, so we just use lingua franca.
				Language: id3v2.EnglishISO6392Code,
				// Use absolute timestamps.
				TimestampFormat: id3v2.SYLTAbsoluteMillisecondsTimestampFormat,
				// Mark as lyrics.
				ContentType: id3v2.SYLTLyricsContentType,
				// Descriptor for lyrics.
				ContentDescriptor: "Lyrics",
				// Parsed synchronized lyrics.
				SynchronizedTexts: parsed.SynchronizedTexts,
			})

			return
		}

		if err != nil {
			logger.Debugf(ctx, "Failed to parse synchronized lyrics, falling back to plain lyrics: %v", err)
		}
	}

	tag.AddUnsynchronisedLyricsFrame(
		//nolint:exhaustruct // ContentDescriptor not available in source data.
		id3v2.UnsynchronisedLyricsFrame{
			Encoding: id3v2.EncodingUTF8,
			Lyrics:   lyrics,
			// Field is required, so we just use lingua franca.
			Language: id3v2.EnglishISO6392Code,
		},
	)
}

// isSynchronizedLyricsType reports whether the lyrics type should be stored as synchronized frames.
func isSynchronizedLyricsType(lyricsType string) bool {
	switch strings.ToLower(strings.TrimSpace(lyricsType)) {
	case LyricsTypeSubtitle, LyricsTypeLRC:
		return true
	default:
		return false
	}
}
