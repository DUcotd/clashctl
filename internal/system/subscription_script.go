package system

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"clashctl/internal/netsec"
)

const MaxPreparedSubscriptionBytes = 20 * 1024 * 1024

type PreparedSubscription struct {
	Body        []byte
	ContentPath string
	InfoPath    string
	FetchDetail string
	TempDir     string
}

// Cleanup removes any temporary files created for the prepared subscription.
func (p *PreparedSubscription) Cleanup() error {
	if p == nil || strings.TrimSpace(p.TempDir) == "" {
		return nil
	}
	return os.RemoveAll(p.TempDir)
}

// ValidateSubscriptionURL validates that a URL is safe for subscription fetching.
func ValidateSubscriptionURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("订阅 URL 不能为空")
	}

	// Must start with http:// or https://
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("仅支持 http/https 协议的订阅 URL")
	}

	// Keep rejecting obviously dangerous shell metacharacters even though the
	// downloader no longer shells out; this avoids surprising behavior if the URL
	// is surfaced elsewhere.
	dangerousChars := []string{";", "|", "`", "$(", "&&", "||", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(rawURL, char) {
			return fmt.Errorf("URL 包含非法字符: %s", char)
		}
	}

	_, err := netsec.ValidateRemoteHTTPURL(rawURL, netsec.URLValidationOptions{ResolveHost: false})
	return err
}

// PrepareSubscriptionURL downloads a subscription URL using a direct Go HTTP
// client. The client validates each dial target at connect time so redirects
// and late DNS changes cannot silently pivot to local/private addresses.
func PrepareSubscriptionURL(rawURL string, timeout time.Duration) (*PreparedSubscription, error) {
	if err := ValidateSubscriptionURL(rawURL); err != nil {
		return nil, err
	}

	workDir, err := os.MkdirTemp("", "clashctl-sub-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	cleanupOnError := true
	defer func() {
		if cleanupOnError {
			_ = os.RemoveAll(workDir)
		}
	}()

	outDir := filepath.Join(workDir, "output")
	if err := EnsureDir(outDir); err != nil {
		return nil, fmt.Errorf("创建订阅输出目录失败: %w", err)
	}

	body, fetchDetail, err := fetchPreparedSubscription(rawURL, timeout)
	if err != nil {
		return nil, err
	}

	contentPath, err := ensurePathWithinBase(outDir, filepath.Join(outDir, "subscription.txt"))
	if err != nil {
		return nil, fmt.Errorf("订阅内容路径不安全: %w", err)
	}
	if err := WriteFileAtomic(contentPath, body, 0600); err != nil {
		return nil, fmt.Errorf("写入订阅内容失败: %w", err)
	}

	infoPath, err := ensurePathWithinBase(outDir, filepath.Join(outDir, "subscription.info"))
	if err != nil {
		return nil, fmt.Errorf("订阅信息路径不安全: %w", err)
	}
	if err := WriteFileAtomic(infoPath, []byte(fetchDetail+"\n"), 0600); err != nil {
		return nil, fmt.Errorf("写入订阅信息失败: %w", err)
	}

	prepared := &PreparedSubscription{
		Body:        body,
		ContentPath: contentPath,
		InfoPath:    infoPath,
		FetchDetail: fetchDetail,
		TempDir:     workDir,
	}
	cleanupOnError = false
	return prepared, nil
}

func fetchPreparedSubscription(rawURL string, timeout time.Duration) ([]byte, string, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("构建订阅请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "clashctl/"+UserAgentVersion)

	resp, err := newPreparedSubscriptionHTTPClient(timeout).Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("下载订阅失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(preview))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return nil, "", fmt.Errorf("下载订阅失败: HTTP %d: %s", resp.StatusCode, msg)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxPreparedSubscriptionBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("读取订阅响应失败: %w", err)
	}
	if int64(len(body)) > MaxPreparedSubscriptionBytes {
		return nil, "", fmt.Errorf("订阅内容过大: %d bytes (最大允许 %d bytes)", len(body), MaxPreparedSubscriptionBytes)
	}

	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	fetchDetail := strings.Join([]string{
		fmt.Sprintf("url=%s", rawURL),
		fmt.Sprintf("final_url=%s", finalURL),
		"fetcher=go-http",
		fmt.Sprintf("status=%d", resp.StatusCode),
		fmt.Sprintf("bytes=%d", len(body)),
	}, "\n")
	return body, fetchDetail, nil
}

func newPreparedSubscriptionHTTPClient(timeout time.Duration) *http.Client {
	timeout = resolvePreparedSubscriptionTimeout(timeout)
	dialer := &net.Dialer{
		Timeout:   ConnectTimeout,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   10,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialPreparedSubscription(ctx, dialer, network, addr, timeout)
		},
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= MaxRedirects {
				return fmt.Errorf("重定向次数过多 (超过 %d 次)，可能存在重定向循环", MaxRedirects)
			}
			return nil
		},
	}
}

func dialPreparedSubscription(ctx context.Context, dialer *net.Dialer, network, addr string, timeout time.Duration) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	resolved, err := netsec.ResolveRemoteHost(host, netsec.URLValidationOptions{ResolveHost: true, Timeout: timeout})
	if err != nil {
		return nil, err
	}
	if resolved == nil || len(resolved.Addrs) == 0 {
		return nil, fmt.Errorf("无法解析主机 %s", host)
	}

	var lastErr error
	for _, addr := range resolved.Addrs {
		ip := strings.TrimSpace(addr.IP.String())
		if ip == "" {
			continue
		}
		if netsec.IsPrivateIP(addr.IP) && !netsec.AllowLocalSubscriptionTargets() {
			if lastErr == nil {
				lastErr = fmt.Errorf("拒绝连接内网/本地地址 %s (DNS rebinding 防护)", ip)
			}
			continue
		}
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("无法连接主机 %s", host)
}

func resolvePreparedSubscriptionTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return DefaultTimeout
	}
	return timeout
}

func readPreparedSubscriptionBody(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > MaxPreparedSubscriptionBytes {
		return nil, fmt.Errorf("订阅内容过大: %d bytes (最大允许 %d bytes)", info.Size(), MaxPreparedSubscriptionBytes)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, MaxPreparedSubscriptionBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxPreparedSubscriptionBytes {
		return nil, fmt.Errorf("订阅内容过大: %d bytes (最大允许 %d bytes)", len(data), MaxPreparedSubscriptionBytes)
	}
	return data, nil
}

// ReadPreparedSubscriptionBody reads a local subscription file with the same
// size limit used for prepared/remote subscription content.
func ReadPreparedSubscriptionBody(path string) ([]byte, error) {
	return readPreparedSubscriptionBody(path)
}

func ensurePathWithinBase(baseDir, path string) (string, error) {
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(baseAbs, pathAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("路径超出允许目录: %s", path)
	}
	return pathAbs, nil
}
