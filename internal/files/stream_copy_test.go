package files

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"testing/synctest"
	"time"
)

// slowFiniteReader simulates a rate-limited source for stream copy throttling tests.
type slowFiniteReader struct {
	// remaining is the number of bytes left to emit.
	remaining int
	// chunkSize is the maximum number of bytes returned per Read call.
	chunkSize int
	// delay is the sleep duration applied before each Read call.
	delay time.Duration
}

// Read returns the next chunk from the simulated slow reader.
func (r *slowFiniteReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}

	if r.delay > 0 {
		time.Sleep(r.delay)
	}

	n := r.chunkSize
	if n <= 0 || n > len(p) {
		n = len(p)
	}

	if n > r.remaining {
		n = r.remaining
	}

	for i := range n {
		p[i] = 'a'
	}

	r.remaining -= n

	return n, nil
}

// TestCopyStream_CopiesAndValidatesExpectedBytes verifies CopyStream copies data and validates expected byte counts.
func TestCopyStream_CopiesAndValidatesExpectedBytes(t *testing.T) {
	t.Parallel()

	var destination bytes.Buffer

	written, err := CopyStream(
		t.Context(),
		&destination,
		bytes.NewBufferString("hello"),
		&CopyStreamOptions{ExpectedBytes: 5},
	)
	if err != nil {
		t.Fatalf("CopyStream returned error: %v", err)
	}

	if written != 5 {
		t.Fatalf("written bytes = %d, want 5", written)
	}

	if got := destination.String(); got != "hello" {
		t.Fatalf("destination = %q, want %q", got, "hello")
	}
}

// TestCopyStream_ReturnsIncompleteCopy verifies CopyStream returns ErrIncompleteCopy when fewer bytes are read.
func TestCopyStream_ReturnsIncompleteCopy(t *testing.T) {
	t.Parallel()

	var destination bytes.Buffer

	written, err := CopyStream(
		t.Context(),
		&destination,
		bytes.NewBufferString("hello"),
		&CopyStreamOptions{ExpectedBytes: 6},
	)
	if !errors.Is(err, ErrIncompleteCopy) {
		t.Fatalf("error = %v, want ErrIncompleteCopy", err)
	}

	if written != 5 {
		t.Fatalf("written bytes = %d, want 5", written)
	}
}

// TestCopyStream_RespectsContextCancellationWithoutSpeedLimit verifies context cancellation works with unlimited copy speed.
func TestCopyStream_RespectsContextCancellationWithoutSpeedLimit(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		var destination bytes.Buffer

		reader := &slowFiniteReader{
			remaining: 256 * 1024,
			chunkSize: 1024,
			delay:     time.Millisecond,
		}

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go func() {
			time.Sleep(5 * time.Millisecond)
			cancel()
		}()

		written, err := CopyStream(ctx, &destination, reader, &CopyStreamOptions{SpeedLimitBytes: 0})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}

		if written >= int64(256*1024) {
			t.Fatalf("written bytes = %d, want partial copy before cancellation", written)
		}
	})
}

// TestCopyStream_ExactMultipleOfSpeedLimit_NoExtraWindowDelay verifies CopyStream does not
// wait an extra throttling window when stream size is an exact speed-limit multiple.
func TestCopyStream_ExactMultipleOfSpeedLimit_NoExtraWindowDelay(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		var destination bytes.Buffer

		ctx, cancel := context.WithTimeout(t.Context(), 750*time.Millisecond)
		defer cancel()

		written, err := CopyStream(
			ctx,
			&destination,
			bytes.NewBufferString("abcd"),
			&CopyStreamOptions{SpeedLimitBytes: 4},
		)
		if err != nil {
			t.Fatalf("CopyStream returned error: %v", err)
		}

		if written != 4 {
			t.Fatalf("written bytes = %d, want 4", written)
		}

		if got := destination.String(); got != "abcd" {
			t.Fatalf("destination = %q, want %q", got, "abcd")
		}
	})
}
