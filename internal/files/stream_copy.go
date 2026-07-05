package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/schollz/progressbar/v3"
)

// CopyStreamOptions configures CopyStream.
type CopyStreamOptions struct {
	// ExpectedBytes enables a final byte-count check when greater than zero.
	ExpectedBytes int64
	// SpeedLimitBytes limits copying to this number of bytes per second when greater than zero.
	SpeedLimitBytes int64
	// ShowProgress mirrors copied bytes to a progress bar when ExpectedBytes is known.
	ShowProgress bool
	// ProgressDescription is displayed near the progress bar.
	ProgressDescription string
}

// byteTokenBucket throttles copy throughput using a token-bucket refill model.
type byteTokenBucket struct {
	// capacity is the maximum number of bytes that can be consumed in one burst.
	capacity float64
	// tokens is the number of bytes currently available for immediate consumption.
	tokens float64
	// refillRate is the sustained byte refill rate per second.
	refillRate float64
	// lastRefill records when tokens were last replenished.
	lastRefill time.Time
}

// ErrIncompleteCopy indicates that fewer bytes were copied than the caller expected.
var ErrIncompleteCopy = errors.New("incomplete stream copy")

// CopyStream copies bytes from source to destination with optional progress, throttling and size validation.
func CopyStream(ctx context.Context, destination io.Writer, source io.Reader, opts *CopyStreamOptions) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	options := opts
	if options == nil {
		options = new(CopyStreamOptions)
	}

	options.normalize()

	writer := destination

	if options.ShowProgress && options.ExpectedBytes > 0 {
		writer = io.MultiWriter(
			destination,
			progressbar.DefaultBytes(options.ExpectedBytes, options.ProgressDescription),
		)
	}

	written, err := copyStream(ctx, writer, source, options.SpeedLimitBytes)
	if err != nil {
		return written, err
	}

	if options.ExpectedBytes > 0 && written != options.ExpectedBytes {
		return written, fmt.Errorf(
			"%w: wrote %s, expected %s",
			ErrIncompleteCopy,
			humanize.IBytes(nonNegativeUint64(written)),
			humanize.IBytes(nonNegativeUint64(options.ExpectedBytes)),
		)
	}

	return written, nil
}

// normalize clamps invalid option values in place.
func (opts *CopyStreamOptions) normalize() {
	if opts == nil {
		return
	}

	if opts.SpeedLimitBytes < 0 {
		opts.SpeedLimitBytes = 0
	}
}

// copyStream copies source to destination with optional byte-rate throttling.
func copyStream(ctx context.Context, destination io.Writer, source io.Reader, speedLimitBytes int64) (int64, error) {
	if speedLimitBytes <= 0 {
		return copyStreamWithoutLimit(ctx, destination, source)
	}

	limiter := newByteTokenBucket(speedLimitBytes)
	buffer := make([]byte, limitedCopyBufferSize(speedLimitBytes))

	var totalWritten int64

	for {
		if err := ctx.Err(); err != nil {
			return totalWritten, err
		}

		readBytes, readErr := source.Read(buffer)
		if readBytes > 0 {
			if err := limiter.wait(ctx, readBytes); err != nil {
				return totalWritten, err
			}

			writtenBytes, writeErr := destination.Write(buffer[:readBytes])
			totalWritten += int64(writtenBytes)

			if writeErr != nil {
				return totalWritten, writeErr
			}

			if writtenBytes != readBytes {
				return totalWritten, io.ErrShortWrite
			}
		}

		if errors.Is(readErr, io.EOF) {
			return totalWritten, nil
		}

		if readErr != nil {
			return totalWritten, readErr
		}
	}
}

// copyStreamWithoutLimit copies source to destination while checking cancellation between read/write steps.
func copyStreamWithoutLimit(ctx context.Context, destination io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, 32*1024)

	var totalWritten int64

	for {
		if err := ctx.Err(); err != nil {
			return totalWritten, err
		}

		readBytes, readErr := source.Read(buffer)
		if readBytes > 0 {
			if err := ctx.Err(); err != nil {
				return totalWritten, err
			}

			writtenBytes, writeErr := destination.Write(buffer[:readBytes])

			totalWritten += int64(writtenBytes)
			if writeErr != nil {
				return totalWritten, writeErr
			}

			if writtenBytes != readBytes {
				return totalWritten, io.ErrShortWrite
			}
		}

		if errors.Is(readErr, io.EOF) {
			return totalWritten, nil
		}

		if readErr != nil {
			return totalWritten, readErr
		}
	}
}

// limitedCopyBufferSize picks a copy buffer size that respects the configured speed limit.
func limitedCopyBufferSize(speedLimitBytes int64) int {
	const defaultCopyBufferSize = 32 * 1024

	if speedLimitBytes <= 0 {
		return defaultCopyBufferSize
	}

	if speedLimitBytes < defaultCopyBufferSize {
		return int(speedLimitBytes)
	}

	return defaultCopyBufferSize
}

// newByteTokenBucket creates a token bucket sized for the given bytes-per-second limit.
func newByteTokenBucket(speedLimitBytes int64) *byteTokenBucket {
	limit := float64(speedLimitBytes)

	return &byteTokenBucket{
		capacity:   limit,
		tokens:     limit,
		refillRate: limit,
		lastRefill: time.Now(),
	}
}

// wait blocks until the bucket has enough tokens to allow copying the requested bytes.
func (bucket *byteTokenBucket) wait(ctx context.Context, bytes int) error {
	if bytes <= 0 {
		return nil
	}

	requiredTokens := float64(bytes)

	for {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		bucket.refill(time.Now())

		if bucket.tokens >= requiredTokens {
			bucket.tokens -= requiredTokens
			return nil
		}

		missingTokens := requiredTokens - bucket.tokens

		waitDuration := time.Duration((missingTokens / bucket.refillRate) * float64(time.Second))
		if waitDuration <= 0 {
			waitDuration = time.Nanosecond
		}

		timer := time.NewTimer(waitDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// refill adds tokens based on elapsed time since the previous refill.
func (bucket *byteTokenBucket) refill(now time.Time) {
	elapsed := now.Sub(bucket.lastRefill)
	if elapsed <= 0 {
		return
	}

	bucket.tokens += elapsed.Seconds() * bucket.refillRate
	if bucket.tokens > bucket.capacity {
		bucket.tokens = bucket.capacity
	}

	bucket.lastRefill = now
}

// nonNegativeUint64 converts int64 to uint64, clamping negatives to zero.
func nonNegativeUint64(value int64) uint64 {
	if value <= 0 {
		return 0
	}

	return uint64(value)
}
