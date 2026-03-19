package cmd

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"clashctl/internal/mihomo"
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
