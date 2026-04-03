# clashctl 安全漏洞修复计划

> 生成时间: 2026-04-03
> 项目版本: v2.7.2
> 状态: 待执行

---

## 漏洞总览

| # | 编号 | 严重性 | 漏洞名称 | 状态 |
|---|------|--------|---------|------|
| 1 | VULN-1 | **严重** | DNS Rebinding SSRF 绕过 | 待修复 |
| 2 | VULN-2 | **严重** | 符号链接 TOCTOU 路径遍历 | 待修复 |
| 3 | VULN-3 | **严重** | 相对路径校验绕过 | 待修复 |
| 4 | VULN-4 | **高** | YAML Bomb 资源耗尽 | 待修复 |
| 5 | VULN-5 | **高** | Shell 配置符号链接攻击 | 待修复 |
| 6 | VULN-6 | **高** | PID 文件符号链接攻击 | 待修复 |
| 7 | VULN-7 | **高** | 代理组 URL 校验遗漏 | 待修复 |
| 8 | VULN-8 | **中** | Mirror URL 拼接注入 | 待修复 |
| 9 | VULN-9 | **中** | 备份恢复校验不足 | 待修复 |
| 10 | VULN-10 | **中** | 日志脱敏覆盖不全 | 待修复 |
| 11 | VULN-12 | **中** | 依赖版本过旧 | 待修复 |
| 12 | VULN-11 | **低** | /proc 扫描信息泄露 | 待修复 |
| 13 | VULN-13 | **低** | 错误信息内部路径泄露 | 待修复 |
| 14 | VULN-14 | **低** | 临时文件权限不足 | 待修复 |

---

## Phase 1: P0 严重漏洞修复

### VULN-1: DNS Rebinding SSRF 绕过

**文件**: `internal/system/subscription_script.go` (第 191-220 行 `dialPreparedSubscription`)

**问题分析**:
当前代码在 `dialPreparedSubscription` 中调用 `netsec.ResolveRemoteHost` 解析 DNS 并校验 IP，但解析和实际连接之间存在时间窗口。攻击者可以：
1. 控制一个域名，初始 DNS 指向公网 IP（通过校验）
2. 在校验完成后、连接发生前，将 DNS 改为内网 IP（如 `127.0.0.1` 或 `169.254.169.254`）
3. 实际连接时命中内网服务

**修复方案**:
在 `dialPreparedSubscription` 的循环中，对每个解析到的 IP 在连接前重新执行私有地址校验。需要导出 `netsec.IsPrivateIP` 函数供调用。

```go
// 修复前 (第 204-215 行):
for _, addr := range resolved.Addrs {
    ip := strings.TrimSpace(addr.IP.String())
    if ip == "" {
        continue
    }
    conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
    if err == nil {
        return conn, nil
    }
    lastErr = err
}

// 修复后:
for _, addr := range resolved.Addrs {
    ip := strings.TrimSpace(addr.IP.String())
    if ip == "" {
        continue
    }
    // DNS rebinding 防护: 连接前再次校验 IP
    if netsec.IsPrivateIP(addr.IP) {
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
```

**需要新增导出函数** (`internal/netsec/url.go`):
```go
// IsPrivateIP checks if an IP address is private/local.
func IsPrivateIP(ip net.IP) bool {
    if ip == nil {
        return false
    }
    if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsUnspecified() {
        return true
    }
    if ip4 := ip.To4(); ip4 != nil {
        privateCIDRs := []string{
            "10.0.0.0/8",
            "172.16.0.0/12",
            "192.168.0.0/16",
            "100.64.0.0/10",
            "169.254.0.0/16",
            "127.0.0.0/8",
        }
        for _, cidr := range privateCIDRs {
            _, network, _ := net.ParseCIDR(cidr)
            if network.Contains(ip4) {
                return true
            }
        }
        return false
    }
    privateCIDRs := []string{
        "::1/128",
        "fc00::/7",
        "fe80::/10",
    }
    for _, cidr := range privateCIDRs {
        _, network, _ := net.ParseCIDR(cidr)
        if network.Contains(ip) {
            return true
        }
    }
    return false
}
```

