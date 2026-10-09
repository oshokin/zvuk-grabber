package http

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"slices"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/config"
	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// LogTransport is a custom http.RoundTripper that logs HTTP requests and responses.
// It wraps another http.RoundTripper and logs debug information for each request/response cycle.
type LogTransport struct {
	// next is the underlying HTTP round tripper.
	next http.RoundTripper
	// maxLogLength is the maximum length of logged request/response data.
	maxLogLength uint64
}

// Static error definitions for better error handling.
var (
	// ErrNilRequest indicates that the HTTP request is nil.
	ErrNilRequest = errors.New("request is nil")
)

// NewLogTransport creates and returns a new instance of LogTransport.
// If maxLogLength is less than or equal to 0, it defaults to config.DefaultMaxLogLength.
func NewLogTransport(next http.RoundTripper, maxLogLength uint64) http.RoundTripper {
	if maxLogLength <= 0 {
		maxLogLength = config.DefaultMaxLogLength
	}

	return &LogTransport{
		next:         next,
		maxLogLength: maxLogLength,
	}
}

// RoundTrip executes a single HTTP transaction and logs the request and response.
// It implements the http.RoundTripper interface.
func (t *LogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	// Skip logging if the logger is not at debug level.
	if !logger.IsDebugLevel() {
		return t.next.RoundTrip(req)
	}

	ctx := req.Context()

	requestDump := t.dumpRequest(req)

	// Record the start time to measure the duration of the request.
	startTime := time.Now()

	// Forward the request to the underlying RoundTripper.
	resp, err := t.next.RoundTrip(req)

	// Calculate the duration of the request.
	duration := time.Since(startTime)

	if err != nil {
		logger.Debugf(ctx, "Request failed: %s %s | Error: %v", req.Method, req.URL.String(), err)

		return nil, err
	}

	responseDump := t.dumpResponse(resp)

	logger.Debugf(ctx, "%s %s [%d] %s\nRequest: %s\nResponse: %s",
		req.Method, req.URL.Path, resp.StatusCode, duration, requestDump, responseDump)

	return resp, nil
}

// CloseIdleConnections closes idle connections on the wrapped transport.
func (t *LogTransport) CloseIdleConnections() {
	if t == nil {
		return
	}

	closeIdleRoundTripper(t.next)
}

// dumpRequest serializes an HTTP request for debug logging with secrets redacted.
func (t *LogTransport) dumpRequest(req *http.Request) string {
	cloned := req.Clone(req.Context())
	redactSensitiveHTTPHeaders(cloned.Header)

	dump, err := httputil.DumpRequest(cloned, true)
	if err != nil {
		return err.Error()
	}

	return t.truncate(dump)
}

// dumpResponse serializes an HTTP response for debug logging with secrets redacted.
func (t *LogTransport) dumpResponse(resp *http.Response) string {
	//nolint:bodyclose // Body is the live response body.
	cloned := t.cloneResponse(resp)
	if cloned == nil {
		return ""
	}

	redactSensitiveHTTPHeaders(cloned.Header)

	contentType := resp.Header.Get("Content-Type")

	dump, err := httputil.DumpResponse(cloned, utils.IsTextContentType(contentType))
	if err != nil {
		return err.Error()
	}

	return t.truncate(dump)
}

// cloneResponse returns a copy with detached Header and Trailer maps.
// Body, Request, and TLS stay shared so DumpResponse can read the live body.
func (t *LogTransport) cloneResponse(resp *http.Response) *http.Response {
	if resp == nil {
		return nil
	}

	return &http.Response{
		Status:           resp.Status,
		StatusCode:       resp.StatusCode,
		Proto:            resp.Proto,
		ProtoMajor:       resp.ProtoMajor,
		ProtoMinor:       resp.ProtoMinor,
		Header:           resp.Header.Clone(),
		Body:             resp.Body,
		ContentLength:    resp.ContentLength,
		TransferEncoding: slices.Clone(resp.TransferEncoding),
		Close:            resp.Close,
		Uncompressed:     resp.Uncompressed,
		Trailer:          resp.Trailer.Clone(),
		Request:          resp.Request,
		TLS:              resp.TLS,
	}
}

// truncate limits dump output to the configured maximum log length.
func (t *LogTransport) truncate(data []byte) string {
	if uint64(len(data)) > t.maxLogLength {
		return string(data[:t.maxLogLength]) + "... [truncated]"
	}

	return string(data)
}
