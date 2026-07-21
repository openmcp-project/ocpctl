package provider

import "fmt"

var registry = map[string]ClusterProvider{}

// Register adds a provider. Call from init() in each provider's file.
func Register(p ClusterProvider) {
	registry[p.Name()] = p
}

// Get returns the named provider or an error.
func Get(name string) (ClusterProvider, error) {
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown cluster provider %q", name)
	}
	return p, nil
}
