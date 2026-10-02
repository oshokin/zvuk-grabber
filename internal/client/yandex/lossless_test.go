package yandex

import (
	"bytes"
	"encoding/binary"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/media/lossless"
)

const (
	// flacMetadataHeaderSize is the FLAC metadata block header:
	// 1-bit last-block flag, 7-bit type, 24-bit length.
	flacMetadataHeaderSize = 4
	// flacStreamInfoSize is the fixed STREAMINFO payload size in the FLAC spec.
	flacStreamInfoSize = 34
	// flacFrameSync0 is the first byte of a native FLAC frame sync.
	flacFrameSync0 = 0xff
	// flacFrameSync1 is the unmasked second sync byte (variable-blocksize frames).
	flacFrameSync1 = 0xf8
	// mp4BoxHeaderSize is a standard ISO BMFF box header (size + type).
	mp4BoxHeaderSize = 8
	// mp4BoxTypeOffset is the fourcc offset inside a standard box header.
	mp4BoxTypeOffset = 4
	// dfLaFullBoxHeaderSize is the dfLa version/flags prefix (must be zero).
	dfLaFullBoxHeaderSize = 4
)

// flacLastMetadataFlag is bit 7 of the metadata header type byte.
const flacLastMetadataFlag byte = 0x80

// TestSignRequest_MatchesYandexWebPlayerEncRaw locks the HMAC against a known web-player vector.
func TestSignRequest_MatchesYandexWebPlayerEncRaw(t *testing.T) {
	t.Parallel()

	sign := SignRequest(1789466815, "18930267", transportEncRaw)
	assert.Equal(t, "hsU3ga/m4+9ZG8yS1BT1kWHLJ9q7QbsgPmL11NkrVBk", sign)
}

// TestBuildFileInfoURL_UsesEncRawTransport signs get-file-info with encraw as the default transport.
func TestBuildFileInfoURL_UsesEncRawTransport(t *testing.T) {
	t.Parallel()

	rawURL := BuildFileInfoURL("https://api.music.yandex.net", "18930267", "", 1789466815)
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	query := parsed.Query()
	assert.Equal(t, "lossless", query.Get("quality"))
	assert.Equal(t, transportEncRaw, query.Get("transports"))
	assert.Equal(t, "flac,aac,he-aac,mp3,flac-mp4,aac-mp4,he-aac-mp4", query.Get("codecs"))
	assert.Equal(t, SignRequest(1789466815, "18930267", transportEncRaw), query.Get("sign"))
}

// TestParseDownloadInfo_FLACMP4EncRaw reads camelCase downloadInfo for a flac-mp4 encraw stream.
func TestParseDownloadInfo_FLACMP4EncRaw(t *testing.T) {
	t.Parallel()

	info, err := ParseDownloadInfo([]byte(`{
		"downloadInfo": {
			"trackId": "18930267",
			"quality": "lossless",
			"codec": "flac-mp4",
			"bitrate": 0,
			"transport": "encraw",
			"key": "6b10ee77e7fb0a377c937f139bd369d9",
			"urls": ["https://example.test/flac-mp4"]
		}
	}`))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "lossless", info.Quality)
	assert.Equal(t, "flac-mp4", info.Codec)
	assert.Equal(t, "6b10ee77e7fb0a377c937f139bd369d9", info.Key)
	assert.Equal(t, []string{"https://example.test/flac-mp4"}, info.URLs)
}

// TestNormalizeLosslessFLAC_PassesThroughNativeFLAC leaves an already-native stream unchanged.
func TestNormalizeLosslessFLAC_PassesThroughNativeFLAC(t *testing.T) {
	t.Parallel()

	native := append(append([]byte{}, []byte("fLaC")...), []byte("metaframes")...)
	got, err := lossless.Normalize(native)
	require.NoError(t, err)
	assert.Equal(t, native, got)
}

// TestNormalizeLosslessFLAC_RemuxesFLACMP4 concatenates dfLa metadata with mdat frames.
func TestNormalizeLosslessFLAC_RemuxesFLACMP4(t *testing.T) {
	t.Parallel()

	metadata := buildStreamInfoMetadata(true)
	first := []byte{flacFrameSync0, flacFrameSync1, 0x01, 0x02}
	second := []byte{flacFrameSync0, flacFrameSync1, 0x03, 0x04}
	container := bytes.Join([][]byte{
		mp4Box("ftyp", []byte("isom")),
		mp4Box("moov", mp4Box("trak", mp4Box("dfLa", append(make([]byte, dfLaFullBoxHeaderSize), metadata...)))),
		mp4Box("mdat", first),
		mp4Box("mdat", second),
	}, nil)

	got, err := lossless.Normalize(container)
	require.NoError(t, err)
	assert.Equal(t, bytes.Join([][]byte{[]byte("fLaC"), metadata, first, second}, nil), got)
}

// TestNormalizeLosslessFLAC_RemuxesNestedMP4InsideMdat unwraps an inner MP4 stored in outer mdat.
func TestNormalizeLosslessFLAC_RemuxesNestedMP4InsideMdat(t *testing.T) {
	t.Parallel()

	metadata := buildStreamInfoMetadata(true)
	first := []byte{flacFrameSync0, flacFrameSync1, 0x11, 0x22}
	second := []byte{flacFrameSync0, flacFrameSync1, 0x33, 0x44}
	inner := bytes.Join([][]byte{
		mp4Box("ftyp", []byte("isom")),
		mp4Box("moov", nil),
		mp4Box("mdat", first),
		mp4Box("mdat", second),
	}, nil)
	container := bytes.Join([][]byte{
		mp4Box("ftyp", []byte("isom")),
		mp4Box("moov", mp4Box("trak", mp4Box("dfLa", append(make([]byte, dfLaFullBoxHeaderSize), metadata...)))),
		mp4Box("mdat", inner),
	}, nil)

	got, err := lossless.Normalize(container)
	require.NoError(t, err)
	assert.Equal(t, bytes.Join([][]byte{[]byte("fLaC"), metadata, first, second}, nil), got)
}

// TestNormalizeLosslessFLAC_RejectsNonFLAC returns ErrNotFLAC for unrelated payloads.
func TestNormalizeLosslessFLAC_RejectsNonFLAC(t *testing.T) {
	t.Parallel()

	_, err := lossless.Normalize([]byte("ID3notaudio"))
	require.Error(t, err)
	assert.ErrorIs(t, err, lossless.ErrNotFLAC)
}

// mp4Box builds a standard ISO BMFF box with a 32-bit size header.
func mp4Box(boxType string, payload []byte) []byte {
	box := make([]byte, mp4BoxHeaderSize+len(payload))
	binary.BigEndian.PutUint32(box, uint32(len(box)))
	copy(box[mp4BoxTypeOffset:], boxType)
	copy(box[mp4BoxHeaderSize:], payload)

	return box
}

// buildStreamInfoMetadata returns a STREAMINFO block, optionally marked last.
func buildStreamInfoMetadata(last bool) []byte {
	metadata := make([]byte, flacMetadataHeaderSize+flacStreamInfoSize)
	if last {
		metadata[0] = flacLastMetadataFlag
	}

	metadata[flacMetadataHeaderSize-1] = byte(flacStreamInfoSize)

	return metadata
}
