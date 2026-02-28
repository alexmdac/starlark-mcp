// No build tag: dedent is a pure utility used by eval cases but tested
// unconditionally so that "go test ./evals/" runs its tests without
// -tags eval. All other files in this package require the eval tag.

package main

import "strings"

// dedent dedents a backtick string for use as an inline literal.
// It strips leading/trailing blank lines, computes the longest common
// whitespace prefix across all non-empty lines, and removes it.
func dedent(s string) string {
	s = strings.TrimLeft(s, "\n")
	s = strings.TrimRight(s, " \t\n")
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")

	// Find longest common whitespace prefix (considering only non-empty lines).
	prefix := ""
	first := true
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		ws := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		if first {
			prefix = ws
			first = false
		} else {
			// Shorten prefix to common length.
			n := len(prefix)
			if len(ws) < n {
				n = len(ws)
			}
			for i := 0; i < n; i++ {
				if prefix[i] != ws[i] {
					n = i
					break
				}
			}
			prefix = prefix[:n]
		}
	}

	// Remove the common prefix from each line.
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
		} else {
			lines[i] = line[len(prefix):]
		}
	}
	return strings.Join(lines, "\n")
}
