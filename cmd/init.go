package cmd

import (
	"fmt"
	"os"

	"github.com/aadidev/debategpt/internal/debate"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a DEBATE.md file in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := "DEBATE.md"

		// Check if already exists
		if _, err := os.Stat(filename); err == nil {
			return fmt.Errorf("DEBATE.md already exists in current directory")
		}

		content := debate.GenerateDefaultDebateFile()
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return fmt.Errorf("write DEBATE.md: %w", err)
		}

		fmt.Println("✅ Created DEBATE.md in current directory")
		fmt.Println("   Edit it to customize debate behavior for this project.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
