package releases

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"clashctl/internal/system"
)

func TestSelectGitHubReleasePrefersStableByDefault(t *testing.T) {
	releases := []GitHubRelease{
		{TagName: "v3.0.0-rc1", Prerelease: true},
		{TagName: "v2.9.0", Prerelease: false},
	}

	got := SelectGitHubRelease(releases, false)
	if got == nil {
		t.Fatal("SelectGitHubRelease() returned nil")
	}
	if got.TagName != "v2.9.0" {
		t.Fatalf("SelectGitHubRelease() = %q, want v2.9.0", got.TagName)
	}
}

func TestSelectGitHubReleaseIncludesPrereleaseWhenRequested(t *testing.T) {
	releases := []GitHubRelease{
		{TagName: "v3.0.0-rc1", Prerelease: true},
		{TagName: "v2.9.0", Prerelease: false},
	}

	got := SelectGitHubRelease(releases, true)
	if got == nil {
		t.Fatal("SelectGitHubRelease() returned nil")
	}
	if got.TagName != "v3.0.0-rc1" {
		t.Fatalf("SelectGitHubRelease() = %q, want v3.0.0-rc1", got.TagName)
	}
}

func TestFindGitHubReleaseAsset(t *testing.T) {
	release := &GitHubRelease{Assets: []GitHubAsset{{Name: "clashctl-linux-amd64", BrowserDownloadURL: "https://example.com/a"}}}

	asset, ok := FindGitHubReleaseAsset(release, "clashctl-linux-amd64")
	if !ok {
		t.Fatal("FindGitHubReleaseAsset() should find requested asset")
	}
	if asset.BrowserDownloadURL != "https://example.com/a" {
		t.Fatalf("asset = %#v", asset)
	}
}

func TestNamedDownloads(t *testing.T) {
	release := &GitHubRelease{Assets: []GitHubAsset{{Name: "clashctl-linux-amd64", BrowserDownloadURL: "https://example.com/a"}}}

	got := NamedDownloads(release)
	want := []system.NamedDownload{{Name: "clashctl-linux-amd64", URL: "https://example.com/a"}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("NamedDownloads() = %#v, want %#v", got, want)
	}
}

func TestDownloadVerifiedGitHubAssetDoesNotUseMirrorByDefault(t *testing.T) {
	t.Setenv("CLASHCTL_ALLOW_UNTRUSTED_MIRROR", "")

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checksums.txt":
			_, _ = fmt.Fprintln(w, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824  clashctl-linux-amd64")
		case "/clashctl-linux-amd64":
			http.Error(w, "upstream down", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	defer origin.Close()

	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checksums.txt":
			_, _ = fmt.Fprintln(w, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824  clashctl-linux-amd64")
		case "/clashctl-linux-amd64":
			_, _ = w.Write([]byte("hello"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mirror.Close()

	dest := filepath.Join(t.TempDir(), "clashctl-linux-amd64")
	err := DownloadVerifiedGitHubAsset(
		system.NamedDownload{Name: "clashctl-linux-amd64", URL: origin.URL + "/clashctl-linux-amd64"},
		system.NamedDownload{Name: "checksums.txt", URL: origin.URL + "/checksums.txt"},
		func(raw string) string { return mirror.URL + "/" + filepath.Base(raw) },
		dest,
	)
	if err == nil {
		t.Fatal("DownloadVerifiedGitHubAsset() should fail when origin download fails and mirror fallback is disabled")
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Fatalf("dest file should not exist, stat err = %v", statErr)
	}
}

func TestDownloadVerifiedGitHubAssetAllowsMirrorWithExplicitOptIn(t *testing.T) {
	t.Setenv("CLASHCTL_ALLOW_UNTRUSTED_MIRROR", "1")

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checksums.txt":
			_, _ = fmt.Fprintln(w, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824  clashctl-linux-amd64")
		case "/clashctl-linux-amd64":
			http.Error(w, "upstream down", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	defer origin.Close()

	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checksums.txt":
			_, _ = fmt.Fprintln(w, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824  clashctl-linux-amd64")
		case "/clashctl-linux-amd64":
			_, _ = w.Write([]byte("hello"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mirror.Close()

	dest := filepath.Join(t.TempDir(), "clashctl-linux-amd64")
	err := DownloadVerifiedGitHubAsset(
		system.NamedDownload{Name: "clashctl-linux-amd64", URL: origin.URL + "/clashctl-linux-amd64"},
		system.NamedDownload{Name: "checksums.txt", URL: origin.URL + "/checksums.txt"},
		func(raw string) string { return mirror.URL + "/" + filepath.Base(raw) },
		dest,
	)
	if err != nil {
		t.Fatalf("DownloadVerifiedGitHubAsset() error = %v", err)
	}
	data, readErr := os.ReadFile(dest)
	if readErr != nil {
		t.Fatalf("ReadFile(dest) error = %v", readErr)
	}
	if string(data) != "hello" {
		t.Fatalf("dest content = %q, want hello", string(data))
	}
}

func TestFetchJSONWithFallbackDoesNotUseMirrorByDefault(t *testing.T) {
	t.Setenv("CLASHCTL_ALLOW_UNTRUSTED_MIRROR", "")

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream down", http.StatusBadGateway)
	}))
	defer origin.Close()

	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(GitHubRelease{TagName: "v9.9.9"})
	}))
	defer mirror.Close()

	var release GitHubRelease
	err := fetchJSONWithFallback(origin.URL+"/release", func(string) string { return mirror.URL + "/release" }, &release)
	if err == nil {
		t.Fatal("fetchJSONWithFallback() should fail when mirror fallback is disabled")
	}
	if release.TagName != "" {
		t.Fatalf("release = %#v, want zero value", release)
	}
}

func TestFetchJSONWithFallbackAllowsMirrorWithExplicitOptIn(t *testing.T) {
	t.Setenv("CLASHCTL_ALLOW_UNTRUSTED_MIRROR", "1")

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream down", http.StatusBadGateway)
	}))
	defer origin.Close()

	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(GitHubRelease{TagName: "v9.9.9"})
	}))
	defer mirror.Close()

	var release GitHubRelease
	err := fetchJSONWithFallback(origin.URL+"/release", func(string) string { return mirror.URL + "/release" }, &release)
	if err != nil {
		t.Fatalf("fetchJSONWithFallback() error = %v", err)
	}
	if release.TagName != "v9.9.9" {
		t.Fatalf("release.TagName = %q, want v9.9.9", release.TagName)
	}
}
