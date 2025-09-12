package ratelimiter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bendiamant/leash-gateway/internal/modules/interface"
	"go.uber.org/zap"
)

// RateLimiter implements a token bucket rate limiter module
type RateLimiter struct {
	name         string
	version      string
	description  string
	author       string
	config       *RateLimiterConfig
	buckets      map[string]*TokenBucket
	mu           sync.RWMutex
	logger       *zap.SugaredLogger
	status       *interfaces.ModuleStatus
	startTime    time.Time
}

// RateLimiterConfig represents rate limiter configuration
type RateLimiterConfig struct {
	Algorithm      string        `yaml:"algorithm" json:"algorithm"`           // token_bucket, fixed_window, sliding_window
	DefaultLimit   int64         `yaml:"default_limit" json:"default_limit"`   // requests per window
	DefaultWindow  time.Duration `yaml:"default_window" json:"default_window"` // time window
	Storage        string        `yaml:"storage" json:"storage"`               // memory, redis
	BurstSize      int64         `yaml:"burst_size" json:"burst_size"`         // max burst allowed
	RefillRate     int64         `yaml:"refill_rate" json:"refill_rate"`       // tokens per second
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	capacity    int64
	tokens      int64
	refillRate  int64
	lastRefill  time.Time
	mu          sync.Mutex
}

// NewRateLimiter creates a new rate limiter module
func NewRateLimiter(logger *zap.SugaredLogger) *RateLimiter {
	return &RateLimiter{
		name:        "rate-limiter",
		version:     "1.0.0",
		description: "Token bucket rate limiter for request throttling",
		author:      "Leash Security",
		buckets:     make(map[string]*TokenBucket),
		logger:      logger,
		status: &interfaces.ModuleStatus{
			State:             interfaces.ModuleStateReady,
			RequestsProcessed: 0,
			ErrorCount:        0,
		},
	}
}

// Metadata methods
func (rl *RateLimiter) Name() string        { return rl.name }
func (rl *RateLimiter) Version() string     { return rl.version }
func (rl *RateLimiter) Type() interfaces.ModuleType { return interfaces.ModuleTypePolicy }
func (rl *RateLimiter) Description() string { return rl.description }
func (rl *RateLimiter) Author() string      { return rl.author }
func (rl *RateLimiter) Dependencies() []string { return []string{} }

// Lifecycle methods
func (rl *RateLimiter) Initialize(ctx context.Context, config *interfaces.ModuleConfig) error {
	rl.logger.Infof("Initializing rate limiter module")

	// Parse configuration
	rateLimiterConfig := &RateLimiterConfig{
		Algorithm:     "token_bucket",
		DefaultLimit:  1000,
		DefaultWindow: time.Hour,
		Storage:       "memory",
		BurstSize:     100,
		RefillRate:    1000, // 1000 tokens per second
	}

	// Override with provided config
	if config != nil && config.Config != nil {
		if algorithm, ok := config.Config["algorithm"].(string); ok {
			rateLimiterConfig.Algorithm = algorithm
		}
		if limit, ok := config.Config["default_limit"].(int); ok {
			rateLimiterConfig.DefaultLimit = int64(limit)
		}
		if window, ok := config.Config["default_window"].(string); ok {
			if duration, err := time.ParseDuration(window); err == nil {
				rateLimiterConfig.DefaultWindow = duration
			}
		}
		if storage, ok := config.Config["storage"].(string); ok {
			rateLimiterConfig.Storage = storage
		}
		if burstSize, ok := config.Config["burst_size"].(int); ok {
			rateLimiterConfig.BurstSize = int64(burstSize)
		}
		if refillRate, ok := config.Config["refill_rate"].(int); ok {
			rateLimiterConfig.RefillRate = int64(refillRate)
		}
	}

	rl.config = rateLimiterConfig
	rl.startTime = time.Now()
	rl.status.State = interfaces.ModuleStateReady

	rl.logger.Infof("Rate limiter initialized with algorithm=%s, limit=%d, window=%v", 
		rateLimiterConfig.Algorithm, rateLimiterConfig.DefaultLimit, rateLimiterConfig.DefaultWindow)

	return nil
}

func (rl *RateLimiter) Start(ctx context.Context) error {
	rl.status.State = interfaces.ModuleStateRunning
	rl.status.StartTime = time.Now()
	rl.logger.Infof("Rate limiter module started")
	return nil
}

func (rl *RateLimiter) Stop(ctx context.Context) error {
	rl.status.State = interfaces.ModuleStateDraining
	rl.logger.Infof("Rate limiter module stopping")
	return nil
}

func (rl *RateLimiter) Shutdown(ctx context.Context) error {
	rl.status.State = interfaces.ModuleStateStopped
	rl.logger.Infof("Rate limiter module shutdown")
	return nil
}

