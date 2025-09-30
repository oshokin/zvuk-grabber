package http

import (
	"net/http"

	"github.com/oshokin/zvuk-grabber/internal/utils"
)

// UserAgentInjector is a custom http.RoundTripper that injects a User-Agent header into HTTP requests.
// It wraps another http.RoundTripper and ensures that a User-Agent header is present in every request.
type UserAgentInjector struct {
	// next is the underlying HTTP round tripper.
	next http.RoundTripper
	// userAgentProvider provides the User-Agent string to inject.
	userAgentProvider utils.UserAgentProvider
}

// userAgentHeader is the HTTP header name for User-Agent.
const userAgentHeader = "User-Agent"

// NewUserAgentInjector creates and returns a new instance of UserAgentInjector.
// It takes an underlying http.RoundTripper and a UserAgentProvider to supply the User-Agent string.
func NewUserAgentInjector(next http.RoundTripper, userAgentProvider utils.UserAgentProvider) http.RoundTripper {
	return &UserAgentInjector{
		next:              next,
		userAgentProvider: userAgentProvider,
	}
}

// RoundTrip executes a single HTTP transaction and injects a User-Agent header if it is missing.
// It implements the http.RoundTripper interface.
func (t *UserAgentInjector) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get(userAgentHeader) == "" {
		req.Header.Set(userAgentHeader, t.userAgentProvider.GetUserAgent())
	}

	return t.next.RoundTrip(req)
}
