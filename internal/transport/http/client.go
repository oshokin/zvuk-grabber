//nolint:err113 // HTTP client keeps request pipeline grouped by operation stages; timeout cancel func is returned to callers.
package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/files"
	"github.com/oshokin/zvuk-grabber/internal/transport/download"
)

// APIErrorDecoder parses provider-specific error payloads.
type APIErrorDecoder interface {
	// Decode parses a response body into a provider-specific API error.
	Decode(body []byte) *DecodedAPIError
}

// APIErrorDecoderFunc adapts a function to APIErrorDecoder.
type APIErrorDecoderFunc func(body []byte) *DecodedAPIError

// DecodedAPIError contains a parsed provider-specific API error and log attrs.
type DecodedAPIError struct {
	// Err is the provider-specific error value.
	Err error
	// LogAttrs are extra structured attributes for request logs.
	LogAttrs []any
}

// ClientOptions controls transport client behavior.
type ClientOptions struct {
	// AudioHTTPClient isolates audio timeouts and socket policy from API and cover requests.
	AudioHTTPClient *http.Client
	// AudioConfig controls the audio retry budget.
	AudioConfig *config.DownloadHTTPConfig
	// HTTPClient is the underlying net/http client.
	HTTPClient *http.Client
	// DefaultHeaders are applied to every outgoing request.
	DefaultHeaders map[string]string
	// ProviderName identifies the provider in request logs.
	ProviderName string
	// RequestTimeout limits regular API request duration.
	RequestTimeout time.Duration
	// DownloadTimeout limits download and stream request duration.
	DownloadTimeout time.Duration
	// APIErrorDecoder parses provider-specific error payloads.
	APIErrorDecoder APIErrorDecoder
}

// RemoteStream contains an opened HTTP response body and transport metadata.
type RemoteStream struct {
	// Body is the readable response stream.
	Body io.ReadCloser
	// ContentLength is the reported content length, or -1 if unknown.
	ContentLength int64
	// ContentType is the HTTP Content-Type header value.
	ContentType string
	// StatusCode is the HTTP response status code.
	StatusCode int
}

// cancelReadCloser propagates stream lifecycle to request context cancellation.
type cancelReadCloser struct {
	// ReadCloser is the underlying response body stream.
	io.ReadCloser

	// cancel releases the request context when the stream closes.
	cancel context.CancelFunc
	// once ensures cancel is invoked only once.
	once sync.Once
}

// Client provides reusable HTTP operations with request logging.
type Client struct {
	// audioHTTPClient performs dedicated audio requests.
	audioHTTPClient *http.Client
	// audioConfig controls retries for each audio URL.
	audioConfig *config.DownloadHTTPConfig
	// httpClient performs the actual HTTP round trips.
	httpClient *http.Client
	// headers are default headers applied to every request.
	headers map[string]string
	// mu protects mutable client state.
	mu sync.RWMutex
	// ctx is the base context for requests without an explicit context.
	ctx context.Context
	// cancel cancels the current base context.
	cancel context.CancelFunc
	// requestTimeout limits regular API request duration.
	requestTimeout time.Duration
	// downloadTimeout limits download and stream request duration.
	downloadTimeout time.Duration
	// providerName identifies the provider in request logs.
	providerName string
	// apiErrorDecoder parses provider-specific error payloads.
	apiErrorDecoder APIErrorDecoder
}

// Decode implements APIErrorDecoder.
func (f APIErrorDecoderFunc) Decode(body []byte) *DecodedAPIError {
	if f == nil {
		return nil
	}

	return f(body)
}

// Close closes the stream and cancels the associated request context.
func (c *cancelReadCloser) Close() error {
	err := c.ReadCloser.Close()
	c.once.Do(c.cancel)

	return err
}

// NewClient creates a transport HTTP client.
func NewClient(options *ClientOptions) *Client {
	if options == nil {
		options = new(ClientOptions)
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = new(http.Client)
	}

	requestTimeout := options.RequestTimeout
	if requestTimeout <= 0 {
		requestTimeout = DefaultTimeout
	}

	downloadTimeout := max(options.DownloadTimeout, 0)

	ctx, cancel := context.WithCancel(context.Background())

	audioClient := options.AudioHTTPClient
	if audioClient == nil {
		audioClient = httpClient
	}

	client := &Client{
		audioHTTPClient: audioClient,
		audioConfig:     options.AudioConfig,
		httpClient:      httpClient,
		headers:         make(map[string]string, len(options.DefaultHeaders)),
		ctx:             ctx,
		cancel:          cancel,
		requestTimeout:  requestTimeout,
		downloadTimeout: downloadTimeout,
		providerName:    strings.TrimSpace(options.ProviderName),
		apiErrorDecoder: options.APIErrorDecoder,
	}

	maps.Copy(client.headers, options.DefaultHeaders)

	return client
}

