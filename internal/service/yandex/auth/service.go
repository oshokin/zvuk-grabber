//nolint:gocognit,err113,gosec,funlen // Auth flow mirrors Yandex protocol and keeps staged helpers plus fixed API constants in one file.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"

	authbrowser "github.com/oshokin/zvuk-grabber/internal/browser"
	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/retry"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// Service performs visible browser authentication for Yandex Music.
type Service struct {
	// cfg holds the application configuration.
	cfg *config.Config
	// browser is the browser instance used for login.
	browser *rod.Browser
	// page is the active browser tab for user interaction.
	page *rod.Page
	// session manages browser lifecycle and cleanup.
	session *authbrowser.Session
	// loginRetryEngine handles auth polling retries.
	loginRetryEngine *retry.Engine
}

// authSnapshot contains extracted browser page state used for auth checks.
type authSnapshot struct {
	// URL is the current page URL.
	URL string
	// Text is the normalized page text.
	Text string
	// Raw contains storage/cookies/script snippets used for token extraction.
	Raw string
}

// oauthTokenResponse represents token exchange JSON responses.
type oauthTokenResponse struct {
	// AccessToken is the OAuth bearer token from the exchange response.
	AccessToken string `json:"access_token"`
	// Error is the OAuth error code when token exchange fails.
	Error string `json:"error"`
	// ErrorDescription is the human-readable OAuth error description.
	ErrorDescription string `json:"error_description"`
}

// yandexSessionCookieInfo describes extracted cookies needed for session-based token exchange.
type yandexSessionCookieInfo struct {
	// Header is the Cookie header value built from session cookies.
	Header string
	// Present lists normalized cookie names found in the browser.
	Present []string
	// Missing lists required session cookie names not yet available.
	Missing []string
	// TotalYandex counts cookies scoped to Yandex domains.
	TotalYandex int
	// SelectedHost is the host used for Ya-Client-Host during exchange.
	SelectedHost string
}

const (
	// maxLoginWaitTime is the maximum duration to wait for user login.
	maxLoginWaitTime = 10 * time.Minute
	// pollInterval is the delay between auth state polling attempts.
	pollInterval = 2 * time.Second
	// maxPageTextLength limits page text captured for token pattern matching.
	maxPageTextLength = 20_000
	// pollDiagnosticEvery controls how often missing-cookie diagnostics are logged.
	pollDiagnosticEvery = 10
	// initialLoadTimeout limits the first page load wait after navigation.
	initialLoadTimeout = 20 * time.Second
	// authFlowChangedHint is appended when token exchange endpoints may have changed.
	authFlowChangedHint = "Yandex auth flow may have changed, please update zvuk-grabber"

	// yandexSessionTokenURL exchanges browser session cookies for a primary OAuth token.
	yandexSessionTokenURL = "https://mobileproxy.passport.yandex.net/1/bundle/oauth/token_by_sessionid"
	// yandexMusicTokenURL exchanges a primary token for a Yandex Music OAuth token.
	yandexMusicTokenURL = "https://oauth.mobile.yandex.net/1/token"
	// defaultYandexMusicLoginURL is the visible browser start page for manual Yandex Music login.
	defaultYandexMusicLoginURL = "https://music.yandex.ru/"

	// These client credentials are public Yandex mobile/client identifiers used by
	// the unofficial Yandex Music auth flow. They are not user secrets, but this
	// flow can break if Yandex changes or revokes these clients.
	// yandexSessionClientID is the public client ID for session token exchange.
	yandexSessionClientID = "c0ebe342af7d48fbbbfcf2d2eedb8f9e"
	// yandexSessionClientSecret is the public client secret for session token exchange.
	yandexSessionClientSecret = "ad0a908f0aa341a182a37ecd75bc319e"
	// yandexMusicClientID is the public client ID for music token exchange.
	yandexMusicClientID = "23cabbbdc6cd418abb4b39c32c41195d"
	// yandexMusicClientSecret is the public client secret for music token exchange.
	yandexMusicClientSecret = "53bc75238f0c4d08a118e51fe9203300"

	// yandexPassportHostHeader is the default Ya-Client-Host for session exchange.
	yandexPassportHostHeader = "passport.yandex.ru"
	// yandexAuthUserAgent is the User-Agent for OAuth token exchange requests.
	yandexAuthUserAgent = "com.yandex.mobile.auth.sdk/7.33.2.733022870"

	// oauthHeaderUserAgent is the HTTP header name for OAuth token exchange User-Agent.
	oauthHeaderUserAgent = "user-agent"
	// oauthFormClientID is the OAuth form field name for client_id.
	oauthFormClientID = "client_id"
	// oauthFormClientSecret is the OAuth form field name for client_secret.
	oauthFormClientSecret = "client_secret"

	// yandexCookieSessionID is the normalized Yandex session cookie name.
	yandexCookieSessionID = "session_id"
	// yandexCookieSessionID2 is the normalized Yandex sessionid2 cookie name.
	yandexCookieSessionID2 = "sessionid2"
	// yandexCookieSessionID2SSL is the normalized Yandex sessionid2_ssl cookie name.
	yandexCookieSessionID2SSL = "sessionid2_ssl"
	// yandexCookieSessionID2SS is the normalized Yandex sessionid2ss cookie name.
	yandexCookieSessionID2SS = "sessionid2ss"

	// sessionEndSignalTargetClosed is an error substring indicating a closed browser target.
	sessionEndSignalTargetClosed = "target closed"
	// sessionEndSignalContextCanceled is an error substring indicating context cancellation.
	sessionEndSignalContextCanceled = "context canceled"
)

