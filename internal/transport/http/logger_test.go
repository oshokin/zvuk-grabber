package http

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// httpDumpRedactionWant is the expected dump and live-header state after redaction.
type httpDumpRedactionWant struct {
	// DumpContains is a substring that must appear in the dump.
	DumpContains string
	// DumpOmits is a secret that must not appear in the dump.
	DumpOmits string
	// LiveHeader is the original header value after dumping.
	LiveHeader string
}

// TestLogTransport_DumpRequestRedactsCookies verifies debug request dumps hide Cookie values.
func TestLogTransport_DumpRequestRedactsCookies(t *testing.T) {
	t.Parallel()

	transport := &LogTransport{
		next:         http.DefaultTransport,
		maxLogLength: 4096,
	}

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"https://zvuk.com/api/tiny/releases",
		http.NoBody,
	)
	require.NoError(t, err)

	req.Header.Set("Cookie", "auth=secret-token")
	req.Header.Set("User-Agent", "TestAgent/1.0")

	want := &httpDumpRedactionWant{
		DumpContains: "Cookie: ***",
		DumpOmits:    "secret-token",
		LiveHeader:   "auth=secret-token",
	}
	got := transport.dumpRequest(req)

	require.Contains(t, got, want.DumpContains)
	require.NotContains(t, got, want.DumpOmits)
	require.Equal(t, want.LiveHeader, req.Header.Get("Cookie"))
}

// TestLogTransport_DumpResponseRedactsSetCookie verifies debug response dumps hide Set-Cookie values.
func TestLogTransport_DumpResponseRedactsSetCookie(t *testing.T) {
	t.Parallel()

	transport := &LogTransport{
		next:         http.DefaultTransport,
		maxLogLength: 4096,
	}

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"https://zvuk.com/api/v2/tiny/profile",
		http.NoBody,
	)
	require.NoError(t, err)

	resp := &http.Response{
		Status:     "307 Temporary Redirect",
		StatusCode: http.StatusTemporaryRedirect,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Set-Cookie":   {"spid=secret-spid"},
			"Content-Type": {"text/html"},
		},
		Body:    io.NopCloser(strings.NewReader("")),
		Request: req,
	}

	want := &httpDumpRedactionWant{
		DumpContains: "Set-Cookie: ***",
		DumpOmits:    "secret-spid",
		LiveHeader:   "spid=secret-spid",
	}
	got := transport.dumpResponse(resp)

	require.Contains(t, got, want.DumpContains)
	require.NotContains(t, got, want.DumpOmits)
	require.Equal(t, want.LiveHeader, resp.Header.Get("Set-Cookie"))
}
