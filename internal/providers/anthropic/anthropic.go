package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bendiamant/leash-gateway/internal/circuitbreaker"
	"github.com/bendiamant/leash-gateway/internal/providers/base"
	"go.uber.org/zap"
)

// AnthropicProvider implements the Provider interface for Anthropic
type AnthropicProvider struct {
	name           string
	config         *base.ProviderConfig
	client         *http.Client
	circuitBreaker *circuitbreaker.CircuitBreaker
	logger         *zap.SugaredLogger
	lastHealth     *base.ProviderHealth
	healthTicker   *time.Ticker
	stopHealth     chan struct{}
}

// AnthropicRequest represents an Anthropic API request
type AnthropicRequest struct {
	Model       string                 `json:"model"`
	Messages    []base.Message         `json:"messages"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature *float64               `json:"temperature,omitempty"`
	TopP        *float64               `json:"top_p,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	StopSequences []string             `json:"stop_sequences,omitempty"`
}

// AnthropicResponse represents an Anthropic API response
type AnthropicResponse struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Role         string    `json:"role"`
	Content      []Content `json:"content"`
	Model        string    `json:"model"`
	StopReason   string    `json:"stop_reason"`
	StopSequence string    `json:"stop_sequence,omitempty"`
	Usage        Usage     `json:"usage"`
}

// Content represents content in Anthropic response
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Usage represents token usage in Anthropic response
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(config *base.ProviderConfig, cbManager *circuitbreaker.Manager, logger *zap.SugaredLogger) *AnthropicProvider {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	// Create circuit breaker
	cb := cbManager.GetOrCreate(config.Name, circuitbreaker.Config{
		MaxFailures:  config.CircuitBreaker.FailureThreshold,
		MinRequests:  config.CircuitBreaker.MinRequests,
		ResetTimeout: config.CircuitBreaker.Timeout,
		OnStateChange: func(name string, from, to circuitbreaker.State) {
			logger.Infof("Circuit breaker %s state changed from %s to %s", name, from, to)
		},
	})

	provider := &AnthropicProvider{
		name:           config.Name,
		config:         config,
		client:         client,
		circuitBreaker: cb,
		logger:         logger,
		stopHealth:     make(chan struct{}),
	}

	// Start health monitoring if enabled
	if config.HealthCheck.Enabled {
		provider.startHealthMonitoring()
	}

	return provider
}

// Metadata methods
func (p *AnthropicProvider) Name() string { return p.name }
func (p *AnthropicProvider) Endpoint() string { return p.config.Endpoint }

func (p *AnthropicProvider) SupportedModels() []string {
	models := make([]string, len(p.config.Models))
	for i, model := range p.config.Models {
		models[i] = model.Name
	}
	return models
}

