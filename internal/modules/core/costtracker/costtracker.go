package costtracker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bendiamant/leash-gateway/internal/modules/interface"
	"go.uber.org/zap"
)

// CostTracker implements a cost tracking and limiting module
type CostTracker struct {
	name        string
	version     string
	description string
	author      string
	config      *CostTrackerConfig
	usage       map[string]*TenantUsage
	logger      *zap.SugaredLogger
	status      *interfaces.ModuleStatus
	startTime   time.Time
	mu          sync.RWMutex
}

// CostTrackerConfig represents cost tracker configuration
type CostTrackerConfig struct {
	Storage           string                    `yaml:"storage" json:"storage"`                       // memory, database
	AggregationWindow time.Duration            `yaml:"aggregation_window" json:"aggregation_window"` // 1h, 24h
	AlertThresholds   []AlertThreshold          `yaml:"alert_thresholds" json:"alert_thresholds"`
	Limits            map[string]CostLimit      `yaml:"limits" json:"limits"` // per-tenant limits
	TrackRequests     bool                      `yaml:"track_requests" json:"track_requests"`
	TrackResponses    bool                      `yaml:"track_responses" json:"track_responses"`
}

// AlertThreshold represents a cost alert threshold
type AlertThreshold struct {
	Threshold    float64 `yaml:"threshold" json:"threshold"`
	Notification string  `yaml:"notification" json:"notification"` // email, webhook, log
	Message      string  `yaml:"message" json:"message"`
}

// CostLimit represents per-tenant cost limits
type CostLimit struct {
	HourlyLimitUSD float64 `yaml:"hourly_limit_usd" json:"hourly_limit_usd"`
	DailyLimitUSD  float64 `yaml:"daily_limit_usd" json:"daily_limit_usd"`
	MonthlyLimitUSD float64 `yaml:"monthly_limit_usd" json:"monthly_limit_usd"`
}

