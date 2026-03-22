package nodes

import (
	"testing"

	"clashctl/internal/mihomo"
)

type fakeControllerClient struct {
	groups      map[string]mihomo.ProxyGroup
	groupDetail *mihomo.ProxyGroupDetail
	proxies     map[string]mihomo.ProxyInfo
	switchGroup string
	switchNode  string
}

func (c *fakeControllerClient) CheckConnection() error { return nil }

func (c *fakeControllerClient) GetAllProxyGroups() (map[string]mihomo.ProxyGroup, error) {
	return c.groups, nil
}

func (c *fakeControllerClient) GetProxyGroupDetail(name string) (*mihomo.ProxyGroupDetail, error) {
	return c.groupDetail, nil
}

func (c *fakeControllerClient) GetAllProxies() (map[string]mihomo.ProxyInfo, error) {
	return c.proxies, nil
}

func (c *fakeControllerClient) SwitchProxy(groupName, nodeName string) error {
	c.switchGroup = groupName
	c.switchNode = nodeName
	return nil
}

func (c *fakeControllerClient) TestProxyGroupNodes(groupName string, maxConcurrent int) (*mihomo.ProxyGroupDetail, error) {
	return c.groupDetail, nil
}

func (c *fakeControllerClient) TestNode(groupName, nodeName string) int {
	return 42
}

func TestListGroupsSortsNames(t *testing.T) {
	svc := &defaultService{newClient: func(string) controllerClient {
		return &fakeControllerClient{groups: map[string]mihomo.ProxyGroup{
			"zeta":  {Type: "Selector", Now: "B", All: []string{"B"}},
			"Alpha": {Type: "Selector", Now: "A", All: []string{"A", "B"}},
		}}
	}}

	groups, err := svc.ListGroups("127.0.0.1:9090")
	if err != nil {
		t.Fatalf("ListGroups() error = %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d", len(groups))
	}
	if groups[0].Name != "Alpha" || groups[1].Name != "zeta" {
		t.Fatalf("groups = %#v", groups)
	}
	if groups[0].NodeCount != 2 || groups[1].Current != "B" {
		t.Fatalf("groups = %#v", groups)
	}
}

func TestGetGroupBuildsNodeEntries(t *testing.T) {
	svc := &defaultService{newClient: func(string) controllerClient {
		return &fakeControllerClient{
			groupDetail: &mihomo.ProxyGroupDetail{Name: "PROXY", Type: "Selector", Now: "Node B", All: []string{"Node A", "Node B"}},
			proxies: map[string]mihomo.ProxyInfo{
				"Node A": {Type: "ss"},
				"Node B": {Type: "vmess"},
			},
		}
	}}

	group, err := svc.GetGroup("127.0.0.1:9090", "PROXY")
	if err != nil {
		t.Fatalf("GetGroup() error = %v", err)
	}
	if group == nil || group.Name != "PROXY" || len(group.Nodes) != 2 {
		t.Fatalf("group = %#v", group)
	}
	if group.Nodes[0].Protocol != "ss" || group.Nodes[1].Protocol != "vmess" {
		t.Fatalf("group = %#v", group)
	}
	if group.Nodes[0].Selected || !group.Nodes[1].Selected {
		t.Fatalf("group = %#v", group)
	}
}
