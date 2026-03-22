package system

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"clashctl/internal/core"
	"clashctl/internal/netsec"
)

//go:embed scripts/prepare-subscription.sh
var prepareSubscriptionScript string

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

	// Reject URLs with shell metacharacters
	dangerousChars := []string{";", "|", "`", "$(", "&&", "||", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(rawURL, char) {
			return fmt.Errorf("URL 包含非法字符: %s", char)
		}
	}

	if _, err := netsec.ValidateRemoteHTTPURL(rawURL, netsec.URLValidationOptions{ResolveHost: true}); err != nil {
		return err
	}

	return nil
}

// PrepareSubscriptionURL downloads a subscription URL via the bundled shell helper.
func PrepareSubscriptionURL(rawURL string, timeout time.Duration) (*PreparedSubscription, error) {
	// Validate URL before passing to shell script
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

	scriptPath := filepath.Join(workDir, "prepare-subscription.sh")
	if err := os.WriteFile(scriptPath, []byte(prepareSubscriptionScript), 0700); err != nil {
		return nil, fmt.Errorf("写入订阅脚本失败: %w", err)
	}

	outDir := filepath.Join(workDir, "output")
	cmd := exec.Command(
		"/bin/sh",
		scriptPath,
		rawURL,
		outDir,
		fmt.Sprintf("%d", int(timeout.Seconds())),
		"clashctl/"+core.AppVersion,
		strconv.FormatInt(MaxPreparedSubscriptionBytes, 10),
	)
	cmd.Env = StripProxyEnv(os.Environ())
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("订阅脚本执行失败: %s", msg)
	}

	contentPath := strings.TrimSpace(string(output))
	if contentPath == "" {
		contentPath = filepath.Join(outDir, "subscription.txt")
	}
	contentPath, err = ensurePathWithinBase(outDir, contentPath)
	if err != nil {
		return nil, fmt.Errorf("订阅脚本输出路径不安全: %w", err)
	}
	body, err := readPreparedSubscriptionBody(contentPath)
	if err != nil {
		return nil, fmt.Errorf("读取订阅内容失败: %w", err)
	}
	infoPath, err := ensurePathWithinBase(outDir, filepath.Join(outDir, "subscription.info"))
	if err != nil {
		return nil, fmt.Errorf("订阅信息路径不安全: %w", err)
	}
	infoData, _ := os.ReadFile(infoPath)

	prepared := &PreparedSubscription{
		Body:        body,
		ContentPath: contentPath,
		InfoPath:    infoPath,
		FetchDetail: strings.TrimSpace(string(infoData)),
		TempDir:     workDir,
	}
	cleanupOnError = false
	return prepared, nil
}

func readPreparedSubscriptionBody(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > MaxPreparedSubscriptionBytes {
		return nil, fmt.Errorf("订阅内容过大: %d bytes (最大允许 %d bytes)", info.Size(), MaxPreparedSubscriptionBytes)
	}
	return os.ReadFile(path)
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
