package zvuk

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
	TrackPath                  string
	CoverPath                  string
	Quality                    TrackQuality
	TrackTags                  map[string]string
	TrackLyrics                *zvuk.Lyrics
	IsCoverEmbeddedToTrackTags bool
}

// TagProcessorImpl provides the default implementation of TagProcessor.
type TagProcessorImpl struct{}

type imageMetadata struct {
	data     []byte
	mimeType string
}

// NewTagProcessor creates a new TagProcessor instance.
func NewTagProcessor() TagProcessor {
	return &TagProcessorImpl{}
}

// WriteTags writes metadata to audio files based on the provided request.
func (tp *TagProcessorImpl) WriteTags(ctx context.Context, req *WriteTagsRequest) error {
	if req.TrackPath == "" {
		return errors.New("track path cannot be empty")
	}

	var image *imageMetadata

	// If a cover path is provided and embedding is enabled, read the cover art
	if req.CoverPath != "" && req.IsCoverEmbeddedToTrackTags {
		imageData, err := os.ReadFile(filepath.Clean(req.CoverPath))
		if err != nil {
			return err
		}

		// Determine the MIME type of the cover art based on its file extension
		imageMIMEType := mime.TypeByExtension(filepath.Ext(req.CoverPath))
		image = &imageMetadata{
			data:     imageData,
			mimeType: imageMIMEType,
		}
	}

	// Write tags based on the track quality (FLAC or MP3)
	if req.Quality == TrackQualityFLAC {
		return tp.writeFLACTags(ctx, req, image)
	}

	return tp.writeMP3Tags(ctx, req, image)
}

func (tp *TagProcessorImpl) writeFLACTags(ctx context.Context, req *WriteTagsRequest, image *imageMetadata) error {
	// Parse the FLAC file
	f, err := flac.ParseFile(filepath.Clean(req.TrackPath))
	if err != nil {
		return err
	}

	// Extract existing FLAC comments (metadata) from the file
	tag, idx, err := tp.extractFLACComment(req.TrackPath)
	if err != nil {
		return err
	}

	// If no existing comments are found, create a new metadata block
	if tag == nil {
		tag = flacvorbis.New()
	}

	// Add tags to the FLAC metadata block
	err = tp.addFLACTags(tag, req)
	if err != nil {
		return err
	}

	// Marshal the updated metadata and update the FLAC file's metadata blocks
	tagMeta := tag.Marshal()
	if idx >= 0 {
		f.Meta[idx] = &tagMeta
	} else {
		f.Meta = append(f.Meta, &tagMeta)
	}

	// Embed the cover art into the FLAC file if provided
	tp.embedFLACCover(ctx, f, image)

	// Save the updated FLAC file
	return f.Save(req.TrackPath)
}

func (tp *TagProcessorImpl) extractFLACComment(filename string) (*flacvorbis.MetaDataBlockVorbisComment, int, error) {
	f, err := flac.ParseFile(filepath.Clean(filename))
	if err != nil {
		return nil, 0, err
	}

	// Iterate through the metadata blocks to find the Vorbis comment block
	for idx, meta := range f.Meta {
		if meta.Type != flac.VorbisComment {
			continue
		}

		// Parse the Vorbis comment block
		var comment *flacvorbis.MetaDataBlockVorbisComment

		comment, err = flacvorbis.ParseFromMetaDataBlock(*meta)
		if err == nil {
			return comment, idx, nil
		}
	}

	// Return nil if no Vorbis comment block is found
	return nil, -1, nil
}

func (tp *TagProcessorImpl) addFLACTags(tag *flacvorbis.MetaDataBlockVorbisComment, req *WriteTagsRequest) error {
	// Map of FLAC tag keys to their corresponding values in req.TrackTags
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

	// Add each tag to the Vorbis comment block
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

	// Create a new FLAC picture block from the image data
	picture, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "", image.data, image.mimeType)
	if err != nil {
		logger.Errorf(ctx, "Failed to embed image to FLAC: %v", err)

		return
	}

	// Add the picture block to the FLAC file's metadata
	pictureMeta := picture.Marshal()
	f.Meta = append(f.Meta, &pictureMeta)
}

