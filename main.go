package main

import (
	"fmt"
	"os"

	"github.com/nickevangelista/aicouncil-cli/internal"
	"github.com/spf13/cobra"
)

func main() {
	// rootCmd is the root CLI command — shown when the user runs `ai-council` with no subcommands
	rootCmd := &cobra.Command{
		Use:   "ai-council",
		Short: "Democratic AI council to elect the best response",
		Long: `
╔═══════════════════════════════════════════╗
║           AI COUNCIL — v1.0               ║
║     Democracy among artificial minds      ║
╚═══════════════════════════════════════════╝

ai-council sends your prompt to multiple AI assistants (Gemini, Kiro,
Copilot), collects their responses, and uses democratic cross-voting to elect
the best answer based on categories like accuracy, clarity, and practicality.

Examples:
  ai-council ask "How to do idiomatic error handling in Go?"
  ai-council ask "Explain recursion with an example" --verbose
  ai-council ask "Refactor this code" --no-vote
  ai-council ask "What's the difference between mutex and channel?" | pbcopy
`,
	}

	// askCmd is the main subcommand: sends a question to the council
	askCmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask a question to the AI council",
		Args:  cobra.ExactArgs(1), // requires exactly 1 argument: the question
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := args[0]

			// Read flags
			configPath, _ := cmd.Flags().GetString("config")
			noVote, _ := cmd.Flags().GetBool("no-vote")
			verbose, _ := cmd.Flags().GetBool("verbose")
			quiet, _ := cmd.Flags().GetBool("quiet")

			// Create the council from the config file
			council, err := internal.NewCouncil(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Quiet mode: no animations, just prints the final answer to stdout
			// Useful for pipes: ai-council ask "..." --quiet | pbcopy
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

			// Normal mode: shows progress and pretty result
			result, err := council.Deliberate(prompt, verbose)
			if err != nil {
				return fmt.Errorf("deliberation failed: %w", err)
			}

			if noVote {
				// No vote: shows all responses side by side
				internal.DisplayAllResponses(result)
			} else {
				// With vote: shows scoreboard and winner
				internal.DisplayResults(result)
			}

			return nil
		},
	}

	// Flags for the ask subcommand
	askCmd.Flags().StringP("config", "c", "config.json", "path to the config file")
	askCmd.Flags().BoolP("no-vote", "n", false, "show all responses without voting")
	askCmd.Flags().BoolP("verbose", "v", false, "show detailed progress for each phase")
	askCmd.Flags().BoolP("quiet", "q", false, "quiet mode: only prints the winning response (good for pipes)")

	// Register the subcommand
	rootCmd.AddCommand(askCmd)

	// Execute — cobra handles usage errors (invalid flags, missing args, etc.)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
