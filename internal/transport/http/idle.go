package http

import "net/http"

// closeIdler is implemented by transports that can drop idle TCP/HTTP2 connections.
type closeIdler interface {
	CloseIdleConnections()
}

// CloseIdleHTTPClient closes idle keep-alives so a finished CLI run can exit.
func CloseIdleHTTPClient(client *http.Client) {
	if client == nil {
		return
	}

	client.CloseIdleConnections()
}

// closeIdleRoundTripper closes idle connections on rt when it supports it.
func closeIdleRoundTripper(rt http.RoundTripper) {
	if closer, ok := rt.(closeIdler); ok {
		closer.CloseIdleConnections()
	}
}
