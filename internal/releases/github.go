package releases

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"clashctl/internal/system"
)

type MirrorFunc func(string) string

// GitHubAsset represents a downloadable asset from GitHub Releases.
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// GitHubRelease represents a GitHub release response.
type GitHubRelease struct {
	TagName    string        `json:"tag_name"`
	Name       string        `json:"name"`
	Prerelease bool          `json:"prerelease"`
	Assets     []GitHubAsset `json:"assets"`
}

// FetchLatestGitHubRelease retrieves the latest stable or prerelease GitHub release.
func FetchLatestGitHubRelease(owner, repo string, includePreRelease bool, mirror MirrorFunc) (*GitHubRelease, error) {
	if !includePreRelease {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
		var release GitHubRelease
		if err := fetchJSONWithFallback(url, mirror, &release); err != nil {
			return nil, fmt.Errorf("获取 GitHub Release 失败: %w", err)
		}
		return &release, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)
	var releases []GitHubRelease
	if err := fetchJSONWithFallback(url, mirror, &releases); err != nil {
		return nil, fmt.Errorf("获取 GitHub Releases 失败: %w", err)
	}
	release := SelectGitHubRelease(releases, includePreRelease)
	if release == nil {
		return nil, fmt.Errorf("未找到可用的发布版本")
	}
	return release, nil
}

// SelectGitHubRelease returns the first usable release entry.
func SelectGitHubRelease(releases []GitHubRelease, includePreRelease bool) *GitHubRelease {
	for i := range releases {
		release := &releases[i]
		if strings.TrimSpace(release.TagName) == "" {
			continue
		}
		if !includePreRelease && release.Prerelease {
			continue
		}
		return release
	}
	return nil
}

// FindGitHubReleaseAsset looks up a named release asset.
func FindGitHubReleaseAsset(release *GitHubRelease, name string) (GitHubAsset, bool) {
	if release == nil {
		return GitHubAsset{}, false
	}
	for _, asset := range release.Assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return GitHubAsset{}, false
}

// NamedDownloads converts release assets to download descriptors.
func NamedDownloads(release *GitHubRelease) []system.NamedDownload {
	if release == nil {
		return nil
	}
	assets := make([]system.NamedDownload, 0, len(release.Assets))
	for _, asset := range release.Assets {
		assets = append(assets, system.NamedDownload{Name: asset.Name, URL: asset.BrowserDownloadURL})
	}
	return assets
}

// DownloadVerifiedGitHubAsset downloads and verifies a release asset.
//
// By default, third-party mirror fallback is disabled for release metadata,
// binaries, and checksums because a mirror that serves the full release chain
// can satisfy integrity checks without proving authenticity. Operators who
// explicitly accept that trade-off may set CLASHCTL_ALLOW_UNTRUSTED_MIRROR=1.
func DownloadVerifiedGitHubAsset(asset, checksumAsset system.NamedDownload, mirror MirrorFunc, destPath string) error {
	if err := downloadVerifiedFile(asset, checksumAsset, destPath); err != nil {
		if !system.AllowUntrustedMirrorDownloads() {
			return err
		}
		mirrorAsset := asset
		mirrorAsset.URL = mirrorURL(asset.URL, mirror)
		mirrorChecksum := checksumAsset
		mirrorChecksum.URL = mirrorURL(checksumAsset.URL, mirror)
		if mirrorAsset.URL != asset.URL {
			if mirrorErr := downloadVerifiedFile(mirrorAsset, mirrorChecksum, destPath); mirrorErr == nil {
				return nil
			}
		}
		return err
	}
	return nil
}

func fetchJSONWithFallback(url string, mirror MirrorFunc, dest any) error {
	doFetch := func(rawURL string) error {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return err
		}
		return system.FetchJSONWithDoer(
			system.NewHTTPClient(15*time.Second, false),
			req,
			dest,
		)
	}
	err := doFetch(url)
	if err == nil {
		return nil
	}
	if !system.AllowUntrustedMirrorDownloads() {
		return err
	}
	mirrorURL := mirrorURL(url, mirror)
	if mirrorURL != url {
		if mirrorErr := doFetch(mirrorURL); mirrorErr == nil {
			return nil
		}
	}
	return err
}

func downloadVerifiedFile(asset, checksumAsset system.NamedDownload, destPath string) error {
	checksumReq, err := http.NewRequest(http.MethodGet, checksumAsset.URL, nil)
	if err != nil {
		return err
	}
	checksumData, err := system.DownloadBytesWithDoer(
		system.NewHTTPClient(2*time.Minute, false),
		checksumReq,
	)
	if err != nil {
		return fmt.Errorf("下载校验文件失败: %w", err)
	}
	want, err := system.ExtractSHA256(checksumData, asset.Name)
	if err != nil {
		return err
	}
	assetReq, err := http.NewRequest(http.MethodGet, asset.URL, nil)
	if err != nil {
		return err
	}
	return system.DownloadFileWithOptions(
		system.NewHTTPClient(5*time.Minute, false),
		assetReq,
		destPath,
		system.DownloadOptions{ExpectedSHA256: want, Atomic: true},
	)
}

func mirrorURL(url string, mirror MirrorFunc) string {
	if mirror == nil {
		return url
	}
	mirrored := strings.TrimSpace(mirror(url))
	if mirrored == "" {
		return url
	}
	return mirrored
}
