package register

import (
	"github.com/aadidev/debategpt/internal/agent"
	"github.com/aadidev/debategpt/internal/agent/providers"
)

func init() {
	agent.DefaultRegistry.Register("openai", providers.NewOpenAIAgent)
	agent.DefaultRegistry.Register("claude", providers.NewClaudeAgent)
	agent.DefaultRegistry.Register("gemini", providers.NewGeminiAgent)
	agent.DefaultRegistry.Register("openai-compatible", providers.NewOpenAICompatAgent)
}
