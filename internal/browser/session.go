package browser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// Options controls browser session startup behavior.
type Options struct {
	// ProfileDirPrefix is the temp directory name prefix for the browser profile.
	ProfileDirPrefix string
	// Headless runs the browser without a visible window when true.
	Headless bool
	// UseStealth opens pages with anti-detection patches when true.
	UseStealth bool
	// SlowMotion adds a delay between browser actions for debugging visibility.
	SlowMotion time.Duration
	// CleanupDelay waits before removing the temp profile after browser close.
	CleanupDelay time.Duration
}

// Session holds an opened browser and page with cleanup metadata.
type Session struct {
	// Browser is the connected rod browser instance.
	Browser *rod.Browser
	// Page is the active browser tab used for user interaction.
	Page *rod.Page

	// tempDir is the filesystem path to the temporary browser profile.
	tempDir string
	// cleanupDelay waits before removing tempDir on Close.
	cleanupDelay time.Duration
}

// defaultCleanupDelay is the default wait before deleting a browser temp profile.
const defaultCleanupDelay = 500 * time.Millisecond

var (
	// errPageNotInitialized is returned when the browser page is not ready.
	errPageNotInitialized = errors.New("browser page is not initialized")
	// errPageInspectionPanic is returned when page inspection panics.
	errPageInspectionPanic = errors.New("browser page inspection panic")
)

// Open creates and connects a browser session.
func Open(ctx context.Context, opts *Options) (*Session, error) {
	if opts == nil {
		opts = &Options{}
	}

	prefix := opts.ProfileDirPrefix
	if prefix == "" {
		prefix = "zvuk-grabber-auth-*"
	}

	tempDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary browser profile: %w", err)
	}

	cleanupDelay := opts.CleanupDelay
	if cleanupDelay <= 0 {
		cleanupDelay = defaultCleanupDelay
	}

	browserLauncher := launcher.New().
		Headless(opts.Headless).
		UserDataDir(tempDir)

	if chromePath, exists := launcher.LookPath(); exists {
		logger.Debugf(ctx, "Using system Chrome installation at: %s", chromePath)
		browserLauncher = browserLauncher.Bin(chromePath)
	} else {
		logger.Debug(ctx, "System Chrome not found, rod may download Chromium")
	}

	launcherURL, err := browserLauncher.Launch()
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browserInstance := rod.New().ControlURL(launcherURL)
	if logger.IsDebugLevel() && opts.SlowMotion > 0 {
		browserInstance = browserInstance.
			Trace(true).
			SlowMotion(opts.SlowMotion)
	}

	if err = browserInstance.Connect(); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	connectedBrowser := browserInstance

	var page *rod.Page
	if opts.UseStealth {
		page, err = stealth.Page(connectedBrowser)
	} else {
		page, err = connectedBrowser.Page(proto.TargetCreateTarget{})
	}

	if err != nil {
		_ = connectedBrowser.Close()
		_ = os.RemoveAll(tempDir)

		return nil, fmt.Errorf("failed to create browser page: %w", err)
	}

	return &Session{
		Browser:      connectedBrowser,
		Page:         page,
		tempDir:      tempDir,
		cleanupDelay: cleanupDelay,
	}, nil
}

// Close shuts down browser resources and removes temporary profile data.
func (s *Session) Close(ctx context.Context) {
	if s == nil {
		return
	}

	if s.Browser != nil {
		if err := s.Browser.Close(); err != nil {
			logger.Debugf(ctx, "Browser close error: %v", err)
		}
	}

	if s.tempDir != "" {
		time.Sleep(s.cleanupDelay)

		if err := os.RemoveAll(s.tempDir); err != nil {
			logger.Debugf(ctx, "Could not clean up browser temp dir %s: %v", s.tempDir, err)
		}
	}
}

// IsAlive reports whether the session page is still available.
func (s *Session) IsAlive(ctx context.Context) bool {
	if s == nil || s.Page == nil {
		return false
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Debugf(ctx, "Browser session panic recovered: %v", r)
		}
	}()

	_, err := s.Page.Info()

	return err == nil
}

// CurrentURL safely reads the current page URL.
func (s *Session) CurrentURL(ctx context.Context) (url string, err error) {
	if s == nil || s.Page == nil {
		return "", errPageNotInitialized
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Debugf(ctx, "CurrentURL panic recovered: %v", r)
			err = fmt.Errorf("%w: %v", errPageInspectionPanic, r)
		}
	}()

	info, err := s.Page.Info()
	if err != nil {
		return "", err
	}

	return info.URL, nil
}

// WaitOrCancel waits for d or returns if the context is canceled.
func WaitOrCancel(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
