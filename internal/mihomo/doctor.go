// Package mihomo provides environment diagnostic functions (doctor).
package mihomo

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"clashctl/internal/system"
)

// CheckResult represents a single diagnostic check.
type CheckResult struct {
	Name    string // e.g. "mihomo binary"
	Passed  bool
	Problem string // empty if passed
	Suggest string // remediation suggestion if failed
}

// RunDoctor performs all environment checks and returns the results.
func RunDoctor(configDir, controllerAddr string, tunMode bool) []CheckResult {
	var results []CheckResult

	// 1. Check mihomo binary
	results = append(results, checkBinary())

	// 2. Check config directory
	results = append(results, checkConfigDir(configDir))

	// 3. Check /dev/net/tun (only if TUN mode)
	if tunMode {
		results = append(results, checkTUNDevice())
	}

	// 3b. Check iptables (required for TUN mode auto-route)
	if tunMode {
		results = append(results, checkIptables())
	}

	// 4. Check root privilege (for TUN mode)
	if tunMode {
		results = append(results, checkRootPrivilege())
	}

	// 5. Check controller port
	results = append(results, checkControllerPort(controllerAddr))

	// 6. Check systemd availability
	results = append(results, checkSystemd())

	// 7. Check disk space
	results = append(results, checkDiskSpace(configDir))

	// 8. Check DNS resolution
	results = append(results, checkDNS())

	// 9. Check if mihomo is running (process check)
	results = append(results, checkMihomoRunning(controllerAddr))

	// 10. Check geodata files (needed for GEOIP/GEOSITE rules)
	results = append(results, checkGeoData(configDir))

	return results
}

func checkBinary() CheckResult {
	binary, err := FindBinary()
	if err != nil {
		return CheckResult{
			Name:    "mihomo 可执行文件",
			Passed:  false,
			Problem: err.Error(),
			Suggest: "运行 'clashctl init' 将自动下载安装，或手动安装: https://github.com/MetaCubeX/mihomo/releases",
		}
	}
	version, _ := GetBinaryVersion()
	return CheckResult{
		Name:   "mihomo 可执行文件",
		Passed: true,
		Problem: fmt.Sprintf("已找到: %s (%s)", binary, version),
	}
}

func checkConfigDir(configDir string) CheckResult {
	if !system.DirExists(configDir) {
		return CheckResult{
			Name:    "配置目录",
			Passed:  false,
			Problem: fmt.Sprintf("目录 %s 不存在", configDir),
			Suggest: "运行 clashctl init 或手动创建目录",
		}
	}
	if err := system.DirWritable(configDir); err != nil {
		return CheckResult{
			Name:    "配置目录",
			Passed:  false,
			Problem: err.Error(),
			Suggest: fmt.Sprintf("请检查 %s 的权限", configDir),
		}
	}
	return CheckResult{
		Name:   "配置目录",
		Passed: true,
	}
}

func checkTUNDevice() CheckResult {
	if _, err := os.Stat("/dev/net/tun"); os.IsNotExist(err) {
		return CheckResult{
			Name:    "TUN 设备 (/dev/net/tun)",
			Passed:  false,
			Problem: "/dev/net/tun 不存在",
			Suggest: "请加载 TUN 内核模块: sudo modprobe tun",
		}
	}
	return CheckResult{
		Name:   "TUN 设备 (/dev/net/tun)",
		Passed: true,
	}
}

func checkIptables() CheckResult {
	if !system.CommandExists("iptables") {
		return CheckResult{
			Name:    "iptables",
			Passed:  false,
			Problem: "未找到 iptables，TUN 模式需要 iptables 进行流量路由",
			Suggest: "安装 iptables: apt install iptables / yum install iptables\n或使用 mixed-port 模式代替 TUN 模式",
		}
	}
	return CheckResult{
		Name:   "iptables",
		Passed: true,
	}
}

func checkRootPrivilege() CheckResult {
	if !system.IsRoot() {
		return CheckResult{
			Name:    "root 权限 (TUN 模式)",
			Passed:  false,
			Problem: "当前进程未以 root 身份运行",
			Suggest: "TUN 模式需要 root 权限，请使用 sudo 运行",
		}
	}
	return CheckResult{
		Name:   "root 权限 (TUN 模式)",
		Passed: true,
	}
}

func checkControllerPort(controllerAddr string) CheckResult {
	if system.CheckPortInUse(controllerAddr) {
		// Port is in use - check if it's actually a working Mihomo instance
		client := NewClient("http://" + controllerAddr)
		if err := client.CheckConnection(); err == nil {
			// Mihomo is running and responding - this is fine
			version, _ := client.Version()
			detail := "端口被 Mihomo 正常占用"
			if version != "" {
				detail += " (" + version + ")"
			}
			return CheckResult{
				Name:   "controller 端口",
				Passed: true,
				Problem: detail,
			}
		}
		return CheckResult{
			Name:    "controller 端口",
			Passed:  false,
			Problem: fmt.Sprintf("端口 %s 已被其他进程占用", controllerAddr),
			Suggest: "请确认没有其他 Mihomo 实例在运行，或修改 external-controller 地址",
		}
	}
	return CheckResult{
		Name:   "controller 端口",
		Passed: true,
	}
}

