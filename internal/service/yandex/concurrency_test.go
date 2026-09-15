package yandex

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/service/stats"
)

// TestDownloadJobs_SequentialWhenMaxConcurrentOne verifies downloads run sequentially when concurrency is one.
func TestDownloadJobs_SequentialWhenMaxConcurrentOne(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		s := &ServiceImpl{
			cfg: &config.Config{
				MaxConcurrentDownloads: 1,
			},
			sessionStats: stats.NewSession(time.Time{}, false),
		}

		var (
			current atomic.Int64
			maxSeen atomic.Int64
		)

		s.executeDownloadJobs(t.Context(), makeJobs(6), func(_ context.Context, _ *trackJob) {
			value := current.Add(1)

			for {
				currentMax := maxSeen.Load()
				if value <= currentMax || maxSeen.CompareAndSwap(currentMax, value) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			current.Add(-1)
		})
		assert.Equal(t, int64(1), maxSeen.Load())
	})
}

// TestDownloadJobs_ConcurrentLimitRespected verifies the concurrent download limit is enforced.
func TestDownloadJobs_ConcurrentLimitRespected(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		s := &ServiceImpl{
			cfg: &config.Config{
				MaxConcurrentDownloads: 3,
			},
			sessionStats: stats.NewSession(time.Time{}, false),
		}

		var (
			current atomic.Int64
			maxSeen atomic.Int64
		)

		s.executeDownloadJobs(t.Context(), makeJobs(20), func(_ context.Context, _ *trackJob) {
			value := current.Add(1)

			for {
				currentMax := maxSeen.Load()
				if value <= currentMax || maxSeen.CompareAndSwap(currentMax, value) {
					break
				}
			}

			time.Sleep(15 * time.Millisecond)
			current.Add(-1)
		})

		require.GreaterOrEqual(t, maxSeen.Load(), int64(2))
		assert.LessOrEqual(t, maxSeen.Load(), int64(3))
	})
}

// TestDownloadJobs_ContextCancellationStopsQueuedWork verifies context cancellation stops queued download jobs.
func TestDownloadJobs_ContextCancellationStopsQueuedWork(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		s := &ServiceImpl{
			cfg: &config.Config{
				MaxConcurrentDownloads: 2,
			},
			sessionStats: stats.NewSession(time.Time{}, false),
		}

		var started atomic.Int64

		ctx, cancel := context.WithCancel(t.Context())

		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		s.executeDownloadJobs(ctx, makeJobs(30), func(_ context.Context, _ *trackJob) {
			started.Add(1)
			time.Sleep(50 * time.Millisecond)
		})
		assert.Less(t, started.Load(), int64(30))
	})
}

// TestDownloadJobs_StatsCorrectUnderParallelExecution verifies download stats remain correct under parallel execution.
func TestDownloadJobs_StatsCorrectUnderParallelExecution(t *testing.T) {
	t.Parallel()

	s := &ServiceImpl{
		cfg: &config.Config{
			MaxConcurrentDownloads: 4,
		},
		sessionStats: stats.NewSession(time.Time{}, false),
	}

	s.executeDownloadJobs(t.Context(), makeJobs(12), func(_ context.Context, _ *trackJob) {
		s.recordProcessed()
		s.recordDownloaded(10)
	})

	snapshot := s.statsSnapshot()
	assert.Equal(t, int64(12), snapshot.Tracks.TotalProcessed)
	assert.Equal(t, int64(12), snapshot.Tracks.Downloaded)
	assert.Equal(t, int64(120), snapshot.BytesDownloaded)
}

// makeJobs builds a slice of empty track jobs for concurrency tests.
func makeJobs(count int) []*trackJob {
	jobs := make([]*trackJob, 0, count)
	for range count {
		jobs = append(jobs, &trackJob{})
	}

	return jobs
}
