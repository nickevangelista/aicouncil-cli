package internal

import (
	"fmt"
	"strings"
)

// ─── ANSI color codes for terminal output ─────────────────────────────────────
// Go has no color library in stdlib, but ANSI codes work in
// any modern terminal (macOS Terminal, iTerm, VS Code, Linux, etc.)
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiPurple = "\033[35m"
	ansiCyan   = "\033[36m"
)

// Colors per agent (A=blue, B=purple, C=yellow)
var agentColors = []string{ansiBlue, ansiPurple, ansiYellow}

// ─── Progress functions (used during execution) ────────────────────────────────

// PrintPhase displays the header for a phase (only in verbose mode)
func PrintPhase(n int, name string, verbose bool) {
	if !verbose {
		// Normal mode: display a simple progress line
		fmt.Printf("%s⟳ Phase %d: %s...%s\n", ansiCyan, n, name, ansiReset)
		return
	}
	fmt.Printf("\n%s%s PHASE %d — %s %s%s\n",
		ansiBold+ansiCyan,
		strings.Repeat("─", 3),
		n, name,
		strings.Repeat("─", 3),
		ansiReset,
	)
}

// PrintAgentWorking displays that an agent is working
func PrintAgentWorking(name string, verbose bool) {
	if verbose {
		fmt.Printf("  %s○ %s...%s\n", ansiDim, name, ansiReset)
	}
}

// PrintAgentDone displays confirmation that an agent finished successfully
func PrintAgentDone(name string, verbose bool) {
	if verbose {
		fmt.Printf("  %s✓ %s%s\n", ansiGreen, name, ansiReset)
	}
}

// PrintAgentError displays an agent error
func PrintAgentError(name string, err error, verbose bool) {
	// Errors always show, regardless of verbose
	fmt.Printf("  %s✗ %s: %s%s\n", ansiRed, name, err.Error(), ansiReset)
}

// PrintSingleWinner warns when only one agent responded
func PrintSingleWinner(name string, verbose bool) {
	fmt.Printf("%s⚠ Only %s responded — returning single response (no voting)%s\n",
		ansiYellow, name, ansiReset)
}

// ─── Result functions ──────────────────────────────────────────────────────────

// DisplayResults shows the voting scoreboard + winning response
func DisplayResults(result *DeliberationResult) {
	// Show each agent's response (summarized)
	printSeparator()
	fmt.Printf("%s%s COUNCIL RESPONSES %s%s\n",
		ansiBold+ansiCyan, strings.Repeat("═", 3), strings.Repeat("═", 3), ansiReset)

	for i, r := range result.Responses {
		if r == nil || r.Err != nil {
			continue
		}
		color := agentColors[i%len(agentColors)]
		preview := truncate(r.Content, 200)
		fmt.Printf("\n%s%s▶ [%s] %s%s\n", ansiBold+color, "", r.Letter, r.Agent.Name, ansiReset)
		fmt.Printf("%s%s%s\n", ansiDim, preview, ansiReset)
	}

	// Show scoreboard if there was voting
	if result.VoteResult != nil && len(result.VoteResult.Totals) > 0 {
		printVotingBoard(result.VoteResult)
	}

	// Show winner and full response
	printWinnerResponse(result)
}

// DisplayAllResponses shows all responses without voting (--no-vote mode)
func DisplayAllResponses(result *DeliberationResult) {
	printSeparator()
	fmt.Printf("%s%s ALL RESPONSES (no voting) %s%s\n",
		ansiBold+ansiCyan, strings.Repeat("═", 3), strings.Repeat("═", 3), ansiReset)

	for i, r := range result.Responses {
		if r == nil {
			continue
		}
		color := agentColors[i%len(agentColors)]
		fmt.Printf("\n%s%s── [%s] %s ──%s\n", ansiBold+color, "", r.Letter, r.Agent.Name, ansiReset)
		if r.Err != nil {
			fmt.Printf("%s✗ Error: %s%s\n", ansiRed, r.Err.Error(), ansiReset)
		} else {
			fmt.Println(r.Content)
		}
	}
	printSeparator()
}

// printVotingBoard displays the voting table in the terminal
func printVotingBoard(vr *VoteResult) {
	printSeparator()
	fmt.Printf("\n%s%s VOTING SCOREBOARD %s%s\n",
		ansiBold+ansiCyan, strings.Repeat("═", 3), strings.Repeat("═", 3), ansiReset)

	// Table header
	fmt.Printf("\n%-14s", "")
	for i, r := range vr.Responses {
		color := agentColors[i%len(agentColors)]
		header := fmt.Sprintf("[%s] %s", r.Letter, r.Agent.Name)
		fmt.Printf("%s%-20s%s", ansiBold+color, header, ansiReset)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("─", 14+20*len(vr.Responses)))

	// Rows per category
	for _, cat := range Categories {
		label := CategoryLabels[cat]
		fmt.Printf("%-14s", label)
		for _, r := range vr.Responses {
			score := vr.AvgScores[r.Letter][cat]
			bar := buildBar(score, 10, 5)
			fmt.Printf("%-20s", bar)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("─", 14+20*len(vr.Responses)))

	// Totals row
	fmt.Printf("%s%-14s%s", ansiBold, "TOTAL", ansiReset)
	for i, r := range vr.Responses {
		total := vr.Totals[r.Letter]
		color := agentColors[i%len(agentColors)]
		fmt.Printf("%s%-20s%s", ansiBold+color, fmt.Sprintf("%.1f pts", total), ansiReset)
	}
	fmt.Println()
}

// printWinnerResponse displays the full winning response
func printWinnerResponse(result *DeliberationResult) {
	printSeparator()

	if result.Winner == nil {
		fmt.Printf("%s✗ Could not determine a winner%s\n", ansiRed, ansiReset)
		return
	}

	// Winner banner
	pts := ""
	if result.VoteResult != nil {
		total := result.VoteResult.Totals[result.Winner.Letter]
		pts = fmt.Sprintf(" — %.1f points", total)
	}

	fmt.Printf("\n%s%s 🏆 WINNER: %s%s%s\n",
		ansiBold+ansiGreen,
		strings.Repeat("═", 3),
		result.Winner.Agent.Name,
		pts,
		ansiReset,
	)
	fmt.Println()
	// Full response (this is the part that goes to stdout for pipes)
	fmt.Println(result.Winner.Content)
	printSeparator()
}

// ─── Helper functions ──────────────────────────────────────────────────────────

// buildBar builds a visual score bar
// Ex: score=7.5, max=10, width=5 → "████░ 7.5"
func buildBar(score, max float64, width int) string {
	if max <= 0 {
		return strings.Repeat("░", width) + " ---"
	}
	ratio := score / max
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}
	filled := int(ratio * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	// Color based on score
	var color string
	switch {
	case score >= 8:
		color = ansiGreen
	case score >= 6:
		color = ansiYellow
	default:
		color = ansiRed
	}

	return fmt.Sprintf("%s%s%s %.1f", color, bar, ansiReset, score)
}

// truncate cuts text at n characters and adds "..." if needed
func truncate(s string, n int) string {
	// Remove extra lines for the preview
	lines := strings.Split(strings.TrimSpace(s), "\n")
	flat := strings.Join(lines[:min(5, len(lines))], " ↵ ")

	if len(flat) <= n {
		return flat
	}
	return flat[:n] + "..."
}

// printSeparator displays a divider line
func printSeparator() {
	fmt.Printf("%s%s%s\n", ansiDim, strings.Repeat("─", 60), ansiReset)
}

// min returns the smaller of two integers (available natively in Go 1.21+)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