// Health methods
func (p *AnthropicProvider) Health(ctx context.Context) (*base.ProviderHealth, error) {
	start := time.Now()

	// Use circuit breaker for health check
	var err error
	healthErr := p.circuitBreaker.Call(func() error {
		// Anthropic doesn't have a simple health endpoint, so we'll use a minimal request
		testReq := &AnthropicRequest{
			Model:     "claude-3-haiku-20240307",
			Messages:  []base.Message{{Role: "user", Content: "Hi"}},
			MaxTokens: 1,
		}

		reqBody, marshalErr := json.Marshal(testReq)
		if marshalErr != nil {
			return marshalErr
		}

		req, reqErr := http.NewRequestWithContext(ctx, "POST", p.config.Endpoint+"/messages", bytes.NewReader(reqBody))
		if reqErr != nil {
			return reqErr
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, respErr := p.client.Do(req)
		if respErr != nil {
			return respErr
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return fmt.Errorf("health check failed with status %d", resp.StatusCode)
		}

		return nil
	})

	responseTime := time.Since(start)
	status := base.HealthStatusHealthy
	message := "Provider is healthy"

	if healthErr != nil {
		status = base.HealthStatusUnhealthy
		message = fmt.Sprintf("Health check failed: %v", healthErr)
		err = healthErr
	}

	health := &base.ProviderHealth{
		Status:       status,
		Message:      message,
		LastCheck:    time.Now(),
		ResponseTime: responseTime,
		Details: map[string]interface{}{
			"endpoint":         p.config.Endpoint,
			"circuit_breaker":  p.circuitBreaker.GetState().String(),
			"supported_models": len(p.config.Models),
		},
	}

	p.lastHealth = health
	return health, err
}

func (p *AnthropicProvider) IsHealthy() bool {
	if p.lastHealth == nil {
		return false
	}
	return p.lastHealth.Status == base.HealthStatusHealthy
}

// Request processing
func (p *AnthropicProvider) ProcessRequest(ctx context.Context, req *base.ProviderRequest) (*base.ProviderResponse, error) {
	start := time.Now()

	// Convert to Anthropic format
	anthropicReq := &AnthropicRequest{
		Model:     req.Model,
		Messages:  req.Messages,
		MaxTokens: 1024, // Default max tokens
		Stream:    false,
	}

	// Add parameters
	if temp, ok := req.Parameters["temperature"].(float64); ok {
		anthropicReq.Temperature = &temp
	}
	if maxTokens, ok := req.Parameters["max_tokens"].(int); ok {
		anthropicReq.MaxTokens = maxTokens
	}
	if topP, ok := req.Parameters["top_p"].(float64); ok {
		anthropicReq.TopP = &topP
	}

	// Marshal request
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var response *base.ProviderResponse
	
	// Use circuit breaker
	callErr := p.circuitBreaker.Call(func() error {
		resp, err := p.makeRequest(ctx, "POST", "/messages", reqBody, req.Headers)
		if err != nil {
			return err
		}
		response = resp
		return nil
	})

	if callErr != nil {
		return nil, callErr
	}

	// Calculate cost
	if response.Usage != nil {
		cost := p.calculateCost(req.Model, response.Usage)
		response.Cost = cost
	}

	response.Latency = time.Since(start)
	return response, nil
}

func (p *AnthropicProvider) ProcessStreamingRequest(ctx context.Context, req *base.ProviderRequest) (*base.StreamingResponse, error) {
	// Similar to ProcessRequest but with streaming enabled
	// Implementation would be similar to OpenAI but with Anthropic's streaming format
	return nil, fmt.Errorf("streaming not yet implemented for Anthropic")
}

// Configuration methods
func (p *AnthropicProvider) UpdateConfig(config *base.ProviderConfig) error {
	p.config = config
	p.client.Timeout = config.Timeout
	return nil
}

func (p *AnthropicProvider) GetConfig() *base.ProviderConfig {
	return p.config
}

// Helper methods
func (p *AnthropicProvider) makeRequest(ctx context.Context, method, path string, body []byte, headers map[string]string) (*base.ProviderResponse, error) {
	url := p.config.Endpoint + path
	
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Set Anthropic-specific headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	for key, value := range p.config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse Anthropic response for usage information
	var usage *base.TokenUsage
	if resp.StatusCode == 200 {
		var anthropicResp AnthropicResponse
		if json.Unmarshal(respBody, &anthropicResp) == nil {
			usage = &base.TokenUsage{
				PromptTokens:     int64(anthropicResp.Usage.InputTokens),
				CompletionTokens: int64(anthropicResp.Usage.OutputTokens),
				TotalTokens:      int64(anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens),
			}
		}
	}

	return &base.ProviderResponse{
		StatusCode: resp.StatusCode,
		Headers:    p.convertHeaders(resp.Header),
		Body:       respBody,
		Usage:      usage,
		Metadata: map[string]string{
			"provider": p.name,
		},
	}, nil
}

func (p *AnthropicProvider) convertHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

func (p *AnthropicProvider) calculateCost(model string, usage *base.TokenUsage) float64 {
	if usage == nil {
		return 0
	}

	// Find model config
	for _, modelConfig := range p.config.Models {
		if modelConfig.Name == model {
			inputCost := float64(usage.PromptTokens) / 1000.0 * modelConfig.CostPer1kInputTokens
			outputCost := float64(usage.CompletionTokens) / 1000.0 * modelConfig.CostPer1kOutputTokens
			return inputCost + outputCost
		}
	}

	return 0 // Unknown model
}

func (p *AnthropicProvider) startHealthMonitoring() {
	p.healthTicker = time.NewTicker(p.config.HealthCheck.Interval)
	
	go func() {
		for {
			select {
			case <-p.healthTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), p.config.HealthCheck.Timeout)
				_, err := p.Health(ctx)
				if err != nil {
					p.logger.Warnf("Health check failed for provider %s: %v", p.name, err)
				}
				cancel()
			case <-p.stopHealth:
				p.healthTicker.Stop()
				return
			}
		}
	}()
}

// Shutdown stops the provider
func (p *AnthropicProvider) Shutdown() error {
	if p.stopHealth != nil {
		close(p.stopHealth)
	}
	if p.healthTicker != nil {
		p.healthTicker.Stop()
	}
	return nil
}