// CloseIdleConnections closes idle API and audio keep-alives so the process can exit.
func (c *Client) CloseIdleConnections() {
	if c == nil {
		return
	}

	CloseIdleHTTPClient(c.httpClient)
	CloseIdleHTTPClient(c.audioHTTPClient)
}

// SetHeader sets or replaces a default request header.
func (c *Client) SetHeader(key, value string) {
	if c == nil || strings.TrimSpace(key) == "" {
		return
	}

	c.mu.Lock()
	c.headers[key] = value
	c.mu.Unlock()
}

// Cancel cancels current requests derived from the internal base context.
func (c *Client) Cancel() {
	if c == nil {
		return
	}

	c.mu.RLock()

	cancel := c.cancel
	c.mu.RUnlock()

	if cancel == nil {
		return
	}

	cancel()
}

// ResetCancel creates a fresh base context for future requests.
func (c *Client) ResetCancel() {
	if c == nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	c.mu.Lock()
	c.ctx = ctx
	c.cancel = cancel
	c.mu.Unlock()
}

// SetDownloadTimeout configures timeout for download/stream endpoints.
func (c *Client) SetDownloadTimeout(timeout time.Duration) {
	if c == nil {
		return
	}

	if timeout < 0 {
		timeout = 0
	}

	c.mu.Lock()
	c.downloadTimeout = timeout
	c.mu.Unlock()
}

// Get sends a GET request without an explicit log context.
func (c *Client) Get(url string) ([]byte, error) {
	return c.GetWithContext(new(RequestLogContext), url)
}

// GetWithContext sends a GET request with structured request logging.
func (c *Client) GetWithContext(reqCtx *RequestLogContext, url string) ([]byte, error) {
	return c.sendRequest(reqCtx, http.MethodGet, url, nil)
}

// GetWithContextAndHeaders sends a GET request with extra headers and logging.
func (c *Client) GetWithContextAndHeaders(
	reqCtx *RequestLogContext,
	url string,
	headers map[string]string,
) ([]byte, error) {
	return c.sendRequestWithHeaders(reqCtx, http.MethodGet, url, nil, headers)
}

// Post sends a POST request without an explicit log context.
func (c *Client) Post(url string, data []byte) ([]byte, error) {
	return c.PostWithContext(new(RequestLogContext), url, data)
}

// PostWithContext sends a POST request with structured request logging.
func (c *Client) PostWithContext(reqCtx *RequestLogContext, url string, data []byte) ([]byte, error) {
	return c.sendRequest(reqCtx, http.MethodPost, url, data)
}

// DownloadBytesWithContext downloads a remote resource as bytes with logging.
func (c *Client) DownloadBytesWithContext(reqCtx *RequestLogContext, url string) ([]byte, error) {
	ctx, cancel := withOptionalTimeout(c.requestContext(reqCtx), c.downloadTimeoutValue())
	defer cancel()

	req, err := c.createRequest(ctx, http.MethodGet, url, nil, nil)
	if err != nil {
		c.logRequest(RequestLogLevelError, reqCtx, "Download request create failed", http.MethodGet, url,
			"error", err,
		)

		return nil, err
	}

	startedAt := time.Now()

	c.logRequest(RequestLogLevelDebug, reqCtx, "Download request started", http.MethodGet, url,
		"headers", SanitizeHeaders(req.Header),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logRequest(RequestLogLevelError, reqCtx, "Download request failed", http.MethodGet, url,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"error", err,
		)

		return nil, fmt.Errorf("failed to download bytes: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logRequest(RequestLogLevelError, reqCtx, "Download response read failed", http.MethodGet, url,
			"status_code", resp.StatusCode,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"error", err,
		)

		return nil, fmt.Errorf("failed to read download response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logRequest(RequestLogLevelError, reqCtx, "Download request finished with bad status", http.MethodGet, url,
			"status_code", resp.StatusCode,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"response_size", FormatByteSize(int64(len(body))),
			"response_preview", ResponsePreview(resp.Header.Get("Content-Type"), body),
		)

		return nil, fmt.Errorf("failed to download bytes: status code %d", resp.StatusCode)
	}

	c.logRequest(RequestLogLevelDebug, reqCtx, "Download request finished", http.MethodGet, url,
		"status_code", resp.StatusCode,
		"duration_ms", time.Since(startedAt).Milliseconds(),
		"response_size", FormatByteSize(int64(len(body))),
	)

	return body, nil
}

