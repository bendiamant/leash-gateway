package base

import (
	"context"
	"time"
)

// Provider represents the base interface for all LLM providers
type Provider interface {
	// Metadata
	Name() string
	Endpoint() string
	SupportedModels() []string
	
	// Health and status
	Health(ctx context.Context) (*ProviderHealth, error)
	IsHealthy() bool
	
	// Request processing
	ProcessRequest(ctx context.Context, req *ProviderRequest) (*ProviderResponse, error)
	ProcessStreamingRequest(ctx context.Context, req *ProviderRequest) (*StreamingResponse, error)
	
	// Configuration
	UpdateConfig(config *ProviderConfig) error
	GetConfig() *ProviderConfig
}

// ProviderHealth represents provider health status
type ProviderHealth struct {
	Status       HealthStatus `json:"status"`
	Message      string       `json:"message"`
	LastCheck    time.Time    `json:"last_check"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorRate    float64      `json:"error_rate"`
	Details      map[string]interface{} `json:"details"`
}

// HealthStatus represents provider health status
type HealthStatus int

const (
	HealthStatusHealthy   HealthStatus = iota
	HealthStatusDegraded
	HealthStatusUnhealthy
	HealthStatusUnknown
)

func (h HealthStatus) String() string {
	switch h {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusDegraded:
		return "degraded"
	case HealthStatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// ProviderConfig represents provider configuration
type ProviderConfig struct {
	Name                   string                 `yaml:"name" json:"name"`
	Endpoint               string                 `yaml:"endpoint" json:"endpoint"`
	Timeout                time.Duration          `yaml:"timeout" json:"timeout"`
	RetryAttempts          int                    `yaml:"retry_attempts" json:"retry_attempts"`
	RetryDelay             time.Duration          `yaml:"retry_delay" json:"retry_delay"`
	RetryBackoffMultiplier float64                `yaml:"retry_backoff_multiplier" json:"retry_backoff_multiplier"`
	MaxRetryDelay          time.Duration          `yaml:"max_retry_delay" json:"max_retry_delay"`
	CircuitBreaker         CircuitBreakerConfig   `yaml:"circuit_breaker" json:"circuit_breaker"`
	HealthCheck            HealthCheckConfig      `yaml:"health_check" json:"health_check"`
	Models                 []ModelConfig          `yaml:"models" json:"models"`
	Headers                map[string]string      `yaml:"headers,omitempty" json:"headers,omitempty"`
	RateLimits             *RateLimitConfig       `yaml:"rate_limits,omitempty" json:"rate_limits,omitempty"`
}

// CircuitBreakerConfig represents circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int           `yaml:"failure_threshold" json:"failure_threshold"`
	SuccessThreshold int           `yaml:"success_threshold" json:"success_threshold"`
	Timeout          time.Duration `yaml:"timeout" json:"timeout"`
	MinRequests      int           `yaml:"min_requests" json:"min_requests"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Interval time.Duration `yaml:"interval" json:"interval"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
	Path     string        `yaml:"path" json:"path"`
	Method   string        `yaml:"method" json:"method"`
}

// ModelConfig represents model configuration and pricing
type ModelConfig struct {
	Name                  string  `yaml:"name" json:"name"`
	CostPer1kInputTokens  float64 `yaml:"cost_per_1k_input_tokens" json:"cost_per_1k_input_tokens"`
	CostPer1kOutputTokens float64 `yaml:"cost_per_1k_output_tokens" json:"cost_per_1k_output_tokens"`
	MaxTokens             int     `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty"`
	SupportsStreaming     bool    `yaml:"supports_streaming" json:"supports_streaming"`
}

// RateLimitConfig represents provider-specific rate limiting
type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute" json:"requests_per_minute"`
	RequestsPerHour   int `yaml:"requests_per_hour" json:"requests_per_hour"`
	RequestsPerDay    int `yaml:"requests_per_day" json:"requests_per_day"`
}

// ProviderRequest represents a request to a provider
type ProviderRequest struct {
	RequestID   string            `json:"request_id"`
	TenantID    string            `json:"tenant_id"`
	Model       string            `json:"model"`
	Messages    []Message         `json:"messages"`
	Parameters  map[string]interface{} `json:"parameters"`
	Headers     map[string]string `json:"headers"`
	Streaming   bool              `json:"streaming"`
	Metadata    map[string]string `json:"metadata"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ProviderResponse represents a response from a provider
type ProviderResponse struct {
	RequestID    string            `json:"request_id"`
	StatusCode   int               `json:"status_code"`
	Headers      map[string]string `json:"headers"`
	Body         []byte            `json:"body"`
	Model        string            `json:"model"`
	Usage        *TokenUsage       `json:"usage,omitempty"`
	Cost         float64           `json:"cost,omitempty"`
	Latency      time.Duration     `json:"latency"`
	Metadata     map[string]string `json:"metadata"`
}

// StreamingResponse represents a streaming response
type StreamingResponse struct {
	RequestID string            `json:"request_id"`
	Headers   map[string]string `json:"headers"`
	Stream    <-chan StreamChunk `json:"-"`
	Metadata  map[string]string `json:"metadata"`
}

// StreamChunk represents a chunk in a streaming response
type StreamChunk struct {
	Data      []byte            `json:"data"`
	Done      bool              `json:"done"`
	Error     error             `json:"error,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

// ProviderRegistry manages multiple providers
type ProviderRegistry interface {
	Register(provider Provider) error
	Unregister(name string) error
	Get(name string) (Provider, error)
	List() []Provider
	GetHealthyProvider(preferredProvider string) (Provider, error)
	HealthCheck(ctx context.Context) map[string]*ProviderHealth
}

// ProviderManager manages provider lifecycle and health
type ProviderManager interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	RouteRequest(ctx context.Context, req *ProviderRequest) (*ProviderResponse, error)
	RouteStreamingRequest(ctx context.Context, req *ProviderRequest) (*StreamingResponse, error)
	GetProviderForModel(model string) (Provider, error)
}
