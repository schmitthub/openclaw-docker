package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"
)

const checkInterval = 12 * time.Hour

type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	ReleaseURL      string
	UpdateAvailable bool
}

type state struct {
	LastChecked   time.Time `json:"last_checked"`
	LatestVersion string    `json:"latest_version"`
	ReleaseURL    string    `json:"release_url"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func DefaultStatePath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(cacheDir) == "" {
		cacheDir = ".cache"
	}

	targetDir := filepath.Join(cacheDir, "openclaw-docker")
	if mkErr := os.MkdirAll(targetDir, 0o755); mkErr != nil {
		return "", fmt.Errorf("create update cache directory: %w", mkErr)
	}

	return filepath.Join(targetDir, "update-state.json"), nil
}

func CheckForUpdate(ctx context.Context, statePath, currentVersion, repo string) (*CheckResult, error) {
	currentVersion = normalizeVersion(currentVersion)
	if currentVersion == "" || strings.EqualFold(currentVersion, "dev") {
		return nil, nil
	}

	current, err := semver.NewVersion(currentVersion)
	if err != nil {
		return nil, nil
	}

	cached := readState(statePath)
	if cached != nil && time.Since(cached.LastChecked) < checkInterval {
		result := cachedResult(current, cached)
		if result != nil {
			return result, nil
		}
	}

	release, err := fetchLatestRelease(ctx, repo)
	if err != nil {
		if cached != nil {
			return cachedResult(current, cached), nil
		}
		return nil, err
	}

	normalizedLatest := normalizeVersion(release.TagName)
	nextState := state{
		LastChecked:   time.Now().UTC(),
		LatestVersion: normalizedLatest,
		ReleaseURL:    release.HTMLURL,
	}
	_ = writeState(statePath, nextState)

	latest, err := semver.NewVersion(normalizedLatest)
	if err != nil {
		return nil, err
	}
	if !latest.GreaterThan(current) {
		return nil, nil
	}

	return &CheckResult{
		CurrentVersion:  current.Original(),
		LatestVersion:   latest.Original(),
		ReleaseURL:      release.HTMLURL,
		UpdateAvailable: true,
	}, nil
}

func fetchLatestRelease(ctx context.Context, repo string) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", strings.TrimSpace(repo))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "openclaw-docker-update-check")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github releases api returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	if strings.TrimSpace(release.TagName) == "" {
		return nil, fmt.Errorf("github release tag is empty")
	}

	return &release, nil
}

func readState(path string) *state {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var s state
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil
	}

	return &s
}

func writeState(path string, s state) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func cachedResult(current *semver.Version, cached *state) *CheckResult {
	latestVersion := normalizeVersion(cached.LatestVersion)
	latest, err := semver.NewVersion(latestVersion)
	if err != nil {
		return nil
	}
	if !latest.GreaterThan(current) {
		return nil
	}

	return &CheckResult{
		CurrentVersion:  current.Original(),
		LatestVersion:   latest.Original(),
		ReleaseURL:      cached.ReleaseURL,
		UpdateAvailable: true,
	}
}

func normalizeVersion(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(raw), "v")
}