**测试** (`internal/system/subscription_script_test.go`):
```go
func TestDialPreparedSubscription_RejectsPrivateIPs(t *testing.T) {
    // 模拟 DNS rebinding: 解析结果中包含私有 IP
    // 验证连接被拒绝
}

func TestIsPrivateIP(t *testing.T) {
    tests := []struct {
        ip       string
        expected bool
    }{
        {"127.0.0.1", true},
        {"10.0.0.1", true},
        {"192.168.1.1", true},
        {"172.16.0.1", true},
        {"169.254.169.254", true},
        {"8.8.8.8", false},
        {"1.1.1.1", false},
        {"::1", true},
        {"fc00::1", true},
        {"fe80::1", true},
    }
    for _, tt := range tests {
        t.Run(tt.ip, func(t *testing.T) {
            ip := net.ParseIP(tt.ip)
            got := netsec.IsPrivateIP(ip)
            if got != tt.expected {
                t.Errorf("IsPrivateIP(%s) = %v, want %v", tt.ip, got, tt.expected)
            }
        })
    }
}
```

---

### VULN-2: 符号链接 TOCTOU 路径遍历

**文件**: `internal/system/fs.go` (第 114-159 行 `ReplaceFile`)

**问题分析**:
`ReplaceFile` 在 `os.Rename` 前仅做了路径校验，但校验和实际 rename 之间存在竞态窗口。攻击者可以：
1. 在校验通过后、rename 前，将目标路径替换为符号链接
2. 导致文件被写入到符号链接指向的任意位置

**修复方案**:
在 `ReplaceFile` 执行 rename 前，对目标路径执行二次校验，并使用 `os.Lstat` 检查是否为符号链接。

```go
// 修复: 在 os.Rename(srcPath, destPath) 前添加:
func ReplaceFile(srcPath, destPath string, opts ReplaceFileOptions) error {
    if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
        return fmt.Errorf("创建目标目录失败: %w", err)
    }

    // 二次校验: 防止 TOCTOU 符号链接攻击
    if err := ValidateOutputPath(destPath); err != nil {
        return fmt.Errorf("目标路径不安全: %w", err)
    }
    if linfo, err := os.Lstat(destPath); err == nil {
        if linfo.Mode()&os.ModeSymlink != 0 {
            return fmt.Errorf("拒绝覆盖符号链接: %s", destPath)
        }
    }

    backupPath, err := ReserveSiblingPath(destPath, ".bak-*")
    // ... 其余代码不变
}
```

**测试** (`internal/system/security_test.go`):
```go
func TestReplaceFile_RejectsSymlink(t *testing.T) {
    tmpDir := t.TempDir()
    target := filepath.Join(tmpDir, "real_target.txt")
    symlink := filepath.Join(tmpDir, "symlink.txt")

    // 创建真实文件
    os.WriteFile(target, []byte("real"), 0644)
    // 创建符号链接指向目标
    os.Symlink(target, symlink)

    src := filepath.Join(tmpDir, "src.txt")
    os.WriteFile(src, []byte("new content"), 0644)

    // 尝试通过符号链接替换应失败
    err := ReplaceFile(src, symlink, ReplaceFileOptions{})
    if err == nil {
        t.Error("ReplaceFile 应拒绝覆盖符号链接")
    }
    if err != nil && !strings.Contains(err.Error(), "符号链接") {
        t.Errorf("错误消息应包含符号链接提示: %v", err)
    }
}
```

---

### VULN-3: 相对路径校验绕过

**文件**: `internal/system/fs.go` (第 192-239 行 `ValidateOutputPath`)

**问题分析**:
第 229 行 `if isRelative { return nil }` 对相对路径直接放行，不做任何校验。攻击者可通过相对路径（如 `../../../etc/passwd` 的变体）绕过白名单检查。

**修复方案**:
移除相对路径直接放行的逻辑，所有路径（包括相对路径）都需解析为绝对路径后校验是否在允许的根目录内。

