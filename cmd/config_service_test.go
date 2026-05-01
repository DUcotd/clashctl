package cmd

import (
	"errors"
	"testing"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
	"clashctl/internal/subscription"
)

func TestBuildExportReport(t *testing.T) {
	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = "https://example.com/sub"
	cfg.Mode = "mixed"
	cfg.MixedPort = 7890

	report := buildExportReport(cfg, "/tmp/config.yaml")
	if report.SubscriptionURL != cfg.SubscriptionURL || report.Mode != "mixed" || report.MixedPort != 7890 {
		t.Fatalf("report = %#v", report)
	}
	if report.OutputPath != "/tmp/config.yaml" || report.Written {
		t.Fatalf("report = %#v", report)
	}
}

func TestFinishInstallReportStoresError(t *testing.T) {
	prev := installJSON
	t.Cleanup(func() { installJSON = prev })
	installJSON = false
	report := &installRunReport{}
	err := finishReport(report, errors.New("need sudo"), false)
	if err == nil || err.Error() != "need sudo" {
		t.Fatalf("finishReport() error = %v", err)
	}
	if report.Error != "need sudo" {
		t.Fatalf("report.Error = %q", report.Error)
	}
}

func TestBuildConfigPathReport(t *testing.T) {
	cfg := &core.AppConfig{ConfigDir: "/etc/mihomo", ProviderPath: "providers/airport.yaml"}
	report := buildConfigPathReport(cfg)

	if report.ConfigDir != "/etc/mihomo" {
		t.Fatalf("ConfigDir = %q", report.ConfigDir)
	}
	if report.ConfigPath != "/etc/mihomo/config.yaml" {
		t.Fatalf("ConfigPath = %q", report.ConfigPath)
	}
	if report.ProviderPath != "/etc/mihomo/providers/airport.yaml" {
		t.Fatalf("ProviderPath = %q", report.ProviderPath)
	}
}

func TestPopulateImportReport(t *testing.T) {
	report := &importRunReport{}
	cfg := &core.AppConfig{Mode: "mixed", MixedPort: 7890}
	plan := &subscription.ResolvedConfigPlan{
		Kind:            subscription.PlanKindStatic,
		ContentKind:     "base64-links",
		DetectedFormat:  "vless",
		Summary:         "parsed 2 proxies",
		ProxyCount:      2,
		VerifyInventory: true,
		UsedProxyEnv:    true,
	}

	populateImportReport(report, cfg, plan)
	if report.PlanKind != "static" || report.ContentKind != "base64-links" || report.DetectedFormat != "vless" {
		t.Fatalf("report = %#v", report)
	}
	if report.ProxyCount != 2 || !report.VerifyInventory || !report.UsedProxyEnv {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildRuntimeStartJSONReport(t *testing.T) {
	report := buildRuntimeStartJSONReport(&mihomo.StartResult{
		Binary:            &mihomo.InstallResult{Path: "/usr/local/bin/mihomo", Version: "1.19.10", ReleaseTag: "v1.19.10", Installed: true},
		GeoData:           &mihomo.GeoDataResult{Downloaded: 2, Files: []mihomo.GeoDataFileResult{{Name: "geoip.dat", Downloaded: true, Required: true}}},
		GeoDataError:      "",
		StartedBy:         "process",
		ServiceStopped:    true,
		ProcessStopped:    true,
		ControllerReady:   true,
		ControllerVersion: "1.19.10",
		Inventory:         &mihomo.ProxyInventory{Loaded: 3, Current: "Node A", Candidates: []string{"Node A", "Node B"}},
		InventoryError:    "",
		Warnings:          []string{"warning one"},
	})

	if report == nil || report.Binary == nil || report.GeoData == nil || report.Inventory == nil {
		t.Fatalf("report = %#v", report)
	}
	if report.Binary.Path != "/usr/local/bin/mihomo" || report.GeoData.Downloaded != 2 || report.Inventory.Loaded != 3 {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Warnings) != 1 || report.Warnings[0] != "warning one" {
		t.Fatalf("warnings = %#v", report.Warnings)
	}
}