func (tp *TagProcessorImpl) writeMP3Tags(ctx context.Context, req *WriteTagsRequest, image *imageMetadata) error {
	// Open the MP3 file for writing metadata
	//nolint:exhaustruct // ParseFrames intentionally omitted when Parse=false (parsing disabled)
	tag, err := id3v2.Open(req.TrackPath, id3v2.Options{Parse: false})
	if err != nil {
		return err
	}

	defer tag.Close()

	// Add metadata tags to the MP3 file
	tp.addMP3Tags(ctx, tag, req)

	// Embed the cover art into the MP3 file if provided
	if image != nil {
		//nolint:exhaustruct // Description field intentionally empty for cover images
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    image.mimeType,
			PictureType: id3v2.PTFrontCover,
			Picture:     image.data,
		})
	}

	// Save the updated MP3 file
	return tag.Save()
}

func (tp *TagProcessorImpl) addMP3Tags(ctx context.Context, tag *id3v2.Tag, req *WriteTagsRequest) {
	// Set default encoding for the tags
	tag.SetDefaultEncoding(id3v2.EncodingUTF8)

	// Add basic metadata tags
	tag.SetAlbum(req.TrackTags["collectionTitle"])
	tag.SetArtist(req.TrackTags["trackArtist"])
	tag.SetGenre(req.TrackTags["trackGenre"])
	tag.SetTitle(req.TrackTags["trackTitle"])
	tag.SetYear(req.TrackTags["releaseYear"])

	// Add track number and total tracks (e.g., "1/10")
	var (
		trackNumber = req.TrackTags["trackNumber"]
		trackCount  = req.TrackTags["trackCount"]
	)

	if trackNumber != "" && trackCount != "" {
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), tag.DefaultEncoding(), trackNumber+"/"+trackCount)
	}

	// Add additional metadata tags
	tag.AddTextFrame(tag.CommonID("Band/Orchestra/Accompaniment"), tag.DefaultEncoding(), req.TrackTags["albumArtist"])
	tag.AddTextFrame(tag.CommonID("Publisher"), tag.DefaultEncoding(), req.TrackTags["recordLabel"])

	// Add lyrics if available
	if req.TrackLyrics != nil {
		lyrics := strings.TrimSpace(req.TrackLyrics.Lyrics)
		if req.TrackLyrics.Type == zvuk.LyricsTypeSubtitle {
			// Parse the LRC file content into a SynchronisedLyricsFrame.
			result, err := id3v2.ParseLRCFile(strings.NewReader(lyrics))
			if err != nil {
				logger.Errorf(ctx, "Failed to parse LRC file content: %v", err)
			}

			// Create a SynchronisedLyricsFrame from the parsed result.
			sylf := id3v2.SynchronisedLyricsFrame{
				Encoding: id3v2.EncodingUTF8,
				// Field is required, so we just use lingua franca
				Language:          id3v2.EnglishISO6392Code,
				TimestampFormat:   id3v2.SYLTAbsoluteMillisecondsTimestampFormat, // Use absolute timestamps.
				ContentType:       id3v2.SYLTLyricsContentType,                   // Mark as lyrics.
				ContentDescriptor: "Lyrics",                                      // Descriptor for lyrics.
				SynchronizedTexts: result.SynchronizedTexts,                      // The actual synchronized lyrics.
			}

			// Add the synchronized lyrics frame to the tag.
			tag.AddSynchronisedLyricsFrame(sylf)
		} else {
			tag.AddUnsynchronisedLyricsFrame(
				//nolint:exhaustruct // ContentDescriptor not available in source data
				id3v2.UnsynchronisedLyricsFrame{
					Encoding: id3v2.EncodingUTF8,
					Lyrics:   lyrics,
					// Field is required, so we just use lingua franca
					Language: id3v2.EnglishISO6392Code,
				})
		}
	}
}
