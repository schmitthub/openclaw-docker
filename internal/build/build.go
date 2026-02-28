package build

import (
	"os/exec"
	"runtime/debug"
	"strings"
)

// Variables injected via ldflags at build time.
// Defaults are used for development builds (go run / go build without flags).
var (
	Version = "DEV"
	Date    = "" // RFC3339 timestamp (e.g. 2026-02-28T14:30:00Z), empty for dev builds
)

func init() {
	if Version == "DEV" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
	}

	if strings.EqualFold(strings.TrimSpace(Version), "DEV") || strings.TrimSpace(Version) == "" {
		Version = fallbackVersion()
	}
}

func fallbackVersion() string {
	if described := runGit("describe", "--tags", "--always", "--dirty"); described != "" {
		return described
	}

	if branch := runGit("branch", "--show-current"); branch != "" {
		return branch
	}

	return "dev"
}

func runGit(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
