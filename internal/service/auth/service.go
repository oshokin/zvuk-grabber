package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-rod/rod"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
)

const (
	// browserSlowMotionDelay is the delay between browser actions for visibility during debugging.
	browserSlowMotionDelay = 200 * time.Millisecond

	// zvukHomeURL is the main Zvuk landing page.
	zvukHomeURL = "https://zvuk.com/"

	// zvukLoginURL is the dedicated login page URL.
	zvukLoginURL = "https://id.zvuk.com/desktop?returnUrl=https://zvuk.com/"

	// zvukDomain is the main Zvuk domain.
	zvukDomain = "zvuk.com"

	// idZvukDomain is the Zvuk ID service domain.
	idZvukDomain = "id.zvuk.com"

	// sberIDDomain is the Sber ID OAuth service domain.
	sberIDDomain = "id.sber.ru"

	// authCookieName is the name of the authentication cookie.
	authCookieName = "auth"

	// avatarButtonSelector is the CSS selector for the avatar button (appears when logged in).
	// Uses attribute selector to avoid CSS Modules hash issues.
	avatarButtonSelector = `[class^="Header_triggerWrapper"]`

	// loginButtonSelector is the CSS selector for the login button (appears when not logged in).
	// Uses attribute selector to avoid CSS Modules hash issues.
	loginButtonSelector = `[class^="Header_authButton"]`

	// loginPollInterval is the interval for polling the login status.
	loginPollInterval = 1 * time.Second

	// maxLoginWaitTime is the maximum time to wait for user to complete login.
	maxLoginWaitTime = 10 * time.Minute

	// sessionEstablishDelay is the delay after login to allow session to fully establish.
	sessionEstablishDelay = 2 * time.Second

	// humanBehaviorMinDelay is the minimum delay for simulated human actions.
	humanBehaviorMinDelay = 500 * time.Millisecond
	// humanBehaviorMaxDelay is the maximum delay for simulated human actions.
	humanBehaviorMaxDelay = 2 * time.Second

	// mouseMovementsPerCheck is the number of random mouse movements to simulate per polling cycle.
	mouseMovementsPerCheck = 2

	// mouseMovementMinDelay is the minimum delay between mouse movements.
	mouseMovementMinDelay = 100 * time.Millisecond
	// mouseMovementMaxDelay is the maximum delay between mouse movements.
	mouseMovementMaxDelay = 400 * time.Millisecond

	// scrollProbability is the probability of scrolling (1 in N).
	scrollProbability = 3
	// scrollMinAmount is the minimum scroll amount in pixels.
	scrollMinAmount = -100
	// scrollMaxAmount is the maximum scroll amount in pixels.
	scrollMaxAmount = 200

	// interactionProbability is the probability of random interaction (1 in N).
	interactionProbability = 5
	// interactionActionCount is the number of possible random interaction actions.
	interactionActionCount = 4

	// smallScrollRange is the range for small random scrolls.
	smallScrollRange = 100
	// smallScrollOffset is the offset to center small scroll range.
	smallScrollOffset = 50

	// pauseMinDelay is the minimum pause duration for human-like pauses.
	pauseMinDelay = 500 * time.Millisecond
	// pauseMaxDelay is the maximum pause duration for human-like pauses.
	pauseMaxDelay = 1500 * time.Millisecond

	// oauthCookieWaitDelay is the delay to wait for OAuth cookies to be set.
	oauthCookieWaitDelay = 2 * time.Second
	// oauthPageLoadDelay is the delay to wait for page to load after OAuth redirect.
	oauthPageLoadDelay = 3 * time.Second

	// minPageHTMLLength is the minimum HTML length to consider page loaded.
	minPageHTMLLength = 100

	// browserCleanupDelay is the delay to wait for Chrome to release file locks before cleanup.
	browserCleanupDelay = 500 * time.Millisecond
)

var (
	// ErrLoginTimeout is returned when login takes too long.
	ErrLoginTimeout = errors.New("login timeout exceeded")

	// ErrBrowserClosed is returned when the browser is closed by the user.
	ErrBrowserClosed = errors.New("browser was closed by user")

	// ErrNavigatedAway is returned when the user navigates away from the login flow.
	ErrNavigatedAway = errors.New("user navigated away from login flow")

	// ErrAuthCookieNotFound is returned when the auth cookie cannot be found after login.
	ErrAuthCookieNotFound = errors.New("auth cookie not found - login may have failed")
)

// Service provides browser-based authentication.
type Service interface {
	// LoginAndExtractToken opens a browser, waits for user to log in, then extracts the auth token.
	LoginAndExtractToken(ctx context.Context) (string, error)
}

// ServiceImpl provides browser-based authentication for Zvuk.
type ServiceImpl struct {
	cfg     *config.Config
	browser *rod.Browser
	page    *rod.Page
	// tempDir stores the temporary profile directory for cleanup.
	tempDir string
}

// NewService creates a new browser authentication service.
func NewService(cfg *config.Config) (*ServiceImpl, error) {
	return &ServiceImpl{
		cfg: cfg,
	}, nil
}

// LoginAndExtractToken opens a browser, waits for user to log in, then extracts the auth token.
func (s *ServiceImpl) LoginAndExtractToken(ctx context.Context) (string, error) {
	logger.Info(ctx, "Starting browser-based authentication")

	// Initialize browser.
	if err := s.initBrowser(ctx); err != nil {
		return "", fmt.Errorf("failed to initialize browser: %w", err)
	}

	defer s.cleanup(ctx)

	// Navigate to login page and wait for user to complete authentication.
	directToken, err := s.waitForUserLogin(ctx)
	if err != nil {
		return "", fmt.Errorf("login failed: %w", err)
	}

	if directToken != "" {
		logger.Info(ctx, "Authentication token retrieved directly from OAuth flow")

		return directToken, nil
	}

	// Extract token from browser cookies.
	token, err := s.extractTokenFromProfile(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to extract token: %w", err)
	}

	logger.Info(ctx, "Authentication token extracted successfully")

	return token, nil
}
