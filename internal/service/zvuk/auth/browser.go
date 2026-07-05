package auth

import (
	"context"

	authbrowser "github.com/oshokin/zvuk-grabber/internal/browser"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// initBrowser initializes the rod browser instance.
func (s *ServiceImpl) initBrowser(ctx context.Context) error {
	logger.Debug(ctx, "Initializing browser")

	session, err := authbrowser.Open(ctx, &authbrowser.Options{
		ProfileDirPrefix: "zvuk-auth-*",
		Headless:         false,
		UseStealth:       true,
		SlowMotion:       browserSlowMotionDelay,
	})
	if err != nil {
		return err
	}

	s.session = session
	s.browser = session.Browser
	s.page = session.Page

	logger.Debug(ctx, "Browser initialized successfully with stealth mode")
	logger.Debugf(ctx, "Page created, ready to navigate")

	return nil
}

// isBrowserAlive checks if the browser is still running.
func (s *ServiceImpl) isBrowserAlive(ctx context.Context) bool {
	if s.session == nil {
		return false
	}

	return s.session.IsAlive(ctx)
}

// getCurrentURL safely gets the current page URL.
func (s *ServiceImpl) getCurrentURL(ctx context.Context) (string, error) {
	if s.session == nil {
		return "", ErrBrowserClosed
	}

	return s.session.CurrentURL(ctx)
}

// cleanup closes the browser and cleans up resources.
func (s *ServiceImpl) cleanup(ctx context.Context) {
	if s.session == nil {
		return
	}

	s.session.Close(ctx)
	s.session = nil
	s.browser = nil
	s.page = nil
}
