package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bendiamant/leash-gateway/internal/modules/interface"
	"go.uber.org/zap"
)

// Logger implements a structured logging module
type Logger struct {
	name        string
	version     string
	description string
	author      string
	config      *LoggerConfig
	logger      *zap.SugaredLogger
	status      *interfaces.ModuleStatus
	startTime   time.Time
}

// LoggerConfig represents logger module configuration
type LoggerConfig struct {
	Destinations []LogDestination `yaml:"destinations" json:"destinations"`
	LogRequests  bool             `yaml:"log_requests" json:"log_requests"`
	LogResponses bool             `yaml:"log_responses" json:"log_responses"`
	RedactPII    bool             `yaml:"redact_pii" json:"redact_pii"`
}

// LogDestination represents a log destination
type LogDestination struct {
	Type     string                 `yaml:"type" json:"type"`         // stdout, file, elasticsearch
	Format   string                 `yaml:"format" json:"format"`     // json, text
	Path     string                 `yaml:"path,omitempty" json:"path,omitempty"`
	URL      string                 `yaml:"url,omitempty" json:"url,omitempty"`
	Index    string                 `yaml:"index,omitempty" json:"index,omitempty"`
	Rotation *RotationConfig        `yaml:"rotation,omitempty" json:"rotation,omitempty"`
	Config   map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// RotationConfig represents log rotation configuration
type RotationConfig struct {
	MaxSize  string `yaml:"max_size" json:"max_size"`
	MaxFiles int    `yaml:"max_files" json:"max_files"`
}

// NewLogger creates a new logger module
func NewLogger(logger *zap.SugaredLogger) *Logger {
	return &Logger{
		name:        "logger",
		version:     "1.0.0",
		description: "Structured request/response logger with multiple destinations",
		author:      "Leash Security",
		logger:      logger,
		status: &interfaces.ModuleStatus{
			State:             interfaces.ModuleStateReady,
			RequestsProcessed: 0,
			ErrorCount:        0,
		},
	}
}

// Metadata methods
func (l *Logger) Name() string                    { return l.name }
func (l *Logger) Version() string                 { return l.version }
func (l *Logger) Type() interfaces.ModuleType     { return interfaces.ModuleTypeSink }
func (l *Logger) Description() string             { return l.description }
func (l *Logger) Author() string                  { return l.author }
func (l *Logger) Dependencies() []string          { return []string{} }

// Lifecycle methods
func (l *Logger) Initialize(ctx context.Context, config *interfaces.ModuleConfig) error {
	l.logger.Infof("Initializing logger module")

	// Parse configuration
	loggerConfig := &LoggerConfig{
		LogRequests:  true,
		LogResponses: false, // Default to false for PII safety
		RedactPII:    true,
		Destinations: []LogDestination{
			{
				Type:   "stdout",
				Format: "json",
			},
		},
	}

	// Override with provided config
	if config != nil && config.Config != nil {
		if destinations, ok := config.Config["destinations"].([]interface{}); ok {
			loggerConfig.Destinations = make([]LogDestination, 0, len(destinations))
			for _, dest := range destinations {
				if destMap, ok := dest.(map[string]interface{}); ok {
					destination := LogDestination{}
					if destType, ok := destMap["type"].(string); ok {
						destination.Type = destType
					}
					if format, ok := destMap["format"].(string); ok {
						destination.Format = format
					}
					if path, ok := destMap["path"].(string); ok {
						destination.Path = path
					}
					loggerConfig.Destinations = append(loggerConfig.Destinations, destination)
				}
			}
		}
		
		if logRequests, ok := config.Config["log_requests"].(bool); ok {
			loggerConfig.LogRequests = logRequests
		}
		if logResponses, ok := config.Config["log_responses"].(bool); ok {
			loggerConfig.LogResponses = logResponses
		}
		if redactPII, ok := config.Config["redact_pii"].(bool); ok {
			loggerConfig.RedactPII = redactPII
		}
	}

	l.config = loggerConfig
	l.startTime = time.Now()
	l.status.State = interfaces.ModuleStateReady

	l.logger.Infof("Logger initialized with %d destinations", len(loggerConfig.Destinations))
	return nil
}

func (l *Logger) Start(ctx context.Context) error {
	l.status.State = interfaces.ModuleStateRunning
	l.status.StartTime = time.Now()
	l.logger.Infof("Logger module started")
	return nil
}

func (l *Logger) Stop(ctx context.Context) error {
	l.status.State = interfaces.ModuleStateDraining
	l.logger.Infof("Logger module stopping")
	return nil
}

func (l *Logger) Shutdown(ctx context.Context) error {
	l.status.State = interfaces.ModuleStateStopped
	l.logger.Infof("Logger module shutdown")
	return nil
}

// Health and status methods
func (l *Logger) Health(ctx context.Context) (*interfaces.HealthStatus, error) {
	return &interfaces.HealthStatus{
		Status:        interfaces.HealthStateHealthy,
		Message:       "Logger is healthy",
		LastCheck:     time.Now(),
		CheckDuration: time.Millisecond,
		Details: map[string]interface{}{
			"destinations":    len(l.config.Destinations),
			"requests_logged": l.status.RequestsProcessed,
		},
	}, nil
}

func (l *Logger) Status() *interfaces.ModuleStatus {
	status := *l.status
	status.LastActivity = time.Now()
	return &status
}

func (l *Logger) Metrics() map[string]interface{} {
	return map[string]interface{}{
		"requests_processed": l.status.RequestsProcessed,
		"errors":            l.status.ErrorCount,
		"destinations":      len(l.config.Destinations),
		"uptime_seconds":    time.Since(l.startTime).Seconds(),
	}
}

// Processing methods
func (l *Logger) ProcessRequest(ctx context.Context, req *interfaces.ProcessRequestContext) (*interfaces.ProcessRequestResult, error) {
	start := time.Now()
	
	if !l.config.LogRequests {
		return &interfaces.ProcessRequestResult{
			Action:         interfaces.ActionContinue,
			ProcessingTime: time.Since(start),
		}, nil
	}

	// Create log entry
	logEntry := map[string]interface{}{
		"timestamp":   req.Timestamp,
		"request_id":  req.RequestID,
		"tenant_id":   req.TenantID,
		"provider":    req.Provider,
		"model":       req.Model,
		"method":      req.Method,
		"path":        req.Path,
		"user_agent":  req.UserAgent,
		"client_ip":   req.ClientIP,
		"body_size":   len(req.Body),
		"type":        "request",
	}

	// Add headers (excluding sensitive ones)
	if headers := l.filterHeaders(req.Headers); len(headers) > 0 {
		logEntry["headers"] = headers
	}

	// Add annotations
	if len(req.Annotations) > 0 {
		logEntry["annotations"] = req.Annotations
	}

	// Log to all destinations
	l.logToDestinations(logEntry)

	l.status.RequestsProcessed++
	l.status.LastActivity = time.Now()

	return &interfaces.ProcessRequestResult{
		Action:         interfaces.ActionContinue,
		ProcessingTime: time.Since(start),
		Annotations: map[string]interface{}{
			"logged": true,
		},
	}, nil
}

func (l *Logger) ProcessResponse(ctx context.Context, resp *interfaces.ProcessResponseContext) (*interfaces.ProcessResponseResult, error) {
	start := time.Now()

	if !l.config.LogResponses {
		return &interfaces.ProcessResponseResult{
			Action:         interfaces.ActionContinue,
			ProcessingTime: time.Since(start),
		}, nil
	}

	// Create log entry
	logEntry := map[string]interface{}{
		"timestamp":        time.Now(),
		"request_id":       resp.RequestID,
		"tenant_id":        resp.TenantID,
		"provider":         resp.Provider,
		"model":            resp.Model,
		"status_code":      resp.StatusCode,
		"response_size":    len(resp.ResponseBody),
		"provider_latency": resp.ProviderLatency.Milliseconds(),
		"total_latency":    resp.TotalLatency.Milliseconds(),
		"type":             "response",
	}

	// Add token usage if available
	if resp.TokensUsed != nil {
		logEntry["tokens"] = map[string]interface{}{
			"prompt":     resp.TokensUsed.PromptTokens,
			"completion": resp.TokensUsed.CompletionTokens,
			"total":      resp.TokensUsed.TotalTokens,
		}
	}

	// Add cost if available
	if resp.CostUSD > 0 {
		logEntry["cost_usd"] = resp.CostUSD
	}

	// Add annotations
	if len(resp.Annotations) > 0 {
		logEntry["annotations"] = resp.Annotations
	}

	// Log to all destinations
	l.logToDestinations(logEntry)

	return &interfaces.ProcessResponseResult{
		Action:         interfaces.ActionContinue,
		ProcessingTime: time.Since(start),
		Annotations: map[string]interface{}{
			"response_logged": true,
		},
	}, nil
}

// Configuration methods
func (l *Logger) ValidateConfig(config *interfaces.ModuleConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	return nil // Logger is very permissive with configuration
}

func (l *Logger) UpdateConfig(ctx context.Context, config *interfaces.ModuleConfig) error {
	if err := l.ValidateConfig(config); err != nil {
		return err
	}

	return l.Initialize(ctx, config)
}

func (l *Logger) GetConfig() *interfaces.ModuleConfig {
	return &interfaces.ModuleConfig{
		Name:     l.name,
		Type:     l.Type().String(),
		Enabled:  l.status.State == interfaces.ModuleStateRunning,
		Priority: 1000, // Low priority for logging (run last)
		Config: map[string]interface{}{
			"destinations":  l.config.Destinations,
			"log_requests":  l.config.LogRequests,
			"log_responses": l.config.LogResponses,
			"redact_pii":    l.config.RedactPII,
		},
	}
}

// logToDestinations logs to all configured destinations
func (l *Logger) logToDestinations(entry map[string]interface{}) {
	for _, dest := range l.config.Destinations {
		switch dest.Type {
		case "stdout":
			l.logToStdout(entry, dest.Format)
		case "file":
			l.logToFile(entry, dest.Path, dest.Format)
		case "elasticsearch":
			// TODO: Implement Elasticsearch logging
			l.logger.Debugf("Elasticsearch logging not yet implemented")
		default:
			l.logger.Warnf("Unknown log destination type: %s", dest.Type)
		}
	}
}

// logToStdout logs to stdout
func (l *Logger) logToStdout(entry map[string]interface{}, format string) {
	switch format {
	case "json":
		if jsonBytes, err := json.Marshal(entry); err == nil {
			fmt.Fprintln(os.Stdout, string(jsonBytes))
		}
	case "text":
		fmt.Fprintf(os.Stdout, "[%s] %s %s %s %s - %v\n",
			entry["timestamp"],
			entry["request_id"],
			entry["tenant_id"],
			entry["provider"],
			entry["method"],
			entry["path"])
	}
}

// logToFile logs to a file
func (l *Logger) logToFile(entry map[string]interface{}, path, format string) {
	// TODO: Implement file logging with rotation
	l.logger.Debugf("File logging to %s not yet implemented", path)
}

// filterHeaders removes sensitive headers from logging
func (l *Logger) filterHeaders(headers map[string]string) map[string]string {
	filtered := make(map[string]string)
	
	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"x-api-key":     true,
		"cookie":        true,
		"set-cookie":    true,
	}

	for key, value := range headers {
		if sensitiveHeaders[key] {
			filtered[key] = "[REDACTED]"
		} else {
			filtered[key] = value
		}
	}

	return filtered
}
