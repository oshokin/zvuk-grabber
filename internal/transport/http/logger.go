package http

import (
	"errors"
	"net/http"
	"net/http/httputil"
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

func (t *LogTransport) dumpRequest(req *http.Request) string {
	// Include the request body in the dump.
	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		return err.Error()
	}

	return t.truncate(dump)
}

func (t *LogTransport) dumpResponse(resp *http.Response) string {
	// Check the Content-Type header to determine if the response body should be dumped.
	contentType := resp.Header.Get("Content-Type")

	dump, err := httputil.DumpResponse(resp, utils.IsTextContentType(contentType))
	if err != nil {
		return err.Error()
	}

	return t.truncate(dump)
}

func (t *LogTransport) truncate(data []byte) string {
	if uint64(len(data)) > t.maxLogLength {
		return string(data[:t.maxLogLength]) + "... [truncated]"
	}

	return string(data)
}
