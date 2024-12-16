package version

// Variables set during build time
var (
	Version   = "unknown" // Set via -ldflags during build
	GitCommit = "unknown" // Set via -ldflags during build
	BuildTime = "unknown" // Set via -ldflags during build
)
