package mihomo

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"clashctl/internal/system"
)

const openAIDoctorTimeout = 8 * time.Second

// OpenAIDoctorReport contains OpenAI/Codex-specific diagnostic checks and hints.
type OpenAIDoctorReport struct {
	Results []CheckResult
	Hints   []string
}

// RunOpenAIDoctor diagnoses the current shell/proxy path for OpenAI/Codex login.
func RunOpenAIDoctor(mixedPort int) *OpenAIDoctorReport {
	report := &OpenAIDoctorReport{}

	report.Results = append(report.Results, checkShellProxyEnv())

	directClient := system.NewHTTPClient(openAIDoctorTimeout, true)
	directEgress, _ := checkEgress("直连出口地区", directClient)
	report.Results = append(report.Results, directEgress)

	directAuth, directAuthOK := checkOpenAIAuth("auth.openai.com（直连）", directClient)
	report.Results = append(report.Results, directAuth)

	directAPI, directAPIOK := checkOpenAIAPI("api.openai.com（直连）", directClient)
	report.Results = append(report.Results, directAPI)

	var proxyCountry string
	var proxyCountryCode string
	proxyAvailable := false
	proxyAuthOK := false
	proxyAPIOK := false

	if mixedPort > 0 {
		proxyURL := fmt.Sprintf("http://127.0.0.1:%d", mixedPort)
		proxyClient, err := system.NewProxyHTTPClient(openAIDoctorTimeout, proxyURL)
		if err != nil {
			report.Results = append(report.Results, CheckResult{
				Name:    "本地 mixed 代理",
				Passed:  false,
				Problem: err.Error(),
				Suggest: "请确认 mixed-port 配置正确",
			})
		} else {
			proxyAvailable = true
			proxyEgress, ok := checkEgress("代理出口地区", proxyClient)
			report.Results = append(report.Results, proxyEgress)
			if ok {
				proxyCountry, proxyCountryCode = extractCountry(proxyEgress.Problem)
			}

			proxyAuth, ok := checkOpenAIAuth("auth.openai.com（代理）", proxyClient)
			proxyAuthOK = ok
			report.Results = append(report.Results, proxyAuth)

			proxyAPI, ok := checkOpenAIAPI("api.openai.com（代理）", proxyClient)
			proxyAPIOK = ok
			report.Results = append(report.Results, proxyAPI)
		}
	}

	directCountry, directCountryCode := extractCountry(directEgress.Problem)
	report.Hints = buildOpenAIHints(
		system.HasProxyEnvForDisplay(),
		proxyAvailable,
		directAuthOK,
		proxyAuthOK,
		directAPIOK,
		proxyAPIOK,
		directCountry,
		directCountryCode,
		proxyCountry,
		proxyCountryCode,
	)

	return report
}

func checkShellProxyEnv() CheckResult {
	envs := system.ProxyEnvForDisplay()
	if len(envs) == 0 {
		return CheckResult{
			Name:    "Shell 代理环境",
			Passed:  false,
			Problem: "当前 shell 未导出 HTTP_PROXY/HTTPS_PROXY/ALL_PROXY",
			Suggest: "如果使用 mixed-port，请先执行 'source ~/.bashrc' 或重新打开终端",
		}
	}
	return CheckResult{
		Name:    "Shell 代理环境",
		Passed:  true,
		Problem: strings.Join(envs, "; "),
	}
}

func checkEgress(name string, client *http.Client) (CheckResult, bool) {
	info, err := system.DetectEgressInfo(client)
	if err != nil {
		return CheckResult{
			Name:    name,
			Passed:  false,
			Problem: err.Error(),
			Suggest: "请检查当前出口是否能访问公共回显服务，或切换代理节点后重试",
		}, false
	}

	detail := info.Country
	if info.CountryCode != "" {
		if detail != "" {
			detail += " (" + info.CountryCode + ")"
		} else {
			detail = info.CountryCode
		}
	}
	if info.IP != "" {
		if detail != "" {
			detail += " - "
		}
		detail += info.IP
	}
	if info.Source != "" {
		detail += " via " + info.Source
	}

	return CheckResult{
		Name:    name,
		Passed:  true,
		Problem: detail,
	}, true
}

