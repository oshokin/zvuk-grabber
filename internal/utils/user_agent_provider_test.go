package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewSimpleUserAgentProvider tests the NewSimpleUserAgentProvider function.
func TestNewSimpleUserAgentProvider(t *testing.T) {
	t.Parallel()

	userAgent := "TestAgent/1.0"
	provider := NewSimpleUserAgentProvider(userAgent)

	assert.NotNil(t, provider)
	assert.Implements(t, (*UserAgentProvider)(nil), provider)
}

// TestSimpleUserAgentProvider_GetUserAgent tests the GetUserAgent method.
func TestSimpleUserAgentProvider_GetUserAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userAgent string
	}{
		{
			name:      "empty user agent",
			userAgent: "",
		},
		{
			name:      "simple user agent",
			userAgent: "Mozilla/5.0",
		},
		{
			name:      "complex user agent",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		},
		{
			name:      "custom user agent",
			userAgent: "ZvukGrabber/1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewSimpleUserAgentProvider(tt.userAgent)
			result := provider.GetUserAgent()
			assert.Equal(t, tt.userAgent, result)
		})
	}
}

// TestSimpleUserAgentProvider_Interface tests that SimpleUserAgentProvider implements UserAgentProvider interface.
func TestSimpleUserAgentProvider_Interface(t *testing.T) {
	t.Parallel()

	provider := NewSimpleUserAgentProvider("test")

	// This should compile without issues, proving the interface is implemented correctly.
	_ = provider

	// Test that we can call the interface method successfully.
	userAgent := provider.GetUserAgent()
	assert.Equal(t, "test", userAgent)
}

// TestSimpleUserAgentProvider_MultipleInstances tests that multiple instances work independently.
func TestSimpleUserAgentProvider_MultipleInstances(t *testing.T) {
	t.Parallel()

	provider1 := NewSimpleUserAgentProvider("Agent1")
	provider2 := NewSimpleUserAgentProvider("Agent2")

	assert.Equal(t, "Agent1", provider1.GetUserAgent())
	assert.Equal(t, "Agent2", provider2.GetUserAgent())
	assert.NotEqual(t, provider1.GetUserAgent(), provider2.GetUserAgent())
}
