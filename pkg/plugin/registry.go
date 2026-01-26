// Package plugin provides plugin management for seedcli
package plugin

import (
	"fmt"
	"sync"

	"github.com/kiridharan/seedcli/pkg/core"
)

// Registry manages all registered components
type Registry struct {
	mu sync.RWMutex

	adapters      map[string]core.Adapter
	schemaEngines map[string]core.SchemaEngine
	generators    map[string]core.Generator
	validators    map[string]core.Validator
	plugins       map[string]core.Plugin
}

// NewRegistry creates a new component registry
func NewRegistry() *Registry {
	return &Registry{
		adapters:      make(map[string]core.Adapter),
		schemaEngines: make(map[string]core.SchemaEngine),
		generators:    make(map[string]core.Generator),
		validators:    make(map[string]core.Validator),
		plugins:       make(map[string]core.Plugin),
	}
}

// =============================================================================
// ADAPTER MANAGEMENT
// =============================================================================

// RegisterAdapter registers a database adapter
func (r *Registry) RegisterAdapter(name string, adapter core.Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = adapter
}

// GetAdapter retrieves a registered adapter
func (r *Registry) GetAdapter(name string) (core.Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.adapters[name]
	return adapter, ok
}

// ListAdapters returns all registered adapter names
func (r *Registry) ListAdapters() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}

// =============================================================================
// SCHEMA ENGINE MANAGEMENT
// =============================================================================

// RegisterSchemaEngine registers a schema engine
func (r *Registry) RegisterSchemaEngine(name string, engine core.SchemaEngine) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schemaEngines[name] = engine
}

// GetSchemaEngine retrieves a registered schema engine
func (r *Registry) GetSchemaEngine(name string) (core.SchemaEngine, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	engine, ok := r.schemaEngines[name]
	return engine, ok
}

// ListSchemaEngines returns all registered schema engine names
func (r *Registry) ListSchemaEngines() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.schemaEngines))
	for name := range r.schemaEngines {
		names = append(names, name)
	}
	return names
}

// =============================================================================
// GENERATOR MANAGEMENT
// =============================================================================

// RegisterGenerator registers a data generator
func (r *Registry) RegisterGenerator(name string, gen core.Generator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.generators[name] = gen
}

// GetGenerator retrieves a registered generator
func (r *Registry) GetGenerator(name string) (core.Generator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	gen, ok := r.generators[name]
	return gen, ok
}

// ListGenerators returns all registered generator names
func (r *Registry) ListGenerators() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.generators))
	for name := range r.generators {
		names = append(names, name)
	}
	return names
}

// =============================================================================
// VALIDATOR MANAGEMENT
// =============================================================================

// RegisterValidator registers a data validator
func (r *Registry) RegisterValidator(name string, val core.Validator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.validators[name] = val
}

// GetValidator retrieves a registered validator
func (r *Registry) GetValidator(name string) (core.Validator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.validators[name]
	return val, ok
}

// ListValidators returns all registered validator names
func (r *Registry) ListValidators() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.validators))
	for name := range r.validators {
		names = append(names, name)
	}
	return names
}

// =============================================================================
// PLUGIN MANAGEMENT
// =============================================================================

// RegisterPlugin registers a plugin
func (r *Registry) RegisterPlugin(plugin core.Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := plugin.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	r.plugins[name] = plugin

	// Register plugin-provided components
	switch p := plugin.(type) {
	case core.AdapterPlugin:
		r.adapters[name] = p.Adapter()
	case core.GeneratorPlugin:
		for _, gen := range p.Generators() {
			r.generators[name+"_"+fmt.Sprintf("%d", gen.Priority())] = gen
		}
	case core.ValidatorPlugin:
		for _, val := range p.Validators() {
			r.validators[name+"_"+val.Name()] = val
		}
	}

	return nil
}

// GetPlugin retrieves a registered plugin
func (r *Registry) GetPlugin(name string) (core.Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	plugin, ok := r.plugins[name]
	return plugin, ok
}

// ListPlugins returns all registered plugin names
func (r *Registry) ListPlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

// LoadPlugins loads plugins from a directory
// TODO: Implement dynamic plugin loading (e.g., via Go plugins or WASM)
func (r *Registry) LoadPlugins(dir string) error {
	// For now, this is a placeholder for future plugin loading support
	// Plugins could be:
	// - Go plugins (.so files)
	// - WASM modules
	// - External processes communicating via RPC
	return nil
}

// =============================================================================
// GLOBAL REGISTRY
// =============================================================================

var globalRegistry = NewRegistry()

// GetRegistry returns the global registry
func GetRegistry() *Registry {
	return globalRegistry
}

// Register provides a fluent interface for registration
type Register struct {
	registry *Registry
}

// NewRegister creates a new registration helper
func NewRegister(r *Registry) *Register {
	return &Register{registry: r}
}

// Adapter registers an adapter
func (r *Register) Adapter(name string, adapter core.Adapter) *Register {
	r.registry.RegisterAdapter(name, adapter)
	return r
}

// SchemaEngine registers a schema engine
func (r *Register) SchemaEngine(name string, engine core.SchemaEngine) *Register {
	r.registry.RegisterSchemaEngine(name, engine)
	return r
}

// Generator registers a generator
func (r *Register) Generator(name string, gen core.Generator) *Register {
	r.registry.RegisterGenerator(name, gen)
	return r
}

// Validator registers a validator
func (r *Register) Validator(name string, val core.Validator) *Register {
	r.registry.RegisterValidator(name, val)
	return r
}

// Plugin registers a plugin
func (r *Register) Plugin(plugin core.Plugin) *Register {
	r.registry.RegisterPlugin(plugin)
	return r
}
