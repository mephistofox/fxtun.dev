package payment

import "fmt"

// Registry maps provider names to Provider instances
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new empty registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

// Get returns a provider by name
func (r *Registry) Get(name string) (Provider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("payment provider %q not registered", name)
	}
	return p, nil
}

// Has checks if a provider is registered
func (r *Registry) Has(name string) bool {
	_, ok := r.providers[name]
	return ok
}
