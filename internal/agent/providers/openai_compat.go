package providers

import (
	"github.com/aadidev/debategpt/internal/agent"
)

// NewOpenAICompatAgent creates an OpenAI-compatible agent that re-uses
// the OpenAI adapter with a custom base URL. Works with Ollama, Groq,
// Together, Mistral, DeepSeek, and any other OpenAI-compatible API.
func NewOpenAICompatAgent(cfg agent.AgentConfig) (agent.StreamAgent, error) {
	if cfg.BaseURL == "" {
		return nil, ErrBaseURLRequired
	}
	return NewOpenAIAgent(cfg)
}

// ErrBaseURLRequired indicates that base_url is required for openai-compatible providers.
var ErrBaseURLRequired = agentError("openai-compatible provider requires base_url in config")

type agentError string

func (e agentError) Error() string { return string(e) }
