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

// splitTrailingPaths separates trailing path-like tokens from the text when the path
// flag was provided but Cobra left extra tokens inside the text positional.
func splitTrailingPaths(text string, existing []string) (string, []string) {
	tokens := strings.Fields(text)
	if len(tokens) == 0 {
		return text, existing
	}

	var trailing []string
	// walk backward collecting path-ish tokens
	for i := len(tokens) - 1; i >= 1; i-- { // keep at least one token for text
		if looksLikePath(tokens[i]) {
			trailing = append(trailing, tokens[i])
			tokens = tokens[:i]
		} else {
			break
		}
	}

	if len(trailing) == 0 {
		return text, existing
	}

	// reverse to restore original order
	for i, j := 0, len(trailing)-1; i < j; i, j = i+1, j-1 {
		trailing[i], trailing[j] = trailing[j], trailing[i]
	}

	newText := strings.Join(tokens, " ")
	paths := append(existing, trailing...)
	return newText, paths
}

func looksLikePath(token string) bool {
	return strings.Contains(token, "/") || strings.HasPrefix(token, ".")
}
