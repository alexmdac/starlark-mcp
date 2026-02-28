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
	numRuns    int
	runsDone     []int
	runsPassed   []int
	runDurations [][]time.Duration
	stopCh     chan struct{}
}

func newDisplay(cs []evalCase, numRuns int) *display {
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
		numRuns:    numRuns,
		runsDone:     make([]int, len(cs)),
		runsPassed:   make([]int, len(cs)),
		runDurations: make([][]time.Duration, len(cs)),
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

func (d *display) finishRun(i int, passed bool, dur time.Duration) {
	d.mu.Lock()
	d.runsDone[i]++
	if passed {
		d.runsPassed[i]++
	}
	d.runDurations[i] = append(d.runDurations[i], dur)
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
		fmt.Fprintf(os.Stderr, "\033[2K") // clear line

		done := d.runsDone[i]
		passed := d.runsPassed[i]
		total := d.numRuns

		if done >= total {
			// Finished — compute avg/min/max duration.
			durs := d.runDurations[i]
			var sum time.Duration
			minD, maxD := durs[0], durs[0]
			for _, dur := range durs {
				sum += dur
				if dur < minD {
					minD = dur
				}
				if dur > maxD {
					maxD = dur
				}
			}
			avgD := sum / time.Duration(len(durs))
			color := colorGreen
			mark := "✔"
			if passed == 0 {
				color = colorRed
				mark = "✘"
			} else if passed < total {
				color = colorYellow
				mark = "◑"
			}
			fmt.Fprintf(os.Stderr, "  %s%s %s%s %s(%d/%d passed, avg %s, min %s, max %s)%s\n",
				color, mark, c.name, colorReset, colorDim, passed, total,
				avgD.Round(time.Millisecond), minD.Round(time.Millisecond), maxD.Round(time.Millisecond), colorReset)
		} else {
			// In progress.
			elapsed := now.Sub(d.startTimes[i]).Round(time.Second)
			fmt.Fprintf(os.Stderr, "  %s%s %s%s %s(%d/%d done, %s)%s\n",
				colorYellow, spinnerFrames[frame], c.name, colorReset, colorDim, done, total, elapsed, colorReset)
		}
	}
}

func printSummary(model string, numRuns int, results []caseResults) {

	// Find the longest case name for column sizing.
	nameWidth := 4 // minimum for "NAME"
	for _, cr := range results {
		if len(cr.ec.name) > nameWidth {
			nameWidth = len(cr.ec.name)
		}
	}

	printSummaryTable(model, numRuns, nameWidth, results)
}

