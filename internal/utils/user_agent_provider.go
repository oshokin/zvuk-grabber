package utils

//go:generate $MOCKGEN -source=user_agent_provider.go -destination=mocks/user_agent_provider_mock.go

// UserAgentProvider is an interface that defines a method for retrieving a User-Agent string.
type UserAgentProvider interface {
	// GetUserAgent returns a User-Agent string.
	GetUserAgent() string
}

// SimpleUserAgentProvider is a basic implementation of the UserAgentProvider interface.
// It provides a static User-Agent string that is set during initialization.
type SimpleUserAgentProvider struct {
	// userAgent is the User-Agent string to return.
	userAgent string
}

// NewSimpleUserAgentProvider creates and returns a new instance of SimpleUserAgentProvider.
func NewSimpleUserAgentProvider(userAgent string) UserAgentProvider {
	return &SimpleUserAgentProvider{userAgent: userAgent}
}

// GetUserAgent returns a User-Agent string.
func (p *SimpleUserAgentProvider) GetUserAgent() string {
	return p.userAgent
}
