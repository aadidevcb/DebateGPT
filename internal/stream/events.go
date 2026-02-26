package stream

import "github.com/aadidev/debategpt/internal/agent"

// MuxEvent wraps a stream event with the agent name it came from.
type MuxEvent struct {
	AgentName string
	Event     agent.StreamEvent
}