var (
	// ErrLoginTimeout is returned when login takes longer than maxLoginWaitTime.
	ErrLoginTimeout = errors.New("yandex login timeout exceeded")
	// ErrTokenNotFound is returned when no OAuth token is found in browser state.
	ErrTokenNotFound = errors.New("yandex music token not found in browser session")
	// ErrBrowserSessionClosed indicates the browser/page session ended before token capture.
	ErrBrowserSessionClosed = errors.New("yandex auth browser session was closed before token detection")
	// ErrRegionUnavailable indicates Yandex Music is blocked in the current region.
	ErrRegionUnavailable = errors.New("yandex music is currently not available in your region")
	// ErrSessionCookiesNotFound indicates that browser session cookies are not yet available.
	ErrSessionCookiesNotFound = errors.New("yandex session cookies were not found in browser")
	// ErrPrimaryTokenExchangeFailed indicates that session cookies couldn't be exchanged for a primary token.
	ErrPrimaryTokenExchangeFailed = errors.New("failed to exchange browser session for primary yandex token")
	// ErrMusicTokenExchangeFailed indicates that a primary token couldn't be exchanged for a music token.
	ErrMusicTokenExchangeFailed = errors.New("failed to exchange primary token for yandex music token")
	// yandexTokenPatterns are regexes used to extract OAuth tokens from page state.
	yandexTokenPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)OAuth\s+(y0__[A-Za-z0-9_\-]+)`),
		regexp.MustCompile(
			`(?i)(?:access[_-]?token|oauth[_-]?token|music[_-]?token|yandex[_-]?music[_-]?token)[^A-Za-z0-9_\-]{0,80}(y0__[A-Za-z0-9_\-]+)`,
		),
		regexp.MustCompile(`\b(y0__[A-Za-z0-9_\-]{20,})\b`),
	}
	// regionUnavailablePhrases are page text markers for geo-blocked Yandex Music.
	regionUnavailablePhrases = []string{
		"currently not available in your region",
		"not available in your region",
		"недоступен в вашем регионе",
		"не доступна в вашем регионе",
	}
	// authSensitiveQuotedBodyPattern redacts quoted secrets in error body previews.
	authSensitiveQuotedBodyPattern = regexp.MustCompile(
		`(?i)("?(?:access_token|refresh_token|token|client_secret|sessionid|session_id|cookie)"?\s*[:=]\s*")([^"]+)(")`,
	)
	// authSensitiveBodyPattern redacts unquoted secrets in error body previews.
	authSensitiveBodyPattern = regexp.MustCompile(
		`(?i)(access_token|refresh_token|token|client_secret|sessionid|session_id|cookie)\s*[:=]\s*[^"'\s,&]+`,
	)
)

// NewAuthService creates a Yandex authentication service.
func NewAuthService(cfg *config.Config) *Service {
	loginRetryEngine, err := newLoginRetryEngine()
	if err != nil {
		loginRetryEngine = nil
	}

	return &Service{
		cfg:              cfg,
		loginRetryEngine: loginRetryEngine,
	}
}

// isAuthRetryableError reports whether a login polling error should be retried.
func isAuthRetryableError(err error) bool {
	return !errors.Is(err, ErrRegionUnavailable) && !errors.Is(err, ErrBrowserSessionClosed)
}

