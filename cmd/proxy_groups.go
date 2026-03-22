package cmd

import (
	"sort"

	"clashctl/internal/mihomo"
)

func sortedProxyGroupNames(groups map[string]mihomo.ProxyGroup) []string {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
