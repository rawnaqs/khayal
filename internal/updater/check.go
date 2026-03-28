package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rawnaqs/khayal/internal/version"
)

type UpdateInfo struct {
	Available     bool   `json:"available"`
	Latest        string `json:"latest"`
	ServerVersion string `json:"server_version"`
}

type releaseResponse struct {
	TagName string `json:"tag_name"`
}

var (
	cacheMu      sync.Mutex
	cacheLatest  string
	cacheChecked time.Time
)

// CheckForUpdate checks for updates with a 24-hour in-memory cache.
func CheckForUpdate() *UpdateInfo {
	current := version.Get()
	info := &UpdateInfo{
		Latest:        current,
		ServerVersion: current,
	}

	cacheMu.Lock()
	if time.Since(cacheChecked) < 24*time.Hour && cacheLatest != "" {
		info.Latest = cacheLatest
		if isNewer(cacheLatest, current) {
			info.Available = true
		}
		cacheMu.Unlock()
		return info
	}
	cacheMu.Unlock()

	latest, err := fetchLatestRelease()
	if err != nil {
		return info
	}

	cacheMu.Lock()
	cacheLatest = latest
	cacheChecked = time.Now()
	cacheMu.Unlock()

	info.Latest = latest
	if isNewer(latest, current) {
		info.Available = true
	}

	return info
}

func fetchLatestRelease() (string, error) {
	url := "https://api.github.com/repos/rawnaqs/khayal/releases/latest"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode release: %w", err)
	}

	return stripVPrefix(release.TagName), nil
}

func stripVPrefix(v string) string {
	return strings.TrimPrefix(v, "v")
}

// isNewer compares two semver strings (e.g., "1.0.1" > "1.0.0").
func isNewer(a, b string) bool {
	if a == b {
		return false
	}

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		var ai, bi int
		if i < len(aParts) {
			_, _ = fmt.Sscanf(aParts[i], "%d", &ai)
		}
		if i < len(bParts) {
			_, _ = fmt.Sscanf(bParts[i], "%d", &bi)
		}
		if ai > bi {
			return true
		}
		if ai < bi {
			return false
		}
	}

	return false
}