// newLoginRetryEngine builds the retry engine used while polling for login completion.
func newLoginRetryEngine() (*retry.Engine, error) {
	return retry.NewEngine(&retry.EngineConfig{
		MaxRetries:  0,
		DelayPolicy: retry.NewRandomRangePolicy(pollInterval, pollInterval),
		IsRetryable: isAuthRetryableError,
		Sleeper:     authbrowser.WaitOrCancel,
	})
}

// LoginAndExtractToken opens a browser, waits for login, and returns the detected token.
func (s *Service) LoginAndExtractToken(ctx context.Context) (string, error) {
	if err := s.initBrowser(ctx); err != nil {
		return "", err
	}
	defer s.cleanup(ctx)

	logger.Infof(ctx, "Opening Yandex Music login page: %s", defaultYandexMusicLoginURL)
	logger.Info(ctx, "Log in to Yandex Music in the opened browser window.")
	logger.Info(ctx, "After login, keep the Yandex Music page open until the token is detected and saved.")

	if err := s.page.Navigate(defaultYandexMusicLoginURL); err != nil {
		return "", fmt.Errorf("failed to open Yandex Music login page: %w", err)
	}

	if err := s.page.Timeout(initialLoadTimeout).WaitLoad(); err != nil {
		logger.Debugf(ctx, "Initial Yandex auth page load wait ended early: %v", err)
	}

	loginCtx, cancelLogin := context.WithTimeout(ctx, maxLoginWaitTime)
	defer cancelLogin()

	var (
		token           string
		lastSnapshotURL string
	)

	engine := s.loginRetryEngine
	if engine == nil {
		var engineErr error

		engine, engineErr = newLoginRetryEngine()
		if engineErr != nil {
			return "", fmt.Errorf("failed to initialize yandex auth retry engine: %w", engineErr)
		}
	}

	err := engine.Run(loginCtx, &retry.Request{
		Operation: func(ctx context.Context) error {
			snapshot, snapshotErr := s.captureAuthSnapshot()
			if snapshotErr != nil {
				// Respect external cancellation immediately.
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}

				if isSessionEndedError(snapshotErr) {
					return fmt.Errorf("%w: %w", ErrBrowserSessionClosed, snapshotErr)
				}

				logger.Debugf(ctx, "Failed to read Yandex auth page state: %v", snapshotErr)

				return snapshotErr
			}

			lastSnapshotURL = snapshot.URL
			if isRegionUnavailablePage(snapshot.Text) {
				if snapshot.URL != "" {
					return fmt.Errorf("%w (page: %s)", ErrRegionUnavailable, snapshot.URL)
				}

				return ErrRegionUnavailable
			}

			extractedToken, extractErr := extractToken(snapshot.Raw)
			if extractErr == nil && extractedToken != "" {
				token = extractedToken

				return nil
			}

			exchangedToken, exchangeErr := s.tryExchangeMusicTokenFromSession(ctx, snapshot.URL)
			if exchangeErr == nil && exchangedToken != "" {
				token = exchangedToken

				return nil
			}

			if exchangeErr == nil {
				exchangeErr = ErrTokenNotFound
			}

			if !errors.Is(exchangeErr, ErrSessionCookiesNotFound) {
				logger.Debugf(ctx, "Yandex session token exchange attempt failed: %v", exchangeErr)
			}

			if !errors.Is(exchangeErr, ErrTokenNotFound) {
				logger.Debugf(ctx, "Unexpected Yandex token extraction error: %v", exchangeErr)
			}

			return exchangeErr
		},
		OnRetry: func(ctx context.Context, info *retry.AttemptInfo) {
			if errors.Is(info.Err, ErrSessionCookiesNotFound) && info.Retry%uint64(pollDiagnosticEvery) == 0 {
				logger.Debugf(
					ctx,
					"Waiting for Yandex auth session cookies (attempt=%d, url=%s): %v",
					info.Retry,
					lastSnapshotURL,
					info.Err,
				)
			}
		},
	})
	if err == nil {
		logger.Info(ctx, "Yandex Music token extracted successfully")
		return token, nil
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", ctxErr
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "", fmt.Errorf("%w: %w", ErrLoginTimeout, ErrTokenNotFound)
	}

	return "", err
}

