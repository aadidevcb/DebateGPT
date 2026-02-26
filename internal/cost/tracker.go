package cost

import (
	"fmt"
	"strings"
	"sync"
)

// CostEntry records cost for a single agent in a single round.
type CostEntry struct {
	Agent    string  `json:"agent"`
	Round    int     `json:"round"`
	Phase    string  `json:"phase"` // "debate" or "judge"
	TokensIn int     `json:"tokens_in"`
	TokensOut int    `json:"tokens_out"`
	CostUSD  float64 `json:"cost_usd"`
}

// Tracker tracks token usage and costs across a debate.
type Tracker struct {
	mu      sync.Mutex
	entries []CostEntry
	budget  float64
}

// NewTracker creates a new cost tracker with an optional budget (0 = unlimited).
func NewTracker(budget float64) *Tracker {
	return &Tracker{
		budget: budget,
	}
}

// Record adds a cost entry.
func (t *Tracker) Record(entry CostEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries = append(t.entries, entry)
}

// RecordUsage records token usage for an agent/round combination.
func (t *Tracker) RecordUsage(agentName string, round int, phase string, tokensIn, tokensOut int) {
	costUSD := CalculateCost(agentName, tokensIn, tokensOut)
	t.Record(CostEntry{
		Agent:     agentName,
		Round:     round,
		Phase:     phase,
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		CostUSD:   costUSD,
	})
}

// TotalCost returns the total cost so far.
func (t *Tracker) TotalCost() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	var total float64
	for _, e := range t.entries {
		total += e.CostUSD
	}
	return total
}

// AgentCost returns total cost for a specific agent.
func (t *Tracker) AgentCost(agentName string) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	var total float64
	for _, e := range t.entries {
		if e.Agent == agentName {
			total += e.CostUSD
		}
	}
	return total
}

// BudgetRemaining returns how much budget remains. Returns -1 if unlimited.
func (t *Tracker) BudgetRemaining() float64 {
	if t.budget <= 0 {
		return -1
	}
	return t.budget - t.TotalCost()
}

// OverBudget returns true if total cost exceeds the budget.
func (t *Tracker) OverBudget() bool {
	if t.budget <= 0 {
		return false
	}
	return t.TotalCost() > t.budget
}

// Summary returns a formatted cost summary string.
func (t *Tracker) Summary(agentNames []string, totalRounds int) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	var sb strings.Builder
	sb.WriteString("\n╭──────────────────── Cost Summary ────────────────────╮\n")

	// Header
	sb.WriteString(fmt.Sprintf("│ %-12s", "Agent"))
	for r := 1; r <= totalRounds; r++ {
		sb.WriteString(fmt.Sprintf(" Round %-5d", r))
	}
	sb.WriteString(fmt.Sprintf(" %-10s %-8s│\n", "Judge", "Total"))

	// Per-agent rows
	for _, name := range agentNames {
		sb.WriteString(fmt.Sprintf("│ %-12s", name))
		var agentTotal float64

		for r := 1; r <= totalRounds; r++ {
			roundCost := t.roundCost(name, r, "debate")
			agentTotal += roundCost
			sb.WriteString(fmt.Sprintf(" $%-9.3f", roundCost))
		}

		judgeCost := t.roundCost(name, 0, "judge")
		agentTotal += judgeCost
		if judgeCost > 0 {
			sb.WriteString(fmt.Sprintf(" $%-8.3f", judgeCost))
		} else {
			sb.WriteString(fmt.Sprintf(" %-10s", "—"))
		}

		sb.WriteString(fmt.Sprintf(" $%-6.3f│\n", agentTotal))
	}

	// Total row
	sb.WriteString("│──────────────────────────────────────────────────────│\n")
	sb.WriteString(fmt.Sprintf("│ Total%*s$%-6.3f│\n", 47, "", t.totalUnsafe()))
	sb.WriteString("╰──────────────────────────────────────────────────────╯\n")

	return sb.String()
}

func (t *Tracker) roundCost(agentName string, round int, phase string) float64 {
	var total float64
	for _, e := range t.entries {
		if e.Agent == agentName && e.Round == round && e.Phase == phase {
			total += e.CostUSD
		}
	}
	return total
}

func (t *Tracker) totalUnsafe() float64 {
	var total float64
	for _, e := range t.entries {
		total += e.CostUSD
	}
	return total
}
