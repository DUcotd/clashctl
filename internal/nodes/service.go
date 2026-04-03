package nodes

import (
	"sort"
	"strings"
	"sync"

	"clashctl/internal/mihomo"
)

type GroupSummary struct {
	Name      string
	Type      string
	Current   string
	NodeCount int
}

type NodeEntry struct {
	Name     string
	Protocol string
	Delay    int
	Selected bool
}

type GroupDetail struct {
	Name    string
	Type    string
	Current string
	Nodes   []NodeEntry
}

type TestProgress struct {
	Index  int
	Delay  int
	Tested int
	Total  int
	Done   bool
	Error  string
}

type Service interface {
	CheckConnection(controllerAddr string) error
	ListGroups(controllerAddr string) ([]GroupSummary, error)
	GetGroup(controllerAddr, groupName string) (*GroupDetail, error)
	SwitchNode(controllerAddr, groupName, nodeName string) error
	TestGroup(controllerAddr, groupName string, concurrency int) (*mihomo.ProxyGroupDetail, error)
	StreamNodeTests(controllerAddr, groupName string, nodes []NodeEntry, maxConcurrent int) <-chan TestProgress
}

type controllerClient interface {
	CheckConnection() error
	GetAllProxyGroups() (map[string]mihomo.ProxyGroup, error)
	GetProxyGroupDetail(name string) (*mihomo.ProxyGroupDetail, error)
	GetAllProxies() (map[string]mihomo.ProxyInfo, error)
	SwitchProxy(groupName, nodeName string) error
	TestProxyGroupNodes(groupName string, maxConcurrent int) (*mihomo.ProxyGroupDetail, error)
	TestNode(groupName, nodeName string) int
}

type defaultService struct {
	newClient func(controllerAddr string) controllerClient
}

func NewService() Service {
	return &defaultService{
		newClient: func(controllerAddr string) controllerClient {
			return mihomo.NewClient("http://" + controllerAddr)
		},
	}
}

func NewServiceWithSecret(secret string) Service {
	return &defaultService{
		newClient: func(controllerAddr string) controllerClient {
			return mihomo.NewClientWithSecret("http://"+controllerAddr, secret)
		},
	}
}

func (s *defaultService) CheckConnection(controllerAddr string) error {
	return s.newClient(controllerAddr).CheckConnection()
}

func (s *defaultService) ListGroups(controllerAddr string) ([]GroupSummary, error) {
	groups, err := s.newClient(controllerAddr).GetAllProxyGroups()
	if err != nil {
		return nil, err
	}

	items := make([]GroupSummary, 0, len(groups))
	for name, group := range groups {
		items = append(items, GroupSummary{
			Name:      name,
			Type:      group.Type,
			Current:   group.Now,
			NodeCount: len(group.All),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func (s *defaultService) GetGroup(controllerAddr, groupName string) (*GroupDetail, error) {
	client := s.newClient(controllerAddr)
	detail, err := client.GetProxyGroupDetail(groupName)
	if err != nil {
		return nil, err
	}

	typeMap := make(map[string]string)
	if allProxies, err := client.GetAllProxies(); err == nil {
		for name, info := range allProxies {
			typeMap[name] = info.Type
		}
	}

	group := &GroupDetail{
		Name:    detail.Name,
		Type:    detail.Type,
		Current: detail.Now,
		Nodes:   make([]NodeEntry, 0, len(detail.All)),
	}
	for _, name := range detail.All {
		group.Nodes = append(group.Nodes, NodeEntry{
			Name:     name,
			Protocol: typeMap[name],
			Selected: name == detail.Now,
		})
	}
	return group, nil
}

func (s *defaultService) SwitchNode(controllerAddr, groupName, nodeName string) error {
	return s.newClient(controllerAddr).SwitchProxy(groupName, nodeName)
}

func (s *defaultService) TestGroup(controllerAddr, groupName string, concurrency int) (*mihomo.ProxyGroupDetail, error) {
	return s.newClient(controllerAddr).TestProxyGroupNodes(groupName, concurrency)
}

func (s *defaultService) StreamNodeTests(controllerAddr, groupName string, nodes []NodeEntry, maxConcurrent int) <-chan TestProgress {
	stream := make(chan TestProgress)
	go func() {
		defer close(stream)
		total := len(nodes)
		if total == 0 {
			stream <- TestProgress{Done: true}
			return
		}
		if maxConcurrent <= 0 {
			maxConcurrent = 10
		}

		client := s.newClient(controllerAddr)
		sem := make(chan struct{}, maxConcurrent)
		results := make(chan TestProgress, total)
		var wg sync.WaitGroup

		for i, node := range nodes {
			wg.Add(1)
			sem <- struct{}{}
			go func(index int, nodeName string) {
				defer wg.Done()
				defer func() { <-sem }()
				results <- TestProgress{Index: index, Delay: client.TestNode(groupName, nodeName)}
			}(i, node.Name)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		tested := 0
		for result := range results {
			tested++
			result.Tested = tested
			result.Total = total
			stream <- result
		}

		stream <- TestProgress{Done: true, Tested: tested, Total: total}
	}()
	return stream
}
