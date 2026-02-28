// Package version provides build version information for zfsguard binaries.
// The Version variable is set at build time via ldflags:
//
//	go build -ldflags "-X github.com/pbek/zfsguard/internal/version.Version=1.2.3"
package version

import "fmt"

// Version is the application version, set at build time via ldflags.
// Falls back to "dev" if not set.
var Version = "dev"

// String returns a formatted version string for display.
func String(binary string) string {
	return fmt.Sprintf("%s version %s", binary, Version)
}
