package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// Registry wraps prometheus registry with custom metrics
type Registry struct {
	*prometheus.Registry
	
	// Request metrics
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestSizeBytes *prometheus.HistogramVec
	ResponseSizeBytes *prometheus.HistogramVec
	
	// Module metrics
	ModuleProcessingDuration *prometheus.HistogramVec
	ModuleExecutions        *prometheus.CounterVec
	ModuleErrors           *prometheus.CounterVec
	
	// Business metrics
	TokensProcessed    *prometheus.CounterVec
	CostAccrued       *prometheus.CounterVec
	PolicyViolations  *prometheus.CounterVec
	PIIDetections     *prometheus.CounterVec
	
	// Provider metrics
	ProviderRequests  *prometheus.CounterVec
	ProviderLatency   *prometheus.HistogramVec
	CircuitBreakerState *prometheus.GaugeVec
	
	// System metrics
	ActiveConnections *prometheus.GaugeVec
	ConfigReloads     *prometheus.CounterVec
	CacheOperations   *prometheus.CounterVec
	
	// SLI/SLO metrics
	SLOCompliance       *prometheus.GaugeVec
	ErrorBudgetRemaining *prometheus.GaugeVec
}

// NewRegistry creates a new metrics registry with all custom metrics
func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()
	
	// Add Go runtime metrics
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	
	registry := &Registry{
		Registry: reg,
	}
	
	// Initialize custom metrics
	registry.initializeMetrics()
	
	return registry
}

// initializeMetrics initializes all custom metrics
func (r *Registry) initializeMetrics() {
	// Request metrics
	r.RequestsTotal = r.registerCounterVec(
		"leash_gateway_requests_total",
		"Total number of requests processed",
		[]string{"tenant", "provider", "model", "status", "method"},
	)
	
	r.RequestDuration = r.registerHistogramVec(
		"leash_gateway_request_duration_seconds",
		"Request processing duration in seconds",
		[]string{"tenant", "provider", "model"},
		[]float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
	)
	
	r.RequestSizeBytes = r.registerHistogramVec(
		"leash_gateway_request_size_bytes",
		"Request size in bytes",
		[]string{"tenant", "provider"},
		prometheus.ExponentialBuckets(100, 2, 10), // 100B to 50KB
	)
	
	r.ResponseSizeBytes = r.registerHistogramVec(
		"leash_gateway_response_size_bytes",
		"Response size in bytes",
		[]string{"tenant", "provider"},
		prometheus.ExponentialBuckets(100, 2, 15), // 100B to 1.6MB
	)
	
	// Module metrics
	r.ModuleProcessingDuration = r.registerHistogramVec(
		"leash_module_processing_duration_seconds",
		"Module processing duration in seconds",
		[]string{"module_name", "module_type", "tenant"},
		[]float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .5},
	)
	
	r.ModuleExecutions = r.registerCounterVec(
		"leash_module_executions_total",
		"Total number of module executions",
		[]string{"module_name", "module_type", "tenant", "status"},
	)
	
	r.ModuleErrors = r.registerCounterVec(
		"leash_module_errors_total",
		"Total number of module errors",
		[]string{"module_name", "module_type", "tenant", "error_type"},
	)
	
	// Business metrics
	r.TokensProcessed = r.registerCounterVec(
		"leash_tokens_processed_total",
		"Total number of tokens processed",
		[]string{"tenant", "provider", "model", "token_type"}, // input, output
	)
	
	r.CostAccrued = r.registerCounterVec(
		"leash_cost_usd_total",
		"Total cost accrued in USD",
		[]string{"tenant", "provider", "model"},
	)
	
	r.PolicyViolations = r.registerCounterVec(
		"leash_policy_violations_total",
		"Total number of policy violations",
		[]string{"tenant", "policy_name", "violation_type", "action"},
	)
	
	r.PIIDetections = r.registerCounterVec(
		"leash_pii_detections_total",
		"Total number of PII detections",
		[]string{"tenant", "pii_type", "location"}, // request, response
	)
	
	// Provider metrics
	r.ProviderRequests = r.registerCounterVec(
		"leash_provider_requests_total",
		"Total requests sent to providers",
		[]string{"provider", "status", "model"},
	)
	
	r.ProviderLatency = r.registerHistogramVec(
		"leash_provider_latency_seconds",
		"Provider response latency in seconds",
		[]string{"provider", "model"},
		[]float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60},
	)
	
	r.CircuitBreakerState = r.registerGaugeVec(
		"leash_circuit_breaker_state",
		"Circuit breaker state (0=closed, 1=open, 2=half-open)",
		[]string{"provider"},
	)
	
	// System metrics
	r.ActiveConnections = r.registerGaugeVec(
		"leash_active_connections",
		"Number of active connections",
		[]string{"type"}, // http, grpc
	)
	
	r.ConfigReloads = r.registerCounterVec(
		"leash_config_reloads_total",
		"Total number of configuration reloads",
		[]string{"status"}, // success, failure
	)
	
	r.CacheOperations = r.registerCounterVec(
		"leash_cache_operations_total",
		"Total cache operations",
		[]string{"operation", "result"}, // get/set/delete, hit/miss/error
	)
	
	// SLI/SLO metrics
	r.SLOCompliance = r.registerGaugeVec(
		"leash_slo_compliance_ratio",
		"SLO compliance ratio (0-1)",
		[]string{"slo_name", "tenant"},
	)
	
	r.ErrorBudgetRemaining = r.registerGaugeVec(
		"leash_error_budget_remaining",
		"Remaining error budget (0-1)",
		[]string{"slo_name", "tenant", "window"}, // 1h, 24h, 30d
	)
}

