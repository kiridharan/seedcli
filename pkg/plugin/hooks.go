// Package plugin provides plugin hooks for seedcli
package plugin

import (
	"context"

	"github.com/kiridharan/seedcli/pkg/core"
)

// HookManager manages and executes plugin hooks
type HookManager struct {
	plugins []core.Plugin
}

// NewHookManager creates a new hook manager
func NewHookManager() *HookManager {
	return &HookManager{
		plugins: []core.Plugin{},
	}
}

// AddPlugin adds a plugin to the hook manager
func (m *HookManager) AddPlugin(plugin core.Plugin) {
	m.plugins = append(m.plugins, plugin)
}

// BeforeSeed executes BeforeSeed hooks for all plugins
func (m *HookManager) BeforeSeed(ctx context.Context, collections []*core.Collection) error {
	for _, plugin := range m.plugins {
		hooks := plugin.Hooks()
		if hooks.BeforeSeed != nil {
			if err := hooks.BeforeSeed(ctx, collections); err != nil {
				return err
			}
		}
	}
	return nil
}

// AfterSeed executes AfterSeed hooks for all plugins
func (m *HookManager) AfterSeed(ctx context.Context, result *core.SeedResult) error {
	for _, plugin := range m.plugins {
		hooks := plugin.Hooks()
		if hooks.AfterSeed != nil {
			if err := hooks.AfterSeed(ctx, result); err != nil {
				return err
			}
		}
	}
	return nil
}

// BeforeInsert executes BeforeInsert hooks for all plugins
func (m *HookManager) BeforeInsert(ctx context.Context, collection string, rows []map[string]interface{}) error {
	for _, plugin := range m.plugins {
		hooks := plugin.Hooks()
		if hooks.BeforeInsert != nil {
			if err := hooks.BeforeInsert(ctx, collection, rows); err != nil {
				return err
			}
		}
	}
	return nil
}

// AfterInsert executes AfterInsert hooks for all plugins
func (m *HookManager) AfterInsert(ctx context.Context, collection string, count int64) error {
	for _, plugin := range m.plugins {
		hooks := plugin.Hooks()
		if hooks.AfterInsert != nil {
			if err := hooks.AfterInsert(ctx, collection, count); err != nil {
				return err
			}
		}
	}
	return nil
}

// OnError executes OnError hooks for all plugins
func (m *HookManager) OnError(ctx context.Context, err error) error {
	for _, plugin := range m.plugins {
		hooks := plugin.Hooks()
		if hooks.OnError != nil {
			if handleErr := hooks.OnError(ctx, err); handleErr != nil {
				return handleErr
			}
		}
	}
	return nil
}

// =============================================================================
// BASE PLUGIN IMPLEMENTATION
// =============================================================================

// BasePlugin provides a base implementation for plugins
type BasePlugin struct {
	name        string
	version     string
	description string
	pluginType  core.PluginType
	config      map[string]interface{}
	hooks       core.PluginHooks
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(name, version, description string, pluginType core.PluginType) *BasePlugin {
	return &BasePlugin{
		name:        name,
		version:     version,
		description: description,
		pluginType:  pluginType,
		config:      make(map[string]interface{}),
		hooks:       core.PluginHooks{},
	}
}

// Name returns the plugin name
func (p *BasePlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *BasePlugin) Version() string {
	return p.version
}

// Description returns a brief description
func (p *BasePlugin) Description() string {
	return p.description
}

// Init initializes the plugin with config
func (p *BasePlugin) Init(config map[string]interface{}) error {
	p.config = config
	return nil
}

// Type returns what the plugin provides
func (p *BasePlugin) Type() core.PluginType {
	return p.pluginType
}

// Hooks returns lifecycle hooks
func (p *BasePlugin) Hooks() core.PluginHooks {
	return p.hooks
}

// SetHooks sets the plugin hooks
func (p *BasePlugin) SetHooks(hooks core.PluginHooks) {
	p.hooks = hooks
}

// GetConfig gets a config value
func (p *BasePlugin) GetConfig(key string) (interface{}, bool) {
	val, ok := p.config[key]
	return val, ok
}

// GetConfigString gets a config value as string
func (p *BasePlugin) GetConfigString(key string, defaultVal string) string {
	val, ok := p.config[key]
	if !ok {
		return defaultVal
	}
	str, ok := val.(string)
	if !ok {
		return defaultVal
	}
	return str
}

// GetConfigInt gets a config value as int
func (p *BasePlugin) GetConfigInt(key string, defaultVal int) int {
	val, ok := p.config[key]
	if !ok {
		return defaultVal
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return defaultVal
	}
}

// GetConfigBool gets a config value as bool
func (p *BasePlugin) GetConfigBool(key string, defaultVal bool) bool {
	val, ok := p.config[key]
	if !ok {
		return defaultVal
	}
	b, ok := val.(bool)
	if !ok {
		return defaultVal
	}
	return b
}
