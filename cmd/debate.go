package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aadidev/debategpt/internal/agent"
	"github.com/aadidev/debategpt/internal/config"
	"github.com/aadidev/debategpt/internal/cost"
	"github.com/aadidev/debategpt/internal/debate"
	"github.com/aadidev/debategpt/internal/judge"
	"github.com/aadidev/debategpt/internal/output"
	_ "github.com/aadidev/debategpt/internal/register" // register providers
	"github.com/aadidev/debategpt/internal/stream"
	"github.com/aadidev/debategpt/internal/tui"
	"github.com/spf13/cobra"
)

var debateCmd = &cobra.Command{
	Use:   "debate [question]",
	Short: "Start a multi-agent debate on a question",
	Long: `Start a debate where multiple LLM agents discuss your question from
different perspectives, critique each other, and produce a synthesized
brainstorm document.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDebate,
}

func init() {
	debateCmd.Flags().StringSlice("agents", nil, "agents to use (e.g., claude,openai,gemini)")
	debateCmd.Flags().Int("rounds", 0, "number of debate rounds (overrides config)")
	debateCmd.Flags().String("judge", "", "judge mode: participant, separate, consensus")
	debateCmd.Flags().Float64("budget", 0, "max budget in USD (0 = unlimited)")
	debateCmd.Flags().StringP("output", "o", "", "output file path")
	debateCmd.Flags().Bool("quick", false, "quick mode: 1 round, cheaper models")
	debateCmd.Flags().Bool("transcript", false, "include full transcript in output")
	debateCmd.Flags().StringSlice("context", nil, "context files to include")
	rootCmd.AddCommand(debateCmd)
}

func runDebate(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Load DEBATE.md
	cwd, _ := os.Getwd()
	debateCfg, err := debate.LoadDebateFiles(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Warning: could not load DEBATE.md: %v\n", err)
		debateCfg = debate.DefaultDebateFileConfig()
	}

	// Apply CLI overrides
	if rounds, _ := cmd.Flags().GetInt("rounds"); rounds > 0 {
		debateCfg.Rounds = rounds
	}
	if judgeMode, _ := cmd.Flags().GetString("judge"); judgeMode != "" {
		debateCfg.JudgeMode = judgeMode
	}
	if quick, _ := cmd.Flags().GetBool("quick"); quick {
		debateCfg.Rounds = 1
	}

	// Load context files
	if contextFiles, _ := cmd.Flags().GetStringSlice("context"); len(contextFiles) > 0 {
		for _, f := range contextFiles {
			data, err := os.ReadFile(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠ Warning: could not read context file %s: %v\n", f, err)
				continue
			}
			debateCfg.Context += fmt.Sprintf("\n\n--- %s ---\n%s", f, string(data))
		}
	}

	// Determine which agents to use
	agentNames, _ := cmd.Flags().GetStringSlice("agents")
	if len(agentNames) == 0 {
		for name := range cfg.Agents {
			agentNames = append(agentNames, name)
		}
	}

	if len(agentNames) == 0 {
		return fmt.Errorf("no agents configured. Add agents to config.yaml or use --agents flag")
	}

	// Create agents from registry
	agents := make(map[string]agent.StreamAgent)
	var agentOrder []string
	agentRoles := make(map[string]string)

	for _, name := range agentNames {
		agentCfg, ok := cfg.Agents[name]
		if !ok {
			fmt.Fprintf(os.Stderr, "⚠ Warning: agent '%s' not found in config, skipping\n", name)
			continue
		}

		provider := agentCfg.Provider
		if provider == "" {
			provider = name // "claude", "openai", "gemini" match their provider names
		}

		a, err := agent.DefaultRegistry.Create(provider, agent.AgentConfig{
			Name:      name,
			Provider:  provider,
			Model:     agentCfg.Model,
			APIKey:    agentCfg.APIKey,
			BaseURL:   agentCfg.BaseURL,
			MaxTokens: agentCfg.MaxTokens,
			Role:      agentCfg.Role,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Warning: could not create agent '%s': %v\n", name, err)
			continue
		}

		agents[name] = a
		agentOrder = append(agentOrder, name)
		agentRoles[name] = agentCfg.Role
	}

	if len(agents) < 2 {
		return fmt.Errorf("need at least 2 agents for a debate, got %d", len(agents))
	}

	// Budget
	budget, _ := cmd.Flags().GetFloat64("budget")
	if budget <= 0 {
		budget = debateCfg.Budget
	}
	if budget <= 0 {
		budget = cfg.Cost.Budget
	}

	// Cost tracker
	costTracker := cost.NewTracker(budget)

	// Create debate view
	debateView := tui.NewDebateView(question, agentOrder, agentRoles, debateCfg.Rounds, budget)

	// Print initial header
	fmt.Println(tui.HeaderStyle.Render(" DebateGPT "))
	fmt.Println()

	// Create orchestrator
	orch := debate.NewOrchestrator(debate.OrchestratorConfig{
		Agents:       agents,
		AgentOrder:   agentOrder,
		DebateConfig: debateCfg,
		CostTracker:  costTracker,
		Budget:       budget,
		OnMuxEvent: func(event stream.MuxEvent) {
			debateView.ProcessEvent(event)
			debateView.SetCost(costTracker.TotalCost())
			// Clear and re-render
			fmt.Print("\033[H\033[2J")
			fmt.Println(tui.HeaderStyle.Render(" DebateGPT "))
			fmt.Println()
			fmt.Print(debateView.Render())
		},
	})

	// Run debate
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	debateView.SetRound(1)
	transcript, err := orch.Run(ctx, question)
	if err != nil {
		return fmt.Errorf("debate failed: %w", err)
	}

	// Clear screen for judge phase
	fmt.Print("\033[H\033[2J")
	fmt.Println(tui.HeaderStyle.Render(" DebateGPT — Judge Synthesizing "))
	fmt.Println()
	fmt.Println(tui.StreamingStyle.Render("🔍 Judge is synthesizing the debate..."))

	// Run judge
	judgeMode := debateCfg.JudgeMode
	if judgeMode == "" {
		judgeMode = "separate"
	}

	var judgeModel agent.StreamAgent
	if cfg.Judge.Model != "" {
		// Try to find the judge model in configured agents
		for _, a := range agents {
			if a.Model() == cfg.Judge.Model {
				judgeModel = a
				break
			}
		}
	}

	j := judge.NewJudge(judgeMode, agents, agentOrder, judgeModel, cfg.Judge.ParticipantAgent)

	judgeCh, err := j.Synthesize(ctx, transcript)
	if err != nil {
		return fmt.Errorf("judge synthesis failed: %w", err)
	}

	// Collect judge output
	var judgeOutput strings.Builder
	for event := range judgeCh {
		switch event.Type {
		case agent.EventDelta:
			judgeOutput.WriteString(event.Delta)
			fmt.Print(event.Delta)
		case agent.EventDone:
			if event.Metrics != nil {
				costTracker.RecordUsage("judge", 0, "judge", event.Metrics.TokensIn, event.Metrics.TokensOut)
			}
		}
	}

	fmt.Println()

	// Generate markdown output
	includeTranscript, _ := cmd.Flags().GetBool("transcript")
	md := output.GenerateMarkdown(question, judgeOutput.String(), transcript, costTracker.TotalCost(), judgeMode, includeTranscript)

	// Save to file
	outputPath, _ := cmd.Flags().GetString("output")
	if outputPath == "" {
		outputDir := cfg.Output.Directory
		if outputDir == "" {
			outputDir = "./brainstorms"
		}
		outputPath = output.GenerateFilename(question, outputDir)
	}

	if err := output.WriteFile(outputPath, md); err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Warning: could not save output: %v\n", err)
	} else {
		fmt.Println()
		fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("📄 Saved to: %s", outputPath)))
	}

	// Cost summary
	fmt.Println(costTracker.Summary(agentOrder, len(transcript.Rounds)))

	return nil
}
