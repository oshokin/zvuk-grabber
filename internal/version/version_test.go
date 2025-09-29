package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestShort tests the Short function.
func TestShort(t *testing.T) {
	t.Parallel()

	result := Short()
	assert.Equal(t, Version, result)
}

// TestFull tests the Full function.
func TestFull(t *testing.T) {
	t.Parallel()

	result := Full()
	expected := "version: " + Version + ", commit: " + Commit + ", built at: " + BuildTime
	assert.Equal(t, expected, result)
}

// TestVersionVariables tests that version variables are properly initialized.
func TestVersionVariables(t *testing.T) {
	t.Parallel()

	// Test that Version is not empty.
	assert.NotEmpty(t, Version)

	// Test that Commit is not empty (should be "none" if not set).
	assert.NotEmpty(t, Commit)

	// Test that BuildTime is not empty (should be "unknown" if not set).
	assert.NotEmpty(t, BuildTime)
}

// TestVersionFormat tests that version follows semantic versioning format.
func TestVersionFormat(t *testing.T) {
	t.Parallel()

	// Basic check that version contains at least one dot.
	assert.Contains(t, Version, ".")

	// Check that version doesn't contain spaces.
	assert.NotContains(t, Version, " ")
}
