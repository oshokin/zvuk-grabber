package zvuk

//go:generate $MOCKGEN -source=tag_processor.go -destination=mocks/tag_processor_mock.go

import (
	"context"
	"errors"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
	"github.com/oshokin/id3v2/v2"

	"github.com/oshokin/zvuk-grabber/internal/client/zvuk"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// TagProcessor defines the interface for writing metadata tags to audio files.
type TagProcessor interface {
	WriteTags(ctx context.Context, req *WriteTagsRequest) error
}

// WriteTagsRequest contains parameters for writing metadata to audio files.
type WriteTagsRequest struct {
	// TrackPath is the file path of the audio track.
	TrackPath string
	// CoverPath is the file path of the cover art image.
	CoverPath string
	// Quality specifies the audio quality level.
	Quality TrackQuality
	// TrackTags contains metadata key-value pairs to write.
	TrackTags map[string]string
	// TrackLyrics contains the lyrics data for the track.
	TrackLyrics *zvuk.Lyrics
	// IsCoverEmbeddedToTrackTags indicates whether cover art is embedded in the audio file.
	IsCoverEmbeddedToTrackTags bool
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
	if req.CoverPath != "" && req.IsCoverEmbeddedToTrackTags {
		imageData, err := os.ReadFile(filepath.Clean(req.CoverPath))
		if err != nil {
			return err
		}

		// Determine the MIME type of the cover art based on its file extension.
		imageMIMEType := mime.TypeByExtension(filepath.Ext(req.CoverPath))
		image = &imageMetadata{
			data:     imageData,
			mimeType: imageMIMEType,
		}
	}

	// Write tags based on the track quality (FLAC or MP3).
	if req.Quality == TrackQualityFLAC {
		return tp.writeFLACTags(ctx, req, image)
	}

	return tp.writeMP3Tags(ctx, req, image)
}

func (tp *TagProcessorImpl) writeFLACTags(ctx context.Context, req *WriteTagsRequest, image *imageMetadata) error {
	// Parse the FLAC file.
	f, err := flac.ParseFile(filepath.Clean(req.TrackPath))
	if err != nil {
		return err
	}

	// Extract existing FLAC comments (metadata) from the file.
	commentResult, err := tp.extractFLACComment(req.TrackPath)
	if err != nil {
		return err
	}

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

func (tp *TagProcessorImpl) extractFLACComment(filename string) (*extractFLACCommentResult, error) {
	f, err := flac.ParseFile(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}

	// Iterate through the metadata blocks to find the Vorbis comment block.
	for idx, meta := range f.Meta {
		if meta.Type != flac.VorbisComment {
			continue
		}

		// Parse the Vorbis comment block.
		var comment *flacvorbis.MetaDataBlockVorbisComment

		comment, err = flacvorbis.ParseFromMetaDataBlock(*meta)
		if err == nil {
			return &extractFLACCommentResult{
				Comment: comment,
				Index:   idx,
			}, nil
		}
	}

	// Return nil comment if no Vorbis comment block is found.
	return &extractFLACCommentResult{
		Comment: nil,
		Index:   -1,
	}, nil
}

func (tp *TagProcessorImpl) addFLACTags(tag *flacvorbis.MetaDataBlockVorbisComment, req *WriteTagsRequest) error {
	// Map of FLAC tag keys to their corresponding values in req.TrackTags.
	flacTags := map[string]string{
		"ALBUM":       req.TrackTags["collectionTitle"],
		"ALBUMARTIST": req.TrackTags["albumArtist"],
		"ARTIST":      req.TrackTags["trackArtist"],
		"COPYRIGHT":   req.TrackTags["recordLabel"],
		"DATE":        req.TrackTags["releaseDate"],
		"GENRE":       req.TrackTags["trackGenre"],
		"PLAYLIST_ID": req.TrackTags["playlistID"],
		"RELEASE_ID":  req.TrackTags["albumID"],
		"TITLE":       req.TrackTags["trackTitle"],
		"TOTALTRACKS": req.TrackTags["trackCount"],
		"TRACK_ID":    req.TrackTags["trackID"],
		"TRACKNUMBER": req.TrackTags["trackNumber"],
		"YEAR":        req.TrackTags["releaseYear"],
	}

	if req.TrackLyrics != nil && strings.TrimSpace(req.TrackLyrics.Lyrics) != "" {
		flacTags["LYRICS"] = req.TrackLyrics.Lyrics
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
		//nolint:exhaustruct // Description field intentionally empty for cover images.
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    image.mimeType,
			PictureType: id3v2.PTFrontCover,
			Picture:     image.data,
		})
	}

	// Save the updated MP3 file.
	return tag.Save()
}

