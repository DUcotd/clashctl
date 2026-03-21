package ui

import (
	"sort"
	"strings"
	"sync"

	"clashctl/internal/mihomo"
)

// NodeService hides controller operations behind a UI-friendly interface.
type NodeService interface {
	LoadGroups(controllerAddr string) ([]GroupItem, error)
	LoadNodes(controllerAddr, groupName string) ([]NodeItem, error)
	SwitchNode(controllerAddr, groupName, nodeName string) error
	StartNodeTest(controllerAddr, groupName string, nodes []NodeItem, maxConcurrent int) <-chan nodeTestProgressMsg
}

type controllerClient interface {
	GetAllProxyGroups() (map[string]mihomo.ProxyGroup, error)
	GetProxyGroupDetail(name string) (*mihomo.ProxyGroupDetail, error)
	GetAllProxies() (map[string]mihomo.ProxyInfo, error)
	SwitchProxy(groupName, nodeName string) error
	TestNode(groupName, nodeName string) int
}

type defaultNodeService struct {
	newClient func(controllerAddr string) controllerClient
}

func newDefaultNodeService() NodeService {
	return &defaultNodeService{
		newClient: func(controllerAddr string) controllerClient {
			return mihomo.NewClient("http://" + controllerAddr)
		},
	}
}

func (s *defaultNodeService) LoadGroups(controllerAddr string) ([]GroupItem, error) {
	client := s.newClient(controllerAddr)
	groups, err := client.GetAllProxyGroups()
	if err != nil {
		return nil, err
	}

	items := make([]GroupItem, 0, len(groups))
	for name, group := range groups {
		items = append(items, GroupItem{
			Name:      name,
			Type:      group.Type,
			Now:       group.Now,
			NodeCount: len(group.All),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func (s *defaultNodeService) LoadNodes(controllerAddr, groupName string) ([]NodeItem, error) {
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

	items := make([]NodeItem, 0, len(detail.All))
	for _, name := range detail.All {
		items = append(items, NodeItem{
			Name:     name,
			Protocol: typeMap[name],
			Selected: name == detail.Now,
		})
	}

	return items, nil
}

func (s *defaultNodeService) SwitchNode(controllerAddr, groupName, nodeName string) error {
	return s.newClient(controllerAddr).SwitchProxy(groupName, nodeName)
}

func (s *defaultNodeService) StartNodeTest(controllerAddr, groupName string, nodes []NodeItem, maxConcurrent int) <-chan nodeTestProgressMsg {
	stream := make(chan nodeTestProgressMsg)
	go func() {
		defer close(stream)
		total := len(nodes)
		if total == 0 {
			stream <- nodeTestProgressMsg{done: true}
			return
		}
		if maxConcurrent <= 0 {
			maxConcurrent = 10
		}

		client := s.newClient(controllerAddr)
		sem := make(chan struct{}, maxConcurrent)
		results := make(chan nodeTestProgressMsg, total)
		var wg sync.WaitGroup

		for i, node := range nodes {
			wg.Add(1)
			sem <- struct{}{}
			go func(index int, nodeName string) {
				defer wg.Done()
				defer func() { <-sem }()
				results <- nodeTestProgressMsg{
					index: index,
					delay: client.TestNode(groupName, nodeName),
				}
			}(i, node.Name)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		tested := 0
		for result := range results {
			tested++
			result.tested = tested
			result.total = total
			stream <- result
		}

		stream <- nodeTestProgressMsg{done: true, tested: tested, total: total}
	}()
	return stream
}
