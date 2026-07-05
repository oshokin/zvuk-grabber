package http

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	applogger "github.com/oshokin/zvuk-grabber/internal/logger"
)

// TestLogRequest_UsesInternalLoggerAndRequestContext verifies request logs use the context logger and stage metadata.
func TestLogRequest_UsesInternalLoggerAndRequestContext(t *testing.T) {
	t.Parallel()

	core, observedLogs := observer.New(zap.DebugLevel)
	ctx := applogger.ToContext(context.Background(), zap.New(core).Sugar())

	WriteRequestLog(
		RequestLogLevelInfo,
		&RequestLogContext{
			Ctx:       ctx,
			Stage:     "download",
			Operation: "open_stream",
		},
		"test request",
		"custom_attr", "value",
	)

	entries := observedLogs.AllUntimed()
	require.Len(t, entries, 1)
	assert.Equal(t, "test request", entries[0].Message)
	assert.Equal(t, "download", entries[0].ContextMap()["stage"])
	assert.Equal(t, "open_stream", entries[0].ContextMap()["operation"])
	assert.Equal(t, "value", entries[0].ContextMap()["custom_attr"])
}

// TestLogRequest_DoesNotRedactWithoutSensitiveAttributes verifies no implicit redaction is applied.
func TestLogRequest_DoesNotRedactWithoutSensitiveAttributes(t *testing.T) {
	t.Parallel()

	core, observedLogs := observer.New(zap.DebugLevel)
	ctx := applogger.ToContext(context.Background(), zap.New(core).Sugar())

	WriteRequestLog(
		RequestLogLevelInfo,
		&RequestLogContext{Ctx: ctx},
		"test request",
		"token", "secret-token",
		"name", "visible",
	)

	entries := observedLogs.AllUntimed()
	require.Len(t, entries, 1)
	assert.Equal(t, "secret-token", entries[0].ContextMap()["token"])
	assert.Equal(t, "visible", entries[0].ContextMap()["name"])
}

// TestLogRequest_RedactsSourceSpecificSensitiveAttributes verifies source-specific fields are redacted.
func TestLogRequest_RedactsSourceSpecificSensitiveAttributes(t *testing.T) {
	t.Parallel()

	core, observedLogs := observer.New(zap.DebugLevel)
	ctx := applogger.ToContext(context.Background(), zap.New(core).Sugar())

	WriteRequestLog(
		RequestLogLevelInfo,
		&RequestLogContext{
			Ctx:                ctx,
			SensitiveFieldKeys: []string{"api_key", "signature_hint"},
		},
		"test request",
		"api_key", "custom-secret",
		"signature_hint", "provider-signature",
		"name", "visible",
	)

	entries := observedLogs.AllUntimed()
	require.Len(t, entries, 1)
	assert.Equal(t, "***", entries[0].ContextMap()["api_key"])
	assert.Equal(t, "***", entries[0].ContextMap()["signature_hint"])
	assert.Equal(t, "visible", entries[0].ContextMap()["name"])
}

// TestSanitizeURL_DoesNotRedactWithoutSensitiveKeys verifies default URL sanitization does not guess keys.
func TestSanitizeURL_DoesNotRedactWithoutSensitiveKeys(t *testing.T) {
	t.Parallel()

	rawURL := "https://cdn.example.test/stream?" +
		"X-Amz-Signature=abcd1234" +
		"&X-Amz-Credential=AKIA%2F20260705%2Fru-central1%2Fs3%2Faws4_request" +
		"&token=very-secret" +
		"&utm_source=newsletter"

	sanitized := SanitizeURL(rawURL)
	parsed, err := url.Parse(sanitized)
	require.NoError(t, err)

	query := parsed.Query()
	assert.Equal(t, "abcd1234", query.Get("X-Amz-Signature"))
	assert.Equal(t, "AKIA/20260705/ru-central1/s3/aws4_request", query.Get("X-Amz-Credential"))
	assert.Equal(t, "very-secret", query.Get("token"))
	assert.Equal(t, "newsletter", query.Get("utm_source"))
}

// TestSanitizeURL_TruncatesLongNonSensitiveQueryValues verifies long non-sensitive values are truncated.
func TestSanitizeURL_TruncatesLongNonSensitiveQueryValues(t *testing.T) {
	t.Parallel()

	rawURL := "https://music.example.test/path?utm_content=" + "abcdefghijklmnopqrstuvwxyz" +
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"

	sanitized := SanitizeURL(rawURL)
	parsed, err := url.Parse(sanitized)
	require.NoError(t, err)

	value := parsed.Query().Get("utm_content")
	assert.Contains(t, value, "...(truncated)")
	assert.LessOrEqual(t, len(value), 110)
}

// TestSanitizeURLWithKeys_RedactsSourceSpecificQueryParams verifies source-specific query keys are redacted.
func TestSanitizeURLWithKeys_RedactsSourceSpecificQueryParams(t *testing.T) {
	t.Parallel()

	rawURL := "https://api.example.test/resource?api_key=abc123&signature_hint=xyz987&token=very-secret&utm_source=feed"
	sanitized := SanitizeURLWithKeys(
		rawURL,
		[]string{"api_key", "signature_hint", "token"},
		nil,
	)

	parsed, err := url.Parse(sanitized)
	require.NoError(t, err)

	query := parsed.Query()
	assert.Equal(t, "***", query.Get("api_key"))
	assert.Equal(t, "***", query.Get("signature_hint"))
	assert.Equal(t, "***", query.Get("token"))
	assert.Equal(t, "feed", query.Get("utm_source"))
}