func checkSystemd() CheckResult {
	if _, err := os.Stat("/run/systemd/system"); os.IsNotExist(err) {
		return CheckResult{
			Name:    "systemd",
			Passed:  true, // Not a failure - expected in containers
			Problem: "未检测到 systemd（容器环境正常），将使用子进程模式管理",
		}
	}
	return CheckResult{
		Name:   "systemd",
		Passed: true,
	}
}

func checkDiskSpace(dir string) CheckResult {
	// Fall back to /tmp if the target dir doesn't exist yet
	checkDir := dir
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		checkDir = "/"
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(checkDir, &stat); err != nil {
		return CheckResult{
			Name:    "磁盘空间",
			Passed:  false,
			Problem: fmt.Sprintf("无法获取磁盘信息: %v", err),
			Suggest: "请检查磁盘状态",
		}
	}

	// Available space in GB
	availGB := float64(stat.Bavail*uint64(stat.Bsize)) / (1024 * 1024 * 1024)

	if availGB < 0.1 {
		return CheckResult{
			Name:    "磁盘空间",
			Passed:  false,
			Problem: fmt.Sprintf("%.1f GB 可用空间不足", availGB),
			Suggest: "请清理磁盘空间",
		}
	}
	return CheckResult{
		Name:   "磁盘空间",
		Passed: true,
		Problem: fmt.Sprintf("%.1f GB 可用", availGB),
	}
}

func checkDNS() CheckResult {
	// Try to resolve a common domain
	addrs, err := system.LookupHost("cp.cloudflare.com")
	if err != nil {
		return CheckResult{
			Name:    "DNS 解析",
			Passed:  false,
			Problem: fmt.Sprintf("DNS 解析失败: %v", err),
			Suggest: "请检查 /etc/resolv.conf 或网络连接",
		}
	}
	return CheckResult{
		Name:   "DNS 解析",
		Passed: true,
		Problem: fmt.Sprintf("cp.cloudflare.com → %s", addrs),
	}
}

func checkMihomoRunning(controllerAddr string) CheckResult {
	client := NewClient("http://" + controllerAddr)
	if err := client.CheckConnection(); err != nil {
		return CheckResult{
			Name:    "Mihomo 运行状态",
			Passed:  false,
			Problem: "Mihomo 未运行或 Controller API 不可达",
			Suggest: "使用 'clashctl start' 启动 Mihomo",
		}
	}

	version, _ := client.Version()
	detail := "Mihomo 正在运行"
	if version != "" {
		detail += " (" + version + ")"
	}
	return CheckResult{
		Name:   "Mihomo 运行状态",
		Passed: true,
		Problem: detail,
	}
}

// CanUseTUN checks if TUN mode is viable on this system.
// Returns true only if /dev/net/tun exists AND iptables is available.
func CanUseTUN() bool {
	// Check /dev/net/tun exists
	if _, err := os.Stat("/dev/net/tun"); os.IsNotExist(err) {
		return false
	}
	// Check iptables is available (needed for auto-route)
	if !system.CommandExists("iptables") {
		return false
	}
	return true
}

func checkGeoData(configDir string) CheckResult {
	files := DefaultGeoDataFiles()
	missing := []string{}
	ready := []string{}

	for _, f := range files {
		path := filepath.Join(configDir, f.Name)
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			if f.Required {
				missing = append(missing, f.Name)
			}
		} else {
			ready = append(ready, fmt.Sprintf("%s (%.1f MB)", f.Name, float64(info.Size())/(1024*1024)))
		}
	}

	if len(missing) == 0 {
		detail := ""
		if len(ready) > 0 {
			detail = fmt.Sprintf("全部就绪: %s", ready)
		}
		return CheckResult{
			Name:    "GeoSite/GeoIP 数据",
			Passed:  true,
			Problem: detail,
		}
	}

	return CheckResult{
		Name:    "GeoSite/GeoIP 数据",
		Passed:  false,
		Problem: fmt.Sprintf("缺少: %s", missing),
		Suggest: "Mihomo 首次启动时会自动下载，或使用 'clashctl init' 预下载",
	}
}

// CheckTUNPermission checks if we can actually create a TUN interface.
func CheckTUNPermission() error {
	fd, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		if pe, ok := err.(*os.PathError); ok {
			if errno, ok := pe.Err.(syscall.Errno); ok {
				if errno == syscall.EPERM || errno == syscall.EACCES {
					return fmt.Errorf("权限不足，无法打开 /dev/net/tun。请使用 sudo 运行")
				}
			}
		}
		return fmt.Errorf("无法打开 /dev/net/tun: %w", err)
	}
	fd.Close()
	return nil
}
