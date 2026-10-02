package zvuk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/media/lossless"
)

// normalizeDownloadedFLAC leaves the original download intact until a complete
// native FLAC exists. It returns a separate closed temp file, never truncating
// the source or renaming over an open Windows handle.
func normalizeDownloadedFLAC(ctx context.Context, source *os.File, target string) (string, error) {
	if !strings.EqualFold(filepath.Ext(target), ".flac") {
		return "", nil
	}

	var head [lossless.ProbeSize]byte

	n, err := source.ReadAt(head[:], 0)
	if lossless.HasFLACMarker(head[:n]) {
		return "", nil
	}

	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	if !lossless.HasFtypMarker(head[:n]) {
		return "", nil
	}

	info, err := source.Stat()
	if err != nil {
		return "", err
	}

	out, err := os.CreateTemp(filepath.Dir(source.Name()), filepath.Base(target)+".remux-*")
	if err != nil {
		return "", err
	}

	success := false

	defer func() {
		_ = out.Close()
		if !success {
			_ = os.Remove(out.Name())
		}
	}()

	if err = out.Chmod(info.Mode().Perm()); err != nil {
		return "", err
	}

	if err = lossless.Remux(ctx, out, source, info.Size()); err != nil {
		return "", fmt.Errorf("remux FLAC: %w", err)
	}

	if err = out.Close(); err != nil {
		return "", err
	}

	success = true

	return out.Name(), nil
}
