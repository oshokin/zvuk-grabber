//nolint:err113 // Table tests intentionally use inline synthetic errors to verify classifiers.
package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-rod/rod/lib/proto"
)

// TestIsRegionUnavailablePage verifies region-block page text detection.
func TestIsRegionUnavailablePage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "english region block",
			text: "Yandex Music\nis currently not available in your region",
			want: true,
		},
		{
			name: "russian region block",
			text: "Сервис Яндекс Музыка недоступен в вашем регионе",
			want: true,
		},
		{
			name: "regular music page",
			text: "Yandex Music: personal recommendations and podcasts",
			want: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := isRegionUnavailablePage(testCase.text)
			if got != testCase.want {
				t.Fatalf("isRegionUnavailablePage() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// TestIsSessionEndedError verifies browser/session-close error detection.
func TestIsSessionEndedError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "target closed",
			err:  errors.New("target closed"),
			want: true,
		},
		{
			name: "websocket disconnected",
			err:  errors.New("websocket: close 1006 (abnormal closure): unexpected EOF"),
			want: true,
		},
		{
			name: "context canceled",
			err:  errors.New("context canceled"),
			want: true,
		},
		{
			name: "non-terminal error",
			err:  errors.New("temporary script execution failure"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := isSessionEndedError(testCase.err)
			if got != testCase.want {
				t.Fatalf("isSessionEndedError() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// TestBuildYandexSessionCookieHeader verifies required-session-cookie extraction.
func TestBuildYandexSessionCookieHeader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		cookies []*proto.NetworkCookie
		want    bool
	}{
		{
			name: "has session cookies",
			cookies: []*proto.NetworkCookie{
				{Name: "Session_id", Value: "abc", Domain: ".yandex.ru"},
				{Name: "sessionid2", Value: "def", Domain: ".passport.yandex.ru"},
				{Name: "yandexuid", Value: "uid", Domain: ".yandex.ru"},
			},
			want: true,
		},
		{
			name: "missing session cookies",
			cookies: []*proto.NetworkCookie{
				{Name: "yandexuid", Value: "uid", Domain: ".yandex.ru"},
			},
			want: false,
		},
		{
			name: "non-yandex domain ignored",
			cookies: []*proto.NetworkCookie{
				{Name: "Session_id", Value: "abc", Domain: "example.com"},
			},
			want: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			header := buildYandexSessionCookieInfo(testCase.cookies).Header

			got := header != ""
			if got != testCase.want {
				t.Fatalf(
					"buildYandexSessionCookieHeader() hasHeader=%v, want %v, header=%q",
					got,
					testCase.want,
					header,
				)
			}
		})
	}
}

// TestTrimBodyForError ensures sensitive tokens are redacted in auth error previews.
func TestTrimBodyForError(t *testing.T) {
	t.Parallel()

	rawBody := []byte(
		`error=invalid_grant access_token=y0__secret_token client_secret=supersecret sessionid=session-cookie`,
	)
	preview := trimBodyForError(rawBody)

	if strings.Contains(preview, "y0__secret_token") {
		t.Fatalf("trimBodyForError() leaked access_token in preview: %q", preview)
	}

	if strings.Contains(preview, "supersecret") {
		t.Fatalf("trimBodyForError() leaked client_secret in preview: %q", preview)
	}

	if strings.Contains(preview, "session-cookie") {
		t.Fatalf("trimBodyForError() leaked sessionid in preview: %q", preview)
	}

	if !strings.Contains(preview, "access_token=***") {
		t.Fatalf("trimBodyForError() did not redact access_token: %q", preview)
	}

	if !strings.Contains(preview, "client_secret=***") {
		t.Fatalf("trimBodyForError() did not redact client_secret: %q", preview)
	}

	if !strings.Contains(preview, "sessionid=***") {
		t.Fatalf("trimBodyForError() did not redact sessionid: %q", preview)
	}
}

// TestTrimBodyForError_RedactsJSONSecrets ensures JSON token fields are redacted in auth error previews.
func TestTrimBodyForError_RedactsJSONSecrets(t *testing.T) {
	t.Parallel()

	rawBody := []byte(
		`{"access_token":"y0__secret","client_secret":"supersecret","sessionid":"session-cookie","error":"bad"}`,
	)
	preview := trimBodyForError(rawBody)

	if strings.Contains(preview, "y0__secret") {
		t.Fatalf("trimBodyForError() leaked access_token in preview: %q", preview)
	}

	if strings.Contains(preview, "supersecret") {
		t.Fatalf("trimBodyForError() leaked client_secret in preview: %q", preview)
	}

	if strings.Contains(preview, "session-cookie") {
		t.Fatalf("trimBodyForError() leaked sessionid in preview: %q", preview)
	}

	if !strings.Contains(preview, `"access_token":"***"`) {
		t.Fatalf("trimBodyForError() did not redact json access_token: %q", preview)
	}

	if !strings.Contains(preview, `"client_secret":"***"`) {
		t.Fatalf("trimBodyForError() did not redact json client_secret: %q", preview)
	}

	if !strings.Contains(preview, `"sessionid":"***"`) {
		t.Fatalf("trimBodyForError() did not redact json sessionid: %q", preview)
	}
}