```go
// 修复前 (第 229-231 行):
if isRelative {
    return nil
}

// 修复后:
// 相对路径也需要校验，解析为绝对路径后检查
if isRelative {
    for _, root := range allowedOutputRoots() {
        if pathWithinRoot(resolvedPath, root) {
            return nil
        }
    }
    return fmt.Errorf("相对路径必须位于允许的目录中: %s", resolvedPath)
}
```

**测试** (`internal/system/security_test.go`):
```go
func TestValidateOutputPath_RelativePathEscapes(t *testing.T) {
    // 相对路径尝试逃逸
    err := ValidateOutputPath("../../etc/passwd")
    if err == nil {
        t.Error("ValidateOutputPath 应拒绝逃逸的相对路径")
    }

    // 相对路径在允许目录内应通过
    err = ValidateOutputPath("config.yaml")
    // 取决于当前工作目录，至少不应直接放行
}
```

---

## Phase 2: P1 高危漏洞修复

### VULN-4: YAML Bomb 资源耗尽

**文件**: `internal/config/loader.go`, `internal/subscription/security.go`

**问题分析**:
YAML 解析无嵌套深度和节点数限制。恶意构造的深度嵌套 YAML 可导致解析时栈溢出或内存耗尽。

**修复方案**:
添加自定义 YAML 解码器，限制嵌套深度和节点总数。

```go
// internal/config/loader.go 新增:
const (
    MaxYAMLDepth  = 50
    MaxYAMLNodes  = 100000
)

func ValidateYAMLBytesWithLimits(data []byte) error {
    var node yaml.Node
    if err := yaml.Unmarshal(data, &node); err != nil {
        return err
    }
    if err := checkYAMLDepth(&node, 0); err != nil {
        return err
    }
    count := countYAMLNodes(&node)
    if count > MaxYAMLNodes {
        return fmt.Errorf("YAML 节点数过多: %d (最大允许 %d)", count, MaxYAMLNodes)
    }
    return nil
}

func checkYAMLDepth(node *yaml.Node, depth int) error {
    if depth > MaxYAMLDepth {
        return fmt.Errorf("YAML 嵌套过深: %d (最大允许 %d)", depth, MaxYAMLDepth)
    }
    for _, child := range node.Content {
        if err := checkYAMLDepth(child, depth+1); err != nil {
            return err
        }
    }
    return nil
}

func countYAMLNodes(node *yaml.Node) int {
    count := 1
    for _, child := range node.Content {
        count += countYAMLNodes(child)
    }
    return count
}
```

**测试**:
```go
func TestValidateYAMLBytesWithLimits_DepthExceeded(t *testing.T) {
    // 构造深度嵌套的 YAML
    var buf strings.Builder
    buf.WriteString("a:\n")
    for i := 0; i < 60; i++ {
        buf.WriteString(strings.Repeat("  ", i+1) + "b:\n")
    }
    buf.WriteString(strings.Repeat("  ", 61) + "c: 1\n")

    err := ValidateYAMLBytesWithLimits([]byte(buf.String()))
    if err == nil {
        t.Error("应拒绝嵌套过深的 YAML")
    }
}
```

---

### VULN-5: Shell 配置符号链接攻击

**文件**: `internal/system/shell_proxy.go` (第 122-132 行 `upsertManagedBlock`)

**问题分析**:
写入 `~/.bashrc`/`~/.zshrc` 前未检查目标文件是否为符号链接，攻击者可创建符号链接将内容写入任意文件。

**修复方案**:
```go
func upsertManagedBlock(path, block string) error {
    // 检查符号链接
    if linfo, err := os.Lstat(path); err == nil {
        if linfo.Mode()&os.ModeSymlink != 0 {
            return fmt.Errorf("拒绝写入符号链接: %s", path)
        }
        // 检查全局可写权限
        if linfo.Mode().Perm()&0002 != 0 {
            return fmt.Errorf("拒绝写入全局可写文件: %s", path)
        }
    }

    // 确保路径在用户主目录下
    home, _ := os.UserHomeDir()
    absPath, _ := filepath.Abs(path)
    if !strings.HasPrefix(absPath, home) {
        return fmt.Errorf("拒绝写入用户主目录外的文件: %s", absPath)
    }

    content, err := readManagedShellFile(path)
    // ... 其余代码不变
}
```

