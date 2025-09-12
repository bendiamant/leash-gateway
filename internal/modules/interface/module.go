package interfaces

import (
	"context"
	"time"
)

// Module represents the core interface that all modules must implement
type Module interface {
	// Metadata
	Name() string
	Version() string
	Type() ModuleType
	Description() string
	Author() string
	Dependencies() []string
	
	// Lifecycle
	Initialize(ctx context.Context, config *ModuleConfig) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Shutdown(ctx context.Context) error
	
	// Health and status
	Health(ctx context.Context) (*HealthStatus, error)
	Status() *ModuleStatus
	Metrics() map[string]interface{}
	
	// Request/Response processing
	ProcessRequest(ctx context.Context, req *ProcessRequestContext) (*ProcessRequestResult, error)
	ProcessResponse(ctx context.Context, resp *ProcessResponseContext) (*ProcessResponseResult, error)
	
	// Configuration
	ValidateConfig(config *ModuleConfig) error
	UpdateConfig(ctx context.Context, config *ModuleConfig) error
	GetConfig() *ModuleConfig
}

// ModuleType represents the type of module
type ModuleType int

const (
	ModuleTypeInspector   ModuleType = iota // Analyze content, detect patterns
	ModuleTypePolicy                        // Enforce rules, make allow/deny decisions
	ModuleTypeTransformer                   // Modify content, redact, inject
	ModuleTypeSink                         // Export data, log, send to external systems
)

func (t ModuleType) String() string {
	switch t {
	case ModuleTypeInspector:
		return "inspector"
	case ModuleTypePolicy:
		return "policy"
	case ModuleTypeTransformer:
		return "transformer"
	case ModuleTypeSink:
		return "sink"
	default:
		return "unknown"
	}
}

// ModuleConfig represents module configuration
type ModuleConfig struct {
	Name        string                 `yaml:"name" json:"name"`
	Type        string                 `yaml:"type" json:"type"`
	Enabled     bool                   `yaml:"enabled" json:"enabled"`
	Priority    int                    `yaml:"priority" json:"priority"`
	Config      map[string]interface{} `yaml:"config" json:"config"`
	Conditions  []Condition            `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	Resources   *ResourceLimits        `yaml:"resources,omitempty" json:"resources,omitempty"`
	Timeouts    *Timeouts              `yaml:"timeouts,omitempty" json:"timeouts,omitempty"`
}

// Condition represents execution conditions
type Condition struct {
	Field    string      `yaml:"field" json:"field"`       // tenant, provider, model, etc.
	Operator string      `yaml:"operator" json:"operator"` // eq, ne, in, not_in, regex
	Value    interface{} `yaml:"value" json:"value"`
}

// ResourceLimits represents resource limits for module execution
type ResourceLimits struct {
	MaxMemoryMB      int           `yaml:"max_memory_mb,omitempty" json:"max_memory_mb,omitempty"`
	MaxCPUPercent    int           `yaml:"max_cpu_percent,omitempty" json:"max_cpu_percent,omitempty"`
	MaxExecutionTime time.Duration `yaml:"max_execution_time,omitempty" json:"max_execution_time,omitempty"`
}

// Timeouts represents timeout configurations
type Timeouts struct {
	Initialization time.Duration `yaml:"initialization,omitempty" json:"initialization,omitempty"`
	Processing     time.Duration `yaml:"processing,omitempty" json:"processing,omitempty"`
	Shutdown       time.Duration `yaml:"shutdown,omitempty" json:"shutdown,omitempty"`
}

// ProcessRequestContext represents the context for request processing
type ProcessRequestContext struct {
	// Request identification
	RequestID   string    `json:"request_id"`
	Timestamp   time.Time `json:"timestamp"`
	
	// Tenant and routing information
	TenantID    string `json:"tenant_id"`
	Provider    string `json:"provider"`
	Model       string `json:"model,omitempty"`
	
	// HTTP request details
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body,omitempty"`
	
	// Additional context
	UserAgent   string            `json:"user_agent,omitempty"`
	ClientIP    string            `json:"client_ip,omitempty"`
	
	// Previous module results
	Annotations map[string]interface{} `json:"annotations,omitempty"`
	
	// Configuration
	ModuleConfig *ModuleConfig `json:"module_config,omitempty"`
}

