package yandex

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// defaultDescriptionName is the filename used for saved collection descriptions.
const defaultDescriptionName = "description.txt"

// saveCollectionDescription writes the album or podcast description sidecar when enabled.
func (s *ServiceImpl) saveCollectionDescription(ctx context.Context, jobs []*trackJob) {
	job := firstDescriptionJob(jobs)
	if job == nil {
		return
	}

	description := strings.TrimSpace(albumDescription(job.album))
	if description == "" {
		return
	}

	descriptionPath := s.collectionDescriptionPath(ctx, job)
	if descriptionPath == "" {
		s.recordDescriptionSkipped()
		return
	}

	unlock := s.pathLocks.Lock(descriptionPath)
	defer unlock()

	if s.useExistingDescription(ctx, descriptionPath, job.kind) {
		return
	}

	if s.cfg.DryRun {
		logger.Infof(
			ctx,
			"[DRY-RUN] Would save %s description to: %s",
			yandexCollectionTitle(job.kind),
			descriptionPath,
		)

		return
	}

	if err := writeSidecarFile(descriptionPath, description, s.cfg.ReplaceDescriptions); err != nil {
		s.recordDescriptionSkipped()
		logger.Errorf(ctx, "Failed to save Yandex description for %q: %v", job.collectionTitle, err)

		return
	}

	s.recordDescriptionSaved()
	logger.Infof(ctx, "Saved %s description to %s", yandexCollectionTitle(job.kind), filepath.Base(descriptionPath))
}

// collectionDescriptionPath returns the output path for a collection description file.
func (s *ServiceImpl) collectionDescriptionPath(ctx context.Context, job *trackJob) string {
	tags := s.buildTags(job)

	targetPath := s.buildTargetPath(ctx, job, tags, preferredQuality(s.cfg.Quality))
	if strings.TrimSpace(targetPath) == "" {
		return ""
	}

	return filepath.Join(filepath.Dir(targetPath), defaultDescriptionName)
}

// useExistingDescription skips writing when a description file already exists.
func (s *ServiceImpl) useExistingDescription(ctx context.Context, descriptionPath, kind string) bool {
	return s.useExistingSidecar(
		ctx,
		descriptionPath,
		s.cfg.ReplaceDescriptions,
		s.recordDescriptionSkipped,
		"Yandex description file check failed: path=%s err=%v",
		yandexCollectionTitle(kind)+" description already exists, skipping save",
	)
}

// firstDescriptionJob returns the first job eligible for description export.
func firstDescriptionJob(jobs []*trackJob) *trackJob {
	for _, job := range jobs {
		if job == nil || job.album == nil {
			continue
		}

		if job.kind == collectionAudiobook || job.kind == collectionPodcast {
			return job
		}
	}

	return nil
}