func checkOpenAIAuth(name string, client *http.Client) (CheckResult, bool) {
	probe, err := system.ProbeEndpoint(client, "https://auth.openai.com/.well-known/openid-configuration")
	if err != nil {
		return CheckResult{
			Name:    name,
			Passed:  false,
			Problem: err.Error(),
			Suggest: "这通常表示当前链路到 auth.openai.com 有问题；如果直连正常而代理失败，请切换节点或给 auth.openai.com 走直连规则",
		}, false
	}

	if strings.Contains(strings.ToLower(probe.BodyPreview), "unsupported_country_region_territory") {
		return CheckResult{
			Name:    name,
			Passed:  false,
			Problem: fmt.Sprintf("HTTP %d，返回 unsupported_country_region_territory", probe.StatusCode),
			Suggest: "当前出口国家/地区可能不在 OpenAI 支持范围内，请切换到支持地区的出口后重试",
		}, false
	}

	if probe.StatusCode != http.StatusOK {
		return CheckResult{
			Name:    name,
			Passed:  false,
			Problem: fmt.Sprintf("HTTP %d (%s)", probe.StatusCode, probe.FinalURL),
			Suggest: "auth.openai.com 已可达但返回异常状态，请检查节点、WAF 或上游出口策略",
		}, false
	}

	return CheckResult{
		Name:    name,
		Passed:  true,
		Problem: fmt.Sprintf("HTTP %d (%s)", probe.StatusCode, probe.FinalURL),
	}, true
}

func checkOpenAIAPI(name string, client *http.Client) (CheckResult, bool) {
	probe, err := system.ProbeEndpoint(client, "https://api.openai.com/v1/models")
	if err != nil {
		return CheckResult{
			Name:    name,
			Passed:  false,
			Problem: err.Error(),
			Suggest: "这通常表示当前链路到 api.openai.com 有问题；如果只在代理路径失败，请切换节点或调整规则",
		}, false
	}

	bodyLower := strings.ToLower(probe.BodyPreview)
	if strings.Contains(bodyLower, "unsupported_country_region_territory") {
		return CheckResult{
			Name:    name,
			Passed:  false,
			Problem: fmt.Sprintf("HTTP %d，返回 unsupported_country_region_territory", probe.StatusCode),
			Suggest: "当前出口国家/地区可能不在 OpenAI 支持范围内，请切换到支持地区的出口后重试",
		}, false
	}

	if probe.StatusCode == http.StatusOK || probe.StatusCode == http.StatusUnauthorized || probe.StatusCode == http.StatusForbidden {
		detail := fmt.Sprintf("HTTP %d (%s)", probe.StatusCode, probe.FinalURL)
		if probe.StatusCode == http.StatusUnauthorized {
			detail += "，说明网络可达，未携带 API key 属于预期"
		}
		return CheckResult{
			Name:    name,
			Passed:  true,
			Problem: detail,
		}, true
	}

	return CheckResult{
		Name:    name,
		Passed:  false,
		Problem: fmt.Sprintf("HTTP %d (%s)", probe.StatusCode, probe.FinalURL),
		Suggest: "api.openai.com 可达但返回异常状态，请检查当前出口或稍后重试",
	}, false
}

func buildOpenAIHints(shellProxy, proxyAvailable, directAuthOK, proxyAuthOK, directAPIOK, proxyAPIOK bool, directCountry, directCountryCode, proxyCountry, proxyCountryCode string) []string {
	var hints []string

	if !shellProxy {
		hints = append(hints, "当前 shell 没有代理环境；如果你依赖 mixed-port 登录 Codex/OpenCode，新开终端前先 source 一次 shell 配置。")
	}
	if proxyAvailable && directAuthOK && !proxyAuthOK {
		hints = append(hints, "直连 auth.openai.com 正常、代理路径失败，问题通常在当前代理节点或规则，而不是 OpenAI OAuth 回调本身。")
	}
	if proxyAvailable && directAPIOK && !proxyAPIOK {
		hints = append(hints, "直连 api.openai.com 正常、代理路径失败，说明 token 交换或后续 API 请求经代理会被当前出口拦截。")
	}
	if directCountryCode != "" && proxyCountryCode != "" && directCountryCode != proxyCountryCode {
		hints = append(hints, fmt.Sprintf("直连出口是 %s，代理出口是 %s；OpenAI 最终按实际发起 token 交换的出口地区做判定。", formatCountry(directCountry, directCountryCode), formatCountry(proxyCountry, proxyCountryCode)))
	}
	if proxyAvailable && proxyCountryCode == "" {
		hints = append(hints, "代理链路连出口地区都拿不到，优先怀疑当前 PROXY 节点或 mixed-port 转发本身。")
	}

	return hints
}

func extractCountry(detail string) (country string, code string) {
	if idx := strings.Index(detail, " via "); idx >= 0 {
		detail = detail[:idx]
	}
	if idx := strings.Index(detail, " - "); idx >= 0 {
		detail = detail[:idx]
	}
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return "", ""
	}
	open := strings.LastIndex(detail, "(")
	close := strings.LastIndex(detail, ")")
	if open >= 0 && close > open {
		country = strings.TrimSpace(detail[:open])
		code = strings.TrimSpace(detail[open+1 : close])
		return country, code
	}
	return detail, ""
}

func formatCountry(country, code string) string {
	if country == "" {
		return code
	}
	if code == "" {
		return country
	}
	return country + " (" + code + ")"
}