// initBrowser creates and configures a visible browser instance for login.
func (s *Service) initBrowser(ctx context.Context) error {
	session, err := authbrowser.Open(ctx, &authbrowser.Options{
		ProfileDirPrefix: "yandex-music-auth-*",
		Headless:         false,
		UseStealth:       true,
		SlowMotion:       200 * time.Millisecond,
	})
	if err != nil {
		return err
	}

	s.session = session
	s.browser = session.Browser
	s.page = session.Page

	logger.Debug(ctx, "Yandex Music auth browser initialized")

	return nil
}

// captureAuthSnapshot returns current page URL/text and auth-related browser storage data.
func (s *Service) captureAuthSnapshot() (snapshot *authSnapshot, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("yandex auth page inspection panic: %v", r)
		}
	}()

	snapshotScript := fmt.Sprintf(`() => {
		const rows = [];
		const addStorage = (name, storage) => {
			try {
				for (let i = 0; i < storage.length; i++) {
					const key = storage.key(i);
					rows.push(name + ':' + key + '=' + storage.getItem(key));
				}
			} catch (e) {}
		};
		addStorage('localStorage', window.localStorage);
		addStorage('sessionStorage', window.sessionStorage);
		try { rows.push('cookie=' + document.cookie); } catch (e) {}
		try { rows.push('url=' + window.location.href); } catch (e) {}
		try {
			let count = 0;
			document.querySelectorAll('script').forEach((script) => {
				if (count >= 20) {
					return;
				}
				const content = (script.textContent || '').trim();
				if (content.length > 0) {
					rows.push(content.slice(0, 4096));
					count++;
				}
			});
		} catch (e) {}
		const text = ((document.body && document.body.innerText) || '').slice(0, %d);
		const url = window.location && window.location.href ? window.location.href : '';
		return { raw: rows.join('\n'), text, url };
	}`, maxPageTextLength)

	data, err := s.page.Eval(snapshotScript)
	if err != nil {
		return nil, err
	}

	values := data.Value.Map()
	snapshot = &authSnapshot{
		Raw:  strings.TrimSpace(values["raw"].Str()),
		Text: strings.TrimSpace(values["text"].Str()),
		URL:  strings.TrimSpace(values["url"].Str()),
	}

	return snapshot, nil
}

// extractToken scans raw browser state and returns a Yandex Music OAuth token.
func extractToken(raw string) (string, error) {
	for _, pattern := range yandexTokenPatterns {
		matches := pattern.FindStringSubmatch(raw)
		if len(matches) >= 2 && strings.TrimSpace(matches[1]) != "" {
			return strings.TrimSpace(matches[1]), nil
		}
	}

	return "", ErrTokenNotFound
}

// tryExchangeMusicTokenFromSession exchanges current browser session cookies for a music token.
func (s *Service) tryExchangeMusicTokenFromSession(ctx context.Context, currentURL string) (string, error) {
	if s.browser == nil {
		return "", ErrSessionCookiesNotFound
	}

	cookies, err := s.browser.GetCookies()
	if err != nil {
		return "", err
	}

	cookieInfo := buildYandexSessionCookieInfo(cookies)
	if cookieInfo.Header == "" {
		return "", fmt.Errorf(
			"%w: missing=%v present=%v total_yandex_cookies=%d",
			ErrSessionCookiesNotFound,
			cookieInfo.Missing,
			cookieInfo.Present,
			cookieInfo.TotalYandex,
		)
	}

	hosts := buildYandexClientHosts(currentURL)

	var (
		primaryToken string
		lastErr      error
		lastHost     string
	)

	for _, host := range hosts {
		primaryToken, err = postOAuthTokenRequest(
			ctx,
			yandexSessionTokenURL,
			map[string]string{
				"ya-client-host":     host,
				"ya-client-cookie":   cookieInfo.Header,
				oauthHeaderUserAgent: yandexAuthUserAgent,
			},
			neturl.Values{
				oauthFormClientID:     {yandexSessionClientID},
				oauthFormClientSecret: {yandexSessionClientSecret},
			},
		)
		if err == nil {
			break
		}

		lastErr = err
		lastHost = host
	}

	if strings.TrimSpace(primaryToken) == "" {
		return "", fmt.Errorf("%w (host=%s, hosts=%v): %w; %s",
			ErrPrimaryTokenExchangeFailed,
			lastHost,
			hosts,
			lastErr,
			authFlowChangedHint,
		)
	}

	musicToken, musicErr := postOAuthTokenRequest(
		ctx,
		yandexMusicTokenURL,
		map[string]string{
			oauthHeaderUserAgent: yandexAuthUserAgent,
		},
		neturl.Values{
			"access_token":    {primaryToken},
			oauthFormClientID: {yandexMusicClientID},
			oauthFormClientSecret: {
				yandexMusicClientSecret,
			},
			"grant_type": {"x-token"},
		},
	)
	if musicErr != nil {
		return "", fmt.Errorf("%w: %w; %s", ErrMusicTokenExchangeFailed, musicErr, authFlowChangedHint)
	}

	return musicToken, nil
}

