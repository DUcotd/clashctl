package system

import (
	"bufio"
	"strings"
)

// SplitLines splits text into non-empty trimmed lines.
// It handles both \n and \r\n line endings.
func SplitLines(text string) []string {
	s := bufio.NewScanner(strings.NewReader(text))
	var out []string
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
