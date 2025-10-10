package auth

import (
	"context"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// getAuthCookie retrieves the auth cookie value if it exists, without logging.
func (s *ServiceImpl) getAuthCookie(ctx context.Context) string {
	defer func() {
		if r := recover(); r != nil {
			logger.Debugf(ctx, "getAuthCookie panic recovered: %v", r)
		}
	}()

	cookies, err := s.page.Cookies([]string{s.cfg.ZvukBaseURL})
	if err != nil {
		return ""
	}

	for _, cookie := range cookies {
		if cookie.Name == authCookieName && cookie.Value != "" {
			return cookie.Value
		}
	}

	return ""
}

// extractTokenFromProfile extracts the auth token from browser cookies.
func (s *ServiceImpl) extractTokenFromProfile(ctx context.Context) (string, error) {
	logger.Info(ctx, "Extracting authentication token from cookies...")

	// Get current page URL.
	currentURL := s.page.MustInfo().URL
	logger.Debugf(ctx, "Current page URL: %s", currentURL)

	// Get all cookies.
	logger.Debug(ctx, "Fetching cookies from browser...")

	cookies := s.page.MustCookies()
	logger.Debugf(ctx, "Found %d cookies", len(cookies))

	// Log all cookies only in debug mode.
	if logger.IsDebugLevel() && len(cookies) > 0 {
		logger.Debug(ctx, "Cookie list:")

		for i, cookie := range cookies {
			logger.Debugf(ctx, "Cookie %d: name=%s, domain=%s, value=%s", i+1, cookie.Name, cookie.Domain, cookie.Value)
		}
	}

	// Find the auth cookie.
	logger.Debugf(ctx, "Looking for '%s' cookie...", authCookieName)

	var authToken string

	for _, cookie := range cookies {
		if cookie.Name == authCookieName {
			authToken = cookie.Value
			logger.Debugf(ctx, "Found '%s' cookie! Length: %d characters", authCookieName, len(authToken))

			break
		}
	}

	if authToken == "" {
		logger.Error(ctx, "Auth cookie not found! Available cookies:")

		for _, cookie := range cookies {
			logger.Errorf(ctx, "%s (domain: %s)", cookie.Name, cookie.Domain)
		}

		return "", ErrAuthCookieNotFound
	}

	logger.Info(ctx, "Token extracted successfully from cookie!")

	return authToken, nil
}
