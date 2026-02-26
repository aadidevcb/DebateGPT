package cmd

import (
	"fmt"
	"os"

	"github.com/aadidev/debategpt/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage DebateGPT configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fmt.Println("📋 Current Configuration:")
		fmt.Println()

		// Agents
		fmt.Println("Agents:")
		for name, agent := range cfg.Agents {
			apiKeyDisplay := "not set"
			if agent.APIKey != "" {
				apiKeyDisplay = agent.APIKey[:8] + "..."
			}
			fmt.Printf("  %s:\n", name)
			fmt.Printf("    model: %s\n", agent.Model)
			fmt.Printf("    api_key: %s\n", apiKeyDisplay)
			if agent.BaseURL != "" {
				fmt.Printf("    base_url: %s\n", agent.BaseURL)
			}
			if agent.Role != "" {
				fmt.Printf("    role: %s\n", agent.Role)
			}
		}
		fmt.Println()

		// Debate
		fmt.Println("Debate:")
		fmt.Printf("  rounds: %d\n", cfg.Debate.Rounds)
		fmt.Printf("  adaptive: %v\n", cfg.Debate.Adaptive)
		fmt.Println()

		// Judge
		fmt.Println("Judge:")
		fmt.Printf("  mode: %s\n", cfg.Judge.Mode)
		if cfg.Judge.Model != "" {
			fmt.Printf("  model: %s\n", cfg.Judge.Model)
		}
		fmt.Println()

		// Cost
		fmt.Println("Cost:")
		if cfg.Cost.Budget > 0 {
			fmt.Printf("  budget: $%.2f\n", cfg.Cost.Budget)
		} else {
			fmt.Println("  budget: unlimited")
		}

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		v := viper.New()
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		configDir := config.ConfigDir()
		configPath := configDir + "/config.yaml"

		// Create config dir if needed
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}

		v.AddConfigPath(configDir)
		_ = v.ReadInConfig()

		v.Set(key, value)

		if err := v.WriteConfigAs(configPath); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		fmt.Printf("✅ Set %s = %s in %s\n", key, value, configPath)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
