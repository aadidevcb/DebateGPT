package agent

import (
	"context"
	"time"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// Response represents a completed LLM response.
type Response struct {
	Content      string        `json:"content"`
	TokensIn     int           `json:"tokens_in"`
	TokensOut    int           `json:"tokens_out"`
	FinishReason string        `json:"finish_reason"`
	Latency      time.Duration `json:"latency"`
}

// Metrics holds token usage and timing info, populated on stream completion.
type Metrics struct {
	TokensIn     int           `json:"tokens_in"`
	TokensOut    int           `json:"tokens_out"`
	Latency      time.Duration `json:"latency"`
	FinishReason string        `json:"finish_reason"`
}

// EventType represents the kind of stream event.
type EventType int

const (
	EventDelta EventType = iota // A text chunk
	EventDone                   // Stream finished
	EventError                  // An error occurred
)

// StreamEvent is emitted by streaming agents.
type StreamEvent struct {
	Type    EventType
	Delta   string   // Text chunk (for EventDelta)
	Metrics *Metrics // Populated on EventDone
	Error   error    // Populated on EventError
}

// AgentConfig holds the configuration for a single agent.
type AgentConfig struct {
	Name       string `mapstructure:"name"`
	Provider   string `mapstructure:"provider"`
	Model      string `mapstructure:"model"`
	APIKey     string `mapstructure:"api_key"`
	BaseURL    string `mapstructure:"base_url"`
	MaxTokens  int    `mapstructure:"max_tokens"`
	Role       string `mapstructure:"role"`
	RolePrompt string `mapstructure:"role_prompt"`
}

// Agent is the basic interface every LLM provider implements.
type Agent interface {
	Name() string
	Model() string
	Generate(ctx context.Context, messages []Message) (Response, error)
}

// StreamAgent extends Agent with streaming support.
type StreamAgent interface {
	Agent
	StreamGenerate(ctx context.Context, messages []Message) (<-chan StreamEvent, error)
}
