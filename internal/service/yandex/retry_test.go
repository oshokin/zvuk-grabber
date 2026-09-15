package yandex

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	yandexclient "github.com/oshokin/zvuk-grabber/internal/client/yandex"
	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
)

// errTemporaryNetwork simulates a transient network failure in retry tests.
var errTemporaryNetwork = errors.New("temporary network issue")

// TestRetryValue_NoRetryOnNoFLACError verifies missing FLAC metadata is not retried.
func TestRetryValue_NoRetryOnNoFLACError(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{
		cfg: &config.Config{
			RetryAttemptsCount: 5,
		},
	}

	calls := 0
	_, err := retryValue(t.Context(), service, "downloading Yandex FLAC audio", func() (string, error) {
		calls++

		return "", fmt.Errorf(
			"%w: expected FLAC lossless, got AAC in MP4 container",
			yandexclient.ErrNoFLACDownloadInfo,
		)
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
	assert.Contains(t, err.Error(), "after 1 attempt(s)")
}

// TestRetryValue_NoRetryOnValidationError verifies API validation errors are not retried.
func TestRetryValue_NoRetryOnValidationError(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{
		cfg: &config.Config{
			RetryAttemptsCount: 5,
		},
	}

	validationErr := &model.ErrorResponse{}
	validationErr.APIError.Name = "validate"
	validationErr.APIError.Message = "Parameters requirements are not met."

	calls := 0
	_, err := retryValue(t.Context(), service, "fetching Yandex lyrics", func() (string, error) {
		calls++
		return "", validationErr
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
	assert.Contains(t, err.Error(), "after 1 attempt(s)")
}

// TestRetryValue_NoRetryOnInvalidSignError verifies invalid sign errors are not retried.
func TestRetryValue_NoRetryOnInvalidSignError(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{
		cfg: &config.Config{
			RetryAttemptsCount: 5,
		},
	}

	invalidSignErr := &model.ErrorResponse{}
	invalidSignErr.APIError.Name = "Invalid Sign"

	calls := 0
	_, err := retryValue(t.Context(), service, "fetching Yandex lyrics", func() (string, error) {
		calls++
		return "", invalidSignErr
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
	assert.Contains(t, err.Error(), "after 1 attempt(s)")
}

// TestRetryValue_RetriesTransientError verifies transient failures are retried until success.
func TestRetryValue_RetriesTransientError(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{
		cfg: &config.Config{
			RetryAttemptsCount: 5,
		},
	}

	calls := 0
	value, err := retryValue(t.Context(), service, "resolving Yandex MP3 link", func() (string, error) {
		calls++
		if calls < 3 {
			return "", errTemporaryNetwork
		}

		return "ok", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", value)
	assert.Equal(t, 3, calls)
}

// TestRetryValue_CancelDuringRetryPause verifies cancellation aborts retry backoff.
func TestRetryValue_CancelDuringRetryPause(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		service := &ServiceImpl{
			cfg: &config.Config{
				RetryAttemptsCount:  5,
				ParsedMinRetryPause: 500 * time.Millisecond,
				ParsedMaxRetryPause: 500 * time.Millisecond,
			},
		}

		ctx, cancel := context.WithCancel(t.Context())

		go func() {
			time.Sleep(25 * time.Millisecond)
			cancel()
		}()

		calls := 0

		_, err := retryValue(
			ctx,
			service,
			"resolving Yandex MP3 link",
			func() (string, error) {
				calls++

				return "", errTemporaryNetwork
			},
		)

		require.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, calls)
	})
}
