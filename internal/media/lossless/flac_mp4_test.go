package lossless

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// failedWriter fails every Write so remux error paths can be tested.
type failedWriter struct{}

// boundedReader caps each ReadAt to max bytes.
type boundedReader struct {
	// ReaderAt is the underlying source.
	io.ReaderAt

	// max is the largest buffer accepted by a single ReadAt.
	max int
}

// TestNormalizeRealMP4 remuxes the checked-in FLAC-in-MP4 fixture to native FLAC.
func TestNormalizeRealMP4(t *testing.T) {
	data, err := os.ReadFile("testdata/tone.mp4")
	require.NoError(t, err)
	normalized, err := Normalize(data)
	require.NoError(t, err)
	require.True(t, HasFLACMarker(normalized))
	require.Less(t, len(normalized), len(data))
	require.Equal(t, flacLastMetadataFlag, normalized[len(flacMagic)])
	native, err := Normalize(normalized)
	require.NoError(t, err)
	require.Equal(t, normalized, native)
}

// TestHasFLACMarkerAndFtypMarker distinguishes native FLAC from an ISO BMFF ftyp header.
func TestHasFLACMarkerAndFtypMarker(t *testing.T) {
	require.True(t, HasFLACMarker([]byte(flacMagic)))
	require.False(t, HasFLACMarker([]byte(flacMagic[:len(flacMagic)-1])))
	require.False(t, HasFtypMarker([]byte(flacMagic)))

	ftyp := make([]byte, ProbeSize)
	copy(ftyp[mp4BoxTypeOffset:], ftypMagic)
	require.True(t, HasFtypMarker(ftyp))
	require.False(t, HasFtypMarker(ftyp[:ProbeSize-1]))
	require.False(t, HasFLACMarker(ftyp))
}

// TestRemuxMetadataAndFrames writes STREAMINFO plus frames and accepts both mdat size encodings.
func TestRemuxMetadataAndFrames(t *testing.T) {
	frames := []byte{flacFrameSync0, flacFrameSync1, 1, 2}
	meta := metadata() // MP4 can leave the last-block bit unset.
	source := container(meta, frames)
	normalized, err := Normalize(source)
	require.NoError(t, err)

	meta[0] = flacLastMetadataFlag
	require.Equal(t, bytes.Join([][]byte{[]byte("fLaC"), meta, frames}, nil), normalized)
	// Extended-size mdat and size=0 (to EOF) are both legal.
	for _, extended := range []bool{false, true} {
		tail := box("mdat", frames)
		if extended {
			tail = make([]byte, mp4ExtendedHeaderSize+len(frames))
			binary.BigEndian.PutUint32(tail, 1)
			copy(tail[mp4BoxTypeOffset:], "mdat")
			binary.BigEndian.PutUint64(tail[mp4BoxHeaderSize:], uint64(len(tail)))
			copy(tail[mp4ExtendedHeaderSize:], frames)
		} else {
			binary.BigEndian.PutUint32(tail, 0)
		}

		variant := bytes.Join([][]byte{source[:len(source)-12], tail}, nil)
		got, normalizeErr := Normalize(variant)
		require.NoError(t, normalizeErr)
		require.Equal(t, normalized, got)
	}
}

// TestRemuxRejectsMalformedContainers covers truncated, encrypted, and nested-invalid inputs.
func TestRemuxRejectsMalformedContainers(t *testing.T) {
	good := container(metadata(), []byte{flacFrameSync0, flacFrameSync1, 1, 2})
	badMetadata := metadata()
	badMetadata[flacMetadataHeaderSize-1] = byte(flacStreamInfoSize - 1)
	frames := []byte{flacFrameSync0, flacFrameSync1}

	cases := []struct {
		// name is the subtest label.
		name string
		// data is the malformed FLAC-in-MP4 payload.
		data []byte
	}{
		{
			name: "truncated",
			data: good[:len(good)-1],
		},
		{
			name: "trailing garbage",
			data: append(bytes.Clone(good), 0),
		},
		{
			name: "metadata length",
			data: container(badMetadata, frames),
		},
		{
			name: "junk before frames",
			data: container(metadata(), []byte{0, flacFrameSync0, flacFrameSync1}),
		},
		{
			name: "invalid sync",
			data: container(metadata(), []byte{flacFrameSync0, 0xfa}),
		},
		{
			name: "multiple tracks",
			data: bytes.Join([][]byte{good, box("moov", box("trak", nil))}, nil),
		},
		{
			name: "fake metadata in free",
			data: bytes.Join([][]byte{
				box("ftyp", nil),
				box("free", box("dfLa", bytes.Join([][]byte{make([]byte, dfLaFullBoxHeaderSize), metadata()}, nil))),
				box("mdat", frames),
			}, nil),
		},
		{
			name: "encrypted",
			data: bytes.Replace(good, []byte("fLaC"), []byte("enca"), 1),
		},
		{
			name: "fragmented",
			data: bytes.Join([][]byte{good, box("moof", nil)}, nil),
		},
		{
			name: "duplicate metadata",
			data: bytes.Join([][]byte{
				good,
				box("dfLa", bytes.Join([][]byte{make([]byte, dfLaFullBoxHeaderSize), metadata()}, nil)),
			}, nil),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Normalize(tc.data)
			require.Error(t, err)
		})
	}
}

