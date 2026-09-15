package yandex

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	yandexclient "github.com/oshokin/zvuk-grabber/internal/client/yandex"
	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/retry"
)

// mp3Resolution holds the resolved MP3 download link and bitrate.
type mp3Resolution struct {
	// link is the signed direct download URL.
	link string
	// bitrate is the resolved MP3 bitrate in kbps.
	bitrate int
}

// retryAttemptsFromConfig returns the configured retry attempt count, defaulting to one.
func retryAttemptsFromConfig(cfg *config.Config) int64 {
	if cfg == nil || cfg.RetryAttemptsCount <= 0 {
		return 1
	}

	return cfg.RetryAttemptsCount
}

// buildYandexRetryEngine constructs the retry engine used by the Yandex service.
func buildYandexRetryEngine(cfg *config.Config) (*retry.Engine, error) {
	attempts := retryAttemptsFromConfig(cfg)

	maxRetries := uint64(1)

	retryClassifier := isRetryableYandexError
	if attempts <= 1 {
		retryClassifier = func(error) bool {
			return false
		}
	} else {
		maxRetries = uint64(attempts - 1)
	}

	minPause, maxPause := time.Duration(0), time.Duration(0)
	if cfg != nil {
		minPause = cfg.ParsedMinRetryPause
		maxPause = cfg.ParsedMaxRetryPause
	}

	return retry.NewEngine(&retry.EngineConfig{
		MaxRetries:  maxRetries,
		DelayPolicy: retry.NewRandomRangePolicy(minPause, maxPause),
		IsRetryable: retryClassifier,
	})
}

// retryValue executes an operation with the service retry policy and returns its result.
func retryValue[T any](ctx context.Context, service *ServiceImpl, operation string, fn func() (T, error)) (T, error) {
	var zero T

	var (
		cfg    *config.Config
		engine *retry.Engine
	)
	if service != nil {
		cfg = service.cfg
		engine = service.retryEngine
	}

	if engine == nil {
		var err error

		engine, err = buildYandexRetryEngine(cfg)
		if err != nil {
			return zero, fmt.Errorf("failed to initialize yandex retry engine: %w", err)
		}
	}

	attempts := retryAttemptsFromConfig(cfg)

	var (
		value        T
		attemptsMade int64
	)

	err := engine.Run(ctx, &retry.Request{
		Operation: func(context.Context) error {
			attemptsMade++

			result, operationErr := fn()
			if operationErr != nil {
				return operationErr
			}

			value = result

			return nil
		},
		OnRetry: func(ctx context.Context, info *retry.AttemptInfo) {
			logger.Warnf(ctx, "Retrying %s after error (%d/%d): %v", operation, info.Retry+1, attempts, info.Err)
		},
	})
	if err == nil {
		return value, nil
	}

	if attemptsMade <= 0 {
		attemptsMade = attempts
	}

	return zero, fmt.Errorf("%s failed after %d attempt(s): %w", operation, attemptsMade, err)
}

// openAudioWithRetry opens a remote audio stream with retry support.
func (s *ServiceImpl) openAudioWithRetry(ctx context.Context, rawURL string) (*yandexclient.RemoteFile, error) {
	return retryValue(ctx, s, "opening Yandex audio stream", func() (*yandexclient.RemoteFile, error) {
		return s.client.OpenAudio(ctx, rawURL)
	})
}

// downloadCoverBytesWithRetry downloads cover image bytes with retry support.
func (s *ServiceImpl) downloadCoverBytesWithRetry(ctx context.Context, rawURL string) ([]byte, error) {
	return retryValue(ctx, s, "downloading Yandex cover", func() ([]byte, error) {
		return s.client.DownloadCoverBytes(ctx, rawURL)
	})
}

// resolveMP3LinkWithRetry resolves an MP3 download link with retry support.
func (s *ServiceImpl) resolveMP3LinkWithRetry(
	ctx context.Context,
	trackID string,
	preferredQuality uint8,
) (string, int, error) {
	result, err := retryValue(ctx, s, "resolving Yandex MP3 link", func() (mp3Resolution, error) {
		link, bitrate, resolveErr := s.client.ResolveMP3Link(ctx, trackID, preferredQuality)
		return mp3Resolution{link: link, bitrate: bitrate}, resolveErr
	})
	if err != nil {
		return "", 0, err
	}

	return result.link, result.bitrate, nil
}

// trackLyricsWithRetry fetches track lyrics with retry support.
func (s *ServiceImpl) trackLyricsWithRetry(ctx context.Context, trackID string) (*model.TrackLyrics, error) {
	return retryValue(ctx, s, "fetching Yandex lyrics", func() (*model.TrackLyrics, error) {
		return s.client.TrackLyrics(ctx, trackID)
	})
}

// isRetryableYandexError reports whether a Yandex API error should trigger a retry.
func isRetryableYandexError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	if errors.Is(err, yandexclient.ErrNoFLACDownloadInfo) || errors.Is(err, yandexclient.ErrNoDownloadURLs) {
		return false
	}

	if apiErr, ok := errors.AsType[*model.ErrorResponse](err); ok {
		if strings.EqualFold(apiErr.APIError.Name, "validate") ||
			strings.EqualFold(apiErr.ResultError.Name, "validate") ||
			strings.EqualFold(apiErr.APIError.Name, "invalid sign") ||
			strings.EqualFold(apiErr.ResultError.Name, "invalid sign") {
			return false
		}
	}

	lowerErr := strings.ToLower(err.Error())
	if strings.Contains(lowerErr, "track lyrics not found") || strings.Contains(lowerErr, "lyrics text is empty") {
		return false
	}

	return true
}
