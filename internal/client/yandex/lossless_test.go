package yandex

import (
	"encoding/binary"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignRequest_MatchesYandexWebPlayerEncRaw(t *testing.T) {
	t.Parallel()

	sign := SignRequest(1789466815, "18930267", transportEncRaw)
	assert.Equal(t, "hsU3ga/m4+9ZG8yS1BT1kWHLJ9q7QbsgPmL11NkrVBk", sign)
}

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

func TestNormalizeLosslessFLAC_PassesThroughNativeFLAC(t *testing.T) {
	t.Parallel()

	native := append(append([]byte{}, flacMagic...), []byte("metaframes")...)
	got, err := normalizeLosslessFLAC(native)
	require.NoError(t, err)
	assert.Equal(t, native, got)
}

func TestNormalizeLosslessFLAC_RemuxesFLACMP4(t *testing.T) {
	t.Parallel()

	metadata := buildStreamInfoMetadata(true)
	first := []byte{0xff, 0xf8, 0x01, 0x02}
	second := []byte{0xff, 0xf8, 0x03, 0x04}
	container := concatBytes(
		mp4Box("ftyp", []byte("isom")),
		mp4Box("moov", mp4Box("trak", mp4Box("dfLa", append([]byte{0, 0, 0, 0}, metadata...)))),
		mp4Box("mdat", first),
		mp4Box("mdat", second),
	)

	got, err := normalizeLosslessFLAC(container)
	require.NoError(t, err)
	assert.Equal(t, concatBytes(flacMagic, metadata, first, second), got)
}

func TestNormalizeLosslessFLAC_RemuxesNestedMP4InsideMdat(t *testing.T) {
	t.Parallel()

	metadata := buildStreamInfoMetadata(true)
	first := []byte{0xff, 0xf8, 0x11, 0x22}
	second := []byte{0xff, 0xf8, 0x33, 0x44}
	inner := concatBytes(
		mp4Box("ftyp", []byte("isom")),
		mp4Box("moov", []byte{0, 0, 0, 0}),
		mp4Box("mdat", first),
		mp4Box("mdat", second),
	)
	container := concatBytes(
		mp4Box("ftyp", []byte("isom")),
		mp4Box("moov", mp4Box("trak", mp4Box("dfLa", append([]byte{0, 0, 0, 0}, metadata...)))),
		mp4Box("mdat", inner),
	)

	got, err := normalizeLosslessFLAC(container)
	require.NoError(t, err)
	assert.Equal(t, concatBytes(flacMagic, metadata, first, second), got)
}

func TestNormalizeLosslessFLAC_RejectsNonFLAC(t *testing.T) {
	t.Parallel()

	_, err := normalizeLosslessFLAC([]byte("ID3notaudio"))
	require.Error(t, err)
	assert.ErrorIs(t, err, errLosslessNotFLAC)
}

func TestIsSupportedLosslessCodec_AcceptsFLACFamily(t *testing.T) {
	t.Parallel()

	assert.True(t, isSupportedLosslessCodec("flac"))
	assert.True(t, isSupportedLosslessCodec("flac-mp4"))
	assert.False(t, isSupportedLosslessCodec("aac-mp4"))
}

func mp4Box(boxType string, payload []byte) []byte {
	box := make([]byte, mp4BoxHeaderSize+len(payload))
	binary.BigEndian.PutUint32(box, uint32(len(box)))
	copy(box[4:], boxType)
	copy(box[mp4BoxHeaderSize:], payload)

	return box
}

func buildStreamInfoMetadata(last bool) []byte {
	metadata := make([]byte, flacMetadataBlockHeaderSize+flacStreamInfoBlockSize)
	if last {
		metadata[0] = 0x80
	}

	metadata[3] = flacStreamInfoBlockSize

	return metadata
}

func concatBytes(parts ...[]byte) []byte {
	var out []byte
	for _, part := range parts {
		out = append(out, part...)
	}

	return out
}
