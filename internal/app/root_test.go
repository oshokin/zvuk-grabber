package app

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// fakeDownloadService records whether the summary ran after success, error, or panic.
type fakeDownloadService struct {
	// failure is returned by DownloadURLs when panicValue is nil.
	failure error
	// panicValue, when set, is panicked from DownloadURLs.
	panicValue any
	// summarized is set when PrintDownloadSummary runs.
	summarized bool
}

// DownloadURLs returns failure, or panics when panicValue is set.
func (s *fakeDownloadService) DownloadURLs(context.Context, []string) error {
	if s.panicValue != nil {
		panic(s.panicValue)
	}

	return s.failure
}

// PrintDownloadSummary marks that the deferred summary ran.
func (s *fakeDownloadService) PrintDownloadSummary(context.Context) {
	s.summarized = true
}

// TestRunDownloadsPropagatesFailuresAndPrintsSummary prints the summary after success, error, and panic.
func TestRunDownloadsPropagatesFailuresAndPrintsSummary(t *testing.T) {
	expected := io.ErrUnexpectedEOF

	ok := new(fakeDownloadService)
	require.NoError(t, runDownloads(t.Context(), ok, nil))
	require.True(t, ok.summarized)

	failed := &fakeDownloadService{failure: expected}
	require.ErrorIs(t, runDownloads(t.Context(), failed, nil), expected)
	require.True(t, failed.summarized)

	panicked := &fakeDownloadService{panicValue: "broken"}

	require.Panics(t, func() {
		err := runDownloads(t.Context(), panicked, nil)
		require.NoError(t, err)
	})
	require.True(t, panicked.summarized)
}

// TestRootReturnsInputAndAuthErrors rejects unsupported URLs, missing tokens, and a canceled context.
func TestRootReturnsInputAndAuthErrors(t *testing.T) {
	urls := [][]string{
		{"https://unsupported.test/123"},
		{"https://zvuk.com/track/123"},
		{"https://music.yandex.ru/album/1/track/2"},
	}

	for _, urls := range urls {
		require.Error(t, ExecuteRootCommand(t.Context(), config.DefaultConfig(), urls))
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	require.ErrorIs(t, ExecuteRootCommand(ctx, config.DefaultConfig(), nil), context.Canceled)
}
