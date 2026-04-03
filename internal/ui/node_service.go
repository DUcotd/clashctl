package ui

import (
	nodeops "clashctl/internal/nodes"
)

// NodeService hides controller operations behind a UI-friendly interface.
type NodeService interface {
	LoadGroups(controllerAddr string) ([]GroupItem, error)
	LoadNodes(controllerAddr, groupName string) ([]NodeItem, error)
	SwitchNode(controllerAddr, groupName, nodeName string) error
	StartNodeTest(controllerAddr, groupName string, nodes []NodeItem, maxConcurrent int) <-chan nodeTestProgressMsg
}

type defaultNodeService struct {
	shared nodeops.Service
}

func newDefaultNodeService(controllerSecret string) NodeService {
	return &defaultNodeService{
		shared: nodeops.NewServiceWithSecret(controllerSecret),
	}
}

func (s *defaultNodeService) LoadGroups(controllerAddr string) ([]GroupItem, error) {
	groups, err := s.shared.ListGroups(controllerAddr)
	if err != nil {
		return nil, err
	}

	items := make([]GroupItem, 0, len(groups))
	for _, group := range groups {
		items = append(items, GroupItem{
			Name:      group.Name,
			Type:      group.Type,
			Now:       group.Current,
			NodeCount: group.NodeCount,
		})
	}
	return items, nil
}

func (s *defaultNodeService) LoadNodes(controllerAddr, groupName string) ([]NodeItem, error) {
	group, err := s.shared.GetGroup(controllerAddr, groupName)
	if err != nil {
		return nil, err
	}

	items := make([]NodeItem, 0, len(group.Nodes))
	for _, node := range group.Nodes {
		items = append(items, NodeItem{
			Name:     node.Name,
			Protocol: node.Protocol,
			Selected: node.Selected,
		})
	}

	return items, nil
}

func (s *defaultNodeService) SwitchNode(controllerAddr, groupName, nodeName string) error {
	return s.shared.SwitchNode(controllerAddr, groupName, nodeName)
}

func (s *defaultNodeService) StartNodeTest(controllerAddr, groupName string, nodes []NodeItem, maxConcurrent int) <-chan nodeTestProgressMsg {
	stream := make(chan nodeTestProgressMsg)
	go func() {
		defer close(stream)
		entries := make([]nodeops.NodeEntry, 0, len(nodes))
		for _, node := range nodes {
			entries = append(entries, nodeops.NodeEntry{Name: node.Name, Protocol: node.Protocol, Delay: node.Delay, Selected: node.Selected})
		}
		for result := range s.shared.StreamNodeTests(controllerAddr, groupName, entries, maxConcurrent) {
			stream <- nodeTestProgressMsg{
				index:  result.Index,
				delay:  result.Delay,
				tested: result.Tested,
				total:  result.Total,
				done:   result.Done,
				err:    result.Error,
			}
		}
	}()
	return stream
}
