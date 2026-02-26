package agent

import (
	"fmt"
	"sync"
)

// AgentFactory creates an Agent from configuration.
type AgentFactory func(cfg AgentConfig) (StreamAgent, error)

// Registry holds all known agent provider factories.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]AgentFactory
}

// NewRegistry creates a new empty agent registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]AgentFactory),
	}
}

// DefaultRegistry is the global agent registry.
var DefaultRegistry = NewRegistry()

// Register adds a new agent provider factory to the registry.
func (r *Registry) Register(name string, factory AgentFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Create instantiates an agent by provider name with the given config.
func (r *Registry) Create(name string, cfg AgentConfig) (StreamAgent, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown agent provider: %s (registered: %v)", name, r.List())
	}
	return factory(cfg)
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// Has checks if a provider is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.factories[name]
	return ok
}
