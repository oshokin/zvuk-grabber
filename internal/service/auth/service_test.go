package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// TestNewService tests the NewService function.
func TestNewService(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		AuthToken: "test_token",
	}

	service, err := NewService(cfg)

	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.cfg)
	assert.Nil(t, service.browser)
	assert.Nil(t, service.page)
}

// TestValidateLoginURL tests the validateLoginURL function.
func TestValidateLoginURL(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{}

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "valid zvuk.com URL",
			url:         "https://zvuk.com/login",
			expectError: false,
		},
		{
			name:        "valid id.zvuk.com URL",
			url:         "https://id.zvuk.com/desktop",
			expectError: false,
		},
		{
			name:        "valid id.sber.ru URL",
			url:         "https://id.sber.ru/oauth",
			expectError: false,
		},
		{
			name:        "invalid URL - different domain",
			url:         "https://google.com",
			expectError: true,
		},
		{
			name:        "invalid URL - malicious site",
			url:         "https://evil.com/phishing",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := service.validateLoginURL(tt.url)

			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrNavigatedAway)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSentinelErrors tests that all sentinel errors are defined and have proper messages.
func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		err   error
		wants string
	}{
		{
			name:  "ErrLoginTimeout",
			err:   ErrLoginTimeout,
			wants: "login timeout exceeded",
		},
		{
			name:  "ErrBrowserClosed",
			err:   ErrBrowserClosed,
			wants: "browser was closed by user",
		},
		{
			name:  "ErrNavigatedAway",
			err:   ErrNavigatedAway,
			wants: "user navigated away from login flow",
		},
		{
			name:  "ErrAuthCookieNotFound",
			err:   ErrAuthCookieNotFound,
			wants: "auth cookie not found - login may have failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Error(t, tt.err)
			assert.Equal(t, tt.wants, tt.err.Error())
		})
	}
}

// TestConstants tests that all constants are properly defined.
func TestConstants(t *testing.T) {
	t.Parallel()

	// Test URL constants.
	assert.Equal(t, "https://id.zvuk.com/desktop?returnUrl=https://zvuk.com/", zvukLoginURL)
	assert.Equal(t, "zvuk.com", zvukDomain)
	assert.Equal(t, "id.zvuk.com", idZvukDomain)
	assert.Equal(t, "id.sber.ru", sberIDDomain)

	// Test cookie name.
	assert.Equal(t, "auth", authCookieName)

	// Test CSS selectors.
	assert.Equal(t, `[class^="Header_triggerWrapper"]`, avatarButtonSelector)
	assert.Equal(t, `[class^="Header_authButton"]`, loginButtonSelector)

	// Test timing constants.
	assert.Equal(t, 200, int(browserSlowMotionDelay.Milliseconds()))
	assert.Equal(t, 1, int(loginPollInterval.Seconds()))
	assert.Equal(t, 10, int(maxLoginWaitTime.Minutes()))
	assert.Equal(t, 2, int(sessionEstablishDelay.Seconds()))
}

// TestServiceImpl_Cleanup tests the cleanup function.
func TestServiceImpl_Cleanup(t *testing.T) {
	t.Parallel()

	service := &ServiceImpl{
		browser: nil, // No browser initialized.
	}

	// Should not panic even with nil browser.
	assert.NotPanics(t, func() {
		service.cleanup(context.Background())
	})
}
