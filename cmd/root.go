package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "debategpt",
	Short: "Multi-agent LLM debate CLI for brainstorming",
	Long: `DebateGPT is a terminal-based coding assistant where multiple LLM agents
(Claude, GPT-4o, Gemini, and more) debate each other across rounds to produce
more accurate and well-reasoned answers.

Built for the brainstorming and architecture phase — before writing code.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path (default: ./config.yaml or ~/.debategpt/config.yaml)")
}