// buildYandexSessionCookieInfo extracts Yandex session cookies and prepares diagnostics.
func buildYandexSessionCookieInfo(cookies []*proto.NetworkCookie) *yandexSessionCookieInfo {
	info := new(yandexSessionCookieInfo)

	if len(cookies) == 0 {
		info.Missing = append(info.Missing,
			"Session_id/"+yandexCookieSessionID2+"/"+yandexCookieSessionID2SSL+"/"+yandexCookieSessionID2SS,
		)

		return info
	}

	// cookiePair holds a normalized Yandex session cookie name and value.
	type cookiePair struct {
		// name is the normalized cookie name.
		name string
		// value is the cookie value.
		value string
	}

	var (
		requiredNames = []string{
			yandexCookieSessionID,
			yandexCookieSessionID2,
			yandexCookieSessionID2SSL,
			yandexCookieSessionID2SS,
		}
		seen          = make(map[string]struct{}, len(cookies))
		lookup        = make(map[string]cookiePair, len(cookies))
		others        = make([]cookiePair, 0, len(cookies))
		foundRequired = make(map[string]struct{}, len(requiredNames))
	)

	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}

		name := strings.TrimSpace(cookie.Name)
		value := strings.TrimSpace(cookie.Value)
		domain := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(cookie.Domain)), ".")

		if name == "" || value == "" {
			continue
		}

		if !strings.Contains(domain, "yandex.") {
			continue
		}

		info.TotalYandex++

		normalizedName := strings.ToLower(name)
		if _, exists := seen[normalizedName]; exists {
			continue
		}

		seen[normalizedName] = struct{}{}

		pair := cookiePair{name: name, value: value}
		lookup[normalizedName] = pair
		others = append(others, pair)
		info.Present = append(info.Present, normalizedName)
	}

	parts := make([]string, 0, len(others))
	requiredFound := false

	for _, requiredName := range requiredNames {
		pair, exists := lookup[requiredName]
		if !exists {
			continue
		}

		requiredFound = true
		foundRequired[requiredName] = struct{}{}

		parts = append(parts, pair.name+"="+pair.value)
	}

	if !requiredFound {
		info.Missing = append(info.Missing, requiredNames...)
		sort.Strings(info.Present)

		return info
	}

	for _, requiredName := range requiredNames {
		if _, ok := foundRequired[requiredName]; ok {
			continue
		}

		info.Missing = append(info.Missing, requiredName)
	}

	sort.Slice(others, func(i, j int) bool {
		return strings.ToLower(others[i].name) < strings.ToLower(others[j].name)
	})

	for _, pair := range others {
		normalizedName := strings.ToLower(pair.name)
		alreadyAdded := slices.Contains(requiredNames, normalizedName)

		if alreadyAdded {
			continue
		}

		parts = append(parts, pair.name+"="+pair.value)
	}

	sort.Strings(info.Present)
	info.Header = strings.Join(parts, "; ")

	return info
}

// appendUniqueNormalized appends value to items once after trim/lowercase normalization.
func appendUniqueNormalized(items []string, seen map[string]struct{}, value string) []string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return items
	}

	if !utils.AddUnique(seen, value) {
		return items
	}

	return append(items, value)
}

