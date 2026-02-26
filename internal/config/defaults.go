package config

import "github.com/spf13/viper"

// SetDefaults applies default configuration values.
func SetDefaults(v *viper.Viper) {
	// Debate defaults
	v.SetDefault("debate.rounds", 2)
	v.SetDefault("debate.max_rounds", 4)
	v.SetDefault("debate.adaptive", true)
	v.SetDefault("debate.critique_style", "structured")

	// Judge defaults
	v.SetDefault("judge.mode", "separate")
	v.SetDefault("judge.model", "")
	v.SetDefault("judge.participant_agent", "")

	// Output defaults
	v.SetDefault("output.format", "markdown")
	v.SetDefault("output.directory", "./brainstorms")
	v.SetDefault("output.include_transcript", false)

	// Cost defaults
	v.SetDefault("cost.budget", 0)
	v.SetDefault("cost.show_estimates", true)
	v.SetDefault("cost.warn_threshold", 0.50)
}
