// Package mihomo provides geodata file management for Mihomo.
package mihomo

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"clashctl/internal/system"
)

var newGeoDataHTTPClient = func() system.HTTPDoer {
	// Geodata is fetched before Mihomo is ready, so proxy env would create a startup loop.
	return system.NewHTTPClient(GeoDataDownloadTimeout, true)
}

const (
	// GeoDataDownloadTimeout is the timeout for downloading geodata files.
	GeoDataDownloadTimeout = 60 * time.Second
	// GeoDataMaxSize is the maximum size for geodata files (50MB).
	GeoDataMaxSize = 50 * 1024 * 1024
)

// GeoDataFile describes a geodata file to download.
type GeoDataFile struct {
	Name     string // e.g. "geosite.dat"
	Required bool   // whether mihomo needs this to start
	SizeHint string // human-readable expected size for UI display
}

// GeoDataFileResult reports the outcome for one geodata file.
type GeoDataFileResult struct {
	Name       string
	Downloaded bool
	Skipped    bool
	Required   bool
	Source     string
	Error      string
}

// GeoDataResult summarizes geodata preparation.
type GeoDataResult struct {
	AlreadyReady bool
	Downloaded   int
	Files        []GeoDataFileResult
}

// DefaultGeoDataFiles returns the list of geodata files mihomo needs.
func DefaultGeoDataFiles() []GeoDataFile {
	return []GeoDataFile{
		{Name: "geosite.dat", Required: true, SizeHint: "~4 MB"},
		{Name: "geoip.dat", Required: true, SizeHint: "~20 MB"},
		{Name: "Country.mmdb", Required: true, SizeHint: "~9 MB"},
	}
}

// GeoDataURL returns the download URL for a geodata file.
// Tries multiple sources for reliability.
func GeoDataURL(filename string) string {
	// Primary: GitHub releases (MetaCubeX/meta-rules-dat)
	return fmt.Sprintf("https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/%s", filename)
}

// GeoDataURLMirror returns a Chinese-friendly mirror URL for geodata files.
func GeoDataURLMirror(filename string) string {
	// Mirror 1: ghfast.top GitHub proxy
	return fmt.Sprintf("https://ghfast.top/https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/%s", filename)
}

// GeoDataURLMirror2 returns an alternative mirror URL.
func GeoDataURLMirror2(filename string) string {
	// Mirror 2: gh-proxy.com GitHub proxy
	return fmt.Sprintf("https://gh-proxy.com/https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/%s", filename)
}

const geoDataMirrorOptInEnv = "CLASHCTL_ALLOW_UNVERIFIED_GEODATA_MIRRORS"

func allowUnverifiedGeoDataMirrors() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(geoDataMirrorOptInEnv)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// EnsureGeoData downloads missing geodata files to configDir.
// Returns the number of files downloaded and any error.
func EnsureGeoData(configDir string) (*GeoDataResult, error) {
	client := newGeoDataHTTPClient()
	result := &GeoDataResult{}

	for _, f := range DefaultGeoDataFiles() {
		destPath := filepath.Join(configDir, f.Name)

		// Skip only when the file looks plausibly complete.
		if info, err := os.Stat(destPath); err == nil && info.Size() > 1024 {
			result.Files = append(result.Files, GeoDataFileResult{
				Name:     f.Name,
				Skipped:  true,
				Required: f.Required,
			})
			continue
		}

		// Only trust the official source by default. Unverified mirrors require explicit opt-in.
		urls := []string{GeoDataURL(f.Name)}
		if allowUnverifiedGeoDataMirrors() {
			urls = append(urls, GeoDataURLMirror(f.Name), GeoDataURLMirror2(f.Name))
		}

		var lastErr error
		fileResult := GeoDataFileResult{
			Name:     f.Name,
			Required: f.Required,
		}
		downloadedOK := false
		for _, url := range urls {
			if err := downloadGeoFile(client, url, destPath); err != nil {
				lastErr = err
				continue
			}
			downloadedOK = true
			fileResult.Source = url
			fileResult.Downloaded = true
			break
		}

		if !downloadedOK {
			fileResult.Error = lastErr.Error()
			result.Files = append(result.Files, fileResult)
			if f.Required {
				return result, fmt.Errorf("下载 %s 失败（所有源均不可用）: %w", f.Name, lastErr)
			}
			continue
		}

		result.Downloaded++
		result.Files = append(result.Files, fileResult)
	}

	result.AlreadyReady = result.Downloaded == 0
	return result, nil
}

// downloadGeoFile downloads a single geodata file, handling .gz decompression.
func downloadGeoFile(client system.HTTPDoer, url, destPath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var reader io.Reader = resp.Body

	// Check magic bytes for gzip (some .dat files are actually gzipped)
	buf := make([]byte, 2)
	n, _ := io.ReadFull(resp.Body, buf)
	if n == 2 && buf[0] == 0x1f && buf[1] == 0x8b {
		gzReader, err := gzip.NewReader(io.MultiReader(bytes.NewReader(buf), resp.Body))
		if err != nil {
			return fmt.Errorf("gzip 解压失败: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	} else if n > 0 {
		reader = io.MultiReader(bytes.NewReader(buf[:n]), resp.Body)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	tmpPath, err := system.CreateSiblingTempFile(destPath, ".tmp-*")
	if err != nil {
		return err
	}
	out, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
		_ = os.Remove(tmpPath)
	}()

	limited := io.LimitReader(reader, GeoDataMaxSize+1)
	written, err := io.Copy(out, limited)
	if err != nil {
		return err
	}
	if written > GeoDataMaxSize {
		return fmt.Errorf("文件超过大小上限 %d bytes", GeoDataMaxSize)
	}
	if written < 1024 {
		return fmt.Errorf("下载内容过小，疑似错误响应")
	}
	if _, err := out.Seek(0, io.SeekStart); err == nil {
		buf := make([]byte, 32)
		n, _ := out.Read(buf)
		lower := bytes.ToLower(bytes.TrimSpace(buf[:n]))
		if bytes.HasPrefix(lower, []byte("<html")) || bytes.HasPrefix(lower, []byte("<!doctype")) {
			return fmt.Errorf("下载内容不是 geodata 文件")
		}
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return system.ReplaceFile(tmpPath, destPath, system.ReplaceFileOptions{})
}

// GeoDataReady checks if all required geodata files exist in configDir.
func GeoDataReady(configDir string) bool {
	for _, f := range DefaultGeoDataFiles() {
		if !f.Required {
			continue
		}
		info, err := os.Stat(filepath.Join(configDir, f.Name))
		if err != nil || info.Size() <= 1024 {
			return false
		}
	}
	return true
}

// WaitForController waits for the controller API to become ready, with retries.
// Returns nil when API is reachable, or error after all retries exhausted.
func WaitForController(addr, secret string, maxRetries int, interval time.Duration) error {
	client := NewClientWithSecret("http://"+addr, secret)

	for i := 0; i < maxRetries; i++ {
		if err := client.CheckConnection(); err == nil {
			return nil
		}
		if i < maxRetries-1 {
			time.Sleep(interval)
		}
	}

	return fmt.Errorf("Controller API 在 %d 秒内未就绪", int(float64(maxRetries)*interval.Seconds()))
}

// NeedGeoData checks if geodata download will be needed on startup.
func NeedGeoData(configDir string) bool {
	return !GeoDataReady(configDir)
}