// OpenStreamWithContext opens a remote file stream and returns the response body.
// The caller must close RemoteStream.Body.
func (c *Client) OpenStreamWithContext(reqCtx *RequestLogContext, url string) (*RemoteStream, error) {
	ctx, cancel := withOptionalTimeout(c.requestContext(reqCtx), c.downloadTimeoutValue())

	req, err := c.createRequest(ctx, http.MethodGet, url, nil, nil)
	if err != nil {
		cancel()
		c.logRequest(RequestLogLevelError, reqCtx, "Stream request create failed", http.MethodGet, url, "error", err)

		return nil, fmt.Errorf("failed to create stream request: %w", err)
	}

	startedAt := time.Now()

	c.logRequest(RequestLogLevelDebug, reqCtx, "Audio stream request started", http.MethodGet, url,
		"headers", SanitizeHeaders(req.Header))

	stream, err := download.Open(c.audioHTTPClient, req, c.audioConfig)
	if err != nil {
		cancel()
		c.logRequest(RequestLogLevelError, reqCtx, "Audio stream request failed", http.MethodGet, url,
			"duration_ms", time.Since(startedAt).Milliseconds(), "error", err)

		return nil, err
	}

	c.logRequest(RequestLogLevelDebug, reqCtx, "Audio stream opened", http.MethodGet, url,
		"status_code", stream.StatusCode(), "duration_ms", time.Since(startedAt).Milliseconds(),
		"content_length", FormatByteSize(stream.TotalBytes()), "content_type", stream.ContentType())

	return &RemoteStream{
		Body:          &cancelReadCloser{ReadCloser: stream, cancel: cancel},
		ContentLength: stream.TotalBytes(),
		ContentType:   stream.ContentType(),
		StatusCode:    stream.StatusCode(),
	}, nil
}

// CopyTo preserves resumable transfer behavior through the context-lifetime wrapper.
func (c *cancelReadCloser) CopyTo(
	ctx context.Context,
	destination io.Writer,
	opts *files.CopyStreamOptions,
) (int64, error) {
	return download.Copy(ctx, destination, c.ReadCloser, opts)
}

// sendRequest sends an HTTP request using default headers only.
func (c *Client) sendRequest(reqCtx *RequestLogContext, method, url string, data []byte) ([]byte, error) {
	return c.sendRequestWithHeaders(reqCtx, method, url, data, nil)
}