// TestRemuxCancellationAndWriterFailure stops on a canceled context or a failed writer.
func TestRemuxCancellationAndWriterFailure(t *testing.T) {
	data := container(metadata(), []byte{flacFrameSync0, flacFrameSync1})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, Remux(ctx, io.Discard, bytes.NewReader(data), int64(len(data))), context.Canceled)
	require.ErrorIs(t, Remux(t.Context(), new(failedWriter), bytes.NewReader(data), int64(len(data))), io.ErrClosedPipe)
}

// Write always returns io.ErrClosedPipe.
func (*failedWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

// ReadAt rejects buffers larger than max, otherwise delegates to ReaderAt.
func (r *boundedReader) ReadAt(b []byte, off int64) (int, error) {
	if len(b) > r.max {
		return 0, io.ErrShortBuffer
	}

	return r.ReaderAt.ReadAt(b, off)
}

// TestRemuxBoundedAudioReads copies a large mdat in chunks no larger than 32KiB.
func TestRemuxBoundedAudioReads(t *testing.T) {
	frames := make([]byte, 2<<20)
	frames[0], frames[1] = flacFrameSync0, flacFrameSync1
	data := container(metadata(), frames)
	require.NoError(
		t,
		Remux(
			t.Context(),
			io.Discard,
			&boundedReader{ReaderAt: bytes.NewReader(data), max: 32 * 1024},
			int64(len(data)),
		),
	)
}

// FuzzNormalize rejects non-FLAC output on any accepted payload.
func FuzzNormalize(f *testing.F) {
	f.Add(container(metadata(), []byte{flacFrameSync0, flacFrameSync1, 1, 2}))
	f.Add([]byte("fLaC"))
	f.Fuzz(func(t *testing.T, data []byte) {
		normalized, err := Normalize(data)
		if err == nil && !HasFLACMarker(normalized) {
			t.Fatal("not FLAC")
		}
	})
}

// box builds a standard ISO BMFF box with a 32-bit size header.
func box(kind string, payload []byte) []byte {
	b := make([]byte, mp4BoxHeaderSize+len(payload))
	binary.BigEndian.PutUint32(b, uint32(len(b)))
	copy(b[mp4BoxTypeOffset:], kind)
	copy(b[mp4BoxHeaderSize:], payload)

	return b
}

// metadata returns a STREAMINFO block with the last-block flag unset.
func metadata() []byte {
	b := make([]byte, flacMetadataHeaderSize+flacStreamInfoSize)
	b[flacMetadataHeaderSize-1] = byte(flacStreamInfoSize)

	return b
}

// container builds a minimal unencrypted single-track FLAC-in-MP4 file.
func container(meta, frames []byte) []byte {
	sample := box("fLaC", bytes.Join([][]byte{
		make([]byte, isoAudioSampleEntryPrefix),
		box("dfLa", bytes.Join([][]byte{make([]byte, dfLaFullBoxHeaderSize), meta}, nil)),
	}, nil))
	stsd := box("stsd", bytes.Join([][]byte{{0, 0, 0, 0, 0, 0, 0, 1}, sample}, nil))

	return bytes.Join([][]byte{
		box("ftyp", []byte("isom")),
		box("moov", box("trak", box("mdia", box("minf", box("stbl", stsd))))),
		box("mdat", frames),
	}, nil)
}
