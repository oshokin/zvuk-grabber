//nolint:nolintlint,revive // utils is a common and acceptable package name for utility functions.
package utils

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oshokin/zvuk-grabber/internal/constants"
)

// TestSafeUint64ToInt64 tests the SafeUint64ToInt64 function.
func TestSafeUint64ToInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    uint64
		expected int64
	}{
		{
			name:     "normal value",
			input:    100,
			expected: 100,
		},
		{
			name:     "zero value",
			input:    0,
			expected: 0,
		},
		{
			name:     "max int64 value",
			input:    9223372036854775807,
			expected: 9223372036854775807,
		},
		{
			name:     "value exceeding max int64",
			input:    9223372036854775808,
			expected: 9223372036854775807,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := SafeUint64ToInt64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSanitizeFilename tests the SanitizeFilename function.
func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "valid filename",
			input:    "test_file.txt",
			expected: "test_file.txt",
		},
		{
			name:     "invalid characters",
			input:    "test<file>.txt",
			expected: "test_file_.txt",
		},
		{
			name:     "Windows reserved name",
			input:    "CON",
			expected: "_CON",
		},
		{
			name:     "trailing dots",
			input:    "test...",
			expected: "test",
		},
		{
			name:     "only dots",
			input:    "...",
			expected: "_",
		},
		{
			name:     "control characters",
			input:    "test\x00file",
			expected: "test_file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRandomPause tests the RandomPause function.
func TestRandomPause(t *testing.T) {
	t.Parallel()

	// Test that RandomPause doesn't panic and returns within reasonable time.
	start := time.Now()
	RandomPause(100*time.Millisecond, 150*time.Millisecond)
	duration := time.Since(start)

	// Should pause for at least 100ms but not more than 200ms (allowing some overhead).
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond)
	assert.Less(t, duration, 200*time.Millisecond)
}

// TestSetFileExtension tests the SetFileExtension function.
func TestSetFileExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		filename  string
		extension string
		replace   bool
		expected  string
	}{
		{
			name:      "add extension to file without extension",
			filename:  "testfile",
			extension: ".txt",
			replace:   false,
			expected:  "testfile.txt",
		},
		{
			name:      "add extension without dot",
			filename:  "testfile",
			extension: "txt",
			replace:   false,
			expected:  "testfile.txt",
		},
		{
			name:      "replace existing extension",
			filename:  "testfile.txt",
			extension: constants.ExtensionMP3,
			replace:   true,
			expected:  "testfile.mp3",
		},
		{
			name:      "keep existing extension when not replacing",
			filename:  "testfile.txt",
			extension: constants.ExtensionMP3,
			replace:   false,
			expected:  "testfile.txt.mp3",
		},
		{
			name:      "same extension",
			filename:  "testfile.txt",
			extension: ".txt",
			replace:   true,
			expected:  "testfile.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := SetFileExtension(tt.filename, tt.extension, tt.replace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsFileExist tests the IsFileExist function.
func TestIsFileExist(t *testing.T) {
	t.Parallel()

	// Create a temporary file.
	tempFile, err := os.CreateTemp(t.TempDir(), "test_file")
	require.NoError(t, err)

	tempFile.Close()                 //nolint:errcheck,gosec // Test cleanup, error is not critical.
	defer os.Remove(tempFile.Name()) //nolint:errcheck // Test cleanup, error is not critical.

	// Test existing file.
	exists, err := IsFileExist(tempFile.Name())
	require.NoError(t, err)
	assert.True(t, exists)

	// Test non-existing file.
	exists, err = IsFileExist("/non/existing/file")
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestReadUniqueLinesFromFile tests the ReadUniqueLinesFromFile function.
func TestReadUniqueLinesFromFile(t *testing.T) {
	t.Parallel()

	// Create a temporary file with test content.
	tempFile, err := os.CreateTemp(t.TempDir(), "test_lines")
	require.NoError(t, err)

	defer os.Remove(tempFile.Name()) //nolint:errcheck // Test cleanup, error is not critical.

	content := "line1\nline2\nline1\nline3\nline2\n"
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	tempFile.Close() //nolint:errcheck,gosec // Test cleanup, error is not critical.

	// Test reading unique lines.
	lines, err := ReadUniqueLinesFromFile(tempFile.Name())
	require.NoError(t, err)
	assert.Len(t, lines, 3)
	assert.Contains(t, lines, "line1")
	assert.Contains(t, lines, "line2")
	assert.Contains(t, lines, "line3")

	// Test non-existing file.
	_, err = ReadUniqueLinesFromFile("/non/existing/file")
	require.Error(t, err)
}

// TestExtractNamedGroup tests the ExtractNamedGroup function.
func TestExtractNamedGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		regex     *regexp.Regexp
		groupName string
		input     string
		expected  string
	}{
		{
			name:      "valid match",
			regex:     regexp.MustCompile(`(?P<id>\d+)`),
			groupName: "id",
			input:     "test123",
			expected:  "123",
		},
		{
			name:      "no match",
			regex:     regexp.MustCompile(`(?P<id>\d+)`),
			groupName: "id",
			input:     "test",
			expected:  "",
		},
		{
			name:      "valid match with name group",
			regex:     regexp.MustCompile(`(?P<name>\w+)`),
			groupName: "name",
			input:     "test",
			expected:  "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ExtractNamedGroup(tt.regex, tt.groupName, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsTextContentType tests the IsTextContentType function.
func TestIsTextContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "text/plain",
			contentType: "text/plain",
			expected:    true,
		},
		{
			name:        "text/html with charset",
			contentType: "text/html; charset=utf-8",
			expected:    true,
		},
		{
			name:        "application/json",
			contentType: "application/json",
			expected:    true,
		},
		{
			name:        "application/samlmetadata+xml",
			contentType: "application/samlmetadata+xml",
			expected:    true,
		},
		{
			name:        "image/jpeg",
			contentType: "image/jpeg",
			expected:    false,
		},
		{
			name:        "text with invalid charset",
			contentType: "text/plain; charset=invalid",
			expected:    false,
		},
		{
			name:        "invalid content type",
			contentType: "invalid/type",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsTextContentType(tt.contentType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMap tests the Map function.
func TestMap(t *testing.T) {
	t.Parallel()

	// Test with string slice.
	input := []string{"hello", "world"}
	result := Map(input, strings.ToUpper)
	expected := []string{"HELLO", "WORLD"}
	assert.Equal(t, expected, result)

	// Test with empty slice.
	empty := []string{}
	result = Map(empty, strings.ToUpper)
	assert.Empty(t, result)
}

// TestMapIterator tests the MapIterator function.
func TestMapIterator(t *testing.T) {
	t.Parallel()

	// Create a simple iter.Seq from a slice using a generator function.
	input := []string{"hello", "world"}

	// Convert slice to iter.Seq using a generator.
	seq := func(yield func(string) bool) {
		for _, s := range input {
			if !yield(s) {
				return
			}
		}
	}

	result := MapIterator(seq, strings.ToUpper)
	expected := []string{"HELLO", "WORLD"}
	assert.Equal(t, expected, result)

	// Test with empty slice.
	emptySeq := func(_ func(string) bool) {
		// No elements to yield.
	}
	emptyResult := MapIterator(emptySeq, strings.ToUpper)
	assert.Empty(t, emptyResult)
}

// TestConstants tests the constants.
func TestConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "image/jpeg", ImageJPEGMimeType)
	assert.Equal(t, "image/png", ImagePNGMimeType)
}
