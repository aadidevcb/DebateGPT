package cost

// ModelPricing holds per-million-token pricing for a model.
type ModelPricing struct {
	InputPerMillion  float64
	OutputPerMillion float64
}

// PricingTable maps model identifiers to their pricing.
// Prices in USD per million tokens (as of early 2026).
var PricingTable = map[string]ModelPricing{
	// OpenAI
	"gpt-4o":       {InputPerMillion: 2.50, OutputPerMillion: 10.00},
	"gpt-4o-mini":  {InputPerMillion: 0.15, OutputPerMillion: 0.60},
	"gpt-4-turbo":  {InputPerMillion: 10.00, OutputPerMillion: 30.00},
	"o1":           {InputPerMillion: 15.00, OutputPerMillion: 60.00},
	"o1-mini":      {InputPerMillion: 3.00, OutputPerMillion: 12.00},

	// Anthropic
	"claude-sonnet-4-20250514":  {InputPerMillion: 3.00, OutputPerMillion: 15.00},
	"claude-3-5-sonnet-20241022": {InputPerMillion: 3.00, OutputPerMillion: 15.00},
	"claude-3-5-haiku-20241022":  {InputPerMillion: 0.80, OutputPerMillion: 4.00},
	"claude-3-opus-20240229":     {InputPerMillion: 15.00, OutputPerMillion: 75.00},

	// Google
	"gemini-2.5-pro":    {InputPerMillion: 1.25, OutputPerMillion: 10.00},
	"gemini-2.5-flash":  {InputPerMillion: 0.15, OutputPerMillion: 0.60},
	"gemini-2.0-flash":  {InputPerMillion: 0.10, OutputPerMillion: 0.40},
	"gemini-1.5-pro":    {InputPerMillion: 1.25, OutputPerMillion: 5.00},

	// DeepSeek
	"deepseek-chat":     {InputPerMillion: 0.27, OutputPerMillion: 1.10},
	"deepseek-reasoner": {InputPerMillion: 0.55, OutputPerMillion: 2.19},

	// Mistral
	"mistral-large-latest": {InputPerMillion: 2.00, OutputPerMillion: 6.00},
	"mistral-small-latest": {InputPerMillion: 0.10, OutputPerMillion: 0.30},
}

// CalculateCost estimates the cost for a given agent's token usage.
// Falls back to a reasonable default if the model isn't in the pricing table.
func CalculateCost(model string, tokensIn, tokensOut int) float64 {
	pricing, ok := PricingTable[model]
	if !ok {
		// Default pricing for unknown models
		pricing = ModelPricing{InputPerMillion: 1.00, OutputPerMillion: 3.00}
	}

	inputCost := float64(tokensIn) / 1_000_000.0 * pricing.InputPerMillion
	outputCost := float64(tokensOut) / 1_000_000.0 * pricing.OutputPerMillion

	return inputCost + outputCost
}