// ProcessResponseContext represents the context for response processing
type ProcessResponseContext struct {
	// Inherits request context
	*ProcessRequestContext
	
	// Response details
	StatusCode      int               `json:"status_code"`
	ResponseHeaders map[string]string `json:"response_headers"`
	ResponseBody    []byte            `json:"response_body,omitempty"`
	
	// Performance metrics
	ProviderLatency time.Duration `json:"provider_latency"`
	TotalLatency    time.Duration `json:"total_latency"`
	
	// Usage information
	TokensUsed    *TokenUsage `json:"tokens_used,omitempty"`
	CostUSD       float64     `json:"cost_usd,omitempty"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

// ProcessRequestResult represents the result of request processing
type ProcessRequestResult struct {
	Action            Action                 `json:"action"`
	ModifiedBody      []byte                 `json:"modified_body,omitempty"`
	AdditionalHeaders map[string]string      `json:"additional_headers,omitempty"`
	BlockReason       string                 `json:"block_reason,omitempty"`
	Annotations       map[string]interface{} `json:"annotations,omitempty"`
	ProcessingTime    time.Duration          `json:"processing_time"`
	Confidence        float64                `json:"confidence,omitempty"` // 0.0-1.0
	Metadata          map[string]string      `json:"metadata,omitempty"`
}

// ProcessResponseResult represents the result of response processing
type ProcessResponseResult struct {
	Action            Action                 `json:"action"`
	ModifiedBody      []byte                 `json:"modified_body,omitempty"`
	ModifiedHeaders   map[string]string      `json:"modified_headers,omitempty"`
	Annotations       map[string]interface{} `json:"annotations,omitempty"`
	ProcessingTime    time.Duration          `json:"processing_time"`
	Metadata          map[string]string      `json:"metadata,omitempty"`
}

// Action represents module actions
type Action int

const (
	ActionContinue   Action = iota // Continue to next module
	ActionBlock                    // Block the request
	ActionTransform                // Transform the request/response
	ActionAnnotate                 // Add annotations but continue
	ActionRetry                    // Retry the request
	ActionRoute                    // Route to different provider
)

func (a Action) String() string {
	switch a {
	case ActionContinue:
		return "continue"
	case ActionBlock:
		return "block"
	case ActionTransform:
		return "transform"
	case ActionAnnotate:
		return "annotate"
	case ActionRetry:
		return "retry"
	case ActionRoute:
		return "route"
	default:
		return "unknown"
	}
}

// HealthStatus represents module health status
type HealthStatus struct {
	Status        HealthState            `json:"status"`
	Message       string                 `json:"message,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
	LastCheck     time.Time              `json:"last_check"`
	CheckDuration time.Duration          `json:"check_duration"`
}

// HealthState represents health state
type HealthState int

const (
	HealthStateHealthy   HealthState = iota
	HealthStateUnhealthy
	HealthStateDegraded
	HealthStateUnknown
)

func (h HealthState) String() string {
	switch h {
	case HealthStateHealthy:
		return "healthy"
	case HealthStateUnhealthy:
		return "unhealthy"
	case HealthStateDegraded:
		return "degraded"
	default:
		return "unknown"
	}
}

// ModuleStatus represents module runtime status
type ModuleStatus struct {
	State             ModuleState            `json:"state"`
	StartTime         time.Time              `json:"start_time,omitempty"`
	LastActivity      time.Time              `json:"last_activity,omitempty"`
	RequestsProcessed int64                  `json:"requests_processed"`
	ErrorCount        int64                  `json:"error_count"`
	AverageLatency    time.Duration          `json:"average_latency"`
	ResourceUsage     *ResourceUsage         `json:"resource_usage,omitempty"`
}

// ModuleState represents module state
type ModuleState int

const (
	ModuleStateLoading      ModuleState = iota
	ModuleStateInitializing
	ModuleStateReady
	ModuleStateRunning
	ModuleStateDraining
	ModuleStateStopped
	ModuleStateFailed
)

func (s ModuleState) String() string {
	switch s {
	case ModuleStateLoading:
		return "loading"
	case ModuleStateInitializing:
		return "initializing"
	case ModuleStateReady:
		return "ready"
	case ModuleStateRunning:
		return "running"
	case ModuleStateDraining:
		return "draining"
	case ModuleStateStopped:
		return "stopped"
	case ModuleStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ResourceUsage represents module resource usage
type ResourceUsage struct {
	MemoryUsageMB   float64   `json:"memory_usage_mb"`
	CPUUsagePercent float64   `json:"cpu_usage_percent"`
	LastUpdated     time.Time `json:"last_updated"`
}

// Registry interface for module management
type Registry interface {
	Register(module Module) error
	Unregister(name string) error
	Get(name string) (Module, error)
	List() []Module
	ListByType(moduleType ModuleType) []Module
	Reload(name string) error
	ValidateModule(module Module) error
}

// Loader interface for module loading
type Loader interface {
	LoadFromFile(path string) (Module, error)
	LoadFromPlugin(path string) (Module, error)
	ValidatePlugin(path string) error
	UnloadModule(name string) error
}
