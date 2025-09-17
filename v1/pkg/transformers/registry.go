package transformers

import (
	"fmt"
	"sync"

	"helm-charts-migrator/v1/pkg/logger"
)

// Transformer defines the interface that all transformers must implement
type Transformer interface {
	// Name returns the unique name of the transformer
	Name() string

	// Description returns a human-readable description
	Description() string

	// Transform applies the transformation to the input data
	Transform(data interface{}) (interface{}, error)

	// Validate checks if this transformer can handle the input
	Validate(data interface{}) error

	// Priority returns the execution priority (lower numbers run first)
	Priority() int
}

// TransformerRegistry manages all registered transformers
type TransformerRegistry struct {
	transformers map[string]Transformer
	order        []string // Maintains registration order
	mu           sync.RWMutex
	log          *logger.NamedLogger
}

// NewTransformerRegistry creates a new transformer registry
func NewTransformerRegistry() *TransformerRegistry {
	return &TransformerRegistry{
		transformers: make(map[string]Transformer),
		order:        []string{},
		log:          logger.WithName("transformer-registry"),
	}
}

// Register adds a new transformer to the registry
func (r *TransformerRegistry) Register(transformer Transformer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := transformer.Name()
	if name == "" {
		return fmt.Errorf("transformer name cannot be empty")
	}

	if _, exists := r.transformers[name]; exists {
		return fmt.Errorf("transformer %s already registered", name)
	}

	r.transformers[name] = transformer
	r.order = append(r.order, name)

	r.log.InfoS("Registered transformer",
		"name", name,
		"description", transformer.Description(),
		"priority", transformer.Priority())

	return nil
}

// Unregister removes a transformer from the registry
func (r *TransformerRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.transformers[name]; !exists {
		return fmt.Errorf("transformer %s not found", name)
	}

	delete(r.transformers, name)

	// Remove from order slice
	newOrder := []string{}
	for _, n := range r.order {
		if n != name {
			newOrder = append(newOrder, n)
		}
	}
	r.order = newOrder

	r.log.InfoS("Unregistered transformer", "name", name)
	return nil
}

// Get returns a transformer by name
func (r *TransformerRegistry) Get(name string) (Transformer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	transformer, exists := r.transformers[name]
	if !exists {
		return nil, fmt.Errorf("transformer %s not found", name)
	}

	return transformer, nil
}

// List returns all registered transformer names
func (r *TransformerRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, len(r.order))
	copy(names, r.order)
	return names
}

// ListByPriority returns transformer names sorted by priority
func (r *TransformerRegistry) ListByPriority() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a slice of transformers with their names
	type transformerInfo struct {
		name     string
		priority int
	}

	infos := make([]transformerInfo, 0, len(r.transformers))
	for name, t := range r.transformers {
		infos = append(infos, transformerInfo{
			name:     name,
			priority: t.Priority(),
		})
	}

	// Sort by priority (bubble sort for simplicity)
	for i := 0; i < len(infos); i++ {
		for j := i + 1; j < len(infos); j++ {
			if infos[j].priority < infos[i].priority {
				infos[i], infos[j] = infos[j], infos[i]
			}
		}
	}

	// Extract names
	names := make([]string, len(infos))
	for i, info := range infos {
		names[i] = info.name
	}

	return names
}

// Apply runs a specific transformer on the data
func (r *TransformerRegistry) Apply(name string, data interface{}) (interface{}, error) {
	transformer, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	// Validate input
	if err := transformer.Validate(data); err != nil {
		return nil, fmt.Errorf("validation failed for transformer %s: %w", name, err)
	}

	// Apply transformation
	result, err := transformer.Transform(data)
	if err != nil {
		return nil, fmt.Errorf("transformation %s failed: %w", name, err)
	}

	r.log.V(3).InfoS("Applied transformer", "name", name)
	return result, nil
}

// ApplyChain runs multiple transformers in sequence
func (r *TransformerRegistry) ApplyChain(names []string, data interface{}) (interface{}, error) {
	result := data

	for _, name := range names {
		transformed, err := r.Apply(name, result)
		if err != nil {
			return nil, fmt.Errorf("chain failed at transformer %s: %w", name, err)
		}
		result = transformed
	}

	r.log.V(2).InfoS("Applied transformer chain", "count", len(names))
	return result, nil
}

// ApplyAll runs all registered transformers in priority order
func (r *TransformerRegistry) ApplyAll(data interface{}) (interface{}, error) {
	names := r.ListByPriority()
	return r.ApplyChain(names, data)
}

// ApplyConditional runs transformers that pass the condition check
func (r *TransformerRegistry) ApplyConditional(data interface{}, condition func(Transformer) bool) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := data
	applied := 0

	// Get transformers in priority order
	for _, name := range r.ListByPriority() {
		transformer := r.transformers[name]

		// Check condition
		if !condition(transformer) {
			continue
		}

		// Validate
		if err := transformer.Validate(result); err != nil {
			r.log.V(3).InfoS("Skipping transformer due to validation",
				"name", name,
				"error", err)
			continue
		}

		// Transform
		transformed, err := transformer.Transform(result)
		if err != nil {
			return nil, fmt.Errorf("conditional transformation %s failed: %w", name, err)
		}

		result = transformed
		applied++
	}

	r.log.V(2).InfoS("Applied conditional transformers", "applied", applied)
	return result, nil
}

// Clear removes all registered transformers
func (r *TransformerRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.transformers = make(map[string]Transformer)
	r.order = []string{}

	r.log.InfoS("Cleared all transformers")
}

// Count returns the number of registered transformers
func (r *TransformerRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.transformers)
}

// GetAllByPriority returns all transformers sorted by priority (lower priority runs first)
func (r *TransformerRegistry) GetAllByPriority() []Transformer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a slice of transformers
	type transformerWithPriority struct {
		transformer Transformer
		priority    int
	}

	items := make([]transformerWithPriority, 0, len(r.transformers))
	for _, t := range r.transformers {
		items = append(items, transformerWithPriority{
			transformer: t,
			priority:    t.Priority(),
		})
	}

	// Sort by priority (lower numbers run first)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].priority < items[i].priority {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// Extract transformers
	result := make([]Transformer, len(items))
	for i, item := range items {
		result[i] = item.transformer
	}

	return result
}

// GetInfo returns information about all registered transformers
func (r *TransformerRegistry) GetInfo() []TransformerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]TransformerInfo, 0, len(r.transformers))
	for _, name := range r.order {
		t := r.transformers[name]
		infos = append(infos, TransformerInfo{
			Name:        t.Name(),
			Description: t.Description(),
			Priority:    t.Priority(),
		})
	}

	return infos
}

// TransformerInfo contains metadata about a transformer
type TransformerInfo struct {
	Name        string
	Description string
	Priority    int
}
