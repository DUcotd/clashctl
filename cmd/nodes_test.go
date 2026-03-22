package cmd

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"clashctl/internal/mihomo"
	nodeops "clashctl/internal/nodes"
)

func TestSortedProxyGroupNames(t *testing.T) {
	groups := map[string]mihomo.ProxyGroup{
		"GLOBAL": {Name: "GLOBAL"},
		"PROXY":  {Name: "PROXY"},
		"auto":   {Name: "auto"},
	}
	got := sortedProxyGroupNames(groups)
	want := []string{"GLOBAL", "PROXY", "auto"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortedProxyGroupNames() = %#v, want %#v", got, want)
	}
}

func TestPrintProxyGroupLatency(t *testing.T) {
	detail := &mihomo.ProxyGroupDetail{
		Name: "PROXY",
		Type: "Selector",
		Now:  "Node A",
		Nodes: []mihomo.ProxyNode{
			{Name: "Node A", Delay: 88, Selected: true},
			{Name: "Node B", Delay: 320, Selected: false},
		},
	}

	var buf bytes.Buffer
	printProxyGroupLatency(&buf, detail)
	out := buf.String()

	for _, want := range []string{
		"📡 代理组: PROXY (select)",
		"当前选中: Node A",
		"测速完成: 2 个节点",
		"Node A",
		"88ms ✨",
		"320ms ⚠️",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q in:\n%s", want, out)
		}
	}
}

func TestBuildNodesTestReportJSONIncludesNodes(t *testing.T) {
	report := buildNodesTestReport(8, []*mihomo.ProxyGroupDetail{{
		Name: "PROXY",
		Type: "Selector",
		Now:  "Node A",
		Nodes: []mihomo.ProxyNode{
			{Name: "Node A", Delay: 88, Selected: true},
		},
	}})

	if report.Concurrency != 8 {
		t.Fatalf("Concurrency = %d, want 8", report.Concurrency)
	}

	var buf bytes.Buffer
	if err := writeJSONTo(&buf, report); err != nil {
		t.Fatalf("writeJSONTo() error = %v", err)
	}
	out := buf.String()
	for _, want := range []string{`"concurrency": 8`, `"groups"`, `"nodes"`, `"delay": 88`} {
		if !strings.Contains(out, want) {
			t.Fatalf("JSON output missing %q in:\n%s", want, out)
		}
	}
}

func TestBuildNodesListReport(t *testing.T) {
	report := buildNodesListReport(&nodeops.GroupDetail{
		Name:    "PROXY",
		Type:    "Selector",
		Current: "Node A",
		Nodes: []nodeops.NodeEntry{
			{Name: "Node A", Selected: true},
			{Name: "Node B", Selected: false},
		},
	})

	if report.Group != "PROXY" || report.Type != "select" || report.Count != 2 {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Nodes) != 2 || !report.Nodes[0].Selected || report.Nodes[1].Selected {
		t.Fatalf("nodes = %#v", report.Nodes)
	}
}

func TestBuildNodesGroupsReportSorted(t *testing.T) {
	report := buildNodesGroupsReport([]nodeops.GroupSummary{
		{Name: "Auto", Type: "url-test", NodeCount: 1},
		{Name: "PROXY", Type: "Selector", Current: "Node A", NodeCount: 2},
	})

	if len(report.Groups) != 2 {
		t.Fatalf("Groups = %#v", report.Groups)
	}
	if report.Groups[0].Name != "Auto" || report.Groups[0].Type != "url-test" {
		t.Fatalf("first group = %#v", report.Groups[0])
	}
	if report.Groups[1].Name != "PROXY" || report.Groups[1].Current != "Node A" || report.Groups[1].NodeCount != 2 {
		t.Fatalf("second group = %#v", report.Groups[1])
	}
}
