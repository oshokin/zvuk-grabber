package auth

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"strings"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/retry"
)

// waitForUserLogin navigates to the dedicated login page and waits for successful authentication.
//
//nolint:funlen // Login instructions require many log statements and monitoring logic.
func (s *ServiceImpl) waitForUserLogin(ctx context.Context) (string, error) {
	logger.Info(ctx, "Opening Zvuk homepage...")

	// Navigate to Zvuk homepage so subsequent OAuth calls run under zvuk.com origin.
	logger.Debugf(ctx, "Navigating to %s", zvukHomeURL)

	// Add random delay before navigation to appear more human.
	randomHumanDelay()

	if err := s.page.Navigate(zvukHomeURL); err != nil {
		return "", fmt.Errorf("failed to navigate to zvuk homepage: %w", err)
	}

	// Wait for page to fully load with random delay.
	randomHumanDelay()

	// Perform some human-like mouse movements after page load.
	s.simulateHumanBehavior(ctx)

	pageInfo, err := s.page.Info()
	if err != nil {
		return "", fmt.Errorf("failed to read zvuk page info: %w", err)
	}

	currentURL := pageInfo.URL
	logger.Debugf(ctx, "Navigation complete. Current URL: %s", currentURL)

	logger.Info(ctx, "")
	logger.Info(ctx, "╔══════════════════════════════════════════════════════════════════╗")
	logger.Info(ctx, "║                      LOGIN INSTRUCTIONS                          ║")
	logger.Info(ctx, "╚══════════════════════════════════════════════════════════════════╝")
	logger.Info(ctx, "")
	logger.Info(ctx, "Please complete the login in the browser:")
	logger.Info(ctx, "")
	logger.Info(ctx, "1. Click the 'Войти' (Login) button in the top right corner")
	logger.Info(ctx, "")
	logger.Info(ctx, "   If a promo overlay blocks the login form, close extra popups and continue login manually")
	logger.Info(ctx, "")
	logger.Info(ctx, "2. Enter your FULL phone number (e.g., +71488251742)")
	logger.Info(ctx, "   NOTE: Include country code and all digits")
	logger.Info(ctx, "")
	logger.Info(ctx, "3. Click 'Войти' button to request SMS code")
	logger.Info(ctx, "")
	logger.Info(ctx, "4. Enter the 5-digit SMS code you receive")
	logger.Info(ctx, "")
	logger.Info(ctx, "5. Wait for OAuth flow to complete (10-30 seconds)")
	logger.Info(ctx, "   You'll see Sber and Zvuk logos connecting")
	logger.Info(ctx, "")
	logger.Info(ctx, "6. DO NOT CLOSE THE BROWSER - let it complete automatically")
	logger.Info(ctx, "")
	logger.Info(ctx, "CRITICAL RULES:")
	logger.Info(ctx, "- ONLY interact with login forms")
	logger.Info(ctx, "- Do NOT close browser manually")
	logger.Info(ctx, "- Do NOT navigate away from Zvuk/Sber domains")
	logger.Info(ctx, "- Tool will auto-detect when login completes")
	logger.Info(ctx, "")
	logger.Info(ctx, "Waiting for login to complete...")
	logger.Info(ctx, "")

	// Wait for login by monitoring the process.
	token, err := s.waitForLoginComplete(ctx)
	if err != nil {
		return "", err
	}

	logger.Info(ctx, "Login completed successfully!")

	// Give the session a moment to fully establish.
	time.Sleep(sessionEstablishDelay)

	return token, nil
}

