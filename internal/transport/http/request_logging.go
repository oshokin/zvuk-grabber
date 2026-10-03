package http

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// RequestLogLevel controls severity for transport request logging.
type RequestLogLevel uint8

// RequestLogContext carries optional structured attributes and context.
type RequestLogContext struct {
	// Ctx is the request-scoped context for log emission.
	Ctx context.Context
	// Stage identifies the high-level workflow stage.
	Stage string
	// Operation identifies the specific request operation.
	Operation string
	// SensitiveFieldKeys adds source-specific structured fields to redact.
	SensitiveFieldKeys []string
	// SensitiveQueryKeys adds source-specific URL query params to redact.
	SensitiveQueryKeys []string
}

const (
	// RequestLogLevelDebug logs at debug severity.
	RequestLogLevelDebug RequestLogLevel = iota
	// RequestLogLevelInfo logs at info severity.
	RequestLogLevelInfo
	// RequestLogLevelWarn logs at warn severity.
	RequestLogLevelWarn
	// RequestLogLevelError logs at error severity.
	RequestLogLevelError
)

const (
	// redactedValue replaces sensitive values in logs.
	redactedValue = "***"
	// headerAuthorization is the Authorization header name.
	headerAuthorization = "authorization"
	// headerCookie is the Cookie header name.
	headerCookie = "cookie"
	// headerSetCookie is the Set-Cookie header name.
	headerSetCookie = "set-cookie"
)

var (
	// sensitiveQuotedValuePattern matches quoted token-like values in text.
	sensitiveQuotedValuePattern = regexp.MustCompile(
		`(?i)("?(?:access_token|refresh_token|token|client_secret|passport|sessionid|session_id)"?\s*[:=]\s*")([^"]+)(")`,
	)
	// sensitiveValuePattern matches unquoted token-like key/value pairs in text.
	sensitiveValuePattern = regexp.MustCompile(
		`(?i)(access_token|refresh_token|token|client_secret|passport|sessionid|session_id)\s*[:=]\s*[^"\s,&]+`,
	)
)

// Context returns the request context, or context.Background when unset.
func (r *RequestLogContext) Context() context.Context {
	if r == nil || r.Ctx == nil {
		return context.Background()
	}

	return r.Ctx
}

// ContextOr returns the request context, or fallback when the request context is unset.
func (r *RequestLogContext) ContextOr(fallback context.Context) context.Context {
	if r != nil && r.Ctx != nil {
		return r.Ctx
	}

	if fallback != nil {
		return fallback
	}

	return context.Background()
}

// Err reports cancellation or deadline from the request context.
func (r *RequestLogContext) Err() error {
	if r == nil || r.Ctx == nil {
		return nil
	}

	return r.Ctx.Err()
}

// WriteRequestLog writes request logs with standard structured attributes.
func WriteRequestLog(level RequestLogLevel, reqCtx *RequestLogContext, msg string, args ...any) {
	ctx := reqCtx.Context()
	attrs := requestLogAttrs(reqCtx, args...)

	switch level {
	case RequestLogLevelDebug:
		logger.DebugKV(ctx, msg, attrs...)
	case RequestLogLevelInfo:
		logger.InfoKV(ctx, msg, attrs...)
	case RequestLogLevelWarn:
		logger.WarnKV(ctx, msg, attrs...)
	case RequestLogLevelError:
		logger.ErrorKV(ctx, msg, attrs...)
	}
}

// SanitizeHeaders redacts sensitive HTTP headers for safe logging.
func SanitizeHeaders(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}

	sanitized := make(map[string]string, len(headers))
	for key, values := range headers {
		if isSensitiveHeaderKey(key) {
			sanitized[key] = redactedValue
			continue
		}

		sanitized[key] = strings.Join(values, ", ")
	}

	return sanitized
}

// SanitizeURL redacts sensitive query parameters and strips fragments.
func SanitizeURL(rawURL string) string {
	return SanitizeURLWithKeys(rawURL, nil, nil)
}

