// Package system provides HTTP download helpers.
package system

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const MaxJSONResponseBytes = 4 * 1024 * 1024

// FetchJSON fetches a JSON document and decodes it into dest.
func FetchJSON(url string, timeout time.Duration, dest any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	return FetchJSONWithDoer(NewHTTPClient(timeout, false), req, dest)
}

// FetchJSONWithDoer fetches a JSON document with the provided HTTP client.
func FetchJSONWithDoer(doer HTTPDoer, req *http.Request, dest any) error {
	resp, err := doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxJSONResponseBytes+1))
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}
	if len(data) > MaxJSONResponseBytes {
		return fmt.Errorf("JSON 响应体过大 (超过 %d bytes)", MaxJSONResponseBytes)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	return nil
}

// DownloadFile downloads a file from url to destPath.
func DownloadFile(url, destPath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	return DownloadFileWithDoer(NewHTTPClient(5*time.Minute, false), req, destPath)
}

// DownloadFileWithDoer downloads a file using the provided HTTP client.
func DownloadFileWithDoer(doer HTTPDoer, req *http.Request, destPath string) error {
	return DownloadFileWithOptions(doer, req, destPath, DownloadOptions{})
}

// DownloadOptions controls download behavior.
type DownloadOptions struct {
	ExpectedSHA256 string
	Atomic         bool
}

// DownloadBytes fetches a URL and returns its body.
func DownloadBytes(url string, timeout time.Duration) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return DownloadBytesWithDoer(NewHTTPClient(timeout, false), req)
}

// DownloadBytesLimit fetches a URL and enforces a maximum response size.
func DownloadBytesLimit(url string, timeout time.Duration, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return DownloadBytesWithDoerLimit(NewHTTPClient(timeout, false), req, maxBytes)
}

// DownloadBytesWithDoer fetches a URL using the provided HTTP client.
func DownloadBytesWithDoer(doer HTTPDoer, req *http.Request) ([]byte, error) {
	return DownloadBytesWithDoerLimit(doer, req, 0)
}

// DownloadBytesWithDoerLimit fetches a URL using the provided HTTP client and size limit.
func DownloadBytesWithDoerLimit(doer HTTPDoer, req *http.Request, maxBytes int64) ([]byte, error) {
	resp, err := doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if maxBytes <= 0 {
		return io.ReadAll(resp.Body)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("响应体过大 (超过 %d bytes)", maxBytes)
	}
	return data, nil
}

// DownloadFileWithOptions downloads a file using the provided HTTP client and options.
func DownloadFileWithOptions(doer HTTPDoer, req *http.Request, destPath string, opts DownloadOptions) error {
	resp, err := doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	path := destPath
	cleanupPath := ""
	if opts.Atomic {
		tmpPath, err := CreateSiblingTempFile(destPath, ".tmp-*")
		if err != nil {
			return err
		}
		path = tmpPath
		cleanupPath = path
		defer os.Remove(cleanupPath)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	hasher := sha256.New()
	writer := io.Writer(out)
	if opts.ExpectedSHA256 != "" {
		writer = io.MultiWriter(out, hasher)
	}

	if _, err = io.Copy(writer, resp.Body); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}

	if opts.ExpectedSHA256 != "" {
		got := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(got, opts.ExpectedSHA256) {
			return fmt.Errorf("SHA256 校验失败: got %s want %s", got, opts.ExpectedSHA256)
		}
	}

	if opts.Atomic {
		if err := ReplaceFile(path, destPath, ReplaceFileOptions{}); err != nil {
			return err
		}
	}
	return nil
}