// waitForLoginComplete monitors login process and validates success by checking for avatar button.
//
//nolint:gocognit,cyclop,nestif,funlen // OAuth flow requires complex monitoring and conditional logic.
func (s *ServiceImpl) waitForLoginComplete(ctx context.Context) (string, error) {
	var (
		lastURL string
		// Track if we've entered Sber ID OAuth flow.
		inSberOAuth bool
		// Log the auth-cookie detection hint only once.
		authCookieDetected bool
		token              string
	)

	loginCtx, cancelLogin := context.WithTimeout(ctx, maxLoginWaitTime)
	defer cancelLogin()

	engine := s.loginPollingEngine
	if engine == nil {
		var engineErr error

		engine, engineErr = newLoginPollingEngine()
		if engineErr != nil {
			return "", fmt.Errorf("failed to initialize login polling retry engine: %w", engineErr)
		}
	}

	runErr := engine.Run(loginCtx, &retry.Request{
		Operation: func(ctx context.Context) error {
			// Check if browser was closed.
			if !s.isBrowserAlive(ctx) {
				return ErrBrowserClosed
			}

			// Get current URL safely.
			currentURL, err := s.getCurrentURL(ctx)
			if err != nil {
				return fmt.Errorf("failed to get current URL: %w", err)
			}

			// Log URL changes for debugging.
			if currentURL != lastURL {
				s.logURLChange(ctx, currentURL)
				lastURL = currentURL
			}

			// Track OAuth flow entry.
			if strings.Contains(currentURL, sberIDDomain) && !inSberOAuth {
				logger.Info(ctx, "Sber ID OAuth flow started")
				logger.Info(ctx, "Waiting for OAuth completion... (this may take 10-30 seconds)")

				inSberOAuth = true
			}

			// If we're in OAuth flow and still on Sber domain, check if OAuth completed.
			// The callback page has CORS issues, so we'll manually navigate to zvuk.com.
			if inSberOAuth && strings.Contains(currentURL, sberIDDomain) {
				// Check if we're on the OAuth callback/completion page.
				// This page typically shows "connecting" animation between Sber and Zvuk logos.
				// The URL contains "authorize" and has authOperationId parameter.
				if strings.Contains(currentURL, "/oauth/authorize") &&
					strings.Contains(currentURL, "authOperationId") {
					// Check page content for completion indicators.
					// The callback page loads but gets stuck due to CORS errors.
					// We detect this and manually complete the redirect.
					pageHTML, evalErr := s.page.HTML()
					if evalErr == nil {
						// If page has loaded (has HTML content), assume OAuth succeeded.
						// The CORS errors prevent automatic redirect, so we do it manually.
						if len(pageHTML) > minPageHTMLLength {
							logger.Info(ctx, "OAuth callback page detected (Sber auth completed)")
							logger.Info(ctx, "Manually redirecting to Zvuk to complete login...")
							logger.Info(ctx, "(Bypassing broken automatic redirect)")

							// Wait a moment for cookies to be set.
							time.Sleep(oauthCookieWaitDelay)

							// Manually navigate to Zvuk.
							if err = s.page.Navigate("https://zvuk.com/"); err != nil {
								return fmt.Errorf("failed to navigate to zvuk after oauth callback: %w", err)
							}

							// Wait for page to load.
							time.Sleep(oauthPageLoadDelay)

							if authCookie := s.getAuthCookie(ctx); authCookie == "" {
								logger.Debug(ctx, "No auth cookie yet, continuing to wait...")
							}
						}
					}
				}
			}

			// If we're in OAuth flow and back on Zvuk domains, auth cookie can appear before
			// login is fully completed. Do not finish early: wait for UI confirmation.
			if inSberOAuth && (strings.HasPrefix(currentURL, "https://id.zvuk.com") ||
				strings.HasPrefix(currentURL, "https://zvuk.com")) {
				if authCookie := s.getAuthCookie(ctx); authCookie != "" {
					if !authCookieDetected {
						logger.Info(ctx, "Auth cookie detected. Waiting for final login confirmation on Zvuk page...")

						authCookieDetected = true
					}
				}
			}

			// Check if login is complete on Zvuk pages.
			if strings.HasPrefix(currentURL, "https://zvuk.com") {
				if loggedIn, checkErr := s.checkIfLoggedIn(ctx); checkErr == nil && loggedIn {
					if authCookie := s.getAuthCookie(ctx); authCookie != "" {
						token = authCookie
						return nil
					}

					token = ""

					return nil
				}
			}

			// Validate user hasn't navigated away.
			if err = s.validateLoginURL(currentURL); err != nil {
				return err
			}

			// Simulate human behavior to avoid bot detection.
			s.simulateHumanBehavior(ctx)

			// Occasionally add extra random interactions.
			//nolint:gosec // Weak random is fine for simulating human behavior.
			if rand.IntN(interactionProbability) == 0 {
				s.simulateRandomPageInteraction(ctx)
			}

			return errLoginPending
		},
	})
	if runErr == nil {
		return token, nil
	}

	if errors.Is(runErr, context.DeadlineExceeded) {
		return "", fmt.Errorf("%w: waited for %v", ErrLoginTimeout, maxLoginWaitTime)
	}

	if errors.Is(runErr, context.Canceled) {
		return "", ctx.Err()
	}

	return "", runErr
}

// logURLChange logs URL changes and page details in debug mode.
func (s *ServiceImpl) logURLChange(ctx context.Context, currentURL string) {
	logger.Debugf(ctx, "URL changed: %s", currentURL)

	if !logger.IsDebugLevel() {
		return
	}

	// Show page title.
	pageInfo, err := s.page.Info()
	if err == nil {
		logger.Debugf(ctx, "Page title: %s", pageInfo.Title)
	}

	// Get full page HTML.
	html, err := s.page.HTML()
	if err == nil {
		logger.Debugf(ctx, "Page HTML (full):\n%s", html)
	}
}

// checkIfLoggedIn checks if the user is logged in by looking for the avatar button.
func (s *ServiceImpl) checkIfLoggedIn(ctx context.Context) (bool, error) {
	logger.Debug(ctx, "On zvuk.com - checking for successful login...")

	// Try to find the avatar button (appears only when logged in).
	avatarExists, _, err := s.page.Has(avatarButtonSelector)
	if err == nil && avatarExists {
		logger.Debug(ctx, "Avatar button found - login successful!")
		return true, nil
	}

	// Also check if "Войти" button still exists (not logged in).
	loginButtonExists, _, err := s.page.Has(loginButtonSelector)
	if err == nil && loginButtonExists {
		logger.Debug(ctx, "Still see 'Войти' button - not logged in yet, waiting...")
	}

	return false, err
}

// validateLoginURL validates that the user hasn't navigated away from allowed domains.
func (s *ServiceImpl) validateLoginURL(currentURL string) error {
	parsedURL, err := url.Parse(currentURL)
	if err == nil && (s.isHostOrSubdomain(parsedURL.Hostname(), zvukDomain) ||
		s.isHostOrSubdomain(parsedURL.Hostname(), sberIDDomain)) {
		return nil
	}

	return fmt.Errorf("%w to: %s", ErrNavigatedAway, currentURL)
}

// isHostOrSubdomain checks if the host is the same as the domain
// or a subdomain of the domain.
func (s *ServiceImpl) isHostOrSubdomain(host, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}
