// Package mihomo provides environment diagnostic functions (doctor).
package mihomo

import (
	"fmt"
	"os"
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

	return results
}

func checkBinary() CheckResult {
	binary, err := FindBinary()
	if err != nil {
		return CheckResult{
			Name:    "mihomo 可执行文件",
			Passed:  false,
			Problem: err.Error(),
			Suggest: "请安装 Mihomo：https://github.com/MetaCubeX/mihomo/releases",
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
		return CheckResult{
			Name:    "controller 端口",
			Passed:  false,
			Problem: fmt.Sprintf("端口 %s 已被占用", controllerAddr),
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
			Passed:  false,
			Problem: "未检测到 systemd",
			Suggest: "此系统不支持 systemd，将无法使用服务管理模式",
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
