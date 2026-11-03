package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type releaseRoundTrip func(*http.Request) (*http.Response, error)

func (f releaseRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFleetReleaseDefaultsToUnifiedProxyAndChecksAssetOrigin(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	for _, test := range []struct {
		repo, assetRepo string
		wantError       bool
	}{
		{"", "HlONGlin/UnifiedProxy", false},
		{"example/custom", "example/custom", false},
		{"", "elsewhere/untrusted", true},
	} {
		t.Run(test.repo+test.assetRepo, func(t *testing.T) {
			t.Setenv("MUI_RELEASE_REPO", test.repo)
			repo := test.repo
			if repo == "" {
				repo = "HlONGlin/UnifiedProxy"
			}
			http.DefaultTransport = releaseRoundTrip(func(r *http.Request) (*http.Response, error) {
				body := ""
				if r.URL.Host == "api.github.com" {
					if r.URL.Path != "/repos/"+repo+"/releases/latest" {
						t.Fatalf("wrong release repository: %s", r.URL)
					}
					if r.Header.Get("User-Agent") != "UnifiedProxy-fleet" {
						t.Fatal("wrong product agent")
					}
					body = fmt.Sprintf(`{"tag_name":"v-test","assets":[{"name":"control-linux-amd64","browser_download_url":"https://github.com/%s/releases/download/v-test/control-linux-amd64"},{"name":"sha256sums.txt","browser_download_url":"https://github.com/%s/releases/download/v-test/sha256sums.txt"}]}`, test.assetRepo, test.assetRepo)
				} else {
					body = strings.Repeat("a", 64) + "  control-linux-amd64"
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})
			result, err := checkFleetRelease(context.Background(), "control", "linux-amd64")
			if (err != nil) != test.wantError {
				t.Fatal(result, err)
			}
			if err == nil && (result.Version != "v-test" || result.SHA256 != strings.Repeat("a", 64)) {
				t.Fatal("release metadata lost", result)
			}
		})
	}
}
