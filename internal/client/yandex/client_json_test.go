package yandex

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseResponse_InvalidJSONReturnsError verifies malformed API JSON fails without relying on stdlib wording.
func TestParseResponse_InvalidJSONReturnsError(t *testing.T) {
	t.Parallel()

	var destination map[string]any

	err := parseResponse([]byte("{"), &destination)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing response")

	var syntaxErr *json.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
}

// TestParseResponse_ValidJSONDecodesPayload verifies well-formed API JSON is accepted.
func TestParseResponse_ValidJSONDecodesPayload(t *testing.T) {
	t.Parallel()

	var destination map[string]string

	err := parseResponse([]byte(`{"id":"1"}`), &destination)
	require.NoError(t, err)
	assert.Equal(t, "1", destination["id"])
}
