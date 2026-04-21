package internal

import (
	"fmt"
	"strings"
)

// ─── Códigos ANSI para cores no terminal ───────────────────────────────────────
// Go não tem uma lib de cores na stdlib, mas códigos ANSI funcionam em
// qualquer terminal moderno (macOS Terminal, iTerm, VS Code, Linux, etc.)
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

// Cores por agente (A=azul, B=roxo, C=amarelo)
var agentColors = []string{ansiBlue, ansiPurple, ansiYellow}

// ─── Funções de progresso (usadas durante a execução) ──────────────────────────

// PrintPhase exibe o cabeçalho de uma fase (só no modo verbose)
func PrintPhase(n int, name string, verbose bool) {
	if !verbose {
		// Modo normal: exibe uma linha de progresso simples
		fmt.Printf("%s⟳ Fase %d: %s...%s\n", ansiCyan, n, name, ansiReset)
		return
	}
	fmt.Printf("\n%s%s FASE %d — %s %s%s\n",
		ansiBold+ansiCyan,
		strings.Repeat("─", 3),
		n, name,
		strings.Repeat("─", 3),
		ansiReset,
	)
}

// PrintAgentWorking exibe que um agente está trabalhando
func PrintAgentWorking(name string, verbose bool) {
	if verbose {
		fmt.Printf("  %s○ %s...%s\n", ansiDim, name, ansiReset)
	}
}

// PrintAgentDone exibe confirmação de que um agente terminou com sucesso
func PrintAgentDone(name string, verbose bool) {
	if verbose {
		fmt.Printf("  %s✓ %s%s\n", ansiGreen, name, ansiReset)
	}
}

// PrintAgentError exibe um erro de agente
func PrintAgentError(name string, err error, verbose bool) {
	// Erros sempre aparecem, independente do verbose
	fmt.Printf("  %s✗ %s: %s%s\n", ansiRed, name, err.Error(), ansiReset)
}

// PrintSingleWinner avisa quando só um agente funcionou
func PrintSingleWinner(name string, verbose bool) {
	fmt.Printf("%s⚠ Só %s respondeu — retornando resposta única (sem votação)%s\n",
		ansiYellow, name, ansiReset)
}

// ─── Funções de resultado ──────────────────────────────────────────────────────

// DisplayResults exibe o placar de votação + resposta vencedora
func DisplayResults(result *DeliberationResult) {
	// Mostra as respostas de cada agente (resumidas)
	printSeparator()
	fmt.Printf("%s%s RESPOSTAS DO CONSELHO %s%s\n",
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

	// Mostra o placar se houver votação
	if result.VoteResult != nil && len(result.VoteResult.Totals) > 0 {
		printVotingBoard(result.VoteResult)
	}

	// Mostra o vencedor e a resposta completa
	printWinnerResponse(result)
}

// DisplayAllResponses exibe todas as respostas sem votação (modo --no-vote)
func DisplayAllResponses(result *DeliberationResult) {
	printSeparator()
	fmt.Printf("%s%s TODAS AS RESPOSTAS (sem votação) %s%s\n",
		ansiBold+ansiCyan, strings.Repeat("═", 3), strings.Repeat("═", 3), ansiReset)

	for i, r := range result.Responses {
		if r == nil {
			continue
		}
		color := agentColors[i%len(agentColors)]
		fmt.Printf("\n%s%s── [%s] %s ──%s\n", ansiBold+color, "", r.Letter, r.Agent.Name, ansiReset)
		if r.Err != nil {
			fmt.Printf("%s✗ Erro: %s%s\n", ansiRed, r.Err.Error(), ansiReset)
		} else {
			fmt.Println(r.Content)
		}
	}
	printSeparator()
}

// printVotingBoard exibe a tabela de votos no terminal
func printVotingBoard(vr *VoteResult) {
	printSeparator()
	fmt.Printf("\n%s%s PLACAR DE VOTAÇÃO %s%s\n",
		ansiBold+ansiCyan, strings.Repeat("═", 3), strings.Repeat("═", 3), ansiReset)

	// Cabeçalho da tabela
	fmt.Printf("\n%-14s", "")
	for i, r := range vr.Responses {
		color := agentColors[i%len(agentColors)]
		header := fmt.Sprintf("[%s] %s", r.Letter, r.Agent.Name)
		fmt.Printf("%s%-20s%s", ansiBold+color, header, ansiReset)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("─", 14+20*len(vr.Responses)))

	// Linhas por categoria
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

	// Linha de totais
	fmt.Printf("%s%-14s%s", ansiBold, "TOTAL", ansiReset)
	for i, r := range vr.Responses {
		total := vr.Totals[r.Letter]
		color := agentColors[i%len(agentColors)]
		fmt.Printf("%s%-20s%s", ansiBold+color, fmt.Sprintf("%.1f pts", total), ansiReset)
	}
	fmt.Println()
}

// printWinnerResponse exibe a resposta vencedora completa
func printWinnerResponse(result *DeliberationResult) {
	printSeparator()

	if result.Winner == nil {
		fmt.Printf("%s✗ Não foi possível determinar um vencedor%s\n", ansiRed, ansiReset)
		return
	}

	// Banner do vencedor
	pts := ""
	if result.VoteResult != nil {
		total := result.VoteResult.Totals[result.Winner.Letter]
		pts = fmt.Sprintf(" — %.1f pontos", total)
	}

	fmt.Printf("\n%s%s 🏆 VENCEDOR: %s%s%s\n",
		ansiBold+ansiGreen,
		strings.Repeat("═", 3),
		result.Winner.Agent.Name,
		pts,
		ansiReset,
	)
	fmt.Println()
	// Resposta completa (essa é a parte que vai para o stdout para pipes)
	fmt.Println(result.Winner.Content)
	printSeparator()
}

// ─── Funções auxiliares ────────────────────────────────────────────────────────

// buildBar constrói uma barra visual de pontuação
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

	// Cor baseada na pontuação
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

// truncate corta um texto em n caracteres e adiciona "..." se necessário
func truncate(s string, n int) string {
	// Remove linhas extras para o preview
	lines := strings.Split(strings.TrimSpace(s), "\n")
	flat := strings.Join(lines[:min(5, len(lines))], " ↵ ")

	if len(flat) <= n {
		return flat
	}
	return flat[:n] + "..."
}

// printSeparator exibe uma linha divisória
func printSeparator() {
	fmt.Printf("%s%s%s\n", ansiDim, strings.Repeat("─", 60), ansiReset)
}

// min retorna o menor de dois inteiros (disponível nativamente em Go 1.21+)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
