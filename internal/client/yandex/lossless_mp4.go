package yandex

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

const (
	// mp4BoxHeaderSize is the standard ISO BMFF box header size in bytes.
	mp4BoxHeaderSize = 8
	// mp4LargeBoxHeaderSize is the 64-bit ISO BMFF box header size in bytes.
	mp4LargeBoxHeaderSize = 16
	// dfLaFullBoxHeaderSize is the FullBox version+flags prefix inside dfLa.
	dfLaFullBoxHeaderSize = 4
	// flacStreamInfoBlockSize is the STREAMINFO metadata payload size.
	flacStreamInfoBlockSize = 34
	// flacMetadataBlockHeaderSize is the FLAC metadata block header size.
	flacMetadataBlockHeaderSize = 4
	// flacStreamInfoType is the STREAMINFO metadata block type.
	flacStreamInfoType = 0
	// flacFrameSyncShifted is go-flac's shifted 14-bit frame sync check.
	flacFrameSyncShifted = 0x3E
)

var (
	flacMagic = []byte("fLaC")
	ftypMagic = []byte("ftyp")
	dfLaMagic = []byte("dfLa")
	mdatMagic = []byte("mdat")
)

// normalizeLosslessFLAC returns native FLAC bytes, remuxing flac-mp4 when needed.
func normalizeLosslessFLAC(data []byte) ([]byte, error) {
	if bytes.HasPrefix(data, flacMagic) {
		return data, nil
	}

	extracted, err := extractFLACFromMP4(data)
	if err != nil {
		return nil, err
	}

	if !bytes.HasPrefix(extracted, flacMagic) {
		return nil, errLosslessNotFLAC
	}

	return extracted, nil
}

// extractFLACFromMP4 rebuilds a native FLAC stream from a FLAC-in-MP4 container.
func extractFLACFromMP4(data []byte) ([]byte, error) {
	if !looksLikeMP4(data) {
		return nil, errLosslessNotFLAC
	}

	metadata, err := extractDfLaMetadata(data)
	if err != nil {
		return nil, err
	}

	frames, err := collectFLACFramePayloads(data)
	if err != nil {
		return nil, err
	}

	native := make([]byte, 0, len(flacMagic)+len(metadata)+len(frames))
	native = append(native, flacMagic...)
	native = append(native, metadata...)
	native = append(native, frames...)

	return native, nil
}

// looksLikeMP4 reports whether data looks like an ISO BMFF container.
func looksLikeMP4(data []byte) bool {
	return isNestedMP4(data) || hasMP4Box(data, dfLaMagic)
}

// isNestedMP4 reports whether payload starts with an ftyp box.
func isNestedMP4(data []byte) bool {
	return len(data) >= mp4BoxHeaderSize && bytes.Equal(data[4:8], ftypMagic)
}

// extractDfLaMetadata returns FLAC metadata blocks stored in the dfLa box.
func extractDfLaMetadata(data []byte) ([]byte, error) {
	for _, payload := range findBoxPayloads(data, dfLaMagic) {
		if len(payload) < dfLaFullBoxHeaderSize+flacMetadataBlockHeaderSize+flacStreamInfoBlockSize {
			continue
		}

		if payload[0] != 0 {
			continue
		}

		metadata := payload[dfLaFullBoxHeaderSize:]
		if metadata[0]&0x7f != flacStreamInfoType {
			continue
		}

		return metadata, nil
	}

	return nil, fmt.Errorf("%w: dfLa", errFLACMP4MissingBoxes)
}

