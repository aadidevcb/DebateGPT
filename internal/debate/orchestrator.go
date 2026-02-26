package debate

import (
	"context"
	"fmt"
	"sync"

	"github.com/aadidev/debategpt/internal/agent"
	"github.com/aadidev/debategpt/internal/cost"
	"github.com/aadidev/debategpt/internal/stream"
	"golang.org/x/sync/errgroup"
)

// Orchestrator manages the debate flow across rounds.
type Orchestrator struct {
	agents      map[string]agent.StreamAgent
	agentOrder  []string
	promptBuilder *PromptBuilder
	costTracker *cost.Tracker
	budgetGuard *cost.BudgetGuard
	config      DebateFileConfig
	onMuxEvent  func(stream.MuxEvent) // callback for TUI updates
}

// OrchestratorConfig holds orchestrator initialization options.
type OrchestratorConfig struct {
	Agents        map[string]agent.StreamAgent
	AgentOrder    []string
	DebateConfig  DebateFileConfig
	CostTracker   *cost.Tracker
	Budget        float64
	OnMuxEvent    func(stream.MuxEvent) // optional callback for streaming events
}

// NewOrchestrator creates a new debate orchestrator.
func NewOrchestrator(cfg OrchestratorConfig) *Orchestrator {
	perspectives := make(map[string]string)
	perspectiveNames := []string{"pragmatist", "architect", "contrarian", "security", "performance"}

	// Match agents to perspectives
	i := 0
	for _, name := range cfg.AgentOrder {
		if i < len(perspectiveNames) {
			if p, ok := cfg.DebateConfig.Perspectives[perspectiveNames[i]]; ok {
				perspectives[name] = p
			}
		}
		i++
	}

	// Override if agent has explicit perspective from config
	for perspName, perspPrompt := range cfg.DebateConfig.Perspectives {
		for _, agentName := range cfg.AgentOrder {
			if agentName == perspName {
				perspectives[agentName] = perspPrompt
			}
		}
	}

	return &Orchestrator{
		agents:        cfg.Agents,
		agentOrder:    cfg.AgentOrder,
		promptBuilder: NewPromptBuilder(perspectives, cfg.DebateConfig.Constraints, cfg.DebateConfig.Context),
		costTracker:   cfg.CostTracker,
		budgetGuard:   cost.NewBudgetGuard(cfg.CostTracker, cfg.Budget),
		config:        cfg.DebateConfig,
		onMuxEvent:    cfg.OnMuxEvent,
	}
}

// Run executes the full debate and returns the completed transcript.
func (o *Orchestrator) Run(ctx context.Context, question string) (*Transcript, error) {
	transcript := NewTranscript(question, o.agentOrder)
	rounds := o.config.Rounds
	if rounds <= 0 {
		rounds = 2
	}

	for roundNum := 1; roundNum <= rounds; roundNum++ {
		// Budget check
		if err := o.budgetGuard.Check(0); err != nil {
			break
		}

		var phase string
		var round Round
		var err error

		switch roundNum {
		case 1:
			phase = "initial"
			round, err = o.runInitialRound(ctx, question, roundNum)
		default:
			if roundNum == rounds {
				phase = "defend"
				round, err = o.runDefendRound(ctx, question, transcript, roundNum)
			} else {
				phase = "critique"
				round, err = o.runCritiqueRound(ctx, question, transcript, roundNum)
			}
		}

		if err != nil {
			return transcript, fmt.Errorf("round %d (%s): %w", roundNum, phase, err)
		}

		round.Number = roundNum
		round.Phase = phase
		transcript.AddRound(round)
	}

	transcript.Finalize()
	return transcript, nil
}

// runInitialRound executes the first round where all agents answer independently.
func (o *Orchestrator) runInitialRound(ctx context.Context, question string, roundNum int) (Round, error) {
	messages := make(map[string][]agent.Message)
	for _, name := range o.agentOrder {
		messages[name] = []agent.Message{
			{Role: "system", Content: o.promptBuilder.SystemPrompt(name)},
			{Role: "user", Content: o.promptBuilder.InitialPrompt(question)},
		}
	}
	return o.runStreamingRound(ctx, messages, roundNum)
}

// runCritiqueRound executes a critique round where agents see others' responses.
func (o *Orchestrator) runCritiqueRound(ctx context.Context, question string, transcript *Transcript, roundNum int) (Round, error) {
	messages := make(map[string][]agent.Message)
	for _, name := range o.agentOrder {
		otherResponses := transcript.FormatOtherResponses(roundNum-1, name)
		messages[name] = []agent.Message{
			{Role: "system", Content: o.promptBuilder.SystemPrompt(name)},
			{Role: "user", Content: o.promptBuilder.CritiquePrompt(question, otherResponses)},
		}
	}
	return o.runStreamingRound(ctx, messages, roundNum)
}

// runDefendRound executes the final defend-or-concede round.
func (o *Orchestrator) runDefendRound(ctx context.Context, question string, transcript *Transcript, roundNum int) (Round, error) {
	messages := make(map[string][]agent.Message)
	for _, name := range o.agentOrder {
		otherResponses := transcript.FormatOtherResponses(roundNum-1, name)
		messages[name] = []agent.Message{
			{Role: "system", Content: o.promptBuilder.SystemPrompt(name)},
			{Role: "user", Content: o.promptBuilder.DefendPrompt(question, otherResponses)},
		}
	}
	return o.runStreamingRound(ctx, messages, roundNum)
}

// runStreamingRound runs a single round with all agents streaming in parallel.
func (o *Orchestrator) runStreamingRound(ctx context.Context, messages map[string][]agent.Message, roundNum int) (Round, error) {
	// Start all streams in parallel
	streams := make(map[string]<-chan agent.StreamEvent)
	var mu sync.Mutex
	g, streamCtx := errgroup.WithContext(ctx)

	for _, name := range o.agentOrder {
		agnt := o.agents[name]
		msgs := messages[name]
		n := name

		g.Go(func() error {
			ch, err := agnt.StreamGenerate(streamCtx, msgs)
			if err != nil {
				return fmt.Errorf("agent %s stream start: %w", n, err)
			}
			mu.Lock()
			streams[n] = ch
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return Round{}, err
	}

	// Multiplex all streams
	mux := stream.NewMultiplexer(streams)
	collector := stream.NewCollector(o.agentOrder)
	muxCh := mux.Listen(ctx)

	for event := range muxCh {
		collector.Process(event)
		if o.onMuxEvent != nil {
			o.onMuxEvent(event)
		}
	}

	// Build round from collected responses
	round := Round{
		Responses: make(map[string]RoundEntry),
	}

	for _, name := range o.agentOrder {
		resp, ok := collector.GetResponse(name)
		if !ok {
			continue
		}

		round.Responses[name] = RoundEntry{
			AgentName: name,
			Content:   resp.Content,
			TokensIn:  resp.TokensIn,
			TokensOut: resp.TokensOut,
			Latency:   resp.Latency,
		}

		// Track cost
		o.costTracker.RecordUsage(name, roundNum, "debate", resp.TokensIn, resp.TokensOut)
	}

	return round, nil
}