// sendRequestWithHeaders sends an HTTP request with optional extra headers.
func (c *Client) sendRequestWithHeaders(
	reqCtx *RequestLogContext,
	method, url string,
	data []byte,
	extraHeaders map[string]string,
) ([]byte, error) {
	ctx, cancel := withOptionalTimeout(c.requestContext(reqCtx), c.requestTimeoutValue())
	defer cancel()

	req, err := c.createRequest(ctx, method, url, data, extraHeaders)
	if err != nil {
		c.logRequest(RequestLogLevelError, reqCtx, "HTTP request create failed", method, url,
			"error", err,
		)

		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	startedAt := time.Now()

	c.logRequest(RequestLogLevelDebug, reqCtx, "HTTP request started", method, url,
		"headers", SanitizeHeaders(req.Header),
		"request_size", FormatByteSize(int64(len(data))),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logRequest(RequestLogLevelError, reqCtx, "HTTP request failed", method, url,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"error", err,
		)

		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logRequest(RequestLogLevelError, reqCtx, "HTTP response read failed", method, url,
			"status_code", resp.StatusCode,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"error", err,
		)

		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	logLevel := RequestLogLevelDebug
	if resp.StatusCode >= http.StatusBadRequest {
		logLevel = RequestLogLevelError
	}

	c.logRequest(logLevel, reqCtx, "HTTP request finished", method, url,
		"status_code", resp.StatusCode,
		"duration_ms", time.Since(startedAt).Milliseconds(),
		"response_size", FormatByteSize(int64(len(body))),
		"response_preview", ResponsePreview(resp.Header.Get("Content-Type"), body),
	)

	if decodedAPIError := c.decodeAPIError(body); decodedAPIError != nil && decodedAPIError.Err != nil {
		attrs := make([]any, 0, len(decodedAPIError.LogAttrs)+2)
		if len(decodedAPIError.LogAttrs) > 0 {
			attrs = append(attrs, decodedAPIError.LogAttrs...)
		} else {
			attrs = append(attrs, "api_error", decodedAPIError.Err.Error())
		}

		c.logRequest(RequestLogLevelError, reqCtx, "API returned error payload", method, url, attrs...)

		return nil, decodedAPIError.Err
	}

	return body, nil
}

// createRequest builds an HTTP request with default and extra headers.
func (c *Client) createRequest(
	ctx context.Context,
	method, url string,
	data []byte,
	extraHeaders map[string]string,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	for key, value := range c.headersSnapshot() {
		req.Header.Set(key, value)
	}

	for key, value := range extraHeaders {
		req.Header.Set(key, value)
	}

	return req, nil
}

// decodeAPIError parses a provider-specific API error from a response body.
func (c *Client) decodeAPIError(body []byte) *DecodedAPIError {
	decoder := c.apiErrorDecoderSnapshot()
	if decoder == nil {
		return nil
	}

	return decoder.Decode(body)
}

// apiErrorDecoderSnapshot returns the configured API error decoder.
func (c *Client) apiErrorDecoderSnapshot() APIErrorDecoder {
	if c == nil {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.apiErrorDecoder
}

// headersSnapshot returns a copy of the default request headers.
func (c *Client) headersSnapshot() map[string]string {
	if c == nil {
		return map[string]string{}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	headers := make(map[string]string, len(c.headers))
	maps.Copy(headers, c.headers)

	return headers
}

// requestTimeoutValue returns the configured API request timeout.
func (c *Client) requestTimeoutValue() time.Duration {
	if c == nil {
		return DefaultTimeout
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.requestTimeout
}

// downloadTimeoutValue returns the configured download request timeout.
func (c *Client) downloadTimeoutValue() time.Duration {
	if c == nil {
		return 0
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.downloadTimeout
}

// providerNameValue returns the provider name used in request logs.
func (c *Client) providerNameValue() string {
	if c == nil {
		return ""
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.providerName
}

// baseContext returns the client's base request context.
func (c *Client) baseContext() context.Context {
	if c == nil {
		return context.Background()
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.ctx == nil {
		return context.Background()
	}

	return c.ctx
}

// ResponsePreview returns a redacted text preview for textual bodies.
func ResponsePreview(contentType string, body []byte) string {
	if len(body) == 0 {
		return ""
	}

	contentType = strings.ToLower(contentType)
	if !strings.Contains(contentType, "json") && !strings.Contains(contentType, "xml") &&
		!strings.Contains(contentType, "text") {
		return ""
	}

	const maxPreviewBytes = 512

	preview := strings.TrimSpace(string(body))
	preview = SanitizeTextSecrets(preview)

	if len(preview) <= maxPreviewBytes {
		return preview
	}

	return preview[:maxPreviewBytes] + "...(truncated)"
}

// logRequest writes a structured transport request log entry.
func (c *Client) logRequest(
	level RequestLogLevel,
	reqCtx *RequestLogContext,
	msg, method, rawURL string,
	args ...any,
) {
	attrs := make([]any, 0, len(args)+6)
	sensitiveFieldKeys := requestSensitiveFieldKeys(reqCtx)
	sensitiveQueryKeys := requestSensitiveQueryKeys(reqCtx)

	if providerName := c.providerNameValue(); providerName != "" {
		attrs = append(attrs, "provider", providerName)
	}

	attrs = append(attrs,
		"method", method,
		"url", SanitizeURLWithKeys(rawURL, sensitiveQueryKeys, sensitiveFieldKeys),
	)

	attrs = append(attrs, args...)
	WriteRequestLog(level, reqCtx, msg, attrs...)
}

// requestContext resolves the effective context for an outgoing request.
func (c *Client) requestContext(reqCtx *RequestLogContext) context.Context {
	return reqCtx.ContextOr(c.baseContext())
}

// withOptionalTimeout wraps a context with a timeout when duration is positive.
func withOptionalTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, timeout)
}
