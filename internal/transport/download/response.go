package download

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// lastModifiedSafetyMargin conservatively allows for clock skew when applying RFC 9110 section 8.8.2.2.
const lastModifiedSafetyMargin = 60 * time.Second

var (
	// errInvalidResponse prevents writing unvalidated or encoded representations.
	errInvalidResponse = errors.New("invalid audio response")

	// errResumeRejected permits a bounded full restart after an unusable range response.
	errResumeRejected = errors.New("CDN continuation rejected; requesting a complete file")
)

// open retries request failures while preserving the current body-retry budget.
func (s *Stream) open(offset int64) error {
	for {
		req := s.request.Clone(s.request.Context())
		req.Header.Set("Accept-Encoding", "identity")
		req.Header.Del("If-Range")

		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
			req.Header.Set("If-Range", s.validator)
		}

		err := s.requestOnce(req, offset)
		if err == nil {
			return nil
		}

		if offset > 0 && errors.Is(err, errResumeRejected) {
			offset = 0
		}

		if retryErr := s.retry(err); retryErr != nil {
			return retryErr
		}
	}
}

// requestOnce closes rejected responses and strips signed URLs from transport errors.
func (s *Stream) requestOnce(req *http.Request, offset int64) error {
	resp, err := s.client.Do(req)
	if err != nil {
		if urlErr, ok := errors.AsType[*url.Error](err); ok {
			return urlErr.Err
		}

		return err
	}

	err = s.accept(resp, offset)
	if err == nil {
		return nil
	}

	_ = resp.Body.Close()
	if offset > 0 &&
		(errors.Is(err, errInvalidResponse) || resp.StatusCode == http.StatusRequestedRangeNotSatisfiable) {
		return fmt.Errorf("%w: %w", errResumeRejected, err)
	}

	return err
}

// accept validates the response before any of its bytes can reach the destination.
func (s *Stream) accept(resp *http.Response, offset int64) error {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return &statusError{code: resp.StatusCode, retryAfter: resp.Header.Get("Retry-After")}
	}

	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return fmt.Errorf("%w: unexpected Content-Encoding", errInvalidResponse)
	}

	total := resp.ContentLength

	validator := responseValidator(resp)
	if resp.StatusCode == http.StatusPartialContent {
		start, end, size, err := parseContentRange(resp.Header.Get("Content-Range"))
		if err != nil {
			return err
		}

		if start != offset || end != size-1 || (resp.ContentLength >= 0 && resp.ContentLength != size-start) {
			return fmt.Errorf("%w: inconsistent Content-Range", errInvalidResponse)
		}

		if offset > 0 && (size != s.total || validator != s.validator) {
			return fmt.Errorf("%w: representation changed during continuation", errInvalidResponse)
		}

		total = size
	}

	s.offset = 0
	if resp.StatusCode == http.StatusPartialContent {
		s.offset = offset
	}

	s.response = resp
	s.total = total
	s.validator = validator

	return nil
}

// responseValidator selects an If-Range validator strong enough to combine partial bodies.
func responseValidator(resp *http.Response) string {
	etag := resp.Header.Get("ETag")
	if etag != "" {
		return strongETag(etag)
	}

	modified := resp.Header.Get("Last-Modified")
	modifiedTime, modifiedErr := http.ParseTime(modified)

	date, dateErr := http.ParseTime(resp.Header.Get("Date"))
	if modifiedErr == nil && dateErr == nil && date.Sub(modifiedTime) >= lastModifiedSafetyMargin {
		return modified
	}

	return ""
}

// parseContentRange accepts a single, complete, bounded byte range without integer overflow.
func parseContentRange(value string) (int64, int64, int64, error) {
	raw, ok := strings.CutPrefix(value, "bytes ")
	span, size, slash := strings.Cut(raw, "/")
	first, last, dash := strings.Cut(span, "-")
	start, startErr := strconv.ParseInt(first, 10, 64)
	end, endErr := strconv.ParseInt(last, 10, 64)

	total, totalErr := strconv.ParseInt(size, 10, 64)
	if !ok || !slash || !dash || startErr != nil || endErr != nil || totalErr != nil || start < 0 || end < start ||
		total <= end {
		return 0, 0, 0, fmt.Errorf("%w: malformed Content-Range", errInvalidResponse)
	}

	return start, end, total, nil
}

// strongETag accepts RFC 9110 opaque tags without weak prefixes or invalid characters.
func strongETag(value string) string {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return ""
	}

	for _, char := range []byte(value[1 : len(value)-1]) {
		if char < 0x21 || char == '"' || char == 0x7f {
			return ""
		}
	}

	return value
}
