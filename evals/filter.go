package main

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
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

var (
	tierSingleRe = regexp.MustCompile(`^\d+$`)
	tierRangeRe  = regexp.MustCompile(`^(\d+)-(\d+)$`)
)

// parseTierSpec parses "" (all), "N" (single tier), or "N-M" (range).
func parseTierSpec(s string) (min, max int, err error) {
	if s == "" {
		return 0, 0, nil
	}
	if tierSingleRe.MatchString(s) {
		n, _ := strconv.Atoi(s)
		if n < 1 {
			return 0, 0, fmt.Errorf("bad tier %q: tiers must be >= 1", s)
		}
		return n, n, nil
	}
	if m := tierRangeRe.FindStringSubmatch(s); m != nil {
		min, _ = strconv.Atoi(m[1])
		max, _ = strconv.Atoi(m[2])
		if min < 1 || max < 1 {
			return 0, 0, fmt.Errorf("bad tier range %q: tiers must be >= 1", s)
		}
		if min > max {
			return 0, 0, fmt.Errorf("bad tier range %q: min > max", s)
		}
		return min, max, nil
	}
	return 0, 0, fmt.Errorf("bad tier %q: expected N or N-M where N,M >= 1", s)
}
