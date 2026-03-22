package cmd

import (
	"bytes"
	"strings"
	"testing"

	"clashctl/internal/mihomo"
)

func TestPrintRuntimeStartResult(t *testing.T) {
	result := &mihomo.StartResult{
		Binary: &mihomo.InstallResult{
			Path:      "/usr/local/bin/mihomo",
			Version:   "1.19.10",
			Installed: true,
		},
		GeoData: &mihomo.GeoDataResult{Downloaded: 2},
		Warnings: []string{
			"warning one",
			"warning two",
		},
		StartedBy:         "process",
		ControllerReady:   true,
		ControllerVersion: "1.19.10",
		Inventory: &mihomo.ProxyInventory{
			Loaded:  3,
			Current: "Node A",
		},
	}

	var buf bytes.Buffer
	printRuntimeStartResult(&buf, result)
	out := buf.String()

	for _, want := range []string{
		"已自动安装 Mihomo: /usr/local/bin/mihomo (1.19.10)",
		"已下载 2 个 GeoSite/GeoIP 数据文件",
		"warning one\nwarning two",
		"Mihomo 已以子进程方式启动",
		"Controller API 可达 (Mihomo 1.19.10)",
		"PROXY 已加载 3 个节点",
		"当前节点: Node A",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q in:\n%s", want, out)
		}
	}
}

func TestPrintRuntimeStartResultCompatibleOnly(t *testing.T) {
	result := &mihomo.StartResult{
		Inventory: &mihomo.ProxyInventory{OnlyCompatible: true},
	}

	var buf bytes.Buffer
	printRuntimeStartResult(&buf, result)
	out := buf.String()

	if !strings.Contains(out, "当前仅剩 COMPATIBLE") {
		t.Fatalf("output missing compatible warning in:\n%s", out)
	}
	if !strings.Contains(out, "clashctl config import --file sub.txt --apply --start") {
		t.Fatalf("output missing import fallback hint in:\n%s", out)
	}
}

func TestPrintInstallStatus(t *testing.T) {
	var buf bytes.Buffer
	printInstallStatus(&buf, "/usr/local/bin/mihomo", "1.19.10")
	out := buf.String()

	if !strings.Contains(out, "Mihomo 已安装: /usr/local/bin/mihomo (1.19.10)") {
		t.Fatalf("unexpected output:\n%s", out)
	}
	if !strings.Contains(out, "如需重新安装，请先卸载当前版本") {
		t.Fatalf("missing reinstall hint:\n%s", out)
	}
}

func TestPrintInstallResult(t *testing.T) {
	var buf bytes.Buffer
	printInstallResult(&buf, &mihomo.InstallResult{
		Path:       "/usr/local/bin/mihomo",
		Version:    "1.19.10",
		ReleaseTag: "v1.19.10",
	})
	out := buf.String()

	for _, want := range []string{
		"Mihomo 已安装到: /usr/local/bin/mihomo",
		"版本: 1.19.10",
		"发布版本: v1.19.10",
		"运行 'sudo clashctl init' 开始配置",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q in:\n%s", want, out)
		}
	}
}