// Health and status methods
func (rl *RateLimiter) Health(ctx context.Context) (*interfaces.HealthStatus, error) {
	return &interfaces.HealthStatus{
		Status:        interfaces.HealthStateHealthy,
		Message:       "Rate limiter is healthy",
		LastCheck:     time.Now(),
		CheckDuration: time.Millisecond,
		Details: map[string]interface{}{
			"active_buckets": len(rl.buckets),
			"algorithm":      rl.config.Algorithm,
			"default_limit":  rl.config.DefaultLimit,
		},
	}, nil
}

func (rl *RateLimiter) Status() *interfaces.ModuleStatus {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	status := *rl.status
	status.LastActivity = time.Now()
	return &status
}

func (rl *RateLimiter) Metrics() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"requests_processed": rl.status.RequestsProcessed,
		"errors":            rl.status.ErrorCount,
		"active_buckets":    len(rl.buckets),
		"uptime_seconds":    time.Since(rl.startTime).Seconds(),
	}
}

// Processing methods
func (rl *RateLimiter) ProcessRequest(ctx context.Context, req *interfaces.ProcessRequestContext) (*interfaces.ProcessRequestResult, error) {
	start := time.Now()
	rl.status.RequestsProcessed++
	rl.status.LastActivity = time.Now()

	// Create bucket key (tenant-based)
	bucketKey := fmt.Sprintf("%s:%s", req.TenantID, req.Provider)
	
	bucket := rl.getBucket(bucketKey)
	
	if !bucket.Allow() {
		rl.logger.Warnf("Rate limit exceeded for tenant %s, provider %s", req.TenantID, req.Provider)
		return &interfaces.ProcessRequestResult{
			Action:         interfaces.ActionBlock,
			BlockReason:    "rate_limit_exceeded",
			ProcessingTime: time.Since(start),
			Annotations: map[string]interface{}{
				"rate_limit_exceeded": true,
				"bucket_key":          bucketKey,
				"limit":               rl.config.DefaultLimit,
			},
		}, nil
	}

	return &interfaces.ProcessRequestResult{
		Action:         interfaces.ActionContinue,
		ProcessingTime: time.Since(start),
		Annotations: map[string]interface{}{
			"rate_limit_checked": true,
			"bucket_key":         bucketKey,
			"tokens_remaining":   bucket.tokens,
		},
	}, nil
}

func (rl *RateLimiter) ProcessResponse(ctx context.Context, resp *interfaces.ProcessResponseContext) (*interfaces.ProcessResponseResult, error) {
	// Rate limiter doesn't need to process responses
	return &interfaces.ProcessResponseResult{
		Action: interfaces.ActionContinue,
	}, nil
}

// Configuration methods
func (rl *RateLimiter) ValidateConfig(config *interfaces.ModuleConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if !config.Enabled {
		return nil // Skip validation for disabled modules
	}

	// Validate algorithm
	if configMap := config.Config; configMap != nil {
		if algorithm, ok := configMap["algorithm"].(string); ok {
			if algorithm != "token_bucket" && algorithm != "fixed_window" && algorithm != "sliding_window" {
				return fmt.Errorf("unsupported algorithm: %s", algorithm)
			}
		}

		// Validate limits
		if limit, ok := configMap["default_limit"].(int); ok {
			if limit <= 0 {
				return fmt.Errorf("default_limit must be positive, got %d", limit)
			}
		}
	}

	return nil
}

func (rl *RateLimiter) UpdateConfig(ctx context.Context, config *interfaces.ModuleConfig) error {
	if err := rl.ValidateConfig(config); err != nil {
		return err
	}

	// Re-initialize with new config
	return rl.Initialize(ctx, config)
}

func (rl *RateLimiter) GetConfig() *interfaces.ModuleConfig {
	return &interfaces.ModuleConfig{
		Name:     rl.name,
		Type:     rl.Type().String(),
		Enabled:  rl.status.State == interfaces.ModuleStateRunning,
		Priority: 100, // High priority for rate limiting
		Config: map[string]interface{}{
			"algorithm":      rl.config.Algorithm,
			"default_limit":  rl.config.DefaultLimit,
			"default_window": rl.config.DefaultWindow.String(),
			"storage":        rl.config.Storage,
			"burst_size":     rl.config.BurstSize,
			"refill_rate":    rl.config.RefillRate,
		},
	}
}

// getBucket gets or creates a token bucket for a key
func (rl *RateLimiter) getBucket(key string) *TokenBucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &TokenBucket{
			capacity:   rl.config.BurstSize,
			tokens:     rl.config.BurstSize,
			refillRate: rl.config.RefillRate,
			lastRefill: time.Now(),
		}
		rl.buckets[key] = bucket
	}

	return bucket
}

// Allow checks if a request is allowed by the token bucket
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	
	// Refill tokens based on elapsed time
	tokensToAdd := int64(elapsed.Seconds()) * tb.refillRate
	tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
	tb.lastRefill = now

	// Check if we have tokens available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// min returns the minimum of two int64 values
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
