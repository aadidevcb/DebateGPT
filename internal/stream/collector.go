package stream

import (
	"strings"
	"sync"

	"github.com/aadidev/debategpt/internal/agent"
)

// Collector accumulates streamed deltas into complete responses per agent.
type Collector struct {
	mu        sync.Mutex
	buffers   map[string]*strings.Builder
	metrics   map[string]*agent.Metrics
	completed map[string]bool
}

// NewCollector creates a new stream collector.
func NewCollector(agentNames []string) *Collector {
	c := &Collector{
		buffers:   make(map[string]*strings.Builder, len(agentNames)),
		metrics:   make(map[string]*agent.Metrics, len(agentNames)),
		completed: make(map[string]bool, len(agentNames)),
	}
	for _, name := range agentNames {
		c.buffers[name] = &strings.Builder{}
	}
	return c
}

// Process handles a MuxEvent and updates internal state.
func (c *Collector) Process(event MuxEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch event.Event.Type {
	case agent.EventDelta:
		if buf, ok := c.buffers[event.AgentName]; ok {
			buf.WriteString(event.Event.Delta)
		}
	case agent.EventDone:
		c.completed[event.AgentName] = true
		c.metrics[event.AgentName] = event.Event.Metrics
	case agent.EventError:
		c.completed[event.AgentName] = true
	}
}

// GetResponse returns the complete accumulated response for an agent.
func (c *Collector) GetResponse(agentName string) (agent.Response, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	buf, ok := c.buffers[agentName]
	if !ok {
		return agent.Response{}, false
	}

	resp := agent.Response{
		Content: buf.String(),
	}
	if m, ok := c.metrics[agentName]; ok {
		resp.TokensIn = m.TokensIn
		resp.TokensOut = m.TokensOut
		resp.Latency = m.Latency
		resp.FinishReason = m.FinishReason
	}
	return resp, true
}

// AllDone checks if all agents have completed.
func (c *Collector) AllDone() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.completed) == len(c.buffers)
}

// GetAllResponses returns responses for all agents.
func (c *Collector) GetAllResponses() map[string]agent.Response {
	c.mu.Lock()
	defer c.mu.Unlock()

	responses := make(map[string]agent.Response, len(c.buffers))
	for name, buf := range c.buffers {
		resp := agent.Response{Content: buf.String()}
		if m, ok := c.metrics[name]; ok {
			resp.TokensIn = m.TokensIn
			resp.TokensOut = m.TokensOut
			resp.Latency = m.Latency
			resp.FinishReason = m.FinishReason
		}
		responses[name] = resp
	}
	return responses
}
