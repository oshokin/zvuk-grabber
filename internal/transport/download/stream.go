// Package download implements bounded retries and validated continuation of audio transfers.
package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/files"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/retry"
)

// Error marks a completed download failure so service retries cannot multiply its budget.
type Error struct {
	// Err is the underlying network, protocol, or local I/O failure.
	Err error
}

// Stream owns one URL's request budget and current response. It is not safe for concurrent use.
// Read exposes the current body; CopyTo additionally handles body retries and file rollback.
type Stream struct {
	// client owns the provider's dedicated audio transport.
	client *http.Client
	// request retains context and headers for subsequent GET requests.
	request *http.Request
	// config contains an immutable snapshot of retry settings.
	config *config.DownloadHTTPConfig
	// response is the currently opened response, closed before every retry.
	response *http.Response
	// offset is the first byte in the accepted response.
	offset int64
	// total is the full representation length, or -1 when unknown.
	total int64
	// validator is a strong ETag or sufficiently old Last-Modified value for If-Range.
	validator string
	// retries counts additional requests across both headers and body failures.
	retries uint64
	// engine sleeps and enforces the shared retry budget for this stream.
	engine *retry.Engine
	// delayPolicy applies exponential backoff and optional Retry-After.
	delayPolicy *retryAfterAwarePolicy
}

// bodyReadError distinguishes source failures from destination write failures.
type bodyReadError struct {
	// error preserves the underlying read error for errors.Is and errors.As.
	error
}

// trackedReader marks non-EOF body errors before the copier can mix them with write errors.
type trackedReader struct {
	// reader is the current HTTP response body.
	reader io.Reader
}

// Error describes a failed transfer without exposing signed request URLs.
func (e *Error) Error() string { return "audio download failed: " + e.Err.Error() }

// Unwrap preserves cancellation and underlying error classification.
func (e *Error) Unwrap() error { return e.Err }

// Unwrap exposes the original read failure.
func (e *bodyReadError) Unwrap() error { return e.error }

// Read preserves returned bytes and wraps only body failures.
func (r *trackedReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		return n, &bodyReadError{error: err}
	}

	return n, err
}

// Open starts a GET with bounded retries. The caller must close the returned stream.
func Open(client *http.Client, request *http.Request, cfg *config.DownloadHTTPConfig) (*Stream, error) {
	if cfg == nil {
		cfg = config.DefaultDownloadHTTPConfig()
	}

	cfg = cfg.Clone()
	delayPolicy := &retryAfterAwarePolicy{
		inner: retry.NewExponentialEqualJitterPolicy(cfg.RetryInitialDelay, cfg.RetryMaxDelay),
	}

	engine, err := newAudioRetryEngine(cfg, delayPolicy)
	if err != nil {
		return nil, &Error{Err: err}
	}

	stream := &Stream{
		client:      client,
		request:     request.Clone(request.Context()),
		config:      cfg,
		total:       -1,
		engine:      engine,
		delayPolicy: delayPolicy,
	}
	if openErr := stream.open(0); openErr != nil {
		return nil, &Error{Err: openErr}
	}

	return stream, nil
}

// Read reads the current body without retrying; use Copy for resumable file transfers.
func (s *Stream) Read(p []byte) (int, error) { return s.response.Body.Read(p) }

// Close releases the current response and its connection.
func (s *Stream) Close() error { return s.response.Body.Close() }

// TotalBytes returns the full representation size, or -1 when it is not advertised.
func (s *Stream) TotalBytes() int64 { return s.total }

// StatusCode returns the current successful HTTP response status.
func (s *Stream) StatusCode() int { return s.response.StatusCode }

// ContentType returns the media type advertised by the current response.
func (s *Stream) ContentType() string { return s.response.Header.Get("Content-Type") }

// Copy dispatches resumable streams to their transfer engine and copies ordinary readers unchanged.
func Copy(ctx context.Context, destination io.Writer, source io.Reader, opts *files.CopyStreamOptions) (int64, error) {
	if stream, ok := source.(interface {
		CopyTo(ctx context.Context, destination io.Writer, opts *files.CopyStreamOptions) (int64, error)
	}); ok {
		return stream.CopyTo(ctx, destination, opts)
	}

	return files.CopyStream(ctx, destination, source, opts)
}

// CopyTo writes into an initially empty destination, resuming only validated representations.
// A destination supporting Seek and Truncate can also recover when a CDN requires a full restart.
func (s *Stream) CopyTo(ctx context.Context, destination io.Writer, opts *files.CopyStreamOptions) (int64, error) {
	options := opts.Clone()

	var written int64

	for {
		options.InitialBytes = written
		options.ExpectedBytes = s.total - written

		body := &trackedReader{reader: s.response.Body}
		copied, err := files.CopyStream(ctx, destination, body, options)

		written += copied
		if err == nil {
			return written, nil
		}

		nextOffset, retryErr := s.recoverBody(ctx, destination, written, err)
		if retryErr != nil {
			return written, &Error{Err: retryErr}
		}

		written = nextOffset
	}
}

// ReadAll spools a resumable transfer before returning complete bytes for decryption.
// No partially downloaded or decrypted data is returned on failure.
func ReadAll(ctx context.Context, source io.Reader, opts *files.CopyStreamOptions) ([]byte, error) {
	file, err := os.CreateTemp("", "zvuk-grabber-audio-*.part")
	if err != nil {
		return nil, &Error{Err: err}
	}

	defer func() { _ = file.Close(); _ = os.Remove(file.Name()) }()

	if _, err = Copy(ctx, file, source, opts); err != nil {
		return nil, err
	}

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, &Error{Err: err}
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, &Error{Err: err}
	}

	return data, nil
}

// recoverBody classifies a failed copy, then either reopens a range or resets the destination.
func (s *Stream) recoverBody(ctx context.Context, destination io.Writer, written int64, err error) (int64, error) {
	_ = s.Close()

	if ctx.Err() != nil {
		return written, ctx.Err()
	}

	var readErr *bodyReadError

	if !errors.As(err, &readErr) && !errors.Is(err, files.ErrIncompleteCopy) {
		return written, err
	}

	if errors.Is(err, files.ErrIncompleteCopy) {
		err = io.ErrUnexpectedEOF
	}

	if retryErr := s.retry(err); retryErr != nil {
		return written, retryErr
	}

	offset := written
	if !s.config.Resume || s.validator == "" || s.total < 0 || offset >= s.total {
		offset = 0
	}

	if err = s.open(offset); err != nil {
		return written, err
	}

	if s.offset > 0 {
		logger.Infof(ctx, "Resuming audio download at byte %d of %d", written, s.total)
		return written, nil
	}

	if written == 0 {
		return 0, nil
	}

	logger.Warnf(ctx, "Restarting audio download from the beginning (continuation disabled or unavailable)")

	if err = resetDestination(destination); err != nil {
		return written, err
	}

	return 0, nil
}

// resetDestination prevents concatenating a full response onto an incomplete file.
func resetDestination(destination io.Writer) error {
	file, ok := destination.(interface {
		io.Seeker
		Truncate(size int64) error
	})
	if !ok {
		return fmt.Errorf("%w: destination cannot restart", errInvalidResponse)
	}

	if err := file.Truncate(0); err != nil {
		return err
	}

	_, err := file.Seek(0, io.SeekStart)

	return err
}
