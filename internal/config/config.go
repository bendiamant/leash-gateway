package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/grpc/keepalive"
)

// Config represents the complete gateway configuration
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Envoy         EnvoyConfig         `mapstructure:"envoy"`
	ModuleHost    ModuleHostConfig    `mapstructure:"module_host"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Redis         RedisConfig         `mapstructure:"redis"`
	Tenants       map[string]Tenant   `mapstructure:"tenants"`
	Providers     map[string]Provider `mapstructure:"providers"`
	Modules       map[string]Module   `mapstructure:"modules"`
	Observability ObservabilityConfig `mapstructure:"observability"`
	Security      SecurityConfig      `mapstructure:"security"`
	FeatureFlags  FeatureFlagsConfig  `mapstructure:"feature_flags"`
	Development   DevelopmentConfig   `mapstructure:"development"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port           int           `mapstructure:"port"`
	Host           string        `mapstructure:"host"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	IdleTimeout    time.Duration `mapstructure:"idle_timeout"`
	MaxHeaderBytes int           `mapstructure:"max_header_bytes"`
}

// EnvoyConfig contains Envoy proxy configuration
type EnvoyConfig struct {
	AdminPort  int    `mapstructure:"admin_port"`
	ConfigPath string `mapstructure:"config_path"`
	StatsPort  int    `mapstructure:"stats_port"`
	LogLevel   string `mapstructure:"log_level"`
}

// ModuleHostConfig contains Module Host gRPC service configuration
type ModuleHostConfig struct {
	GRPCPort       int                    `mapstructure:"grpc_port"`
	HealthPort     int                    `mapstructure:"health_port"`
	MaxRecvMsgSize int                    `mapstructure:"max_recv_msg_size"`
	MaxSendMsgSize int                    `mapstructure:"max_send_msg_size"`
	Keepalive      KeepaliveConfig        `mapstructure:"keepalive"`
}

// KeepaliveConfig contains gRPC keepalive configuration
type KeepaliveConfig struct {
	Time                time.Duration `mapstructure:"time"`
	Timeout             time.Duration `mapstructure:"timeout"`
	PermitWithoutStream bool          `mapstructure:"permit_without_stream"`
}

// KeepaliveParams returns gRPC keepalive parameters
func (c ModuleHostConfig) KeepaliveParams() keepalive.ServerParameters {
	return keepalive.ServerParameters{
		Time:    c.Keepalive.Time,
		Timeout: c.Keepalive.Timeout,
	}
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	URL             string        `mapstructure:"url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	MigrationsPath  string        `mapstructure:"migrations_path"`
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	URL          string        `mapstructure:"url"`
	MaxRetries   int           `mapstructure:"max_retries"`
	RetryDelay   time.Duration `mapstructure:"retry_delay"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
}

// Tenant represents a tenant configuration
type Tenant struct {
	Name        string               `mapstructure:"name"`
	Description string               `mapstructure:"description"`
	Policies    []string             `mapstructure:"policies"`
	Quotas      TenantQuotas         `mapstructure:"quotas"`
	RateLimits  []RateLimit          `mapstructure:"rate_limits"`
	Providers   map[string]Provider  `mapstructure:"providers"`
}

// TenantQuotas represents tenant usage quotas
type TenantQuotas struct {
	RequestsPerHour int     `mapstructure:"requests_per_hour"`
	RequestsPerDay  int     `mapstructure:"requests_per_day"`
	CostLimitUSD    float64 `mapstructure:"cost_limit_usd"`
}

// RateLimit represents a rate limiting rule
type RateLimit struct {
	Name       string                 `mapstructure:"name"`
	Limit      int                    `mapstructure:"limit"`
	Window     string                 `mapstructure:"window"`
	Conditions []map[string]interface{} `mapstructure:"conditions"`
}

// Provider represents a provider configuration
type Provider struct {
	Endpoint                 string                 `mapstructure:"endpoint"`
	Timeout                  time.Duration          `mapstructure:"timeout"`
	RetryAttempts           int                    `mapstructure:"retry_attempts"`
	RetryDelay              time.Duration          `mapstructure:"retry_delay"`
	RetryBackoffMultiplier  float64                `mapstructure:"retry_backoff_multiplier"`
	MaxRetryDelay           time.Duration          `mapstructure:"max_retry_delay"`
	CircuitBreaker          CircuitBreakerConfig   `mapstructure:"circuit_breaker"`
	HealthCheck             HealthCheckConfig      `mapstructure:"health_check"`
	Models                  []ModelConfig          `mapstructure:"models"`
}

// CircuitBreakerConfig represents circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int           `mapstructure:"failure_threshold"`
	SuccessThreshold int           `mapstructure:"success_threshold"`
	Timeout          time.Duration `mapstructure:"timeout"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Interval time.Duration `mapstructure:"interval"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Path     string        `mapstructure:"path"`
}

// ModelConfig represents model pricing configuration
type ModelConfig struct {
	Name                   string  `mapstructure:"name"`
	CostPer1kInputTokens   float64 `mapstructure:"cost_per_1k_input_tokens"`
	CostPer1kOutputTokens  float64 `mapstructure:"cost_per_1k_output_tokens"`
}

// Module represents a module configuration
type Module struct {
	Enabled    bool                   `mapstructure:"enabled"`
	Type       string                 `mapstructure:"type"`
	Priority   int                    `mapstructure:"priority"`
	Config     map[string]interface{} `mapstructure:"config"`
	Conditions []map[string]interface{} `mapstructure:"conditions"`
}

// ObservabilityConfig contains observability configuration
type ObservabilityConfig struct {
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Tracing   TracingConfig   `mapstructure:"tracing"`
	Profiling ProfilingConfig `mapstructure:"profiling"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled    bool              `mapstructure:"enabled"`
	Port       int               `mapstructure:"port"`
	Path       string            `mapstructure:"path"`
	Collectors []string          `mapstructure:"collectors"`
	Labels     map[string]string `mapstructure:"labels"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level       string `mapstructure:"level"`
	Format      string `mapstructure:"format"`
	Output      string `mapstructure:"output"`
	AddSource   bool   `mapstructure:"add_source"`
	Development bool   `mapstructure:"development"`
}

// TracingConfig contains tracing configuration
type TracingConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	ServiceName string        `mapstructure:"service_name"`
	Endpoint    string        `mapstructure:"endpoint"`
	Sampler     SamplerConfig `mapstructure:"sampler"`
}

// SamplerConfig contains sampler configuration
type SamplerConfig struct {
	Type  string  `mapstructure:"type"`
	Param float64 `mapstructure:"param"`
}

// ProfilingConfig contains profiling configuration
type ProfilingConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	APIKeys            APIKeysConfig        `mapstructure:"api_keys"`
	CORS               CORSConfig           `mapstructure:"cors"`
	RateLimiting       RateLimitingConfig   `mapstructure:"rate_limiting"`
	RequestSizeLimits  RequestSizeLimits    `mapstructure:"request_size_limits"`
}

// APIKeysConfig contains API key configuration
type APIKeysConfig struct {
	HeaderName string `mapstructure:"header_name"`
	Prefix     string `mapstructure:"prefix"`
	MinLength  int    `mapstructure:"min_length"`
	MaxLength  int    `mapstructure:"max_length"`
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	ExposeHeaders  []string `mapstructure:"expose_headers"`
	MaxAge         int      `mapstructure:"max_age"`
}

// RateLimitingConfig contains rate limiting configuration
type RateLimitingConfig struct {
	Global GlobalRateLimit `mapstructure:"global"`
	PerIP  PerIPRateLimit  `mapstructure:"per_ip"`
}

// GlobalRateLimit contains global rate limiting configuration
type GlobalRateLimit struct {
	Enabled bool   `mapstructure:"enabled"`
	Limit   int    `mapstructure:"limit"`
	Window  string `mapstructure:"window"`
}

// PerIPRateLimit contains per-IP rate limiting configuration
type PerIPRateLimit struct {
	Enabled bool   `mapstructure:"enabled"`
	Limit   int    `mapstructure:"limit"`
	Window  string `mapstructure:"window"`
}

// RequestSizeLimits contains request size limits
type RequestSizeLimits struct {
	MaxBodySize   string `mapstructure:"max_body_size"`
	MaxHeaderSize string `mapstructure:"max_header_size"`
}

// FeatureFlagsConfig contains feature flags
type FeatureFlagsConfig struct {
	EnableStreaming             bool `mapstructure:"enable_streaming"`
	EnableCaching               bool `mapstructure:"enable_caching"`
	EnableRequestSigning        bool `mapstructure:"enable_request_signing"`
	EnableResponseCompression   bool `mapstructure:"enable_response_compression"`
	EnableRequestDeduplication  bool `mapstructure:"enable_request_deduplication"`
}

// DevelopmentConfig contains development/debug settings
type DevelopmentConfig struct {
	DebugMode     bool `mapstructure:"debug_mode"`
	MockProviders bool `mapstructure:"mock_providers"`
	LogRequests   bool `mapstructure:"log_requests"`
	LogResponses  bool `mapstructure:"log_responses"`
	EnablePprof   bool `mapstructure:"enable_pprof"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	v := viper.New()

	// Set config file path
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/gateway/config.yaml"
	}

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Enable environment variable substitution
	v.AutomaticEnv()
	v.SetEnvPrefix("LEASH")

	// Set defaults
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use defaults and env vars
			fmt.Printf("Config file not found at %s, using defaults\n", configPath)
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Parse duration strings
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "120s")

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "120s")
	v.SetDefault("server.max_header_bytes", 1048576)

	// Module Host defaults
	v.SetDefault("module_host.grpc_port", 50051)
	v.SetDefault("module_host.health_port", 8081)
	v.SetDefault("module_host.max_recv_msg_size", 4194304)
	v.SetDefault("module_host.max_send_msg_size", 4194304)
	v.SetDefault("module_host.keepalive.time", "30s")
	v.SetDefault("module_host.keepalive.timeout", "5s")
	v.SetDefault("module_host.keepalive.permit_without_stream", true)

	// Observability defaults
	v.SetDefault("observability.metrics.enabled", true)
	v.SetDefault("observability.metrics.port", 9090)
	v.SetDefault("observability.metrics.path", "/metrics")
	v.SetDefault("observability.logging.level", "info")
	v.SetDefault("observability.logging.format", "json")
	v.SetDefault("observability.logging.output", "stdout")
	v.SetDefault("observability.logging.add_source", true)
	v.SetDefault("observability.logging.development", false)
}

// validate validates the configuration
func validate(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.ModuleHost.GRPCPort <= 0 || config.ModuleHost.GRPCPort > 65535 {
		return fmt.Errorf("invalid module host gRPC port: %d", config.ModuleHost.GRPCPort)
	}

	if config.ModuleHost.HealthPort <= 0 || config.ModuleHost.HealthPort > 65535 {
		return fmt.Errorf("invalid module host health port: %d", config.ModuleHost.HealthPort)
	}

	// Validate observability config
	if config.Observability.Metrics.Port <= 0 || config.Observability.Metrics.Port > 65535 {
		return fmt.Errorf("invalid metrics port: %d", config.Observability.Metrics.Port)
	}

	return nil
}
