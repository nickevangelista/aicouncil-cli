package main

import (
	"fmt"
	"os"

	"github.com/seu-usuario/ai-council/internal"
	"github.com/spf13/cobra"
)

func main() {
	// rootCmd é o comando raiz do CLI — aparece quando o usuário roda `ai-council` sem subcomandos
	rootCmd := &cobra.Command{
		Use:   "ai-council",
		Short: "Conselho democrático de IAs para eleger a melhor resposta",
		Long: `
╔═══════════════════════════════════════════╗
║           AI COUNCIL — v1.0               ║
║  Democracia entre inteligências artificiais║
╚═══════════════════════════════════════════╝

ai-council envia seu prompt para múltiplos assistentes de IA (Gemini, Kiro,
Copilot), coleta as respostas e usa votação democrática cruzada para eleger
a melhor resposta com base em categorias como precisão, clareza e praticidade.

Exemplos:
  ai-council ask "Como fazer uma API REST em Go?"
  ai-council ask "Explica recursão com exemplo" --verbose
  ai-council ask "Refatora esse código" --no-vote
  ai-council ask "Qual a diferença entre mutex e channel?" | pbcopy
`,
	}

	// askCmd é o subcomando principal: envia uma pergunta ao conselho
	askCmd := &cobra.Command{
		Use:   "ask [pergunta]",
		Short: "Faz uma pergunta ao conselho de IAs",
		Args:  cobra.ExactArgs(1), // exige exatamente 1 argumento: a pergunta
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := args[0]

			// Lê as flags
			configPath, _ := cmd.Flags().GetString("config")
			noVote, _ := cmd.Flags().GetBool("no-vote")
			verbose, _ := cmd.Flags().GetBool("verbose")
			quiet, _ := cmd.Flags().GetBool("quiet")

			// Cria o conselho a partir do arquivo de configuração
			council, err := internal.NewCouncil(configPath)
			if err != nil {
				return fmt.Errorf("erro ao carregar configuração: %w", err)
			}

			// Modo silencioso: sem animações, só imprime a resposta final no stdout
			// Útil para pipes: ai-council ask "..." --quiet | pbcopy
			if quiet {
				result, err := council.Deliberate(prompt, false)
				if err != nil {
					return err
				}
				if result.Winner != nil {
					fmt.Println(result.Winner.Content)
				}
				return nil
			}

			// Modo normal: mostra progresso e resultado bonito
			result, err := council.Deliberate(prompt, verbose)
			if err != nil {
				return fmt.Errorf("erro na deliberação: %w", err)
			}

			if noVote {
				// Sem votação: mostra todas as respostas lado a lado
				internal.DisplayAllResponses(result)
			} else {
				// Com votação: mostra placar e vencedor
				internal.DisplayResults(result)
			}

			return nil
		},
	}

	// Flags do subcomando ask
	askCmd.Flags().StringP("config", "c", "config.json", "caminho para o arquivo de configuração")
	askCmd.Flags().BoolP("no-vote", "n", false, "mostra todas as respostas sem realizar votação")
	askCmd.Flags().BoolP("verbose", "v", false, "mostra progresso detalhado de cada fase")
	askCmd.Flags().BoolP("quiet", "q", false, "modo silencioso: só imprime a resposta vencedora (bom para pipes)")

	// Registra o subcomando
	rootCmd.AddCommand(askCmd)

	// Executa — o cobra cuida de erros de uso (flags inválidas, args faltando, etc.)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}