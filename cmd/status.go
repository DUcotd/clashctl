package cmd

import (
	"fmt"
	"io"
	"os"

	"clashctl/internal/core"
	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
	"clashctl/internal/system"
)

var statusJSON bool

type statusServiceReport struct {
	Active bool   `json:"active"`
	Mode   string `json:"mode"`
	Error  string `json:"error,omitempty"`
}

type statusBinaryReport struct {
	Found   bool   `json:"found"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}

type statusConfigReport struct {
	Dir            string `json:"dir"`
	Mode           string `json:"mode"`
	MixedPort      int    `json:"mixed_port,omitempty"`
	ControllerAddr string `json:"controller_addr"`
}

type statusProxyEnvReport struct {
	Configured bool     `json:"configured"`
	Entries    []string `json:"entries,omitempty"`
	Messages   []string `json:"messages,omitempty"`
}

type statusControllerReport struct {
	Reachable bool   `json:"reachable"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
}

type statusProxyGroupReport struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Current   string `json:"current,omitempty"`
	NodeCount int    `json:"node_count"`
}

type statusInventoryReport struct {
	Loaded         int      `json:"loaded"`
	Current        string   `json:"current,omitempty"`
	Candidates     []string `json:"candidates,omitempty"`
	OnlyCompatible bool     `json:"only_compatible"`
}

type statusReport struct {
	Service        statusServiceReport      `json:"service"`
	Binary         statusBinaryReport       `json:"binary"`
	Config         statusConfigReport       `json:"config"`
	ProxyEnv       statusProxyEnvReport     `json:"proxy_env"`
	Controller     statusControllerReport   `json:"controller"`
	Groups         []statusProxyGroupReport `json:"groups,omitempty"`
	GroupsError    string                   `json:"groups_error,omitempty"`
	Inventory      *statusInventoryReport   `json:"inventory,omitempty"`
	InventoryError string                   `json:"inventory_error,omitempty"`
	Warnings       []string                 `json:"warnings,omitempty"`
}

func runStatus(cmd *cobra.Command, args []string, cfg *core.AppConfig) error {
	proxyEnv := system.ProxyEnvForDisplay()

	// Check systemd service
	serviceActive := false
	var serviceErr error
	if mihomo.HasSystemd() {
		serviceActive, serviceErr = mihomo.ServiceStatus(mihomo.DefaultServiceName)
	}

	client := mihomo.NewClientWithSecret("http://"+cfg.ControllerAddr, cfg.ControllerSecret)
	controllerErr := client.CheckConnection()
	controllerOK := controllerErr == nil
	controllerVersion := ""
	if controllerOK {
		controllerVersion, _ = client.Version()
	}

	// Check binary
	binary, err := mihomo.FindBinary()
	binaryVersion := ""
	if err == nil {
		binaryVersion, _ = mihomo.GetBinaryVersion()
	}

	var groups map[string]mihomo.ProxyGroup
	var groupsErr error
	var inventory *mihomo.ProxyInventory
	var inventoryErr error
	if controllerOK {
		groups, groupsErr = client.GetAllProxyGroups()
		inventory, inventoryErr = client.InspectProxyInventory("PROXY")
	}

	report := buildStatusReport(&statusReportInput{
		Cfg: cfg, ProxyEnv: proxyEnv,
		ServiceActive: serviceActive, ServiceErr: serviceErr,
		Binary: binary, BinaryVersion: binaryVersion, BinaryErr: err,
		ControllerVersion: controllerVersion, ControllerErr: controllerErr,
		Groups: groups, GroupsErr: groupsErr,
		Inventory: inventory, InventoryErr: inventoryErr,
	})
	if statusJSON {
		return writeJSON(report)
	}

	fmt.Println("📊 Mihomo 状态")
	fmt.Println()
	return printStatusReport(os.Stdout, report)
}

type statusReportInput struct {
	Cfg               *core.AppConfig
	ProxyEnv          []string
	ServiceActive     bool
	ServiceErr        error
	Binary            string
	BinaryVersion     string
	BinaryErr         error
	ControllerVersion string
	ControllerErr     error
	Groups            map[string]mihomo.ProxyGroup
	GroupsErr         error
	Inventory         *mihomo.ProxyInventory
	InventoryErr      error
}

func buildStatusReport(in *statusReportInput) *statusReport {
	report := &statusReport{
		Service: statusServiceReport{},
		Binary: statusBinaryReport{
			Found: in.BinaryErr == nil,
			Path:  in.Binary,
		},
		Config: statusConfigReport{
			Dir:            in.Cfg.ConfigDir,
			Mode:           in.Cfg.Mode,
			ControllerAddr: in.Cfg.ControllerAddr,
		},
		ProxyEnv: statusProxyEnvReport{
			Configured: len(in.ProxyEnv) > 0,
			Entries:    append([]string(nil), in.ProxyEnv...),
			Messages:   proxyStatusLines(in.Cfg, in.ProxyEnv),
		},
		Controller: statusControllerReport{
			Reachable: in.ControllerErr == nil,
			Version:   in.ControllerVersion,
		},
	}
	if in.Cfg.Mode == "mixed" {
		report.Config.MixedPort = in.Cfg.MixedPort
	}
	if in.ServiceErr != nil {
		report.Service.Error = in.ServiceErr.Error()
	}
	if in.ServiceActive {
		report.Service.Active = true
		report.Service.Mode = "systemd"
	} else if in.ControllerErr == nil {
		report.Service.Active = true
		report.Service.Mode = "process"
	} else {
		report.Service.Mode = "stopped"
	}
	if in.BinaryErr != nil {
		report.Binary.Error = in.BinaryErr.Error()
	} else {
		report.Binary.Version = in.BinaryVersion
	}
	if in.ControllerErr != nil {
		report.Controller.Error = in.ControllerErr.Error()
	}
	if in.GroupsErr != nil {
		report.GroupsError = in.GroupsErr.Error()
	} else if len(in.Groups) > 0 {
		report.Groups = make([]statusProxyGroupReport, 0, len(in.Groups))
		for _, name := range sortedProxyGroupNames(in.Groups) {
			group := in.Groups[name]
			report.Groups = append(report.Groups, statusProxyGroupReport{
				Name:      name,
				Type:      mihomo.NormalizeProxyType(group.Type),
				Current:   group.Now,
				NodeCount: len(group.All),
			})
		}
	}
	if in.InventoryErr != nil {
		report.InventoryError = in.InventoryErr.Error()
	} else if in.Inventory != nil {
		report.Inventory = &statusInventoryReport{
			Loaded:         in.Inventory.Loaded,
			Current:        in.Inventory.Current,
			Candidates:     append([]string(nil), in.Inventory.Candidates...),
			OnlyCompatible: in.Inventory.OnlyCompatible,
		}
		if in.Inventory.OnlyCompatible {
			report.Warnings = append(report.Warnings, "订阅节点未成功加载；当前仅剩 COMPATIBLE，可改用 clashctl config import 生成静态配置")
		}
	}
	return report
}