// TenantUsage represents usage tracking for a tenant
type TenantUsage struct {
	TenantID      string                 `json:"tenant_id"`
	HourlyUsage   map[string]float64     `json:"hourly_usage"`   // hour -> cost
	DailyUsage    map[string]float64     `json:"daily_usage"`    // date -> cost
	MonthlyUsage  map[string]float64     `json:"monthly_usage"`  // month -> cost
	TotalCost     float64                `json:"total_cost"`
	RequestCount  int64                  `json:"request_count"`
	LastUpdated   time.Time              `json:"last_updated"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// NewCostTracker creates a new cost tracker module
func NewCostTracker(logger *zap.SugaredLogger) *CostTracker {
	return &CostTracker{
		name:        "cost-tracker",
		version:     "1.0.0",
		description: "Cost tracking and limiting module for monitoring LLM usage costs",
		author:      "Leash Security",
		usage:       make(map[string]*TenantUsage),
		logger:      logger,
		status: &interfaces.ModuleStatus{
			State:             interfaces.ModuleStateReady,
			RequestsProcessed: 0,
			ErrorCount:        0,
		},
	}
}

// Metadata methods
func (ct *CostTracker) Name() string                    { return ct.name }
func (ct *CostTracker) Version() string                 { return ct.version }
func (ct *CostTracker) Type() interfaces.ModuleType     { return interfaces.ModuleTypeSink }
func (ct *CostTracker) Description() string             { return ct.description }
func (ct *CostTracker) Author() string                  { return ct.author }
func (ct *CostTracker) Dependencies() []string          { return []string{} }

// Lifecycle methods
func (ct *CostTracker) Initialize(ctx context.Context, config *interfaces.ModuleConfig) error {
	ct.logger.Infof("Initializing cost tracker module")

	// Parse configuration
	trackerConfig := &CostTrackerConfig{
		Storage:           "memory",
		AggregationWindow: time.Hour,
		TrackRequests:     true,
		TrackResponses:    true,
		AlertThresholds: []AlertThreshold{
			{Threshold: 100.0, Notification: "log", Message: "Cost threshold exceeded"},
		},
		Limits: make(map[string]CostLimit),
	}

	// Override with provided config
	if config != nil && config.Config != nil {
		if storage, ok := config.Config["storage"].(string); ok {
			trackerConfig.Storage = storage
		}
		if window, ok := config.Config["aggregation_window"].(string); ok {
			if duration, err := time.ParseDuration(window); err == nil {
				trackerConfig.AggregationWindow = duration
			}
		}
		if trackRequests, ok := config.Config["track_requests"].(bool); ok {
			trackerConfig.TrackRequests = trackRequests
		}
		if trackResponses, ok := config.Config["track_responses"].(bool); ok {
			trackerConfig.TrackResponses = trackResponses
		}
		
		// Parse alert thresholds
		if thresholds, ok := config.Config["alert_thresholds"].([]interface{}); ok {
			trackerConfig.AlertThresholds = make([]AlertThreshold, 0, len(thresholds))
			for _, threshold := range thresholds {
				if thresholdMap, ok := threshold.(map[string]interface{}); ok {
					alert := AlertThreshold{}
					if th, ok := thresholdMap["threshold"].(float64); ok {
						alert.Threshold = th
					}
					if notif, ok := thresholdMap["notification"].(string); ok {
						alert.Notification = notif
					}
					if msg, ok := thresholdMap["message"].(string); ok {
						alert.Message = msg
					}
					trackerConfig.AlertThresholds = append(trackerConfig.AlertThresholds, alert)
				}
			}
		}
	}

	ct.config = trackerConfig
	ct.startTime = time.Now()
	ct.status.State = interfaces.ModuleStateReady

	ct.logger.Infof("Cost tracker initialized with storage=%s, window=%v, %d alert thresholds", 
		trackerConfig.Storage, trackerConfig.AggregationWindow, len(trackerConfig.AlertThresholds))

	return nil
}

func (ct *CostTracker) Start(ctx context.Context) error {
	ct.status.State = interfaces.ModuleStateRunning
	ct.status.StartTime = time.Now()
	ct.logger.Infof("Cost tracker module started")
	return nil
}

func (ct *CostTracker) Stop(ctx context.Context) error {
	ct.status.State = interfaces.ModuleStateDraining
	ct.logger.Infof("Cost tracker module stopping")
	return nil
}

func (ct *CostTracker) Shutdown(ctx context.Context) error {
	ct.status.State = interfaces.ModuleStateStopped
	ct.logger.Infof("Cost tracker module shutdown")
	return nil
}

// Health and status methods
func (ct *CostTracker) Health(ctx context.Context) (*interfaces.HealthStatus, error) {
	return &interfaces.HealthStatus{
		Status:        interfaces.HealthStateHealthy,
		Message:       "Cost tracker is healthy",
		LastCheck:     time.Now(),
		CheckDuration: time.Millisecond,
		Details: map[string]interface{}{
			"tracked_tenants":   len(ct.usage),
			"storage":           ct.config.Storage,
			"alert_thresholds":  len(ct.config.AlertThresholds),
		},
	}, nil
}

func (ct *CostTracker) Status() *interfaces.ModuleStatus {
	status := *ct.status
	status.LastActivity = time.Now()
	return &status
}

func (ct *CostTracker) Metrics() map[string]interface{} {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	totalCost := 0.0
	totalRequests := int64(0)
	
	for _, usage := range ct.usage {
		totalCost += usage.TotalCost
		totalRequests += usage.RequestCount
	}

	return map[string]interface{}{
		"requests_processed": ct.status.RequestsProcessed,
		"errors":            ct.status.ErrorCount,
		"tracked_tenants":   len(ct.usage),
		"total_cost_usd":    totalCost,
		"total_requests":    totalRequests,
		"uptime_seconds":    time.Since(ct.startTime).Seconds(),
	}
}

// Processing methods
func (ct *CostTracker) ProcessRequest(ctx context.Context, req *interfaces.ProcessRequestContext) (*interfaces.ProcessRequestResult, error) {
	start := time.Now()
	
	if !ct.config.TrackRequests {
		return &interfaces.ProcessRequestResult{
			Action:         interfaces.ActionContinue,
			ProcessingTime: time.Since(start),
		}, nil
	}

	// Estimate cost for request (basic estimation)
	estimatedCost := ct.estimateRequestCost(req)

	ct.status.RequestsProcessed++
	ct.status.LastActivity = time.Now()

	return &interfaces.ProcessRequestResult{
		Action:         interfaces.ActionContinue,
		ProcessingTime: time.Since(start),
		Annotations: map[string]interface{}{
			"estimated_cost_usd": estimatedCost,
			"cost_tracked":       true,
		},
	}, nil
}

func (ct *CostTracker) ProcessResponse(ctx context.Context, resp *interfaces.ProcessResponseContext) (*interfaces.ProcessResponseResult, error) {
	start := time.Now()

	if !ct.config.TrackResponses {
		return &interfaces.ProcessResponseResult{
			Action:         interfaces.ActionContinue,
			ProcessingTime: time.Since(start),
		}, nil
	}

	// Calculate actual cost from response
	actualCost := ct.calculateResponseCost(resp)
	
	// Track usage
	ct.trackUsage(resp.TenantID, resp.Provider, resp.Model, actualCost)

	// Check for alert thresholds
	ct.checkAlertThresholds(resp.TenantID, actualCost)

	return &interfaces.ProcessResponseResult{
		Action:         interfaces.ActionContinue,
		ProcessingTime: time.Since(start),
		Annotations: map[string]interface{}{
			"actual_cost_usd": actualCost,
			"cost_tracked":    true,
		},
	}, nil
}

// Configuration methods
func (ct *CostTracker) ValidateConfig(config *interfaces.ModuleConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if configMap := config.Config; configMap != nil {
		if storage, ok := configMap["storage"].(string); ok {
			if storage != "memory" && storage != "database" {
				return fmt.Errorf("invalid storage type: %s", storage)
			}
		}
	}

	return nil
}

func (ct *CostTracker) UpdateConfig(ctx context.Context, config *interfaces.ModuleConfig) error {
	if err := ct.ValidateConfig(config); err != nil {
		return err
	}

	return ct.Initialize(ctx, config)
}

func (ct *CostTracker) GetConfig() *interfaces.ModuleConfig {
	return &interfaces.ModuleConfig{
		Name:     ct.name,
		Type:     ct.Type().String(),
		Enabled:  ct.status.State == interfaces.ModuleStateRunning,
		Priority: 900, // Low priority for cost tracking (run near end)
		Config: map[string]interface{}{
			"storage":            ct.config.Storage,
			"aggregation_window": ct.config.AggregationWindow.String(),
			"alert_thresholds":   ct.config.AlertThresholds,
			"track_requests":     ct.config.TrackRequests,
			"track_responses":    ct.config.TrackResponses,
		},
	}
}

// Helper methods
func (ct *CostTracker) estimateRequestCost(req *interfaces.ProcessRequestContext) float64 {
	// Simple estimation based on request size
	// In reality, this would use model-specific token estimation
	bodySize := len(req.Body)
	estimatedTokens := bodySize / 4 // Rough estimate: 4 chars per token
	
	// Use a default cost per token (would be model-specific in reality)
	costPer1kTokens := 0.002 // Default cost
	return float64(estimatedTokens) / 1000.0 * costPer1kTokens
}

func (ct *CostTracker) calculateResponseCost(resp *interfaces.ProcessResponseContext) float64 {
	if resp.CostUSD > 0 {
		return resp.CostUSD
	}

	// Fallback calculation if cost not provided
	if resp.TokensUsed != nil {
		// Use default pricing (would be provider/model specific)
		inputCost := float64(resp.TokensUsed.PromptTokens) / 1000.0 * 0.0015
		outputCost := float64(resp.TokensUsed.CompletionTokens) / 1000.0 * 0.002
		return inputCost + outputCost
	}

	return 0
}

func (ct *CostTracker) trackUsage(tenantID, provider, model string, cost float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	usage, exists := ct.usage[tenantID]
	if !exists {
		usage = &TenantUsage{
			TenantID:     tenantID,
			HourlyUsage:  make(map[string]float64),
			DailyUsage:   make(map[string]float64),
			MonthlyUsage: make(map[string]float64),
			Metadata:     make(map[string]interface{}),
		}
		ct.usage[tenantID] = usage
	}

	now := time.Now()
	hourKey := now.Format("2006-01-02-15")
	dayKey := now.Format("2006-01-02")
	monthKey := now.Format("2006-01")

	// Update usage
	usage.HourlyUsage[hourKey] += cost
	usage.DailyUsage[dayKey] += cost
	usage.MonthlyUsage[monthKey] += cost
	usage.TotalCost += cost
	usage.RequestCount++
	usage.LastUpdated = now

	// Update metadata
	usage.Metadata["last_provider"] = provider
	usage.Metadata["last_model"] = model
	usage.Metadata["last_cost"] = cost

	ct.logger.Debugf("Tracked usage for tenant %s: $%.6f (total: $%.6f)", 
		tenantID, cost, usage.TotalCost)
}

func (ct *CostTracker) checkAlertThresholds(tenantID string, cost float64) {
	ct.mu.RLock()
	usage, exists := ct.usage[tenantID]
	ct.mu.RUnlock()

	if !exists {
		return
	}

	// Check daily usage against thresholds
	today := time.Now().Format("2006-01-02")
	dailyCost := usage.DailyUsage[today]

	for _, threshold := range ct.config.AlertThresholds {
		if dailyCost >= threshold.Threshold {
			ct.sendAlert(tenantID, dailyCost, threshold)
		}
	}
}

func (ct *CostTracker) sendAlert(tenantID string, cost float64, threshold AlertThreshold) {
	message := threshold.Message
	if message == "" {
		message = fmt.Sprintf("Cost threshold exceeded for tenant %s: $%.2f >= $%.2f", 
			tenantID, cost, threshold.Threshold)
	}

	switch threshold.Notification {
	case "log":
		ct.logger.Warnf("COST ALERT: %s", message)
	case "email":
		// TODO: Implement email notifications
		ct.logger.Infof("EMAIL ALERT: %s", message)
	case "webhook":
		// TODO: Implement webhook notifications
		ct.logger.Infof("WEBHOOK ALERT: %s", message)
	default:
		ct.logger.Warnf("Unknown notification type: %s", threshold.Notification)
	}
}

// GetTenantUsage returns usage information for a tenant
func (ct *CostTracker) GetTenantUsage(tenantID string) (*TenantUsage, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	usage, exists := ct.usage[tenantID]
	if !exists {
		return nil, fmt.Errorf("no usage data for tenant %s", tenantID)
	}

	// Return a copy to avoid race conditions
	usageCopy := *usage
	return &usageCopy, nil
}

// GetAllUsage returns usage information for all tenants
func (ct *CostTracker) GetAllUsage() map[string]*TenantUsage {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	result := make(map[string]*TenantUsage)
	for tenantID, usage := range ct.usage {
		usageCopy := *usage
		result[tenantID] = &usageCopy
	}

	return result
}

// ResetUsage resets usage data for a tenant
func (ct *CostTracker) ResetUsage(tenantID string) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if _, exists := ct.usage[tenantID]; !exists {
		return fmt.Errorf("no usage data for tenant %s", tenantID)
	}

	delete(ct.usage, tenantID)
	ct.logger.Infof("Reset usage data for tenant %s", tenantID)
	return nil
}
