package yandex

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"go.uber.org/zap"

	"github.com/oshokin/zvuk-grabber/internal/files"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/media"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

var (
	// errAudioPayloadNotInitialized is returned when writeAudioFile receives a nil payload.
	errAudioPayloadNotInitialized = errors.New("audio payload is not initialized")
	// errYandexAudioStreamOpenerNotConfig is returned when streaming MP3 without an opener.
	errYandexAudioStreamOpenerNotConfig = errors.New("yandex audio stream opener is not configured")
	// errAudioPayloadWithoutStreamOrBytes is returned when payload has neither bytes nor URL.
	errAudioPayloadWithoutStreamOrBytes = errors.New("audio payload does not contain stream or bytes")
)

// writeAudioFile streams or copies audio to a temp file, tags it, and atomically renames it.
//
//nolint:funlen // Function keeps the write pipeline linear and readable.
func (s *ServiceImpl) writeAudioFile(
	ctx context.Context,
	targetPath string,
	payload *audioPayload,
	job *trackJob,
	tags map[string]string,
	lyrics string,
) (int64, error) {
	if payload == nil {
		return 0, errAudioPayloadNotInitialized
	}

	tempFile, err := os.CreateTemp(filepath.Dir(targetPath), filepath.Base(targetPath)+tempFilePattern)
	if err != nil {
		return 0, err
	}

	var (
		tempPath = tempFile.Name()
		cleanup  = true
	)

	defer func() {
		_ = tempFile.Close()

		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	var (
		source        io.Reader
		sourceCloser  io.Closer
		expectedBytes = payload.contentLength
	)

	switch {
	case len(payload.data) > 0:
		source = bytes.NewReader(payload.data)
		if expectedBytes <= 0 {
			expectedBytes = int64(len(payload.data))
		}
	case strings.TrimSpace(payload.streamURL) != "":
		if s.client == nil {
			return 0, errYandexAudioStreamOpenerNotConfig
		}

		remoteFile, openErr := s.openAudioWithRetry(ctx, payload.streamURL)
		if openErr != nil {
			return 0, openErr
		}

		source = remoteFile.Body

		sourceCloser = remoteFile.Body
		if expectedBytes <= 0 {
			expectedBytes = remoteFile.ContentLength
		}
	default:
		return 0, errAudioPayloadWithoutStreamOrBytes
	}

	if sourceCloser != nil {
		defer sourceCloser.Close()
	}

	written, err := s.copyAudioStream(ctx, tempFile, source, expectedBytes)
	if err != nil {
		return 0, err
	}

	if err = tempFile.Close(); err != nil {
		return written, err
	}

	cover := s.downloadCover(ctx, targetPath, job)
	if err = s.tagProcessor.WriteTags(ctx, &media.WriteTagsRequest{
		TrackPath:  tempPath,
		CoverPath:  cover.path,
		Quality:    payload.quality,
		Tags:       tags,
		Lyrics:     strings.TrimSpace(lyrics),
		EmbedCover: cover.path != "",
	}); err != nil {
		return 0, err
	}

	if err = utils.RenameFile(tempPath, targetPath, s.cfg.ReplaceTracks); err != nil {
		return 0, err
	}

	cleanup = false

	return written, nil
}

// copyAudioStream copies source audio into destination with optional progress and throttling.
func (s *ServiceImpl) copyAudioStream(
	ctx context.Context,
	destination io.Writer,
	source io.Reader,
	expectedBytes int64,
) (int64, error) {
	return files.CopyStream(ctx, destination, source, &files.CopyStreamOptions{
		ExpectedBytes:       expectedBytes,
		SpeedLimitBytes:     s.cfg.ParsedDownloadSpeedLimit,
		ShowProgress:        logger.Level() <= zap.InfoLevel && s.cfg.MaxConcurrentDownloads == 1,
		ProgressDescription: "Downloading",
	})
}

// downloadCover fetches or reuses a cover image and returns its local path.
func (s *ServiceImpl) downloadCover(ctx context.Context, targetPath string, job *trackJob) *coverResult {
	coverURL := coverURL(job)
	if coverURL == "" {
		s.recordCoverSkipped()
		return &coverResult{}
	}

	coverPath := s.buildCoverPath(targetPath, job)
	if strings.TrimSpace(coverPath) == "" {
		s.recordCoverSkipped()
		return &coverResult{}
	}

	unlock := s.pathLocks.Lock(coverPath)
	defer unlock()

	if !s.cfg.ReplaceCovers {
		exists, existsErr := utils.IsFileExist(coverPath)
		if existsErr != nil {
			logger.Debugf(ctx, "Yandex cover file check failed: path=%s err=%v", coverPath, existsErr)
		} else if exists {
			s.recordCoverSkipped()
			logger.Infof(ctx, "%s cover already exists, skipping download", yandexCollectionTitle(job.kind))

			return &coverResult{
				path: coverPath,
			}
		}
	}

	if s.client == nil {
		s.recordCoverSkipped()
		logger.Errorf(ctx, "Yandex download client is not configured for cover download")

		return &coverResult{}
	}

	data, err := s.downloadCoverBytesWithRetry(ctx, coverURL)
	if err != nil {
		s.recordCoverSkipped()
		logger.Errorf(ctx, "Failed to download Yandex cover for %q: %v", job.track.FullTitle(), err)

		return &coverResult{}
	}

	if contentType := http.DetectContentType(data); !strings.HasPrefix(contentType, "image/") {
		s.recordCoverSkipped()
		logger.Errorf(ctx, "Yandex cover response is not an image for %q: %s", job.track.FullTitle(), contentType)

		return &coverResult{}
	}

	if err = os.WriteFile(coverPath, data, files.DefaultFilePermissions); err != nil {
		s.recordCoverSkipped()
		logger.Errorf(ctx, "Failed to write Yandex cover for %q: %v", job.track.FullTitle(), err)

		return &coverResult{}
	}

	s.recordCoverDownloaded()
	logger.Infof(ctx, "Successfully downloaded %s cover", yandexCollectionTitle(job.kind))
	logger.Debugf(ctx, "Downloaded Yandex cover: size=%s path=%s", humanize.IBytes(uint64(len(data))), coverPath)

	return &coverResult{
		path: coverPath,
	}
}
