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
	"time"

	"clashctl/internal/system"
)

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

// EnsureGeoData downloads missing geodata files to configDir.
// Returns the number of files downloaded and any error.
func EnsureGeoData(configDir string) (*GeoDataResult, error) {
	client := system.NewHTTPClient(GeoDataDownloadTimeout, false)
	result := &GeoDataResult{}

	for _, f := range DefaultGeoDataFiles() {
		destPath := filepath.Join(configDir, f.Name)

		// Skip if already exists and is not empty
		if info, err := os.Stat(destPath); err == nil && info.Size() > 0 {
			result.Files = append(result.Files, GeoDataFileResult{
				Name:     f.Name,
				Skipped:  true,
				Required: f.Required,
			})
			continue
		}

		// Try multiple sources
		urls := []string{
			GeoDataURL(f.Name),
			GeoDataURLMirror(f.Name),
			GeoDataURLMirror2(f.Name),
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

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, io.LimitReader(reader, GeoDataMaxSize))
	return err
}

// GeoDataReady checks if all required geodata files exist in configDir.
func GeoDataReady(configDir string) bool {
	for _, f := range DefaultGeoDataFiles() {
		if !f.Required {
			continue
		}
		info, err := os.Stat(filepath.Join(configDir, f.Name))
		if err != nil || info.Size() == 0 {
			return false
		}
	}
	return true
}

// WaitForController waits for the controller API to become ready, with retries.
// Returns nil when API is reachable, or error after all retries exhausted.
func WaitForController(addr string, maxRetries int, interval time.Duration) error {
	client := NewClient("http://" + addr)

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
