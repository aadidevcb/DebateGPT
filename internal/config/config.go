package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config is the top-level application configuration.
type Config struct {
	Agents map[string]AgentCfg `mapstructure:"agents"`
	Debate DebateCfg           `mapstructure:"debate"`
	Judge  JudgeCfg            `mapstructure:"judge"`
	Output OutputCfg           `mapstructure:"output"`
	Cost   CostCfg             `mapstructure:"cost"`
}

// AgentCfg holds per-agent settings.
type AgentCfg struct {
	Provider  string `mapstructure:"provider"`
	Model     string `mapstructure:"model"`
	APIKey    string `mapstructure:"api_key"`
	BaseURL   string `mapstructure:"base_url"`
	MaxTokens int    `mapstructure:"max_tokens"`
	Role      string `mapstructure:"role"`
}

// DebateCfg holds debate-specific settings.
type DebateCfg struct {
	Rounds        int    `mapstructure:"rounds"`
	MaxRounds     int    `mapstructure:"max_rounds"`
	Adaptive      bool   `mapstructure:"adaptive"`
	CritiqueStyle string `mapstructure:"critique_style"`
}

// JudgeCfg holds judge configuration.
type JudgeCfg struct {
	Mode             string `mapstructure:"mode"`
	Model            string `mapstructure:"model"`
	ParticipantAgent string `mapstructure:"participant_agent"`
}

// OutputCfg holds output settings.
type OutputCfg struct {
	Format            string `mapstructure:"format"`
	Directory         string `mapstructure:"directory"`
	IncludeTranscript bool   `mapstructure:"include_transcript"`
}

// CostCfg holds cost/budget settings.
type CostCfg struct {
	Budget        float64 `mapstructure:"budget"`
	ShowEstimates bool    `mapstructure:"show_estimates"`
	WarnThreshold float64 `mapstructure:"warn_threshold"`
}

// Load reads configuration from file, env vars, and applies defaults.
func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	SetDefaults(v)

	// Config file locations
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.debategpt")

	// Environment variables
	v.SetEnvPrefix("DEBATEGPT")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	// Resolve env vars in api_key fields
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Resolve environment variable references in API keys
	for name, agentCfg := range cfg.Agents {
		if strings.HasPrefix(agentCfg.APIKey, "${") && strings.HasSuffix(agentCfg.APIKey, "}") {
			envVar := strings.TrimSuffix(strings.TrimPrefix(agentCfg.APIKey, "${"), "}")
			agentCfg.APIKey = os.Getenv(envVar)
			cfg.Agents[name] = agentCfg
		}
	}

	return &cfg, nil
}

// ConfigDir returns the config directory path.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".debategpt"
	}
	return filepath.Join(home, ".debategpt")
}
