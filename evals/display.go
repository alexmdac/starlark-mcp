//go:build eval

package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// display manages live terminal output for eval progress.
type display struct {
	mu         sync.Mutex
	cases      []evalCase
	sorted     []int // indices into cases, sorted lexicographically
	startTimes []time.Time
	done       []bool
	passed     []bool
	attempts   []int
	durations  []time.Duration
	stopCh     chan struct{}
}

func newDisplay(cs []evalCase) *display {
	now := time.Now()
	sorted := make([]int, len(cs))
	for i := range sorted {
		sorted[i] = i
	}
	sort.Slice(sorted, func(a, b int) bool {
		return cs[sorted[a]].name < cs[sorted[b]].name
	})
	d := &display{
		cases:      cs,
		sorted:     sorted,
		startTimes: make([]time.Time, len(cs)),
		done:       make([]bool, len(cs)),
		passed:     make([]bool, len(cs)),
		attempts:   make([]int, len(cs)),
		durations:  make([]time.Duration, len(cs)),
		stopCh:     make(chan struct{}),
	}
	for i := range cs {
		d.startTimes[i] = now
	}
	// Print initial lines.
	for range cs {
		fmt.Fprint(os.Stderr, "\n")
	}
	d.render()
	go d.loop()
	return d
}

func (d *display) loop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.render()
		}
	}
}

func (d *display) finish(i int, passed bool, attempts int, dur time.Duration) {
	d.mu.Lock()
	d.done[i] = true
	d.passed[i] = passed
	d.attempts[i] = attempts
	d.durations[i] = dur
	d.mu.Unlock()
	d.render()
}

func (d *display) stop() {
	close(d.stopCh)
	d.render()
}

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
	colorCyan   = "\033[36m"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (d *display) render() {
	d.mu.Lock()
	defer d.mu.Unlock()

	n := len(d.cases)
	// Move cursor up n lines.
	fmt.Fprintf(os.Stderr, "\033[%dA", n)

	now := time.Now()
	frame := int(now.UnixMilli()/80) % len(spinnerFrames)
	for _, i := range d.sorted {
		c := d.cases[i]
		// Clear line and write status.
		fmt.Fprintf(os.Stderr, "\033[2K")
		if d.done[i] {
			if d.passed[i] {
				fmt.Fprintf(os.Stderr, "  %s✔ %s%s %s(%s, %d attempts)%s\n",
					colorGreen, c.name, colorReset, colorDim, d.durations[i].Round(time.Millisecond), d.attempts[i], colorReset)
			} else {
				fmt.Fprintf(os.Stderr, "  %s✘ %s%s %s(%s, %d attempts)%s\n",
					colorRed, c.name, colorReset, colorDim, d.durations[i].Round(time.Millisecond), d.attempts[i], colorReset)
			}
		} else {
			elapsed := now.Sub(d.startTimes[i]).Round(time.Second)
			fmt.Fprintf(os.Stderr, "  %s%s %s%s %s(%s)%s\n",
				colorYellow, spinnerFrames[frame], c.name, colorReset, colorDim, elapsed, colorReset)
		}
	}
}

func printSummary(model string, results []evalResult) {
	tierNames := map[int]string{
		1: "BASICS",
		2: "SIMPLE ALGORITHMS",
		3: "INTERMEDIATE",
		4: "HARD",
		5: "EXPERT",
		6: "CHALLENGING",
	}

	// Find the highest tier in use.
	maxTier := 0
	for _, r := range results {
		if r.ec.tier > maxTier {
			maxTier = r.ec.tier
		}
	}

	// Find the longest case name for column sizing.
	nameWidth := 4 // minimum for "NAME"
	for _, r := range results {
		if len(r.ec.name) > nameWidth {
			nameWidth = len(r.ec.name)
		}
	}

	//  " ✔ %-Ns  %5s  %5s  %7s  %8s"
	tableWidth := 3 + nameWidth + 2 + 5 + 2 + 5 + 2 + 10 + 2 + 10
	headerFmt := fmt.Sprintf("%%s   %%-%ds  %%5s  %%5s  %%10s  %%10s%%s\n", nameWidth)
	rowFmt := fmt.Sprintf(" %%s%%s%%s %%-%ds  %%s%%5d  %%5.2f  %%10s  %%10s%%s\n", nameWidth)

	fmt.Printf("\n%s%s%s\n", colorCyan, strings.Repeat("═", tableWidth), colorReset)
	fmt.Printf("%s%sEVAL RESULTS — model: %s%s\n", colorBold, colorCyan, model, colorReset)
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("═", tableWidth), colorReset)
	fmt.Printf(headerFmt, colorDim, "NAME", "TRIES", "SCORE", "LLM", "STARLARK", colorReset)

	totalPassed := 0
	totalCases := 0
	totalScore := 0.0
	totalTokensIn := 0
	totalTokensOut := 0

	for tier := 1; tier <= maxTier; tier++ {
		var tierResults []evalResult
		for _, r := range results {
			if r.ec.tier == tier {
				tierResults = append(tierResults, r)
			}
		}
		if len(tierResults) == 0 {
			continue
		}
		sort.Slice(tierResults, func(a, b int) bool {
			return tierResults[a].ec.name < tierResults[b].ec.name
		})

		fmt.Printf("\n%s%sTIER %d: %s%s\n", colorBold, colorCyan, tier, tierNames[tier], colorReset)

		tierPassed := 0
		tierTotal := len(tierResults)
		tierScore := 0.0

		for _, r := range tierResults {
			var mark, color string
			if r.Passed {
				mark = "✔"
				color = colorGreen
				tierPassed++
			} else {
				mark = "✘"
				color = colorRed
			}
			fmt.Printf(rowFmt,
				color, mark, colorReset, r.ec.name, colorDim, r.Attempts, r.Score, r.LLMTime.Round(time.Second), r.StarlarkTime.Round(time.Millisecond), colorReset)
			tierScore += r.Score
			totalTokensIn += r.TokensIn
			totalTokensOut += r.TokensOut
		}

		fmt.Printf("   %sTier score: %.2f (%d/%d passed)%s\n",
			colorDim, tierScore/float64(tierTotal), tierPassed, tierTotal, colorReset)

		totalPassed += tierPassed
		totalCases += tierTotal
		totalScore += tierScore
	}

	fmt.Printf("\n%s%s%s\n", colorCyan, strings.Repeat("─", tableWidth), colorReset)
	fmt.Printf("%s%sOVERALL: %.2f (%d/%d passed)  tokens: %d in, %d out%s\n",
		colorBold, colorCyan, totalScore/float64(totalCases), totalPassed, totalCases, totalTokensIn, totalTokensOut, colorReset)
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("─", tableWidth), colorReset)

	// Print details for failed cases.
	for _, r := range results {
		if r.Passed || len(r.Outputs) == 0 {
			continue
		}
		fmt.Printf("\n%s%sFAILED: %s%s\n", colorBold, colorRed, r.ec.name, colorReset)
		for i, out := range r.Outputs {
			fmt.Printf("%sAttempt %d:%s\n%s\n", colorDim, i+1, colorReset, out)
		}
	}
}