// buildYandexClientHosts creates Ya-Client-Host candidates for token_by_sessionid exchange.
func buildYandexClientHosts(currentURL string) []string {
	seen := make(map[string]struct{}, 8)
	hosts := make([]string, 0, 8)
	hosts = appendUniqueNormalized(hosts, seen, yandexPassportHostHeader)

	if parsedURL, err := neturl.Parse(currentURL); err == nil {
		host := strings.ToLower(strings.TrimSpace(parsedURL.Hostname()))
		hosts = appendUniqueNormalized(hosts, seen, host)

		switch {
		case strings.HasSuffix(host, ".yandex.com"), host == "yandex.com":
			hosts = appendUniqueNormalized(hosts, seen, "passport.yandex.com")
			hosts = appendUniqueNormalized(hosts, seen, "music.yandex.com")
			hosts = appendUniqueNormalized(hosts, seen, "yandex.com")
		case strings.HasSuffix(host, ".yandex.kz"), host == "yandex.kz":
			hosts = appendUniqueNormalized(hosts, seen, "passport.yandex.kz")
			hosts = appendUniqueNormalized(hosts, seen, "music.yandex.kz")
			hosts = appendUniqueNormalized(hosts, seen, "yandex.kz")
		case strings.HasSuffix(host, ".yandex.by"), host == "yandex.by":
			hosts = appendUniqueNormalized(hosts, seen, "passport.yandex.by")
			hosts = appendUniqueNormalized(hosts, seen, "music.yandex.by")
			hosts = appendUniqueNormalized(hosts, seen, "yandex.by")
		case strings.HasSuffix(host, ".yandex.uz"), host == "yandex.uz":
			hosts = appendUniqueNormalized(hosts, seen, "passport.yandex.uz")
			hosts = appendUniqueNormalized(hosts, seen, "music.yandex.uz")
			hosts = appendUniqueNormalized(hosts, seen, "yandex.uz")
		default:
			hosts = appendUniqueNormalized(hosts, seen, "passport.yandex.ru")
			hosts = appendUniqueNormalized(hosts, seen, "music.yandex.ru")
			hosts = appendUniqueNormalized(hosts, seen, "yandex.ru")
		}
	}

	hosts = appendUniqueNormalized(hosts, seen, "passport.yandex.ru")
	hosts = appendUniqueNormalized(hosts, seen, "music.yandex.ru")
	hosts = appendUniqueNormalized(hosts, seen, "yandex.ru")

	return hosts
}

// postOAuthTokenRequest posts form data and parses an OAuth access_token response.
func postOAuthTokenRequest(
	ctx context.Context,
	endpoint string,
	extraHeaders map[string]string,
	form neturl.Values,
) (string, error) {
	requestBody := form.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(requestBody))
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	request.Header.Set("Accept", "application/json")

	for key, value := range extraHeaders {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}

		request.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 20 * time.Second}

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("status %d: %s", response.StatusCode, trimBodyForError(body))
	}

	var tokenResponse oauthTokenResponse
	if err = json.Unmarshal(body, &tokenResponse); err != nil {
		return "", fmt.Errorf("invalid token response: %w", err)
	}

	if strings.TrimSpace(tokenResponse.AccessToken) == "" {
		if tokenResponse.Error != "" || tokenResponse.ErrorDescription != "" {
			return "", fmt.Errorf("%s: %s", tokenResponse.Error, tokenResponse.ErrorDescription)
		}

		return "", errors.New("response does not contain access_token")
	}

	return strings.TrimSpace(tokenResponse.AccessToken), nil
}

// trimBodyForError returns a compact body preview for transport errors.
func trimBodyForError(body []byte) string {
	const maxLength = 400

	normalized := strings.Join(strings.Fields(string(body)), " ")
	normalized = authSensitiveQuotedBodyPattern.ReplaceAllString(normalized, `$1***$3`)

	normalized = authSensitiveBodyPattern.ReplaceAllStringFunc(normalized, func(raw string) string {
		index := strings.IndexAny(raw, ":=")
		if index == -1 {
			return "***"
		}

		return raw[:index+1] + "***"
	})
	if len(normalized) <= maxLength {
		return normalized
	}

	return normalized[:maxLength] + "..."
}

// isRegionUnavailablePage reports whether page text matches known region-block phrases.
func isRegionUnavailablePage(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}

	normalized := strings.ToLower(strings.Join(strings.Fields(text), " "))
	for _, phrase := range regionUnavailablePhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}

	return false
}

// isSessionEndedError reports whether an error indicates a closed browser/page session.
func isSessionEndedError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	closedSignals := []string{
		sessionEndSignalTargetClosed,
		"session with given id not found",
		"cannot find context",
		"websocket",
		"connection reset",
		"broken pipe",
		"eof",
		sessionEndSignalContextCanceled,
	}

	for _, signal := range closedSignals {
		if strings.Contains(message, signal) {
			return true
		}
	}

	return false
}

// cleanup closes browser resources and removes temporary profile data.
func (s *Service) cleanup(ctx context.Context) {
	if s.session == nil {
		return
	}

	s.session.Close(ctx)
	s.session = nil
	s.browser = nil
	s.page = nil
}
