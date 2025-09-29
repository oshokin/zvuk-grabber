package version

import "fmt"

var (
	// Version is the semantic version of the build. It can be overridden via ldflags.
	Version = "1.0.0"
	// Commit is the short git SHA embedded at build time (or "none").
	Commit = "none"
	// BuildTime is the UTC build timestamp embedded at build time.
	BuildTime = "unknown"
)

// Short returns only the semantic version string.
func Short() string {
	return Version
}

// Full returns a human-readable version string with commit and build time.
func Full() string {
	return fmt.Sprintf("version: %s, commit: %s, built at: %s", Version, Commit, BuildTime)
}
