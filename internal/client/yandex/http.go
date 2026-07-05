package yandex

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	httptransport "github.com/oshokin/zvuk-grabber/internal/transport/http"
)

// HttpClient wraps the shared transport HTTP client for Yandex Music API calls.
type HttpClient struct {
	// transport performs authenticated API and download requests.
	transport *httptransport.Client
}

// yandexAPIErrorDecoder parses Yandex Music API error payloads.
type yandexAPIErrorDecoder struct{}

const (
	// UserAgent is the default User-Agent for Yandex Music API requests.
	UserAgent = "Yandex-Music-API"
	// yandexMusicClientHeaderValue identifies Android client family for signed endpoints.
	yandexMusicClientHeaderValue = "YandexMusicAndroid/24023621"
	// DefaultRequestTimeout is the default timeout for Yandex API requests.
	DefaultRequestTimeout = 30 * time.Second
	// providerNameYandex identifies Yandex in transport request logs.
	providerNameYandex = "yandex"
	// headerUserAgent is the HTTP User-Agent header name.
	headerUserAgent = "User-Agent"
	// headerYandexMusicClient is the Yandex API client identity header.
	headerYandexMusicClient = "X-Yandex-Music-Client"
)

// errHTTPClientNotConfigured is returned when HttpClient transport is nil.
var errHTTPClientNotConfigured = errors.New("yandex http client is not configured")

// NewHttpClient creates a Yandex Music HTTP client with default settings.
func NewHttpClient() *HttpClient {
	return &HttpClient{
		transport: httptransport.NewClient(&httptransport.ClientOptions{
			HTTPClient: &http.Client{},
			DefaultHeaders: map[string]string{
				headerUserAgent:         UserAgent,
				headerYandexMusicClient: yandexMusicClientHeaderValue,
				"Content-Type":          "application/json",
			},
			ProviderName:    providerNameYandex,
			RequestTimeout:  DefaultRequestTimeout,
			DownloadTimeout: 0,
			APIErrorDecoder: new(yandexAPIErrorDecoder),
		}),
	}
}

// SetToken configures the OAuth bearer token for subsequent requests.
func (c *HttpClient) SetToken(token string) {
	if c == nil || c.transport == nil {
		return
	}

	c.transport.SetHeader("Authorization", "OAuth "+token)
}

// SetHeader sets a default request header on the underlying transport client.
func (c *HttpClient) SetHeader(key, value string) {
	if c == nil || c.transport == nil {
		return
	}

	c.transport.SetHeader(key, value)
}

// Cancel cancels in-flight requests on the underlying transport client.
func (c *HttpClient) Cancel() {
	if c == nil || c.transport == nil {
		return
	}

	c.transport.Cancel()
}

// ResetCancel recreates the base request context on the underlying transport client.
func (c *HttpClient) ResetCancel() {
	if c == nil || c.transport == nil {
		return
	}

	c.transport.ResetCancel()
}

// SetDownloadTimeout configures the timeout used for download requests.
func (c *HttpClient) SetDownloadTimeout(timeout time.Duration) {
	if c == nil || c.transport == nil {
		return
	}

	c.transport.SetDownloadTimeout(timeout)
}

// GetWithContext sends a GET request and returns the response body.
func (c *HttpClient) GetWithContext(reqCtx *httptransport.RequestLogContext, url string) ([]byte, error) {
	if c == nil || c.transport == nil {
		return nil, errHTTPClientNotConfigured
	}

	return c.transport.GetWithContext(reqCtx, url)
}

// GetWithContextAndHeaders sends a GET request with extra headers and returns the response body.
func (c *HttpClient) GetWithContextAndHeaders(
	reqCtx *httptransport.RequestLogContext,
	url string,
	headers map[string]string,
) ([]byte, error) {
	if c == nil || c.transport == nil {
		return nil, errHTTPClientNotConfigured
	}

	return c.transport.GetWithContextAndHeaders(reqCtx, url, headers)
}

// OpenStreamWithContext opens a remote stream for streaming downloads.
func (c *HttpClient) OpenStreamWithContext(
	reqCtx *httptransport.RequestLogContext,
	url string,
) (*httptransport.RemoteStream, error) {
	if c == nil || c.transport == nil {
		return nil, errHTTPClientNotConfigured
	}

	return c.transport.OpenStreamWithContext(reqCtx, url)
}

// DownloadBytesWithContext downloads a remote resource as bytes.
func (c *HttpClient) DownloadBytesWithContext(reqCtx *httptransport.RequestLogContext, url string) ([]byte, error) {
	if c == nil || c.transport == nil {
		return nil, errHTTPClientNotConfigured
	}

	return c.transport.DownloadBytesWithContext(reqCtx, url)
}

// responsePreview returns a redacted text preview of an HTTP response body.
func responsePreview(contentType string, body []byte) string {
	return httptransport.ResponsePreview(contentType, body)
}

// Decode parses a Yandex Music API error response body.
func (*yandexAPIErrorDecoder) Decode(body []byte) *httptransport.DecodedAPIError {
	var errorResp model.ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil || !errorResp.IsError() {
		return nil
	}

	return &httptransport.DecodedAPIError{
		Err: &errorResp,
		LogAttrs: []any{
			"api_error", errorResp.APIError.Name,
			"api_message", errorResp.APIError.Message,
		},
	}
}
