package cmd

import "strings"

// normalizePaths expands comma-separated path lists and trims whitespace.
// It preserves ordering and drops empty entries.
func normalizePaths(raw []string) []string {
	paths := make([]string, 0, len(raw))
	for _, v := range raw {
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			paths = append(paths, p)
		}
	}
	return paths
}
