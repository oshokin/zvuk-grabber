// Package lossless normalizes native FLAC and unencrypted, non-fragmented,
// single-track FLAC-in-MP4. It does not decode or re-encode audio.
package lossless

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// span is a byte range of FLAC frames inside an mdat box.
type span struct {
	// offset is the start of the frame range in the source.
	offset int64
	// size is the number of frame bytes.
	size int64
}

// parser walks a non-fragmented, single-track FLAC-in-MP4 file.
type parser struct {
	// ctx cancels the walk and the audio copy.
	ctx context.Context
	// source is the FLAC-in-MP4 byte stream.
	source io.ReaderAt
	// metadata is native FLAC metadata extracted from dfLa.
	metadata []byte
	// frames are mdat ranges that contain native FLAC frames.
	frames []*span
	// tracks is the number of trak boxes visited.
	tracks int
	// entries is the number of fLaC sample entries visited.
	entries int
	// boxes is the number of boxes visited.
	boxes int
}

// contextReader aborts a copy when the parse context is canceled.
type contextReader struct {
	// ctx is checked before each Read.
	ctx context.Context
	// reader is the audio section being copied.
	reader io.Reader
}

// mp4Box is one ISO BMFF box after its header has been parsed.
type mp4Box struct {
	// kind is the box fourcc.
	kind string
	// body is the first byte of the box payload.
	body int64
	// end is the first byte after the box.
	end int64
}

const (
	// flacMagic is the native FLAC stream marker.
	flacMagic = "fLaC"
	// ftypMagic is the ISO BMFF file-type box fourcc.
	ftypMagic = "ftyp"
	// maxMetadata is the largest accepted concatenated FLAC metadata payload.
	maxMetadata = 16 << 20
	// maxDepth is the maximum nested box depth walked by the parser.
	maxDepth = 24
	// maxBoxes is the maximum number of boxes visited in one remux.
	maxBoxes = 100000

	// flacMetadataHeaderSize is the FLAC metadata block header:
	// 1-bit last-block flag, 7-bit type, 24-bit length.
	flacMetadataHeaderSize = 4
	// flacStreamInfoSize is the fixed STREAMINFO payload size in the FLAC spec.
	flacStreamInfoSize = 34
	// flacMetadataTypeMask isolates the 7-bit metadata block type.
	flacMetadataTypeMask = 0x7f
	// flacStreamInfoType is metadata block type 0 (STREAMINFO).
	flacStreamInfoType = 0
	// flacMaxMetadataType is the last defined FLAC metadata block type (PICTURE).
	flacMaxMetadataType = 6
	// flacFrameSync0 is the first byte of a native FLAC frame sync.
	flacFrameSync0 = 0xff
	// flacFrameSync1 is the unmasked second sync byte (variable-blocksize frames).
	flacFrameSync1 = 0xf8
	// flacFrameSync1Mask ignores the blocking-strategy bit in the second sync byte.
	flacFrameSync1Mask = 0xfe

	// dfLaFullBoxHeaderSize is the dfLa version/flags prefix (must be zero).
	dfLaFullBoxHeaderSize = 4
	// minDFLaPayloadSize is version/flags plus one STREAMINFO metadata block.
	minDFLaPayloadSize = dfLaFullBoxHeaderSize + flacMetadataHeaderSize + flacStreamInfoSize
	// mp4BoxHeaderSize is a standard ISO BMFF box header (size + type).
	mp4BoxHeaderSize = 8
	// ProbeSize is bytes needed to distinguish native FLAC from an ftyp box.
	ProbeSize = mp4BoxHeaderSize
	// mp4ExtendedHeaderSize is an ISO BMFF header with a 64-bit largesize field.
	mp4ExtendedHeaderSize = 16
	// mp4BoxTypeOffset is the fourcc offset inside a standard box header.
	mp4BoxTypeOffset = 4
	// isoAudioSampleEntryPrefix is bytes before child boxes in AudioSampleEntry v0.
	isoAudioSampleEntryPrefix = 28
)

// flacLastMetadataFlag is bit 7 of the metadata header type byte.
const flacLastMetadataFlag byte = 0x80

var (
	// ErrNotFLAC is returned when the payload is not native FLAC and not FLAC-in-MP4.
	ErrNotFLAC = errors.New("lossless response is not a FLAC stream")
	// ErrInvalidMP4 is returned when FLAC-in-MP4 is fragmented, encrypted, or malformed.
	ErrInvalidMP4 = errors.New("unsupported or malformed FLAC-in-MP4")
)

// HasFLACMarker reports whether prefix starts with the native FLAC magic.
func HasFLACMarker(prefix []byte) bool {
	return len(prefix) >= len(flacMagic) && string(prefix[:len(flacMagic)]) == flacMagic
}

// HasFtypMarker reports whether prefix is a complete ISO BMFF ftyp box header.
func HasFtypMarker(prefix []byte) bool {
	return len(prefix) >= mp4BoxHeaderSize &&
		string(prefix[mp4BoxTypeOffset:mp4BoxHeaderSize]) == ftypMagic
}