// registerCounterVec creates and registers a counter vector
func (r *Registry) registerCounterVec(name, help string, labels []string) *prometheus.CounterVec {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	r.Registry.MustRegister(counter)
	return counter
}

// registerHistogramVec creates and registers a histogram vector
func (r *Registry) registerHistogramVec(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    help,
			Buckets: buckets,
		},
		labels,
	)
	r.Registry.MustRegister(histogram)
	return histogram
}

// registerGaugeVec creates and registers a gauge vector
func (r *Registry) registerGaugeVec(name, help string, labels []string) *prometheus.GaugeVec {
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	r.Registry.MustRegister(gauge)
	return gauge
}

// RecordHTTPMetrics records HTTP request metrics
func (r *Registry) RecordHTTPMetrics(tenant, provider, model, method string, status int, duration float64, requestSize, responseSize int64) {
	labels := prometheus.Labels{
		"tenant":   tenant,
		"provider": provider,
		"model":    model,
		"method":   method,
		"status":   fmt.Sprintf("%d", status),
	}
	
	r.RequestsTotal.With(labels).Inc()
	r.RequestDuration.WithLabelValues(tenant, provider, model).Observe(duration)
	r.RequestSizeBytes.WithLabelValues(tenant, provider).Observe(float64(requestSize))
	r.ResponseSizeBytes.WithLabelValues(tenant, provider).Observe(float64(responseSize))
}

// RecordBusinessMetrics records business-related metrics
func (r *Registry) RecordBusinessMetrics(tenant, provider, model string, inputTokens, outputTokens int64, cost float64) {
	r.TokensProcessed.WithLabelValues(tenant, provider, model, "input").Add(float64(inputTokens))
	r.TokensProcessed.WithLabelValues(tenant, provider, model, "output").Add(float64(outputTokens))
	r.CostAccrued.WithLabelValues(tenant, provider, model).Add(cost)
}

// RecordModuleMetrics records module execution metrics
func (r *Registry) RecordModuleMetrics(moduleName, moduleType, tenant, status string, duration float64) {
	r.ModuleExecutions.WithLabelValues(moduleName, moduleType, tenant, status).Inc()
	r.ModuleProcessingDuration.WithLabelValues(moduleName, moduleType, tenant).Observe(duration)
}

// RecordModuleError records module error metrics
func (r *Registry) RecordModuleError(moduleName, moduleType, tenant, errorType string) {
	r.ModuleErrors.WithLabelValues(moduleName, moduleType, tenant, errorType).Inc()
}
