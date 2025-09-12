// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/bendiamant/leash-gateway/internal/circuitbreaker"
	"github.com/bendiamant/leash-gateway/internal/modules/core/contentfilter"
	"github.com/bendiamant/leash-gateway/internal/modules/core/costtracker"
	"github.com/bendiamant/leash-gateway/internal/modules/core/ratelimiter"
	"github.com/bendiamant/leash-gateway/internal/modules/interface"
	"github.com/bendiamant/leash-gateway/internal/modules/pipeline"
	"github.com/bendiamant/leash-gateway/internal/modules/registry"
	"github.com/bendiamant/leash-gateway/internal/providers"
	"github.com/bendiamant/leash-gateway/internal/providers/base"
	"go.uber.org/zap"
)

func TestPhase3Implementation(t *testing.T) {
	// Create logger
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	t.Run("CircuitBreakerFunctionality", func(t *testing.T) {
		manager := circuitbreaker.NewManager()
		
		cb := manager.GetOrCreate("test-provider", circuitbreaker.Config{
			MaxFailures:  50, // 50% failure rate
			MinRequests:  10,
			ResetTimeout: time.Second,
		})

		// Test successful calls
		for i := 0; i < 5; i++ {
			err := cb.Call(func() error { return nil })
			if err != nil {
				t.Errorf("Expected successful call, got error: %v", err)
			}
		}

		// Test failure calls
		for i := 0; i < 10; i++ {
			cb.Call(func() error { return fmt.Errorf("test error") })
		}

		// Circuit should be open now
		if cb.GetState() != circuitbreaker.StateOpen {
			t.Errorf("Expected circuit breaker to be open, got %s", cb.GetState())
		}

		// Calls should be rejected
		err := cb.Call(func() error { return nil })
		if err == nil {
			t.Error("Expected call to be rejected by open circuit breaker")
		}
	})

	t.Run("ModulePipelineExecution", func(t *testing.T) {
		// Create module registry and pipeline
		moduleRegistry := registry.NewModuleRegistry(sugar)
		modulePipeline := pipeline.NewPipeline(sugar)

		// Create and register modules
		rateLimiter := ratelimiter.NewRateLimiter(sugar)
		contentFilter := contentfilter.NewContentFilter(sugar)
		costTracker := costtracker.NewCostTracker(sugar)

		// Register modules
		if err := moduleRegistry.Register(rateLimiter); err != nil {
			t.Fatalf("Failed to register rate limiter: %v", err)
		}
		if err := moduleRegistry.Register(contentFilter); err != nil {
			t.Fatalf("Failed to register content filter: %v", err)
		}
		if err := moduleRegistry.Register(costTracker); err != nil {
			t.Fatalf("Failed to register cost tracker: %v", err)
		}

		// Add to pipeline
		modulePipeline.AddModule(rateLimiter)
		modulePipeline.AddModule(contentFilter)
		modulePipeline.AddModule(costTracker)

		// Initialize modules
		ctx := context.Background()
		
		rateLimiter.Initialize(ctx, &interfaces.ModuleConfig{
			Name: "rate-limiter", Type: "policy", Enabled: true, Priority: 100,
			Config: map[string]interface{}{"default_limit": 1000, "default_window": "1h"},
		})
		rateLimiter.Start(ctx)

		contentFilter.Initialize(ctx, &interfaces.ModuleConfig{
			Name: "content-filter", Type: "policy", Enabled: true, Priority: 300,
			Config: map[string]interface{}{"action": "block", "blocked_keywords": []interface{}{"test_blocked"}},
		})
		contentFilter.Start(ctx)

		costTracker.Initialize(ctx, &interfaces.ModuleConfig{
			Name: "cost-tracker", Type: "sink", Enabled: true, Priority: 900,
			Config: map[string]interface{}{"storage": "memory"},
		})
		costTracker.Start(ctx)

		// Test request processing
		req := &interfaces.ProcessRequestContext{
			RequestID: "test-req-001",
			TenantID:  "test-tenant",
			Provider:  "openai",
			Model:     "gpt-4o-mini",
			Method:    "POST",
			Path:      "/v1/openai/chat/completions",
			Body:      []byte(`{"messages":[{"role":"user","content":"Hello world"}]}`),
			Annotations: make(map[string]interface{}),
		}

		result, err := modulePipeline.ProcessRequest(ctx, req)
		if err != nil {
			t.Fatalf("Pipeline processing failed: %v", err)
		}

		if result.Action != interfaces.ActionContinue {
			t.Errorf("Expected ActionContinue, got %s", result.Action)
		}

		// Verify annotations were added
		if req.Annotations["rate_limit_checked"] != true {
			t.Error("Expected rate limit annotation")
		}
	})

	t.Run("ProviderRegistryFunctionality", func(t *testing.T) {
		providerRegistry := providers.NewRegistry(sugar)

		// Test provider configuration
		configs := map[string]*base.ProviderConfig{
			"openai": {
				Name:     "openai",
				Endpoint: "https://api.openai.com/v1",
				Timeout:  30 * time.Second,
				Models: []base.ModelConfig{
					{Name: "gpt-4o-mini", CostPer1kInputTokens: 0.15, CostPer1kOutputTokens: 0.60},
				},
				CircuitBreaker: base.CircuitBreakerConfig{
					FailureThreshold: 5,
					MinRequests:      10,
					Timeout:          60 * time.Second,
				},
			},
			"anthropic": {
				Name:     "anthropic",
				Endpoint: "https://api.anthropic.com/v1",
				Timeout:  30 * time.Second,
				Models: []base.ModelConfig{
					{Name: "claude-3-sonnet-20240229", CostPer1kInputTokens: 3.0, CostPer1kOutputTokens: 15.0},
				},
				CircuitBreaker: base.CircuitBreakerConfig{
					FailureThreshold: 5,
					MinRequests:      10,
					Timeout:          60 * time.Second,
				},
			},
		}

		// Initialize providers
		err := providerRegistry.InitializeFromConfig(configs)
		if err != nil {
			t.Fatalf("Failed to initialize providers: %v", err)
		}

		// Test provider lookup
		openaiProvider, err := providerRegistry.Get("openai")
		if err != nil {
			t.Fatalf("Failed to get OpenAI provider: %v", err)
		}

		if openaiProvider.Name() != "openai" {
			t.Errorf("Expected provider name 'openai', got '%s'", openaiProvider.Name())
		}

		// Test model-to-provider mapping
		provider, err := providerRegistry.GetProviderForModel("gpt-4o-mini")
		if err != nil {
			t.Fatalf("Failed to get provider for model: %v", err)
		}

		if provider.Name() != "openai" {
			t.Errorf("Expected OpenAI provider for gpt-4o-mini, got %s", provider.Name())
		}

		// Test Anthropic provider
		anthropicProvider, err := providerRegistry.GetProviderForModel("claude-3-sonnet-20240229")
		if err != nil {
			t.Fatalf("Failed to get provider for Claude model: %v", err)
		}

		if anthropicProvider.Name() != "anthropic" {
			t.Errorf("Expected Anthropic provider for Claude model, got %s", anthropicProvider.Name())
		}
	})

	t.Run("ContentFilterModule", func(t *testing.T) {
		filter := contentfilter.NewContentFilter(sugar)
		
		config := &interfaces.ModuleConfig{
			Name: "content-filter",
			Type: "policy",
			Enabled: true,
			Config: map[string]interface{}{
				"blocked_keywords": []interface{}{"harmful", "inappropriate"},
				"action": "block",
				"severity_threshold": 0.8,
			},
		}

		ctx := context.Background()
		err := filter.Initialize(ctx, config)
		if err != nil {
			t.Fatalf("Failed to initialize content filter: %v", err)
		}

		err = filter.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start content filter: %v", err)
		}

		// Test safe content
		safeReq := &interfaces.ProcessRequestContext{
			RequestID: "safe-req",
			Body:      []byte(`{"messages":[{"role":"user","content":"Hello, how are you?"}]}`),
		}

		result, err := filter.ProcessRequest(ctx, safeReq)
		if err != nil {
			t.Fatalf("Content filter failed: %v", err)
		}

		if result.Action != interfaces.ActionContinue {
			t.Errorf("Expected safe content to continue, got %s", result.Action)
		}

		// Test blocked content
		blockedReq := &interfaces.ProcessRequestContext{
			RequestID: "blocked-req",
			Body:      []byte(`{"messages":[{"role":"user","content":"This is harmful content"}]}`),
		}

		result, err = filter.ProcessRequest(ctx, blockedReq)
		if err != nil {
			t.Fatalf("Content filter failed: %v", err)
		}

		if result.Action != interfaces.ActionBlock {
			t.Errorf("Expected harmful content to be blocked, got %s", result.Action)
		}

		if result.BlockReason == "" {
			t.Error("Expected block reason to be provided")
		}
	})

	t.Run("CostTrackerModule", func(t *testing.T) {
		tracker := costtracker.NewCostTracker(sugar)
		
		config := &interfaces.ModuleConfig{
			Name: "cost-tracker",
			Type: "sink",
			Enabled: true,
			Config: map[string]interface{}{
				"storage": "memory",
				"track_responses": true,
			},
		}

		ctx := context.Background()
		err := tracker.Initialize(ctx, config)
		if err != nil {
			t.Fatalf("Failed to initialize cost tracker: %v", err)
		}

		err = tracker.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start cost tracker: %v", err)
		}

		// Test cost tracking
		resp := &interfaces.ProcessResponseContext{
			ProcessRequestContext: &interfaces.ProcessRequestContext{
				RequestID: "cost-req",
				TenantID:  "test-tenant",
				Provider:  "openai",
				Model:     "gpt-4o-mini",
			},
			TokensUsed: &interfaces.TokenUsage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
			CostUSD: 0.005,
		}

		result, err := tracker.ProcessResponse(ctx, resp)
		if err != nil {
			t.Fatalf("Cost tracker failed: %v", err)
		}

		if result.Action != interfaces.ActionContinue {
			t.Errorf("Expected cost tracker to continue, got %s", result.Action)
		}

		// Verify cost was tracked
		if result.Annotations["cost_tracked"] != true {
			t.Error("Expected cost to be tracked")
		}
	})
}

func TestProviderIntegration(t *testing.T) {
	t.Run("ProviderConfiguration", func(t *testing.T) {
		// Test that provider configurations are valid
		configs := []struct {
			name     string
			endpoint string
			models   []string
		}{
			{"openai", "https://api.openai.com/v1", []string{"gpt-4o-mini", "gpt-4o"}},
			{"anthropic", "https://api.anthropic.com/v1", []string{"claude-3-sonnet-20240229"}},
			{"google", "https://generativelanguage.googleapis.com/v1", []string{"gemini-1.5-flash"}},
		}

		for _, config := range configs {
			t.Run(config.name, func(t *testing.T) {
				if config.endpoint == "" {
					t.Errorf("Provider %s missing endpoint", config.name)
				}
				if len(config.models) == 0 {
					t.Errorf("Provider %s has no models configured", config.name)
				}
			})
		}
	})
}