func printStatusReport(w io.Writer, report *statusReport) error {
	if report.Service.Active {
		switch report.Service.Mode {
		case "systemd":
			fmt.Fprintln(w, "  服务状态: ✅ 运行中 (systemd)")
		case "process":
			fmt.Fprintln(w, "  服务状态: ✅ 运行中 (子进程/API 可达)")
		default:
			fmt.Fprintf(w, "  服务状态: ✅ 运行中 (%s)\n", report.Service.Mode)
		}
	} else {
		fmt.Fprintln(w, "  服务状态: ❌ 未运行")
	}
	if report.Service.Error != "" {
		fmt.Fprintf(w, "  systemd 检查: ⚠ %s\n", report.Service.Error)
	}

	if report.Binary.Found {
		fmt.Fprintf(w, "  可执行文件: ✅ %s\n", report.Binary.Path)
		if report.Binary.Version != "" {
			fmt.Fprintf(w, "  版本: %s\n", report.Binary.Version)
		}
	} else {
		fmt.Fprintf(w, "  可执行文件: ❌ %s\n", report.Binary.Error)
	}

	fmt.Fprintf(w, "  配置目录: %s\n", report.Config.Dir)
	fmt.Fprintf(w, "  运行模式: %s\n", modeLabel(report.Config.Mode))
	if report.Config.Mode == "mixed" {
		fmt.Fprintf(w, "  mixed-port: %d\n", report.Config.MixedPort)
	}
	for _, line := range report.ProxyEnv.Messages {
		fmt.Fprintf(w, "  %s\n", line)
	}

	if report.Controller.Reachable {
		fmt.Fprintf(w, "  Controller API: ✅ 可达")
		if report.Controller.Version != "" {
			fmt.Fprintf(w, " (Mihomo %s)", report.Controller.Version)
		}
		fmt.Fprintln(w)
	} else {
		fmt.Fprintf(w, "  Controller API: ❌ %s\n", report.Controller.Error)
	}

	if report.GroupsError != "" {
		fmt.Fprintf(w, "  代理组信息: ⚠ %s\n", report.GroupsError)
	}
	if len(report.Groups) > 0 {
		fmt.Fprintln(w, "\n  ── 代理组 ──")
		for _, group := range report.Groups {
			marker := "  "
			if group.Current != "" {
				marker = "▸ "
			}
			fmt.Fprintf(w, "\n  %s%s [%s]\n", marker, group.Name, group.Type)
			if group.Current != "" {
				fmt.Fprintf(w, "     当前: %s\n", group.Current)
			}
			fmt.Fprintf(w, "     节点数: %d\n", group.NodeCount)
		}
	}
	if report.Inventory != nil && report.Inventory.OnlyCompatible {
		fmt.Fprintln(w, "\n  ⚠ 订阅节点未成功加载；当前仅剩 COMPATIBLE。")
		fmt.Fprintln(w, "    常见原因: 服务器无法直连订阅 URL，或 provider 拉取失败。")
		fmt.Fprintln(w, "    可改用 'clashctl config import --file sub.txt -o /etc/mihomo/config.yaml' 生成静态配置。")
	}
	if report.InventoryError != "" {
		fmt.Fprintf(w, "\n  订阅节点检查: ⚠ %s\n", report.InventoryError)
	}
	return nil
}

func modeLabel(mode string) string {
	switch mode {
	case "tun":
		return "TUN (透明接管)"
	case "mixed":
		return "mixed-port"
	default:
		return mode
	}
}

func proxyStatusLines(cfg *core.AppConfig, proxyEnv []string) []string {
	if len(proxyEnv) > 0 {
		lines := []string{"Shell 代理: ✅ 已设置"}
		for _, entry := range proxyEnv {
			lines = append(lines, "  "+entry)
		}
		return lines
	}

	if cfg.Mode == "tun" {
		return []string{"Shell 代理: 未设置 (TUN 模式通常不需要)"}
	}

	return []string{
		"Shell 代理: ⚠ 未设置",
		fmt.Sprintf("  当前为 mixed-port 模式；像 codex/opencode/Node.js 这类 CLI 需要显式导出 HTTP_PROXY/HTTPS_PROXY/ALL_PROXY，并为 Node.js 额外设置 NODE_USE_ENV_PROXY=1（代理地址 127.0.0.1:%d）", cfg.MixedPort),
	}
}
