package stream

import (
	"context"
	"sync"

	"github.com/aadidev/debategpt/internal/agent"
)

// Multiplexer fans out to N agent streams and merges them into a single channel.
type Multiplexer struct {
	streams map[string]<-chan agent.StreamEvent
}

// NewMultiplexer creates a new multiplexer from a map of agent name → stream channels.
func NewMultiplexer(streams map[string]<-chan agent.StreamEvent) *Multiplexer {
	return &Multiplexer{streams: streams}
}

// Listen merges all agent streams into a single MuxEvent channel.
// The returned channel is closed when all agent streams are done.
func (m *Multiplexer) Listen(ctx context.Context) <-chan MuxEvent {
	out := make(chan MuxEvent, len(m.streams)*10)
	var wg sync.WaitGroup

	for name, ch := range m.streams {
		wg.Add(1)
		go func(agentName string, stream <-chan agent.StreamEvent) {
			defer wg.Done()
			for {
				select {
				case event, ok := <-stream:
					if !ok {
						return
					}
					select {
					case out <- MuxEvent{AgentName: agentName, Event: event}:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(name, ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