// collectFLACFramePayloads concatenates mdat payloads that contain FLAC frames.
// Yandex wraps a complete inner MP4 inside the outer mdat, so this walks nested ftyp containers.
func collectFLACFramePayloads(data []byte) ([]byte, error) {
	var frames []byte

	err := walkTopLevelBoxes(data, func(boxType []byte, payload []byte) error {
		if !bytes.Equal(boxType, mdatMagic) {
			return nil
		}

		switch {
		case isNestedMP4(payload):
			nested, nestedErr := collectFLACFramePayloads(payload)
			if nestedErr != nil {
				return nestedErr
			}

			frames = append(frames, nested...)
		case isFLACFrameStart(payload):
			frames = append(frames, payload...)
		default:
			if idx := indexFLACSync(payload); idx >= 0 {
				frames = append(frames, payload[idx:]...)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(frames) == 0 || !isFLACFrameStart(frames) {
		return nil, fmt.Errorf("%w: mdat", errFLACMP4MissingBoxes)
	}

	return frames, nil
}

// isFLACFrameStart reports whether payload begins with a FLAC frame sync code.
func isFLACFrameStart(payload []byte) bool {
	return len(payload) >= 2 && payload[0] == 0xFF && payload[1]>>2 == flacFrameSyncShifted
}

// indexFLACSync returns the first FLAC frame sync offset, or -1 if none exists.
func indexFLACSync(data []byte) int {
	for i := 0; i+1 < len(data); i++ {
		if data[i] == 0xFF && data[i+1]>>2 == flacFrameSyncShifted {
			return i
		}
	}

	return -1
}

// walkTopLevelBoxes invokes fn for every top-level ISO BMFF box.
func walkTopLevelBoxes(data []byte, fn func(boxType, payload []byte) error) error {
	offset := 0
	for offset+mp4BoxHeaderSize <= len(data) {
		total, header, ok := parseMP4BoxHeader(data, offset)
		if !ok {
			return fmt.Errorf("%w: truncated top-level box at %d", errFLACMP4MissingBoxes, offset)
		}

		if err := fn(data[offset+4:offset+8], data[offset+header:offset+total]); err != nil {
			return err
		}

		offset += total
	}

	return nil
}

// findBoxPayloads returns payloads for every well-formed box with the given type.
func findBoxPayloads(data, boxType []byte) [][]byte {
	payloads := make([][]byte, 0, 1)

	for offset := 0; offset+mp4BoxHeaderSize <= len(data); offset++ {
		if !bytes.Equal(data[offset+4:offset+8], boxType) {
			continue
		}

		total, header, ok := parseMP4BoxHeader(data, offset)
		if !ok {
			continue
		}

		payloads = append(payloads, data[offset+header:offset+total])
	}

	return payloads
}

// hasMP4Box reports whether a well-formed box of the given type exists.
func hasMP4Box(data, boxType []byte) bool {
	return len(findBoxPayloads(data, boxType)) > 0
}

// parseMP4BoxHeader parses an ISO BMFF box header at offset.
func parseMP4BoxHeader(data []byte, offset int) (total, header int, ok bool) {
	if offset < 0 || offset+mp4BoxHeaderSize > len(data) {
		return 0, 0, false
	}

	size := binary.BigEndian.Uint32(data[offset:])
	header = mp4BoxHeaderSize

	var total64 uint64

	switch size {
	case 0:
		total64 = nonNegativeUint64(len(data) - offset)
	case 1:
		if offset+mp4LargeBoxHeaderSize > len(data) {
			return 0, 0, false
		}

		total64 = binary.BigEndian.Uint64(data[offset+8:])
		header = mp4LargeBoxHeaderSize
	default:
		total64 = uint64(size)
	}

	remaining := nonNegativeUint64(len(data) - offset)
	if total64 < nonNegativeUint64(header) || total64 > remaining {
		return 0, 0, false
	}

	total, ok = intFromUint64(total64)
	if !ok {
		return 0, 0, false
	}

	return total, header, true
}

// nonNegativeUint64 converts a non-negative int to uint64.
func nonNegativeUint64(value int) uint64 {
	if value <= 0 {
		return 0
	}

	return uint64(value)
}

// intFromUint64 converts uint64 to int when the value fits.
func intFromUint64(value uint64) (int, bool) {
	if value > uint64(math.MaxInt) {
		return 0, false
	}

	return int(value), true
}