func printSummaryTable(model string, numRuns int, nameWidth int, results []caseResults) {
	// Columns: mark NAME  PASS_RATE  AVG_SCORE  AVG_TRIES  TOKENS
	passColW := 9 // "5/5 100%"
	tableWidth := 3 + nameWidth + 2 + passColW + 2 + 9 + 2 + 9 + 2 + 14
	headerFmt := fmt.Sprintf("%%s   %%-%ds  %%-%ds  %%9s  %%9s  %%14s%%s\n", nameWidth, passColW)
	rowFmt := fmt.Sprintf(" %%s%%s%%s %%-%ds  %%s%%-%ds  %%9.2f  %%9.1f  %%6d / %%6d%%s\n", nameWidth, passColW)

	fmt.Printf("\n%s%s%s\n", colorCyan, strings.Repeat("═", tableWidth), colorReset)
	fmt.Printf("%s%sEVAL RESULTS — model: %s, %d runs per case%s\n", colorBold, colorCyan, model, numRuns, colorReset)
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("═", tableWidth), colorReset)
	fmt.Printf(headerFmt, colorDim, "NAME", "PASS RATE", "AVG SCORE", "AVG TRIES", "TOKENS IN/OUT", colorReset)

	totalPassed := 0
	totalRuns := 0
	totalScore := 0.0
	totalTokensIn := 0
	totalTokensOut := 0

	for tier := 1; tier <= maxTier; tier++ {
		var tierCases []caseResults
		for _, cr := range results {
			if cr.ec.tier == tier {
				tierCases = append(tierCases, cr)
			}
		}
		if len(tierCases) == 0 {
			continue
		}
		sort.Slice(tierCases, func(a, b int) bool {
			return tierCases[a].ec.name < tierCases[b].ec.name
		})

		fmt.Printf("\n%s%sTIER %d: %s%s\n", colorBold, colorCyan, tier, tierNames[tier], colorReset)

		tierPassed := 0
		tierRuns := 0
		tierScore := 0.0

		for _, cr := range tierCases {
			passed := 0
			scoreSum := 0.0
			attemptsSum := 0
			tokensIn := 0
			tokensOut := 0
			for _, r := range cr.Runs {
				if r.Passed {
					passed++
				}
				scoreSum += r.Score
				attemptsSum += r.Attempts
				tokensIn += r.TokensIn
				tokensOut += r.TokensOut
			}
			n := len(cr.Runs)
			avgScore := scoreSum / float64(n)
			avgAttempts := float64(attemptsSum) / float64(n)
			passRate := float64(passed) / float64(n) * 100

			var mark, color string
			if passed == n {
				mark = "✔"
				color = colorGreen
			} else if passed == 0 {
				mark = "✘"
				color = colorRed
			} else {
				mark = "◑"
				color = colorYellow
			}

			passStr := fmt.Sprintf("%d/%d %3.0f%%", passed, n, passRate)

			fmt.Printf(rowFmt,
				color, mark, colorReset, cr.ec.name, colorDim,
				passStr, avgScore, avgAttempts, tokensIn, tokensOut, colorReset)

			tierPassed += passed
			tierRuns += n
			tierScore += scoreSum
			totalTokensIn += tokensIn
			totalTokensOut += tokensOut
		}

		tierPassRate := float64(tierPassed) / float64(tierRuns) * 100
		fmt.Printf("   %sTier: %.2f avg score, %d/%d runs passed (%.0f%%)%s\n",
			colorDim, tierScore/float64(tierRuns), tierPassed, tierRuns, tierPassRate, colorReset)

		totalPassed += tierPassed
		totalRuns += tierRuns
		totalScore += tierScore
	}

	overallPassRate := float64(totalPassed) / float64(totalRuns) * 100
	fmt.Printf("\n%s%s%s\n", colorCyan, strings.Repeat("─", tableWidth), colorReset)
	fmt.Printf("%s%sOVERALL: %.2f avg score, %d/%d runs passed (%.0f%%)  tokens: %d in, %d out%s\n",
		colorBold, colorCyan, totalScore/float64(totalRuns), totalPassed, totalRuns, overallPassRate, totalTokensIn, totalTokensOut, colorReset)
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("─", tableWidth), colorReset)

	// Print failed run details for cases that didn't pass every run.
	for _, cr := range results {
		var failedRuns []evalResult
		for _, r := range cr.Runs {
			if !r.Passed {
				failedRuns = append(failedRuns, r)
			}
		}
		if len(failedRuns) == 0 {
			continue
		}

		passedCount := len(cr.Runs) - len(failedRuns)
		if passedCount > 0 {
			fmt.Printf("\n%s%sFAILED RUNS (%d/%d failed): %s%s\n",
				colorBold, colorYellow, len(failedRuns), len(cr.Runs), cr.ec.name, colorReset)
		} else {
			fmt.Printf("\n%s%sFAILED (all %d runs): %s%s\n",
				colorBold, colorRed, len(cr.Runs), cr.ec.name, colorReset)
		}
		for ri, r := range failedRuns {
			if len(r.Outputs) == 0 {
				fmt.Printf("%sRun %d: no output%s\n", colorDim, ri+1, colorReset)
				continue
			}
			last := r.Outputs[len(r.Outputs)-1]
			fmt.Printf("%sRun %d (last output):%s\n%s\n", colorDim, ri+1, colorReset, last)
		}
	}
}
