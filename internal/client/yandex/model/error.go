package model

import "strings"

// ErrorResponse represents a Yandex Music API error payload.
//
//nolint:errname // Type name follows external API payload naming.
type ErrorResponse struct {
	// InvocationInfo contains request execution diagnostics.
	InvocationInfo struct {
		// ReqID is the provider request identifier.
		ReqID string `json:"req-id"`
		// Hostname is the API host that served the request.
		Hostname string `json:"hostname"`
		// ExecDurationMillis is the server-side execution time in milliseconds.
		ExecDurationMillis int `json:"exec-duration-millis"`
	} `json:"invocationInfo"`

	// APIError contains the top-level API error block.
	APIError struct {
		// Name is the API error code.
		Name string `json:"name"`
		// Message is the human-readable API error message.
		Message string `json:"message"`
	} `json:"error"`

	// ResultError contains an alternate error block nested under result.
	ResultError struct {
		// Name is the result error code.
		Name string `json:"name"`
		// Message is the human-readable result error message.
		Message string `json:"message"`
	} `json:"result"`
}

// IsError reports whether the response contains an API error payload.
func (e *ErrorResponse) IsError() bool {
	return e.APIError.Name != "" || e.ResultError.Name != ""
}

// Error returns the best available error message from the response payload.
func (e *ErrorResponse) Error() string {
	if e.APIError.Message != "" {
		return e.APIError.Message
	}

	if e.APIError.Name != "" {
		return e.APIError.Name
	}

	if e.ResultError.Message != "" {
		return e.ResultError.Message
	}

	return e.ResultError.Name
}

// ErrorName returns the API error code from the payload.
func (e *ErrorResponse) ErrorName() string {
	if e == nil {
		return ""
	}

	if e.APIError.Name != "" {
		return e.APIError.Name
	}

	return e.ResultError.Name
}

// IsNotFound reports whether the payload is a terminal not-found / missing-lyrics error.
func (e *ErrorResponse) IsNotFound() bool {
	if e == nil {
		return false
	}

	haystack := strings.ToLower(strings.TrimSpace(e.ErrorName() + " " + e.Error()))

	return strings.Contains(haystack, "not found") || strings.Contains(haystack, "no lyrics")
}