// Normalize returns native FLAC bytes, sharing the streaming parser with Zvuk.
func Normalize(data []byte) ([]byte, error) {
	if bytes.HasPrefix(data, []byte(flacMagic)) {
		return data, nil
	}

	var out bytes.Buffer
	if err := Remux(context.Background(), &out, bytes.NewReader(data), int64(len(data))); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// Remux writes a native FLAC stream using bounded audio buffers. Source and
// destination must be distinct; callers discard the destination on any error.
func Remux(ctx context.Context, dst io.Writer, src io.ReaderAt, size int64) error {
	var head [mp4BoxHeaderSize]byte

	if size < mp4BoxHeaderSize {
		return ErrNotFLAC
	}

	if _, err := src.ReadAt(head[:], 0); err != nil {
		return err
	}

	if string(head[mp4BoxTypeOffset:]) != ftypMagic {
		return ErrNotFLAC
	}

	p := &parser{ctx: ctx, source: src}
	if err := p.walk(0, size, 0); err != nil {
		return err
	}

	if len(p.metadata) == 0 || len(p.frames) == 0 {
		return fmt.Errorf("%w: missing dfLa or mdat", ErrInvalidMP4)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := io.Copy(dst, bytes.NewReader(append([]byte(flacMagic), p.metadata...))); err != nil {
		return err
	}

	buffer := make([]byte, 32*1024)

	for _, part := range p.frames {
		reader := io.NewSectionReader(src, part.offset, part.size)
		if _, err := io.CopyBuffer(dst, &contextReader{ctx: ctx, reader: reader}, buffer); err != nil {
			return err
		}
	}

	return nil
}

// Read stops copying as soon as the parse context is canceled.
func (r *contextReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	return r.reader.Read(b)
}

// readBox parses one ISO BMFF box header at offset and returns its body range.
func (p *parser) readBox(offset, end int64) (*mp4Box, error) {
	var header [16]byte

	if offset < 0 || end-offset < mp4BoxHeaderSize {
		return nil, fmt.Errorf("%w: truncated box header", ErrInvalidMP4)
	}

	if _, err := p.source.ReadAt(header[:mp4BoxHeaderSize], offset); err != nil {
		return nil, err
	}

	size, skip := int64(binary.BigEndian.Uint32(header[:mp4BoxTypeOffset])), int64(mp4BoxHeaderSize)
	switch size {
	case 0:
		size = end - offset
	case 1:
		if end-offset < mp4ExtendedHeaderSize {
			return nil, fmt.Errorf("%w: truncated extended header", ErrInvalidMP4)
		}

		if _, err := p.source.ReadAt(header[mp4BoxHeaderSize:], offset+mp4BoxHeaderSize); err != nil {
			return nil, err
		}

		large := binary.BigEndian.Uint64(header[mp4BoxHeaderSize:])
		if large > math.MaxInt64 {
			return nil, fmt.Errorf("%w: extended size overflow", ErrInvalidMP4)
		}

		size, skip = int64(large), mp4ExtendedHeaderSize
	}

	if size < skip || size > end-offset {
		return nil, fmt.Errorf("%w: invalid box size", ErrInvalidMP4)
	}

	return &mp4Box{
		kind: string(header[mp4BoxTypeOffset:mp4BoxHeaderSize]),
		body: offset + skip,
		end:  offset + size,
	}, nil
}

// walk visits every box in the half-open byte range [start, end).
func (p *parser) walk(start, end int64, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("%w: nesting limit", ErrInvalidMP4)
	}

	for offset := start; offset < end; {
		if err := p.ctx.Err(); err != nil {
			return err
		}

		p.boxes++
		if p.boxes > maxBoxes {
			return fmt.Errorf("%w: box count limit", ErrInvalidMP4)
		}

		box, err := p.readBox(offset, end)
		if err != nil {
			return err
		}

		if err = p.visit(box, depth); err != nil {
			return err
		}

		offset = box.end
	}

	return nil
}

// visit dispatches a box by fourcc and rejects fragmented or encrypted media.
func (p *parser) visit(box *mp4Box, depth int) error {
	switch box.kind {
	case "moof", "mvex", "encv", "enca", "sinf", "senc":
		return fmt.Errorf("%w: fragmented/encrypted media (%s)", ErrInvalidMP4, box.kind)
	case "trak":
		p.tracks++
		if p.tracks > 1 {
			return fmt.Errorf("%w: multiple tracks", ErrInvalidMP4)
		}

		return p.walk(box.body, box.end, depth+1)
	case "moov", "mdia", "minf", "stbl":
		return p.walk(box.body, box.end, depth+1)
	case "stsd":
		return p.sampleDescription(box, depth)
	case flacMagic:
		return p.sampleEntry(box, depth)
	case "dfLa":
		return p.readMetadata(box)
	case "mdat":
		return p.mediaData(box, depth)
	}

	return nil
}

// sampleDescription requires a single unencrypted fLaC sample entry in stsd.
func (p *parser) sampleDescription(box *mp4Box, depth int) error {
	if box.end-box.body < 16 {
		return fmt.Errorf("%w: short stsd", ErrInvalidMP4)
	}

	var info [16]byte
	if _, err := p.source.ReadAt(info[:], box.body); err != nil {
		return err
	}

	if binary.BigEndian.Uint32(info[:4]) != 0 || binary.BigEndian.Uint32(info[4:8]) != 1 ||
		string(info[12:]) != flacMagic {
		return fmt.Errorf("%w: unsupported sample descriptions", ErrInvalidMP4)
	}

	return p.walk(box.body+8, box.end, depth+1)
}

// sampleEntry walks child boxes after an ISO AudioSampleEntry version-0 prefix.
func (p *parser) sampleEntry(box *mp4Box, depth int) error {
	p.entries++
	if p.entries > 1 || box.end-box.body < isoAudioSampleEntryPrefix {
		return fmt.Errorf("%w: invalid FLAC sample entry", ErrInvalidMP4)
	}
	// ISO AudioSampleEntry version 0 has 28 bytes before child boxes.
	var version [2]byte
	if _, err := p.source.ReadAt(version[:], box.body+8); err != nil {
		return err
	}

	if binary.BigEndian.Uint16(version[:]) != 0 {
		return fmt.Errorf("%w: audio sample entry version", ErrInvalidMP4)
	}

	return p.walk(box.body+isoAudioSampleEntryPrefix, box.end, depth+1)
}

// readMetadata extracts native FLAC metadata from a dfLa box.
func (p *parser) readMetadata(box *mp4Box) error {
	if p.metadata != nil || box.end-box.body < minDFLaPayloadSize || box.end-box.body > maxMetadata {
		return fmt.Errorf("%w: invalid/duplicate dfLa", ErrInvalidMP4)
	}

	data := make([]byte, int(box.end-box.body))
	if _, err := p.source.ReadAt(data, box.body); err != nil {
		return err
	}

	if binary.BigEndian.Uint32(data[:dfLaFullBoxHeaderSize]) != 0 {
		return fmt.Errorf("%w: dfLa version/flags", ErrInvalidMP4)
	}

	metadata, err := p.normalizeMetadata(data[dfLaFullBoxHeaderSize:])
	if err != nil {
		return err
	}

	p.metadata = metadata

	return nil
}

// mediaData records FLAC frames or recurses into a nested MP4 wrapped in mdat.
func (p *parser) mediaData(box *mp4Box, depth int) error {
	var prefix [8]byte

	n := min(int64(len(prefix)), box.end-box.body)
	if n < 2 {
		return fmt.Errorf("%w: empty mdat", ErrInvalidMP4)
	}

	if _, err := p.source.ReadAt(prefix[:n], box.body); err != nil {
		return err
	}

	if n == mp4BoxHeaderSize && string(prefix[mp4BoxTypeOffset:]) == ftypMagic {
		// Yandex can wrap an inner MP4 in an outer mdat.
		return p.walk(box.body, box.end, depth+1)
	}

	if prefix[0] != flacFrameSync0 || prefix[1]&flacFrameSync1Mask != flacFrameSync1 {
		return fmt.Errorf("%w: mdat does not start with FLAC frames", ErrInvalidMP4)
	}

	p.frames = append(p.frames, &span{offset: box.body, size: box.end - box.body})

	return nil
}

// normalizeMetadata validates every block boundary and sets the native last-block
// flag. Never scan arbitrary bytes for dfLa or a frame sync marker.
func (p *parser) normalizeMetadata(data []byte) ([]byte, error) {
	for offset := 0; ; {
		if len(data)-offset < flacMetadataHeaderSize {
			return nil, fmt.Errorf("%w: truncated FLAC metadata", ErrInvalidMP4)
		}

		kind := data[offset] & flacMetadataTypeMask

		length := int(data[offset+1])<<16 | int(data[offset+2])<<8 | int(data[offset+3])
		if kind > flacMaxMetadataType ||
			(offset == 0 && (kind != flacStreamInfoType || length != flacStreamInfoSize)) ||
			(offset > 0 && kind == flacStreamInfoType) ||
			length > len(data)-offset-flacMetadataHeaderSize {
			return nil, fmt.Errorf("%w: invalid FLAC metadata block", ErrInvalidMP4)
		}

		end := offset + flacMetadataHeaderSize + length
		if data[offset]&flacLastMetadataFlag != 0 && end != len(data) {
			return nil, fmt.Errorf("%w: data after last metadata block", ErrInvalidMP4)
		}

		data[offset] &= flacMetadataTypeMask
		if end == len(data) {
			data[offset] |= flacLastMetadataFlag
			return data, nil
		}

		offset = end
	}
}
