package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bendiamant/leash-gateway/internal/modules/interface"
	"go.uber.org/zap"
)

// ModuleRegistry implements the Registry interface
type ModuleRegistry struct {
	modules map[string]interfaces.Module
	mu      sync.RWMutex
	logger  *zap.SugaredLogger
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry(logger *zap.SugaredLogger) *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]interfaces.Module),
		logger:  logger,
	}
}

// Register registers a module in the registry
func (r *ModuleRegistry) Register(module interfaces.Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := module.Name()
	if name == "" {
		return fmt.Errorf("module name cannot be empty")
	}

	// Check if module already exists
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("module %s already registered", name)
	}

	// Validate module
	if err := r.ValidateModule(module); err != nil {
		return fmt.Errorf("module validation failed: %w", err)
	}

	// Register module
	r.modules[name] = module
	r.logger.Infof("Module %s (type: %s, version: %s) registered successfully", 
		name, module.Type().String(), module.Version())

	return nil
}

// Unregister removes a module from the registry
func (r *ModuleRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	module, exists := r.modules[name]
	if !exists {
		return fmt.Errorf("module %s not found", name)
	}

	// Stop module before unregistering
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := module.Stop(ctx); err != nil {
		r.logger.Warnf("Error stopping module %s: %v", name, err)
	}

	if err := module.Shutdown(ctx); err != nil {
		r.logger.Warnf("Error shutting down module %s: %v", name, err)
	}

	delete(r.modules, name)
	r.logger.Infof("Module %s unregistered successfully", name)

	return nil
}

// Get retrieves a module by name
func (r *ModuleRegistry) Get(name string) (interfaces.Module, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	module, exists := r.modules[name]
	if !exists {
		return nil, fmt.Errorf("module %s not found", name)
	}

	return module, nil
}

// List returns all registered modules
func (r *ModuleRegistry) List() []interfaces.Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modules := make([]interfaces.Module, 0, len(r.modules))
	for _, module := range r.modules {
		modules = append(modules, module)
	}

	return modules
}

// ListByType returns modules of a specific type
func (r *ModuleRegistry) ListByType(moduleType interfaces.ModuleType) []interfaces.Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var modules []interfaces.Module
	for _, module := range r.modules {
		if module.Type() == moduleType {
			modules = append(modules, module)
		}
	}

	return modules
}

// Reload reloads a specific module
func (r *ModuleRegistry) Reload(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	module, exists := r.modules[name]
	if !exists {
		return fmt.Errorf("module %s not found", name)
	}

	// Stop the module
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := module.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop module %s: %w", name, err)
	}

	// Start the module again
	if err := module.Start(ctx); err != nil {
		return fmt.Errorf("failed to restart module %s: %w", name, err)
	}

	r.logger.Infof("Module %s reloaded successfully", name)
	return nil
}

// ValidateModule validates a module before registration
func (r *ModuleRegistry) ValidateModule(module interfaces.Module) error {
	// Check required fields
	if module.Name() == "" {
		return fmt.Errorf("module name is required")
	}

	if module.Version() == "" {
		return fmt.Errorf("module version is required")
	}

	if module.Description() == "" {
		return fmt.Errorf("module description is required")
	}

	// Validate module type
	moduleType := module.Type()
	if moduleType < interfaces.ModuleTypeInspector || moduleType > interfaces.ModuleTypeSink {
		return fmt.Errorf("invalid module type: %d", moduleType)
	}

	// Check dependencies
	dependencies := module.Dependencies()
	for _, dep := range dependencies {
		if _, exists := r.modules[dep]; !exists {
			return fmt.Errorf("dependency %s not found", dep)
		}
	}

	return nil
}

// GetModulesByPriority returns modules sorted by priority (ascending)
func (r *ModuleRegistry) GetModulesByPriority(moduleType interfaces.ModuleType) []interfaces.Module {
	modules := r.ListByType(moduleType)
	
	// Sort by priority (lower number = higher priority)
	for i := 0; i < len(modules)-1; i++ {
		for j := i + 1; j < len(modules); j++ {
			iPriority := r.getModulePriority(modules[i])
			jPriority := r.getModulePriority(modules[j])
			if iPriority > jPriority {
				modules[i], modules[j] = modules[j], modules[i]
			}
		}
	}
	
	return modules
}

// getModulePriority extracts priority from module config
func (r *ModuleRegistry) getModulePriority(module interfaces.Module) int {
	config := module.GetConfig()
	if config != nil {
		return config.Priority
	}
	return 500 // Default priority
}

// HealthCheck performs health check on all modules
func (r *ModuleRegistry) HealthCheck(ctx context.Context) map[string]*interfaces.HealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[string]*interfaces.HealthStatus)
	
	for name, module := range r.modules {
		health, err := module.Health(ctx)
		if err != nil {
			results[name] = &interfaces.HealthStatus{
				Status:        interfaces.HealthStateUnhealthy,
				Message:       fmt.Sprintf("Health check failed: %v", err),
				LastCheck:     time.Now(),
				CheckDuration: 0,
			}
		} else {
			results[name] = health
		}
	}

	return results
}

// GetMetrics collects metrics from all modules
func (r *ModuleRegistry) GetMetrics() map[string]map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make(map[string]map[string]interface{})
	
	for name, module := range r.modules {
		moduleMetrics := module.Metrics()
		if moduleMetrics != nil {
			metrics[name] = moduleMetrics
		}
	}

	return metrics
}
