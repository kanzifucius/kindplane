package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

const (
	// GitHubReleasesURL is the API endpoint for the latest release
	GitHubReleasesURL = "https://api.github.com/repos/kanzifucius/kindplane/releases/latest"

	// CheckTimeout is the maximum time to wait for the GitHub API
	CheckTimeout = 2 * time.Second
)

// CheckResult contains the result of a version check
type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseURL      string
}

// githubRelease represents the GitHub API response for a release
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckForUpdate queries the GitHub API to check if a newer version is available
func CheckForUpdate(currentVersion string) (*CheckResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, GitHubReleasesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "kindplane-version-check")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := &CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  release.TagName,
		ReleaseURL:     release.HTMLURL,
	}

	// Compare versions using semver
	updateAvailable, err := isNewerVersion(currentVersion, release.TagName)
	if err != nil {
		// If we can't parse versions, assume no update (fail safe)
		return result, nil
	}

	result.UpdateAvailable = updateAvailable
	return result, nil
}

// isNewerVersion compares two version strings and returns true if latest is newer than current
func isNewerVersion(current, latest string) (bool, error) {
	// Normalise version strings (ensure they have 'v' prefix for semver)
	current = normaliseVersion(current)
	latest = normaliseVersion(latest)

	// Handle "dev" version - always show updates for dev builds
	if current == "dev" || current == "vdev" {
		return false, nil // Don't nag developers
	}

	currentVer, err := semver.NewVersion(current)
	if err != nil {
		return false, fmt.Errorf("failed to parse current version %q: %w", current, err)
	}

	latestVer, err := semver.NewVersion(latest)
	if err != nil {
		return false, fmt.Errorf("failed to parse latest version %q: %w", latest, err)
	}

	return latestVer.GreaterThan(currentVer), nil
}

// normaliseVersion ensures the version string is in a format semver can parse
func normaliseVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "dev" || v == "none" {
		return "dev"
	}
	// Remove 'v' prefix if present for consistent handling
	v = strings.TrimPrefix(v, "v")
	return v
}
