package subscription

import (
	"fmt"
	"strings"
	"time"

	"clashctl/internal/config"
	"clashctl/internal/core"
	"clashctl/internal/system"

	"gopkg.in/yaml.v3"
)

// PlanKind identifies the resolved config style.
type PlanKind string

const (
	PlanKindStatic PlanKind = "static"
	PlanKindYAML   PlanKind = "yaml"
)

// ResolvedConfigPlan is a write-ready Mihomo config plan.
type ResolvedConfigPlan struct {
	Kind            PlanKind
	ContentKind     string
	DetectedFormat  string
	Summary         string
	FetchDetail     string
	UsedProxyEnv    bool
	ProxyCount      int
	VerifyInventory bool
	Warnings        []string
	RemovedFields   []string
	Sanitized       bool
	MihomoConfig    *core.MihomoConfig
	RawYAML         []byte
}

// RenderYAML renders the plan to YAML.
func (p *ResolvedConfigPlan) RenderYAML() ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("配置计划为空")
	}
	if len(p.RawYAML) > 0 {
		return append([]byte{}, p.RawYAML...), nil
	}
	if p.MihomoConfig == nil {
		return nil, fmt.Errorf("未生成可写入的配置")
	}
	return core.RenderYAML(p.MihomoConfig)
}

// Save writes the plan to disk with backup and validation.
func (p *ResolvedConfigPlan) Save(path string) (string, error) {
	if p == nil {
		return "", fmt.Errorf("配置计划为空")
	}
	if len(p.RawYAML) > 0 {
		return config.SaveRawYAML(p.RawYAML, path)
	}
	if p.MihomoConfig == nil {
		return "", fmt.Errorf("未生成可写入的配置")
	}
	return config.SaveMihomoConfig(p.MihomoConfig, path)
}

// Resolver resolves subscription inputs into write-ready config plans.
type Resolver struct {
	prepareURL func(string, time.Duration) (*system.PreparedSubscription, error)
}

// NewResolver creates a Resolver with the default remote fetcher.
func NewResolver() *Resolver {
	return &Resolver{
		prepareURL: system.PrepareSubscriptionURL,
	}
}

// ResolveRemoteURL resolves a remote subscription URL into a config plan.
func (r *Resolver) ResolveRemoteURL(cfg *core.AppConfig, rawURL string, timeout time.Duration) (*ResolvedConfigPlan, error) {
	prepared, err := r.prepareURL(rawURL, timeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = prepared.Cleanup()
	}()

	plan, err := r.ResolveContent(cfg, prepared.Body)
	if err != nil {
		contentKind := system.ProbeContentKind(prepared.Body)
		if contentKind == "unknown" && looksLikeProviderConfig(prepared.Body) {
			return nil, fmt.Errorf("检测到 provider-only 订阅；出于安全原因，已禁止保留远程 provider/rule-provider URL，请改用可直接展开为静态节点的订阅")
		}
		if contentKind == "html" || contentKind == "empty" {
			return nil, fmt.Errorf("订阅返回了不可用内容 (%s): %s", contentKind, previewSubscriptionBody(prepared.Body))
		}
		if contentKind == "unknown" {
			return nil, fmt.Errorf("订阅返回了无法识别的内容: %s", previewSubscriptionBody(prepared.Body))
		}
		return nil, err
	}

	plan.FetchDetail = prepared.FetchDetail
	plan.UsedProxyEnv = system.HasProxyEnvForDisplay()
	return plan, nil
}

// ResolveContent resolves raw subscription content into a config plan.
func (r *Resolver) ResolveContent(cfg *core.AppConfig, content []byte) (*ResolvedConfigPlan, error) {
	contentKind := system.ProbeContentKind(content)
	switch contentKind {
	case "raw-links", "base64-links":
		parsed, err := Parse(content)
		if err != nil {
			return nil, err
		}
		return &ResolvedConfigPlan{
			Kind:            PlanKindStatic,
			ContentKind:     contentKind,
			DetectedFormat:  parsed.DetectedFormat,
			Summary:         fmt.Sprintf("已解析 %d 个节点，使用静态配置", len(parsed.Names)),
			ProxyCount:      len(parsed.Names),
			VerifyInventory: true,
			MihomoConfig:    core.BuildStaticMihomoConfig(cfg, parsed.Proxies, parsed.Names),
		}, nil
	case "mihomo-yaml":
		patched, err := PatchRemoteYAML(content, cfg)
		if err != nil {
			return nil, err
		}
		return &ResolvedConfigPlan{
			Kind:            PlanKindYAML,
			ContentKind:     contentKind,
			DetectedFormat:  contentKind,
			Summary:         "检测到 Mihomo/Clash YAML，已转为本地静态配置",
			VerifyInventory: true,
			Warnings:        patched.Warnings,
			RemovedFields:   patched.RemovedFields,
			Sanitized:       patched.Sanitized,
			RawYAML:         patched.YAML,
		}, nil
	default:
		return nil, fmt.Errorf("未识别的订阅内容格式: %s", contentKind)
	}
}

func previewSubscriptionBody(body []byte) string {
	preview := strings.TrimSpace(string(body))
	if preview == "" {
		return "空响应"
	}
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	return preview
}

func looksLikeProviderConfig(body []byte) bool {
	var doc map[string]any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return false
	}
	_, hasProviders := doc["proxy-providers"]
	_, hasProxies := doc["proxies"]
	_, hasGroups := doc["proxy-groups"]
	return hasProviders && !hasProxies && !hasGroups
}
