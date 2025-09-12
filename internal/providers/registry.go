package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bendiamant/leash-gateway/internal/circuitbreaker"
	"github.com/bendiamant/leash-gateway/internal/providers/anthropic"
	"github.com/bendiamant/leash-gateway/internal/providers/base"
	"github.com/bendiamant/leash-gateway/internal/providers/openai"
	"go.uber.org/zap"
)

// Registry implements the ProviderRegistry interface
type Registry struct {
	providers     map[string]base.Provider
	cbManager     *circuitbreaker.Manager
	logger        *zap.SugaredLogger
	mu            sync.RWMutex
	healthTicker  *time.Ticker
	stopHealth    chan struct{}
}

// NewRegistry creates a new provider registry
func NewRegistry(logger *zap.SugaredLogger) *Registry {
	return &Registry{
		providers:  make(map[string]base.Provider),
		cbManager:  circuitbreaker.NewManager(),
		logger:     logger,
		stopHealth: make(chan struct{}),
	}
}

// Register registers a provider
func (r *Registry) Register(provider base.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	r.providers[name] = provider
	r.logger.Infof("Provider %s registered successfully", name)

	return nil
}

// Unregister removes a provider
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	// Shutdown provider if it supports it
	if shutdowner, ok := provider.(interface{ Shutdown() error }); ok {
		if err := shutdowner.Shutdown(); err != nil {
			r.logger.Warnf("Error shutting down provider %s: %v", name, err)
		}
	}

	delete(r.providers, name)
	r.logger.Infof("Provider %s unregistered", name)

	return nil
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (base.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// List returns all registered providers
func (r *Registry) List() []base.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]base.Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}

	return providers
}

// GetHealthyProvider returns a healthy provider, preferring the specified one
func (r *Registry) GetHealthyProvider(preferredProvider string) (base.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try preferred provider first
	if preferredProvider != "" {
		if provider, exists := r.providers[preferredProvider]; exists && provider.IsHealthy() {
			return provider, nil
		}
	}

	// Fall back to any healthy provider
	for _, provider := range r.providers {
		if provider.IsHealthy() {
			return provider, nil
		}
	}

	return nil, fmt.Errorf("no healthy providers available")
}

// HealthCheck performs health checks on all providers
func (r *Registry) HealthCheck(ctx context.Context) map[string]*base.ProviderHealth {
	r.mu.RLock()
	providers := make(map[string]base.Provider)
	for name, provider := range r.providers {
		providers[name] = provider
	}
	r.mu.RUnlock()

	results := make(map[string]*base.ProviderHealth)
	
	for name, provider := range providers {
		health, err := provider.Health(ctx)
		if err != nil {
			results[name] = &base.ProviderHealth{
				Status:    base.HealthStatusUnhealthy,
				Message:   fmt.Sprintf("Health check failed: %v", err),
				LastCheck: time.Now(),
			}
		} else {
			results[name] = health
		}
	}

	return results
}

// GetProviderForModel determines which provider to use for a given model
func (r *Registry) GetProviderForModel(model string) (base.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Simple model-to-provider mapping
	if strings.HasPrefix(model, "gpt-") {
		if provider, exists := r.providers["openai"]; exists {
			return provider, nil
		}
	}
	
	if strings.HasPrefix(model, "claude-") {
		if provider, exists := r.providers["anthropic"]; exists {
			return provider, nil
		}
	}

	// Check all providers for model support
	for _, provider := range r.providers {
		for _, supportedModel := range provider.SupportedModels() {
			if supportedModel == model {
				return provider, nil
			}
		}
	}

	return nil, fmt.Errorf("no provider found for model %s", model)
}

// InitializeFromConfig initializes providers from configuration
func (r *Registry) InitializeFromConfig(configs map[string]*base.ProviderConfig) error {
	for name, config := range configs {
		config.Name = name
		
		var provider base.Provider
		var err error

		switch name {
		case "openai":
			provider = openai.NewOpenAIProvider(config, r.cbManager, r.logger)
		case "anthropic":
			provider = anthropic.NewAnthropicProvider(config, r.cbManager, r.logger)
		default:
			r.logger.Warnf("Unknown provider type: %s", name)
			continue
		}

		if err := r.Register(provider); err != nil {
			return fmt.Errorf("failed to register provider %s: %w", name, err)
		}
	}

	return nil
}

// StartHealthMonitoring starts periodic health monitoring
func (r *Registry) StartHealthMonitoring(interval time.Duration) {
	r.healthTicker = time.NewTicker(interval)
	
	go func() {
		for {
			select {
			case <-r.healthTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				r.HealthCheck(ctx)
				cancel()
			case <-r.stopHealth:
				r.healthTicker.Stop()
				return
			}
		}
	}()
}

// StopHealthMonitoring stops health monitoring
func (r *Registry) StopHealthMonitoring() {
	if r.stopHealth != nil {
		close(r.stopHealth)
	}
	if r.healthTicker != nil {
		r.healthTicker.Stop()
	}
}

// Shutdown shuts down all providers
func (r *Registry) Shutdown() error {
	r.StopHealthMonitoring()

	r.mu.Lock()
	defer r.mu.Unlock()

	for name, provider := range r.providers {
		if shutdowner, ok := provider.(interface{ Shutdown() error }); ok {
			if err := shutdowner.Shutdown(); err != nil {
				r.logger.Errorf("Error shutting down provider %s: %v", name, err)
			}
		}
	}

	return nil
}
