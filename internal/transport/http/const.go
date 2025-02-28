package http

import "time"

const (
	// DefaultTimeout is the default timeout duration for HTTP requests.
	DefaultTimeout = 60 * time.Second

	// DefaultUserAgent is the default User-Agent string used for HTTP requests.
	// It mimics a common browser User-Agent to avoid being blocked by servers.
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36" //nolint: lll
)