**测试**:
```go
func TestUpsertManagedBlock_RejectsSymlink(t *testing.T) {
    tmpDir := t.TempDir()
    target := filepath.Join(tmpDir, "target.txt")
    symlink := filepath.Join(tmpDir, "symlink.txt")
    os.WriteFile(target, []byte("target"), 0644)
    os.Symlink(target, symlink)

    err := upsertManagedBlock(symlink, "test block")
    if err == nil {
        t.Error("应拒绝写入符号链接")
    }
}
```

---

### VULN-6: PID 文件符号链接攻击

**文件**: `internal/mihomo/process.go` (第 208-213 行 `writePIDFile`)

**问题分析**:
`os.WriteFile` 会跟随符号链接，若 PID 文件路径是符号链接，可覆盖任意文件。

**修复方案**:
```go
func writePIDFile(configDir string, pid int) error {
    if err := os.MkdirAll(configDir, 0755); err != nil {
        return err
    }
    path := pidFilePath(configDir)

    // 检查符号链接
    if linfo, err := os.Lstat(path); err == nil {
        if linfo.Mode()&os.ModeSymlink != 0 {
            return fmt.Errorf("PID 文件路径是符号链接: %s", path)
        }
    }

    return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0600)
}
```

**测试**:
```go
func TestWritePIDFile_RejectsSymlink(t *testing.T) {
    tmpDir := t.TempDir()
    target := filepath.Join(tmpDir, "real_pid")
    symlink := filepath.Join(tmpDir, "clashctl.pid")
    os.WriteFile(target, []byte("0"), 0644)
    os.Symlink(target, symlink)

    err := writePIDFile(tmpDir, 12345)
    if err == nil {
        t.Error("writePIDFile 应拒绝符号链接")
    }
}
```

---

### VULN-7: 代理组 URL 全量校验

**文件**: `internal/subscription/yaml_profile.go` (第 274-284 行)

**问题分析**:
仅对 `url-test`/`fallback`/`load-balance` 类型的代理组校验 URL，`select` 类型或其他类型的 URL 字段不校验。

**修复方案**:
```go
// 修复: 对所有代理组中的 URL 字段统一校验
func sanitizeProxyGroups(value any) []any {
    list, ok := value.([]any)
    if !ok {
        return nil
    }
    allowedKeys := map[string]bool{
        "name":      true,
        "type":      true,
        "proxies":   true,
        "use":       true,
        "url":       true,
        "interval":  true,
        "lazy":      true,
        "tolerance": true,
        "filter":    true,
    }
    out := make([]any, 0, len(list))
    for _, entry := range list {
        group, ok := entry.(map[string]any)
        if !ok {
            continue
        }
        cleaned := map[string]any{}
        groupType, _ := group["type"].(string)
        for key, groupValue := range group {
            lowerKey := strings.ToLower(key)
            if !allowedKeys[lowerKey] {
                continue
            }
            // 所有类型的 URL 都校验
            if lowerKey == "url" {
                urlValue, ok := groupValue.(string)
                if !ok {
                    continue
                }
                if _, err := netsec.ValidateRemoteHTTPURL(urlValue, netsec.URLValidationOptions{ResolveHost: true}); err != nil {
                    continue
                }
            }
            cleaned[key] = cloneYAMLValue(groupValue)
        }
        if len(cleaned) > 0 {
            out = append(out, cleaned)
        }
    }
    return out
}
```

**测试**:
```go
func TestSanitizeProxyGroups_ValidatesAllURLs(t *testing.T) {
    // select 类型代理组含内网 URL 应被过滤
    groups := []any{
        map[string]any{
            "name": "PROXY",
            "type": "select",
            "url":  "http://192.168.1.1/health",
        },
    }
    result := sanitizeProxyGroups(groups)
    if len(result) > 0 {
        t.Error("应过滤含内网 URL 的代理组")
    }
}
```

---

## Phase 3: P2 中危漏洞修复

### VULN-8: Mirror URL 安全拼接

**文件**: `internal/mihomo/installer.go` (第 56-73 行 `GetGitHubMirrorURL`)

