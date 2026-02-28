# Package: `internal/update`

Checks GitHub releases for newer CLI versions. Called from `main.go` after command execution.

## Flow

1. `DefaultStatePath()` — resolves cache path for `update-state.json`
2. `CheckForUpdate(ctx, statePath, currentVersion, repo)` — checks GitHub API
3. Returns `*CheckResult` with `UpdateAvailable`, `LatestVersion`, `ReleaseURL`
4. Skips check if current version is `"DEV"` or unparseable
5. Caches result for 12 hours to avoid API rate limits

## GitHub API

Fetches `https://api.github.com/repos/{repo}/releases/latest` with 2-second timeout.

## Types

- `CheckResult{CurrentVersion, LatestVersion, ReleaseURL, UpdateAvailable}`
- `state{LastChecked, LatestVersion, ReleaseURL}` — cached in `update-state.json`
