package main

import (
	"fmt"
	"path"
	"strconv"
	"strings"
)

// filterCases returns the subset of cases matching the given glob pattern and tier range.
// An empty filter or tier means "match all".
func filterCases(all []evalCase, pattern, tierSpec string) ([]evalCase, error) {
	minTier, maxTier, err := parseTierSpec(tierSpec)
	if err != nil {
		return nil, err
	}

	var out []evalCase
	for _, ec := range all {
		if minTier > 0 && (ec.tier < minTier || ec.tier > maxTier) {
			continue
		}
		if pattern != "" {
			matched, err := path.Match(pattern, ec.name)
			if err != nil {
				return nil, fmt.Errorf("bad filter pattern: %w", err)
			}
			if !matched {
				continue
			}
		}
		out = append(out, ec)
	}
	return out, nil
}

// parseTierSpec parses "" (all), "N" (single tier), or "N-M" (range).
func parseTierSpec(s string) (min, max int, err error) {
	if s == "" {
		return 0, 0, nil
	}
	if i := strings.Index(s, "-"); i >= 0 {
		min, err = strconv.Atoi(s[:i])
		if err != nil {
			return 0, 0, fmt.Errorf("bad tier range %q: %w", s, err)
		}
		max, err = strconv.Atoi(s[i+1:])
		if err != nil {
			return 0, 0, fmt.Errorf("bad tier range %q: %w", s, err)
		}
		if min > max {
			return 0, 0, fmt.Errorf("bad tier range %q: min > max", s)
		}
		return min, max, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, 0, fmt.Errorf("bad tier %q: %w", s, err)
	}
	return n, n, nil
}
