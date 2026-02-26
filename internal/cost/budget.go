package cost

import "fmt"

// BudgetGuard enforces a budget limit before each round.
type BudgetGuard struct {
	tracker *Tracker
	budget  float64
}

// NewBudgetGuard creates a budget guard. Budget of 0 means unlimited.
func NewBudgetGuard(tracker *Tracker, budget float64) *BudgetGuard {
	return &BudgetGuard{tracker: tracker, budget: budget}
}

// Check returns an error if starting the next round would likely exceed the budget.
// estimatedCost is the projected cost of the next round.
func (g *BudgetGuard) Check(estimatedCost float64) error {
	if g.budget <= 0 {
		return nil // unlimited
	}

	spent := g.tracker.TotalCost()
	if spent+estimatedCost > g.budget {
		return fmt.Errorf(
			"budget exceeded: spent $%.3f + estimated $%.3f = $%.3f > budget $%.3f",
			spent, estimatedCost, spent+estimatedCost, g.budget,
		)
	}
	return nil
}

// EstimateRoundCost gives a rough estimate of one round's cost
// based on the number of agents, average input/output tokens, and the model.
func EstimateRoundCost(agentModels []string, avgInputTokens, avgOutputTokens int) float64 {
	var total float64
	for _, model := range agentModels {
		total += CalculateCost(model, avgInputTokens, avgOutputTokens)
	}
	return total
}