// SanitizeURLWithKeys redacts URL query parameters using source-specific key lists.
func SanitizeURLWithKeys(rawURL string, sensitiveQueryKeys, sensitiveFieldKeys []string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return rawURL
	}

	parsed.Fragment = ""

	query := parsed.Query()
	if len(query) > 0 {
		sensitiveQueryKeySet := normalizeSensitiveKeySet(sensitiveQueryKeys)
		sensitiveFieldKeySet := normalizeSensitiveKeySet(sensitiveFieldKeys)

		sanitizedQuery := url.Values{}

		for key, values := range query {
			if isSensitiveQueryKeyWithCustom(key, sensitiveQueryKeySet, sensitiveFieldKeySet) {
				sanitizedQuery.Set(key, redactedValue)
				continue
			}

			for _, value := range values {
				if len(value) > 96 {
					value = value[:96] + "...(truncated)"
				}

				sanitizedQuery.Add(key, value)
			}
		}

		parsed.RawQuery = sanitizedQuery.Encode()
	}

	return parsed.String()
}

// SanitizeTextSecrets redacts token-like values in free-form text.
func SanitizeTextSecrets(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	value = sensitiveQuotedValuePattern.ReplaceAllString(value, `$1`+redactedValue+`$3`)

	return sensitiveValuePattern.ReplaceAllStringFunc(value, func(raw string) string {
		index := strings.IndexAny(raw, ":=")
		if index == -1 {
			return redactedValue
		}

		return raw[:index+1] + redactedValue
	})
}

// FormatByteSize formats bytes for logs and returns "unknown" for negatives.
func FormatByteSize(size int64) string {
	if size < 0 {
		return "unknown"
	}

	return humanize.IBytes(uint64(size))
}

// requestSensitiveFieldKeys returns source-specific structured fields that must be redacted.
func requestSensitiveFieldKeys(reqCtx *RequestLogContext) []string {
	if reqCtx == nil {
		return nil
	}

	return reqCtx.SensitiveFieldKeys
}

// requestSensitiveQueryKeys returns source-specific query params that must be redacted.
func requestSensitiveQueryKeys(reqCtx *RequestLogContext) []string {
	if reqCtx == nil {
		return nil
	}

	return reqCtx.SensitiveQueryKeys
}

// requestLogAttrs builds structured log attributes from context and caller args.
func requestLogAttrs(reqCtx *RequestLogContext, args ...any) []any {
	attrs := make([]any, 0, len(args)+10)

	if reqCtx != nil {
		if reqCtx.Stage != "" {
			attrs = append(attrs, "stage", reqCtx.Stage)
		}

		if reqCtx.Operation != "" {
			attrs = append(attrs, "operation", reqCtx.Operation)
		}
	}

	attrs = append(attrs, args...)

	return sanitizeArgsWithKeys(attrs, requestSensitiveFieldKeys(reqCtx))
}

// sanitizeArgsWithKeys redacts sensitive values in structured log attributes using source-specific keys.
func sanitizeArgsWithKeys(args []any, sensitiveFieldKeys []string) []any {
	if len(args) == 0 {
		return nil
	}

	sensitiveFieldKeySet := normalizeSensitiveKeySet(sensitiveFieldKeys)

	sanitized := make([]any, len(args))
	copy(sanitized, args)

	for i := 0; i+1 < len(sanitized); i += 2 {
		key, ok := sanitized[i].(string)
		if !ok || !isSensitiveFieldKeyWithCustom(key, sensitiveFieldKeySet) {
			continue
		}

		sanitized[i+1] = redactedValue
	}

	return sanitized
}

// isSensitiveHeaderKey reports whether an HTTP header should be redacted in logs.
func isSensitiveHeaderKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))

	return normalized == headerAuthorization ||
		normalized == headerCookie ||
		normalized == headerSetCookie ||
		normalized == "proxy-authorization"
}

// isSensitiveFieldKeyWithCustom reports whether a structured log field should be redacted.
func isSensitiveFieldKeyWithCustom(key string, customKeySet map[string]struct{}) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}

	_, ok := customKeySet[normalized]

	return ok
}

// isSensitiveQueryKeyWithCustom reports whether a URL query parameter should be redacted.
func isSensitiveQueryKeyWithCustom(
	key string,
	customQueryKeySet map[string]struct{},
	customFieldKeySet map[string]struct{},
) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}

	if _, ok := customQueryKeySet[normalized]; ok {
		return true
	}

	return isSensitiveFieldKeyWithCustom(normalized, customFieldKeySet)
}

// normalizeSensitiveKeySet normalizes key casing and whitespace for fast membership checks.
func normalizeSensitiveKeySet(keys []string) map[string]struct{} {
	if len(keys) == 0 {
		return nil
	}

	result := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "" {
			continue
		}

		result[normalized] = struct{}{}
	}

	return result
}