func (tp *TagProcessorImpl) addMP3Tags(ctx context.Context, tag *id3v2.Tag, req *WriteTagsRequest) {
	// Set default encoding for the tags.
	tag.SetDefaultEncoding(id3v2.EncodingUTF8)

	// Add basic metadata tags.
	tag.SetAlbum(req.TrackTags["collectionTitle"])
	tag.SetArtist(req.TrackTags["trackArtist"])
	tag.SetGenre(req.TrackTags["trackGenre"])
	tag.SetTitle(req.TrackTags["trackTitle"])
	tag.SetYear(req.TrackTags["releaseYear"])

	// Add track number and total tracks (e.g., "1/10").
	var (
		trackNumber = req.TrackTags["trackNumber"]
		trackCount  = req.TrackTags["trackCount"]
	)

	if trackNumber != "" && trackCount != "" {
		tag.AddTextFrame(
			tag.CommonID("Track number/Position in set"),
			tag.DefaultEncoding(),
			trackNumber+"/"+trackCount,
		)
	}

	// Add additional metadata tags.
	tag.AddTextFrame(tag.CommonID("Band/Orchestra/Accompaniment"), tag.DefaultEncoding(), req.TrackTags["albumArtist"])
	tag.AddTextFrame(tag.CommonID("Publisher"), tag.DefaultEncoding(), req.TrackTags["recordLabel"])

	// Add lyrics if available.
	if req.TrackLyrics == nil {
		return
	}

	lyrics := strings.TrimSpace(req.TrackLyrics.Lyrics)

	// If the lyrics type is subtitle, parse the LRC file content into a SynchronisedLyricsFrame.
	if req.TrackLyrics.Type == zvuk.LyricsTypeSubtitle {
		// Parse the LRC file content into a SynchronisedLyricsFrame.
		result, err := id3v2.ParseLRCFile(strings.NewReader(lyrics))
		if err != nil {
			logger.Errorf(ctx, "Failed to parse LRC file content: %v", err)
		}

		// Create a SynchronisedLyricsFrame from the parsed result.
		sylf := id3v2.SynchronisedLyricsFrame{
			Encoding: id3v2.EncodingUTF8,
			// Field is required, so we just use lingua franca.
			Language: id3v2.EnglishISO6392Code,
			// Use absolute timestamps.
			TimestampFormat: id3v2.SYLTAbsoluteMillisecondsTimestampFormat,
			// Mark as lyrics.
			ContentType: id3v2.SYLTLyricsContentType,
			// Descriptor for lyrics.
			ContentDescriptor: "Lyrics",
			// The actual synchronized lyrics.
			SynchronizedTexts: result.SynchronizedTexts,
		}

		// Add the synchronized lyrics frame to the tag.
		tag.AddSynchronisedLyricsFrame(sylf)

		return
	}

	// Add the unsynchronised lyrics frame to the tag.
	tag.AddUnsynchronisedLyricsFrame(
		//nolint:exhaustruct // ContentDescriptor not available in source data.
		id3v2.UnsynchronisedLyricsFrame{
			Encoding: id3v2.EncodingUTF8,
			Lyrics:   lyrics,
			// Field is required, so we just use lingua franca.
			Language: id3v2.EnglishISO6392Code,
		})
}
