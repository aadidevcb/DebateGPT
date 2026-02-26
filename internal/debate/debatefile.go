package debate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DebateFileConfig holds configuration parsed from DEBATE.md files.
type DebateFileConfig struct {
	Context       string            `json:"context"`
	Perspectives  map[string]string `json:"perspectives"`
	Constraints   []string          `json:"constraints"`
	Rounds        int               `json:"rounds"`
	JudgeMode     string            `json:"judge_mode"`
	CritiqueStyle string            `json:"critique_style"`
	Temperature   float64           `json:"temperature"`
	FocusAreas    []string          `json:"focus_areas"`
	Budget        float64           `json:"budget"`
}

// DefaultDebateFileConfig returns sensible defaults.
func DefaultDebateFileConfig() DebateFileConfig {
	return DebateFileConfig{
		Perspectives: map[string]string{
			"pragmatist": "Optimize for simplicity, shipping fast, and maintainability. Prefer proven tools and minimal dependencies.",
			"architect":  "Design for extensibility, scalability, and clean abstractions. Think about the 10x growth case.",
			"contrarian": "Challenge every assumption. Propose unconventional alternatives and identify hidden risks.",
		},
		Rounds:        2,
		JudgeMode:     "separate",
		CritiqueStyle: "structured",
		Temperature:   0.7,
	}
}

// LoadDebateFiles walks up from startDir to root, collecting and merging DEBATE.md files.
// Deeper files override shallower ones, lists are appended.
func LoadDebateFiles(startDir string) (DebateFileConfig, error) {
	configs, err := collectDebateFiles(startDir)
	if err != nil {
		return DefaultDebateFileConfig(), err
	}

	// Also check ~/.debategpt/DEBATE.md
	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".debategpt", "DEBATE.md")
		if globalCfg, err := parseDebateFile(globalPath); err == nil {
			configs = append([]DebateFileConfig{globalCfg}, configs...)
		}
	}

	if len(configs) == 0 {
		return DefaultDebateFileConfig(), nil
	}

	result := DefaultDebateFileConfig()
	for _, cfg := range configs {
		result = mergeConfigs(result, cfg)
	}

	return result, nil
}

// collectDebateFiles walks up from dir to /, collecting DEBATE.md files.
func collectDebateFiles(dir string) ([]DebateFileConfig, error) {
	var configs []DebateFileConfig
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	for {
		debatePath := filepath.Join(absDir, "DEBATE.md")
		if cfg, err := parseDebateFile(debatePath); err == nil {
			configs = append([]DebateFileConfig{cfg}, configs...) // prepend (shallower first)
		}

		parent := filepath.Dir(absDir)
		if parent == absDir {
			break // reached root
		}
		absDir = parent
	}

	return configs, nil
}

// parseDebateFile parses a single DEBATE.md file.
func parseDebateFile(path string) (DebateFileConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return DebateFileConfig{}, err
	}
	defer file.Close()

	cfg := DebateFileConfig{
		Perspectives: make(map[string]string),
	}

	scanner := bufio.NewScanner(file)
	var currentSection string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect section headers
		if strings.HasPrefix(trimmed, "## ") {
			currentSection = strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			continue
		}

		// Skip empty lines and top-level headers
		if trimmed == "" || strings.HasPrefix(trimmed, "# ") {
			continue
		}

		switch currentSection {
		case "context":
			if cfg.Context != "" {
				cfg.Context += "\n"
			}
			cfg.Context += trimmed

		case "perspectives":
			if strings.HasPrefix(trimmed, "- ") {
				trimmed = strings.TrimPrefix(trimmed, "- ")
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
					cfg.Perspectives[key] = val
				}
			}

		case "constraints":
			if strings.HasPrefix(trimmed, "- ") {
				cfg.Constraints = append(cfg.Constraints, strings.TrimPrefix(trimmed, "- "))
			}

		case "debate style":
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				switch key {
				case "rounds":
					if n, err := strconv.Atoi(val); err == nil {
						cfg.Rounds = n
					}
				case "judge":
					cfg.JudgeMode = val
				case "critique_format":
					cfg.CritiqueStyle = val
				case "temperature":
					if f, err := strconv.ParseFloat(val, 64); err == nil {
						cfg.Temperature = f
					}
				}
			}

		case "focus areas":
			if strings.HasPrefix(trimmed, "- ") {
				cfg.FocusAreas = append(cfg.FocusAreas, strings.TrimPrefix(trimmed, "- "))
			}
		}
	}

	return cfg, scanner.Err()
}

// mergeConfigs merges two configs; override takes precedence, lists are appended.
func mergeConfigs(base, override DebateFileConfig) DebateFileConfig {
	if override.Context != "" {
		base.Context = override.Context
	}
	for k, v := range override.Perspectives {
		base.Perspectives[k] = v
	}
	base.Constraints = append(base.Constraints, override.Constraints...)
	if override.Rounds > 0 {
		base.Rounds = override.Rounds
	}
	if override.JudgeMode != "" {
		base.JudgeMode = override.JudgeMode
	}
	if override.CritiqueStyle != "" {
		base.CritiqueStyle = override.CritiqueStyle
	}
	if override.Temperature > 0 {
		base.Temperature = override.Temperature
	}
	base.FocusAreas = append(base.FocusAreas, override.FocusAreas...)
	if override.Budget > 0 {
		base.Budget = override.Budget
	}
	return base
}

// GenerateDefaultDebateFile returns the content of a starter DEBATE.md.
func GenerateDefaultDebateFile() string {
	return fmt.Sprintf(`# Debate Rules

## Context
Describe your project here. This context is shared with all agents.

## Perspectives
- pragmatist: "Optimize for simplicity, shipping fast, and maintainability"
- architect: "Design for extensibility, scalability, and clean abstractions"
- contrarian: "Challenge every assumption, propose unconventional alternatives"

## Constraints
- Add project-specific constraints here

## Debate Style
rounds: 2
judge: separate
critique_format: structured
temperature: 0.7

## Focus Areas
- Architecture decisions
- Trade-offs and risks
`)
}
