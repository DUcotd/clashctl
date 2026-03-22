package cmd

import (
	"fmt"
	"io"
	"strings"

	"clashctl/internal/mihomo"
)

func printRuntimeStartResult(w io.Writer, result *mihomo.StartResult) {
	if result == nil {
		return
	}
	if result.Binary != nil && result.Binary.Installed {
		fmt.Fprintf(w, "📦 已自动安装 Mihomo: %s", result.Binary.Path)
		if result.Binary.Version != "" {
			fmt.Fprintf(w, " (%s)", result.Binary.Version)
		}
		fmt.Fprintln(w)
	}
	if result.GeoData != nil && result.GeoData.Downloaded > 0 {
		fmt.Fprintf(w, "📦 已下载 %d 个 GeoSite/GeoIP 数据文件\n", result.GeoData.Downloaded)
	}
	if result.GeoDataError != "" {
		fmt.Fprintf(w, "⚠️  GeoData 预下载失败: %s (Mihomo 会自动重试)\n", result.GeoDataError)
	}
	if result.ServiceStopped {
		fmt.Fprintln(w, "🧹 已停止旧的 systemd 服务")
	}
	if result.ProcessStopped {
		fmt.Fprintln(w, "🧹 已清理旧进程")
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintf(w, "⚠️  %s\n", strings.Join(result.Warnings, "\n"))
	}

	switch result.StartedBy {
	case "systemd":
		fmt.Fprintln(w, "✅ 通过 systemd 启动成功")
	case "process":
		fmt.Fprintln(w, "✅ Mihomo 已以子进程方式启动")
	}

	if result.ControllerReady {
		fmt.Fprint(w, "✅ Controller API 可达")
		if result.ControllerVersion != "" {
			fmt.Fprintf(w, " (Mihomo %s)", result.ControllerVersion)
		}
		fmt.Fprintln(w)
	}

	if result.Inventory != nil && result.Inventory.OnlyCompatible {
		fmt.Fprintln(w, "⚠️  Controller API 已启动，但订阅节点未成功加载；当前仅剩 COMPATIBLE")
		fmt.Fprintln(w, "   建议: 在本地下载订阅后执行 'clashctl config import --file sub.txt --apply --start'")
	} else if result.Inventory != nil {
		fmt.Fprintf(w, "✅ PROXY 已加载 %d 个节点\n", result.Inventory.Loaded)
		if result.Inventory.Current != "" {
			fmt.Fprintf(w, "   当前节点: %s\n", result.Inventory.Current)
		}
	}
}

func printInstallStatus(w io.Writer, path, version string) {
	fmt.Fprintf(w, "Mihomo 已安装: %s", path)
	if version != "" {
		fmt.Fprintf(w, " (%s)", version)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "如需重新安装，请先卸载当前版本")
}

func printInstallResult(w io.Writer, result *mihomo.InstallResult) {
	if result == nil {
		return
	}
	fmt.Fprintf(w, "✅ Mihomo 已安装到: %s\n", result.Path)
	if result.Version != "" {
		fmt.Fprintf(w, "   版本: %s\n", result.Version)
	}
	if result.ReleaseTag != "" {
		fmt.Fprintf(w, "   发布版本: %s\n", result.ReleaseTag)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "💡 运行 'sudo clashctl init' 开始配置")
}