**问题分析**:
字符串拼接 `mirror + "/" + strings.TrimPrefix(originalURL, "https://")` 不安全，若 mirror 环境变量包含路径可导致 URL 结构异常。

**修复方案**:
```go
import "net/url"

func GetGitHubMirrorURL(originalURL string) string {
    if customMirror := os.Getenv("CLASHCTL_GITHUB_MIRROR"); customMirror != "" {
        mirror := strings.TrimRight(customMirror, "/")
        if strings.HasPrefix(originalURL, "https://github.com/") || strings.HasPrefix(originalURL, "https://api.github.com/") {
            mirrorURL, err := url.JoinPath(mirror, strings.TrimPrefix(originalURL, "https://"))
            if err == nil {
                return mirrorURL
            }
        }
    }
    // ... 其余逻辑不变
}
```

### VULN-9: 备份恢复校验增强

**文件**: `cmd/backup.go` (第 269-358 行 `runRestore`)

**问题分析**:
备份文件名仅做基本路径检查，未校验扩展名。恢复目标路径未做白名单校验。

**修复方案**:
```go
func resolveBackupPath(backupDir, backupName string) (string, error) {
    name := strings.TrimSpace(backupName)
    if name == "" {
        return "", fmt.Errorf("备份文件名不能为空")
    }
    if filepath.Base(name) != name {
        return "", fmt.Errorf("备份文件名不合法: %s", backupName)
    }
    // 扩展名白名单
    ext := strings.ToLower(filepath.Ext(name))
    if ext != ".yaml" && ext != ".yml" {
        return "", fmt.Errorf("备份文件扩展名不合法: %s (仅支持 .yaml/.yml)", ext)
    }
    // ... 其余代码不变
}
```

### VULN-10: 日志脱敏增强

**文件**: `internal/app/logger.go`

**问题分析**:
当前正则脱敏可能遗漏 base64 编码的 token 或新型密钥格式。

**修复方案**:
添加更多脱敏模式，包括 base64 token 检测。

### VULN-12: 依赖版本更新

**文件**: `go.mod`

**更新计划**:
```
golang.org/x/sys    v0.12.0  -> v0.38.0
golang.org/x/text   v0.3.8   -> v0.30.0
golang.org/x/term   v0.6.0   -> v0.36.0
golang.org/x/sync   v0.1.0   -> v0.18.0
```

---

## Phase 4: P3 低危漏洞修复

### VULN-11: /proc 扫描限制

**文件**: `internal/mihomo/process.go` (第 263-291 行 `findManagedProcessPIDs`)

**修复方案**:
添加当前 PID namespace 检查，仅在 `/proc/self/ns/pid` 匹配时扫描。

### VULN-13: 错误信息路径脱敏

**文件**: 多处错误消息

**修复方案**:
在错误消息中使用 `filepath.Base(path)` 替代完整路径。

### VULN-14: 临时文件权限加固

**文件**: `internal/system/fs.go` (第 49-64 行 `CreateSiblingTempFile`)

**修复方案**:
```go
func CreateSiblingTempFile(targetPath, patternSuffix string) (string, error) {
    dir := filepath.Dir(targetPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return "", err
    }
    tmpFile, err := os.CreateTemp(dir, filepath.Base(targetPath)+patternSuffix)
    if err != nil {
        return "", err
    }
    tmpPath := tmpFile.Name()
    // 显式设置安全权限
    if err := os.Chmod(tmpPath, 0600); err != nil {
        _ = os.Remove(tmpPath)
        return "", err
    }
    if err := tmpFile.Close(); err != nil {
        _ = os.Remove(tmpPath)
        return "", err
    }
    return tmpPath, nil
}
```

---

## 执行顺序

1. **Phase 1** (P0) → VULN-1, VULN-2, VULN-3
2. **Phase 2** (P1) → VULN-4, VULN-5, VULN-6, VULN-7
3. **Phase 3** (P2) → VULN-8, VULN-9, VULN-10, VULN-12
4. **Phase 4** (P3) → VULN-11, VULN-13, VULN-14
5. **最终验证** → 运行 `make test` 全量测试
