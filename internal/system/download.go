// Package system provides HTTP download helpers.
package system

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

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

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
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
	resp, err := doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
