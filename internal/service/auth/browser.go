package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// initBrowser initializes the rod browser instance.
func (s *ServiceImpl) initBrowser(ctx context.Context) error {
	logger.Debug(ctx, "Initializing browser")

	// Create a temporary directory for incognito-like session.
	// This avoids session persistence and provides a clean slate each time.
	// More importantly, it helps evade bot detection by not reusing profiles.
	tempDir, err := os.MkdirTemp("", "zvuk-auth-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary user data directory: %w", err)
	}

	logger.Debugf(ctx, "Using temporary profile directory: %s", tempDir)
	logger.Info(ctx, "Running in incognito mode (fresh browser profile)")

	// Store tempDir for cleanup.
	s.tempDir = tempDir

	// Try to find existing Chrome installation first.
	chromePath, exists := launcher.LookPath()

	var launcherURL string

	if exists {
		// Use system Chrome if available.
		logger.Debugf(ctx, "Using system Chrome installation at: %s", chromePath)
		launcherURL = launcher.New().
			// User needs to see the browser to log in.
			Headless(false).
			// Use temporary directory for incognito-like behavior.
			UserDataDir(tempDir).
			// Use system Chrome.
			Bin(chromePath).
			MustLaunch()
	} else {
		// Fall back to downloading Chromium.
		logger.Debug(ctx, "System Chrome not found, downloading Chromium")

		launcherURL = launcher.New().
			// User needs to see the browser to log in.
			Headless(false).
			// Use temporary directory for incognito-like behavior.
			UserDataDir(tempDir).
			MustLaunch()
	}

	logger.Debugf(ctx, "Browser launched at: %s", launcherURL)
	logger.Debugf(ctx, "User data directory: %s", tempDir)

	// Create browser instance.
	browserInstance := rod.New().ControlURL(launcherURL)

	// Enable trace and slow motion only in debug mode.
	if logger.IsDebugLevel() {
		logger.Debug(ctx, "Debug mode enabled - enabling browser trace and slow motion")

		browserInstance = browserInstance.
			// Enable tracing - logs all CDP events.
			Trace(true).
			// Slow down actions for visibility.
			SlowMotion(browserSlowMotionDelay)
	}

	s.browser = browserInstance.MustConnect()

	// Create a stealth-enabled page to evade bot detection.
	s.page = stealth.MustPage(s.browser)

	logger.Debug(ctx, "Browser initialized successfully with stealth mode")
	logger.Debugf(ctx, "Page created, ready to navigate")

	return nil
}

// isBrowserAlive checks if the browser is still running.
func (s *ServiceImpl) isBrowserAlive(ctx context.Context) bool {
	defer func() {
		// Recover from panic if browser is dead.
		if r := recover(); r != nil {
			// Browser panicked, log it in debug mode.
			logger.Debugf(ctx, "Browser panic recovered: %v", r)
		}
	}()

	// Try to get page info - will panic if browser/page is closed.
	_, err := s.page.Info()

	return err == nil
}

// getCurrentURL safely gets the current page URL.
func (s *ServiceImpl) getCurrentURL(ctx context.Context) (string, error) {
	defer func() {
		if r := recover(); r != nil {
			// Browser or page was closed, log it in debug mode.
			logger.Debugf(ctx, "getCurrentURL panic recovered: %v", r)
		}
	}()

	info, err := s.page.Info()
	if err != nil {
		return "", err
	}

	return info.URL, nil
}

// cleanup closes the browser and cleans up resources.
func (s *ServiceImpl) cleanup(ctx context.Context) {
	if s.browser != nil {
		// Close browser and wait for it to fully terminate.
		if err := s.browser.Close(); err != nil {
			logger.Debugf(ctx, "Browser close error (expected): %v", err)
		}
	}

	// Clean up temporary profile directory.
	if s.tempDir != "" {
		// Give Chrome a moment to release file locks.
		time.Sleep(browserCleanupDelay)

		if err := os.RemoveAll(s.tempDir); err != nil {
			// This can fail on Windows or if Chrome hasn't fully exited.
			// Not a critical error, so just debug log it.
			logger.Debugf(ctx, "Could not clean up temp directory %s: %v", s.tempDir, err)
		}
	}
}
