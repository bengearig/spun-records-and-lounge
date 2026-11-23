package feature

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	features map[string]Feature
}

func NewRegistry() *Registry {
	return &Registry{features: map[string]Feature{}}
}

// Default is populated by init() in feature packages.
var Default = NewRegistry()

func (r *Registry) Register(f Feature) {
	name := f.Name()
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.features[name]; ok {
		panic(fmt.Sprintf("feature %q is already registered", name))
	}
	r.features[name] = f
}

func (r *Registry) Lookup(name string) (Feature, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lookup(name)
}

func (r *Registry) lookup(name string) (Feature, error) {
	if f, ok := r.features[name]; ok {
		return f, nil
	}
	known := "none"
	if names := r.names(); len(names) > 0 {
		known = strings.Join(names, ", ")
	}
	return nil, fmt.Errorf("unknown feature %q (known features: %s)", name, known)
}

func (r *Registry) names() []string {
	names := make([]string, 0, len(r.features))
	for name := range r.features {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// All returns every registered feature sorted by name.
func (r *Registry) All() []Feature {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := r.names()
	out := make([]Feature, 0, len(names))
	for _, name := range names {
		out = append(out, r.features[name])
	}
	return out
}

// Enabled returns the features set to true in cfg, sorted by name. Every
// name in cfg must be registered, even when set to false.
func (r *Registry) Enabled(cfg map[string]bool) ([]Feature, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(cfg))
	for name := range cfg {
		names = append(names, name)
	}
	slices.Sort(names)
	var out []Feature
	for _, name := range names {
		f, err := r.lookup(name)
		if err != nil {
			return nil, err
		}
		if cfg[name] {
			out = append(out, f)
		}
	}
	return out, nil
}
