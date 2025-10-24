package main

import "fmt"

// Version information - set at build time via -ldflags
var (
	// Version is the semantic version string (e.g., "1.0.0")
	// Can be overridden at build time with: -ldflags "-X main.Version=x.y.z"
	Version = "dev"

	// BuildDate is the date/time when the binary was built
	// Can be set at build time with: -ldflags "-X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	BuildDate = "unknown"

	// GitCommit is the git commit hash
	// Can be set at build time with: -ldflags "-X main.GitCommit=$(git rev-parse --short HEAD)"
	GitCommit = "unknown"
)

// VersionString returns a formatted version string with build information
func VersionString() string {
	return fmt.Sprintf("BEM Server %s (built %s, commit %s)", Version, BuildDate, GitCommit)
}

// ShortVersion returns just the version number
func ShortVersion() string {
	return Version
}
