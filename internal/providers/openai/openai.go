package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bendiamant/leash-gateway/internal/circuitbreaker"
	"github.com/bendiamant/leash-gateway/internal/providers/base"
	"go.uber.org/zap"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	name           string
	config         *base.ProviderConfig
	client         *http.Client
	circuitBreaker *circuitbreaker.CircuitBreaker
	logger         *zap.SugaredLogger
	lastHealth     *base.ProviderHealth
	healthTicker   *time.Ticker
	stopHealth     chan struct{}
}

// OpenAIRequest represents an OpenAI API request
type OpenAIRequest struct {
	Model       string                 `json:"model"`
	Messages    []base.Message         `json:"messages"`
	Temperature *float64               `json:"temperature,omitempty"`
	MaxTokens   *int                   `json:"max_tokens,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Stop        []string               `json:"stop,omitempty"`
	TopP        *float64               `json:"top_p,omitempty"`
	User        string                 `json:"user,omitempty"`
}

// OpenAIResponse represents an OpenAI API response
type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a choice in the OpenAI response
type Choice struct {
	Index        int     `json:"index"`
	Message      base.Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage in OpenAI response
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config *base.ProviderConfig, cbManager *circuitbreaker.Manager, logger *zap.SugaredLogger) *OpenAIProvider {
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

	provider := &OpenAIProvider{
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
func (p *OpenAIProvider) Name() string { return p.name }
func (p *OpenAIProvider) Endpoint() string { return p.config.Endpoint }

func (p *OpenAIProvider) SupportedModels() []string {
	models := make([]string, len(p.config.Models))
	for i, model := range p.config.Models {
		models[i] = model.Name
	}
	return models
}

// Health methods
func (p *OpenAIProvider) Health(ctx context.Context) (*base.ProviderHealth, error) {
	start := time.Now()

	// Use circuit breaker for health check
	var err error
	healthErr := p.circuitBreaker.Call(func() error {
		req, reqErr := http.NewRequestWithContext(ctx, "GET", p.config.Endpoint+"/models", nil)
		if reqErr != nil {
			return reqErr
		}

		resp, respErr := p.client.Do(req)
		if respErr != nil {
			return respErr
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
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

func (p *OpenAIProvider) IsHealthy() bool {
	if p.lastHealth == nil {
		return false
	}
	return p.lastHealth.Status == base.HealthStatusHealthy
}

// Request processing
func (p *OpenAIProvider) ProcessRequest(ctx context.Context, req *base.ProviderRequest) (*base.ProviderResponse, error) {
	start := time.Now()

	// Convert to OpenAI format
	openaiReq := &OpenAIRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
	}

	// Add parameters
	if temp, ok := req.Parameters["temperature"].(float64); ok {
		openaiReq.Temperature = &temp
	}
	if maxTokens, ok := req.Parameters["max_tokens"].(int); ok {
		openaiReq.MaxTokens = &maxTokens
	}
	if topP, ok := req.Parameters["top_p"].(float64); ok {
		openaiReq.TopP = &topP
	}
	if user, ok := req.Parameters["user"].(string); ok {
		openaiReq.User = user
	}

	// Marshal request
	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var response *base.ProviderResponse
	
	// Use circuit breaker
	callErr := p.circuitBreaker.Call(func() error {
		resp, err := p.makeRequest(ctx, "POST", "/chat/completions", reqBody, req.Headers)
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

func (p *OpenAIProvider) ProcessStreamingRequest(ctx context.Context, req *base.ProviderRequest) (*base.StreamingResponse, error) {
	// Convert to OpenAI format with streaming enabled
	openaiReq := &OpenAIRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true,
	}

	// Add parameters
	if temp, ok := req.Parameters["temperature"].(float64); ok {
		openaiReq.Temperature = &temp
	}
	if maxTokens, ok := req.Parameters["max_tokens"].(int); ok {
		openaiReq.MaxTokens = &maxTokens
	}

	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal streaming request: %w", err)
	}

	// Create streaming request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.Endpoint+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Make streaming request with circuit breaker
	var httpResp *http.Response
	callErr := p.circuitBreaker.Call(func() error {
		resp, err := p.client.Do(httpReq)
		if err != nil {
			return err
		}
		if resp.StatusCode >= 400 {
			resp.Body.Close()
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		httpResp = resp
		return nil
	})

	if callErr != nil {
		return nil, callErr
	}

	// Create streaming response
	streamChan := make(chan base.StreamChunk, 10)
	go p.processStreamingResponse(httpResp, streamChan)

	return &base.StreamingResponse{
		RequestID: req.RequestID,
		Headers:   p.convertHeaders(httpResp.Header),
		Stream:    streamChan,
		Metadata: map[string]string{
			"provider": p.name,
			"model":    req.Model,
		},
	}, nil
}

// Configuration methods
func (p *OpenAIProvider) UpdateConfig(config *base.ProviderConfig) error {
	p.config = config
	p.client.Timeout = config.Timeout
	
	// Update circuit breaker if needed
	// This would typically involve recreating the circuit breaker
	
	return nil
}

func (p *OpenAIProvider) GetConfig() *base.ProviderConfig {
	return p.config
}

// Helper methods
func (p *OpenAIProvider) makeRequest(ctx context.Context, method, path string, body []byte, headers map[string]string) (*base.ProviderResponse, error) {
	url := p.config.Endpoint + path
	
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
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

	// Parse OpenAI response for usage information
	var usage *base.TokenUsage
	if resp.StatusCode == 200 {
		var openaiResp OpenAIResponse
		if json.Unmarshal(respBody, &openaiResp) == nil {
			usage = &base.TokenUsage{
				PromptTokens:     int64(openaiResp.Usage.PromptTokens),
				CompletionTokens: int64(openaiResp.Usage.CompletionTokens),
				TotalTokens:      int64(openaiResp.Usage.TotalTokens),
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

func (p *OpenAIProvider) processStreamingResponse(resp *http.Response, streamChan chan base.StreamChunk) {
	defer resp.Body.Close()
	defer close(streamChan)

	scanner := io.Reader(resp.Body)
	buffer := make([]byte, 4096)

	for {
		n, err := scanner.Read(buffer)
		if err != nil {
			if err != io.EOF {
				streamChan <- base.StreamChunk{
					Error: err,
					Done:  true,
				}
			} else {
				streamChan <- base.StreamChunk{
					Done: true,
				}
			}
			break
		}

		streamChan <- base.StreamChunk{
			Data: buffer[:n],
			Done: false,
		}
	}
}

func (p *OpenAIProvider) convertHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

func (p *OpenAIProvider) calculateCost(model string, usage *base.TokenUsage) float64 {
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

func (p *OpenAIProvider) startHealthMonitoring() {
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
func (p *OpenAIProvider) Shutdown() error {
	if p.stopHealth != nil {
		close(p.stopHealth)
	}
	if p.healthTicker != nil {
		p.healthTicker.Stop()
	}
	return nil
}
