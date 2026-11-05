package app

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var fleetRepoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type fleetRelease struct {
	Version  string `json:"version"`
	Current  string `json:"current"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
	Platform string `json:"platform"`
}

func checkFleetRelease(ctx context.Context, role, platform string) (fleetRelease, error) {
	// GOARCH reports arm without its ISA version. The armv5 release runs on
	// all supported 32-bit ARM CPUs; callers can explicitly request v6/v7.
	if platform == "linux-arm" {
		platform = "linux-armv5"
	}
	repo := os.Getenv("UNIFIEDPROXY_RELEASE_REPO")
	if repo == "" {
		repo = "HlONGlin/UnifiedProxy"
	}
	if !fleetRepoPattern.MatchString(repo) {
		return fleetRelease{}, errors.New("invalid UNIFIEDPROXY_RELEASE_REPO")
	}
	if role != "node" && role != "control" {
		return fleetRelease{}, errors.New("invalid release role")
	}
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" || len(via) > 10 {
			return errors.New("invalid release redirect")
		}
		return nil
	}}
	get := func(url string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "UnifiedProxy-fleet")
		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			return nil, fmt.Errorf("release lookup HTTP %d", res.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(res.Body, (2<<20)+1))
		if len(data) > 2<<20 {
			return nil, errors.New("release metadata too large")
		}
		return data, err
	}
	data, err := get("https://api.github.com/repos/" + repo + "/releases/latest")
	if err != nil {
		return fleetRelease{}, err
	}
	var metadata struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err = json.Unmarshal(data, &metadata); err != nil {
		return fleetRelease{}, err
	}
	asset := role + "-" + platform
	if strings.HasPrefix(platform, "windows-") {
		asset += ".exe"
	}
	result := fleetRelease{Version: metadata.Tag, Current: appVersion, Platform: platform}
	sumURL := ""
	for _, item := range metadata.Assets {
		if item.Name == asset {
			result.URL = item.URL
		}
		if item.Name == "sha256sums.txt" {
			sumURL = item.URL
		}
	}
	if result.URL == "" || sumURL == "" {
		return result, errors.New("release has no fleet binary and checksum for this platform; configure UNIFIEDPROXY_RELEASE_REPO to the merged repository")
	}
	prefix := "https://github.com/" + repo + "/releases/download/"
	if !strings.HasPrefix(result.URL, prefix) || !strings.HasPrefix(sumURL, prefix) {
		return result, errors.New("release assets point outside the configured repository")
	}
	data, err = get(sumURL)
	if err != nil {
		return result, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || path.Base(strings.TrimPrefix(fields[1], "*")) != asset {
			continue
		}
		sum, err := hex.DecodeString(fields[0])
		if err == nil && len(sum) == 32 {
			result.SHA256 = strings.ToLower(fields[0])
			break
		}
	}
	if result.SHA256 == "" {
		return result, errors.New("release checksum missing")
	}
	return result, nil
}

func (a *App) handleFleetRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, 405, apiResponse{})
		return
	}
	role := r.URL.Query().Get("role")
	if role == "" {
		role = a.role
	}
	platform := r.URL.Query().Get("platform")
	if platform == "" {
		platform = runtime.GOOS + "-" + runtime.GOARCH
	}
	// Platform is matched to asset names, never used as a filesystem path.
	release, err := checkFleetRelease(r.Context(), role, platform)
	if err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error(), Data: release})
		return
	}
	writeJSON(w, 200, apiResponse{OK: true, Data: release})
}
