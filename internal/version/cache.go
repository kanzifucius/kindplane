package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// CacheFileName is the name of the cache file
	CacheFileName = "version-cache.json"

	// CacheDuration is how long the cache is valid
	CacheDuration = 24 * time.Hour
)

// Cache represents the cached version check result
type Cache struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
	ReleaseURL    string    `json:"release_url"`
}

// getCacheDir returns the directory for storing the cache file
func getCacheDir() (string, error) {
	// Use XDG_CONFIG_HOME if set, otherwise ~/.config
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	cacheDir := filepath.Join(configDir, "kindplane")
	return cacheDir, nil
}

// getCachePath returns the full path to the cache file
func getCachePath() (string, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, CacheFileName), nil
}

// LoadCache loads the cached version check result from disk
func LoadCache() (*Cache, error) {
	cachePath, err := getCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// SaveCache saves the version check result to disk
func SaveCache(latestVersion, releaseURL string) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return err
	}

	// Create the cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, CacheFileName)

	cache := Cache{
		LastCheck:     time.Now(),
		LatestVersion: latestVersion,
		ReleaseURL:    releaseURL,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// ShouldCheck returns true if we should check for updates (cache expired or missing)
func ShouldCheck() bool {
	cache, err := LoadCache()
	if err != nil {
		// Cache doesn't exist or is corrupted, should check
		return true
	}

	// Check if cache has expired
	return time.Since(cache.LastCheck) > CacheDuration
}

// GetCachedResult returns a CheckResult from the cache, or nil if cache is invalid
func GetCachedResult(currentVersion string) *CheckResult {
	cache, err := LoadCache()
	if err != nil {
		return nil
	}

	// Check if cache has expired
	if time.Since(cache.LastCheck) > CacheDuration {
		return nil
	}

	result := &CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  cache.LatestVersion,
		ReleaseURL:     cache.ReleaseURL,
	}

	// Determine if update is available
	updateAvailable, err := isNewerVersion(currentVersion, cache.LatestVersion)
	if err == nil {
		result.UpdateAvailable = updateAvailable
	}

	return result
}
