package contentfilter

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bendiamant/leash-gateway/internal/modules/interface"
	"go.uber.org/zap"
)

// ContentFilter implements a content filtering module
type ContentFilter struct {
	name        string
	version     string
	description string
	author      string
	config      *ContentFilterConfig
	patterns    []*regexp.Regexp
	logger      *zap.SugaredLogger
	status      *interfaces.ModuleStatus
	startTime   time.Time
}

// ContentFilterConfig represents content filter configuration
type ContentFilterConfig struct {
	BlockedKeywords    []string  `yaml:"blocked_keywords" json:"blocked_keywords"`
	BlockedPatterns    []string  `yaml:"blocked_patterns" json:"blocked_patterns"`
	SeverityThreshold  float64   `yaml:"severity_threshold" json:"severity_threshold"`
	Action             string    `yaml:"action" json:"action"` // block, warn, annotate, redact
	CaseSensitive      bool      `yaml:"case_sensitive" json:"case_sensitive"`
	CheckRequests      bool      `yaml:"check_requests" json:"check_requests"`
	CheckResponses     bool      `yaml:"check_responses" json:"check_responses"`
	RedactionText      string    `yaml:"redaction_text" json:"redaction_text"`
}

// DetectionResult represents content detection result
type DetectionResult struct {
	Detected   bool     `json:"detected"`
	Matches    []string `json:"matches"`
	Confidence float64  `json:"confidence"`
	Action     string   `json:"action"`
	Message    string   `json:"message"`
}

// NewContentFilter creates a new content filter module
func NewContentFilter(logger *zap.SugaredLogger) *ContentFilter {
	return &ContentFilter{
		name:        "content-filter",
		version:     "1.0.0",
		description: "Content filtering module for detecting and blocking inappropriate content",
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
func (cf *ContentFilter) Name() string                    { return cf.name }
func (cf *ContentFilter) Version() string                 { return cf.version }
func (cf *ContentFilter) Type() interfaces.ModuleType     { return interfaces.ModuleTypePolicy }
func (cf *ContentFilter) Description() string             { return cf.description }
func (cf *ContentFilter) Author() string                  { return cf.author }
func (cf *ContentFilter) Dependencies() []string          { return []string{} }

// Lifecycle methods
func (cf *ContentFilter) Initialize(ctx context.Context, config *interfaces.ModuleConfig) error {
	cf.logger.Infof("Initializing content filter module")

	// Parse configuration
	filterConfig := &ContentFilterConfig{
		BlockedKeywords:   []string{"inappropriate", "harmful"},
		SeverityThreshold: 0.8,
		Action:            "block",
		CaseSensitive:     false,
		CheckRequests:     true,
		CheckResponses:    true,
		RedactionText:     "[FILTERED]",
	}

	// Override with provided config
	if config != nil && config.Config != nil {
		if keywords, ok := config.Config["blocked_keywords"].([]interface{}); ok {
			filterConfig.BlockedKeywords = make([]string, len(keywords))
			for i, keyword := range keywords {
				if str, ok := keyword.(string); ok {
					filterConfig.BlockedKeywords[i] = str
				}
			}
		}
		
		if patterns, ok := config.Config["blocked_patterns"].([]interface{}); ok {
			filterConfig.BlockedPatterns = make([]string, len(patterns))
			for i, pattern := range patterns {
				if str, ok := pattern.(string); ok {
					filterConfig.BlockedPatterns[i] = str
				}
			}
		}

		if threshold, ok := config.Config["severity_threshold"].(float64); ok {
			filterConfig.SeverityThreshold = threshold
		}
		if action, ok := config.Config["action"].(string); ok {
			filterConfig.Action = action
		}
		if caseSensitive, ok := config.Config["case_sensitive"].(bool); ok {
			filterConfig.CaseSensitive = caseSensitive
		}
		if checkRequests, ok := config.Config["check_requests"].(bool); ok {
			filterConfig.CheckRequests = checkRequests
		}
		if checkResponses, ok := config.Config["check_responses"].(bool); ok {
			filterConfig.CheckResponses = checkResponses
		}
		if redactionText, ok := config.Config["redaction_text"].(string); ok {
			filterConfig.RedactionText = redactionText
		}
	}

	// Compile regex patterns
	cf.patterns = make([]*regexp.Regexp, len(filterConfig.BlockedPatterns))
	for i, pattern := range filterConfig.BlockedPatterns {
		flags := 0
		if !filterConfig.CaseSensitive {
			flags = regexp.IgnoreCase
		}
		
		regex, err := regexp.Compile(fmt.Sprintf("(?%s)%s", "", pattern))
		if err != nil {
			return fmt.Errorf("invalid regex pattern %s: %w", pattern, err)
		}
		cf.patterns[i] = regex
	}

	cf.config = filterConfig
	cf.startTime = time.Now()
	cf.status.State = interfaces.ModuleStateReady

	cf.logger.Infof("Content filter initialized with %d keywords, %d patterns, action=%s", 
		len(filterConfig.BlockedKeywords), len(filterConfig.BlockedPatterns), filterConfig.Action)

	return nil
}

func (cf *ContentFilter) Start(ctx context.Context) error {
	cf.status.State = interfaces.ModuleStateRunning
	cf.status.StartTime = time.Now()
	cf.logger.Infof("Content filter module started")
	return nil
}

func (cf *ContentFilter) Stop(ctx context.Context) error {
	cf.status.State = interfaces.ModuleStateDraining
	cf.logger.Infof("Content filter module stopping")
	return nil
}

func (cf *ContentFilter) Shutdown(ctx context.Context) error {
	cf.status.State = interfaces.ModuleStateStopped
	cf.logger.Infof("Content filter module shutdown")
	return nil
}

// Health and status methods
func (cf *ContentFilter) Health(ctx context.Context) (*interfaces.HealthStatus, error) {
	return &interfaces.HealthStatus{
		Status:        interfaces.HealthStateHealthy,
		Message:       "Content filter is healthy",
		LastCheck:     time.Now(),
		CheckDuration: time.Millisecond,
		Details: map[string]interface{}{
			"blocked_keywords": len(cf.config.BlockedKeywords),
			"blocked_patterns": len(cf.patterns),
			"action":           cf.config.Action,
		},
	}, nil
}

func (cf *ContentFilter) Status() *interfaces.ModuleStatus {
	status := *cf.status
	status.LastActivity = time.Now()
	return &status
}

func (cf *ContentFilter) Metrics() map[string]interface{} {
	return map[string]interface{}{
		"requests_processed": cf.status.RequestsProcessed,
		"errors":            cf.status.ErrorCount,
		"blocked_keywords":  len(cf.config.BlockedKeywords),
		"blocked_patterns":  len(cf.patterns),
		"uptime_seconds":    time.Since(cf.startTime).Seconds(),
	}
}

// Processing methods
func (cf *ContentFilter) ProcessRequest(ctx context.Context, req *interfaces.ProcessRequestContext) (*interfaces.ProcessRequestResult, error) {
	start := time.Now()
	
	if !cf.config.CheckRequests {
		return &interfaces.ProcessRequestResult{
			Action:         interfaces.ActionContinue,
			ProcessingTime: time.Since(start),
		}, nil
	}

	// Extract content from request body
	content, err := cf.extractContentFromRequest(req.Body)
	if err != nil {
		cf.logger.Warnf("Failed to extract content from request: %v", err)
		return &interfaces.ProcessRequestResult{
			Action:         interfaces.ActionContinue,
			ProcessingTime: time.Since(start),
		}, nil
	}

	// Check content
	result := cf.checkContent(content)
	cf.status.RequestsProcessed++
	cf.status.LastActivity = time.Now()

	if result.Detected && result.Confidence >= cf.config.SeverityThreshold {
		switch cf.config.Action {
		case "block":
			cf.logger.Warnf("Blocking request %s due to content violation: %s", req.RequestID, result.Message)
			return &interfaces.ProcessRequestResult{
				Action:      interfaces.ActionBlock,
				BlockReason: fmt.Sprintf("Content violation: %s", result.Message),
				ProcessingTime: time.Since(start),
				Annotations: map[string]interface{}{
					"content_filter_detected": true,
					"matches":                 result.Matches,
					"confidence":              result.Confidence,
					"action":                  "block",
				},
			}, nil
		case "redact":
			// Redact content and continue
			redactedBody := cf.redactContent(req.Body, result.Matches)
			return &interfaces.ProcessRequestResult{
				Action:       interfaces.ActionTransform,
				ModifiedBody: redactedBody,
				ProcessingTime: time.Since(start),
				Annotations: map[string]interface{}{
					"content_filter_redacted": true,
					"matches":                 result.Matches,
					"confidence":              result.Confidence,
				},
			}, nil
		default: // warn, annotate
			cf.logger.Warnf("Content warning for request %s: %s", req.RequestID, result.Message)
		}
	}

	return &interfaces.ProcessRequestResult{
		Action:         interfaces.ActionContinue,
		ProcessingTime: time.Since(start),
		Annotations: map[string]interface{}{
			"content_filter_checked": true,
			"content_safe":           !result.Detected,
		},
	}, nil
}

func (cf *ContentFilter) ProcessResponse(ctx context.Context, resp *interfaces.ProcessResponseContext) (*interfaces.ProcessResponseResult, error) {
	start := time.Now()

	if !cf.config.CheckResponses {
		return &interfaces.ProcessResponseResult{
			Action:         interfaces.ActionContinue,
			ProcessingTime: time.Since(start),
		}, nil
	}

	// Extract content from response
	content, err := cf.extractContentFromResponse(resp.ResponseBody)
	if err != nil {
		cf.logger.Warnf("Failed to extract content from response: %v", err)
		return &interfaces.ProcessResponseResult{
			Action:         interfaces.ActionContinue,
			ProcessingTime: time.Since(start),
		}, nil
	}

	// Check content
	result := cf.checkContent(content)

	if result.Detected && result.Confidence >= cf.config.SeverityThreshold {
		if cf.config.Action == "redact" {
			// Redact response content
			redactedBody := cf.redactContent(resp.ResponseBody, result.Matches)
			return &interfaces.ProcessResponseResult{
				Action:       interfaces.ActionTransform,
				ModifiedBody: redactedBody,
				ProcessingTime: time.Since(start),
				Annotations: map[string]interface{}{
					"response_content_redacted": true,
					"matches":                   result.Matches,
				},
			}, nil
		}
	}

	return &interfaces.ProcessResponseResult{
		Action:         interfaces.ActionContinue,
		ProcessingTime: time.Since(start),
		Annotations: map[string]interface{}{
			"response_content_checked": true,
			"content_safe":             !result.Detected,
		},
	}, nil
}

// Configuration methods
func (cf *ContentFilter) ValidateConfig(config *interfaces.ModuleConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if configMap := config.Config; configMap != nil {
		if action, ok := configMap["action"].(string); ok {
			validActions := map[string]bool{
				"block": true, "warn": true, "annotate": true, "redact": true,
			}
			if !validActions[action] {
				return fmt.Errorf("invalid action: %s", action)
			}
		}

		if threshold, ok := configMap["severity_threshold"].(float64); ok {
			if threshold < 0 || threshold > 1 {
				return fmt.Errorf("severity_threshold must be between 0 and 1, got %f", threshold)
			}
		}
	}

	return nil
}

func (cf *ContentFilter) UpdateConfig(ctx context.Context, config *interfaces.ModuleConfig) error {
	if err := cf.ValidateConfig(config); err != nil {
		return err
	}

	return cf.Initialize(ctx, config)
}

func (cf *ContentFilter) GetConfig() *interfaces.ModuleConfig {
	return &interfaces.ModuleConfig{
		Name:     cf.name,
		Type:     cf.Type().String(),
		Enabled:  cf.status.State == interfaces.ModuleStateRunning,
		Priority: 300, // Medium priority for content filtering
		Config: map[string]interface{}{
			"blocked_keywords":    cf.config.BlockedKeywords,
			"blocked_patterns":    cf.config.BlockedPatterns,
			"severity_threshold":  cf.config.SeverityThreshold,
			"action":              cf.config.Action,
			"case_sensitive":      cf.config.CaseSensitive,
			"check_requests":      cf.config.CheckRequests,
			"check_responses":     cf.config.CheckResponses,
		},
	}
}

// Helper methods
func (cf *ContentFilter) extractContentFromRequest(body []byte) (string, error) {
	if len(body) == 0 {
		return "", nil
	}

	// Try to parse as JSON (LLM request)
	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		// If not JSON, treat as plain text
		return string(body), nil
	}

	// Extract messages content
	var content strings.Builder
	if messages, ok := requestData["messages"].([]interface{}); ok {
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if msgContent, ok := msgMap["content"].(string); ok {
					content.WriteString(msgContent)
					content.WriteString(" ")
				}
			}
		}
	}

	return content.String(), nil
}

func (cf *ContentFilter) extractContentFromResponse(body []byte) (string, error) {
	if len(body) == 0 {
		return "", nil
	}

	// Try to parse as JSON (LLM response)
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return string(body), nil
	}

	// Extract choices content
	var content strings.Builder
	if choices, ok := responseData["choices"].([]interface{}); ok {
		for _, choice := range choices {
			if choiceMap, ok := choice.(map[string]interface{}); ok {
				if message, ok := choiceMap["message"].(map[string]interface{}); ok {
					if msgContent, ok := message["content"].(string); ok {
						content.WriteString(msgContent)
						content.WriteString(" ")
					}
				}
			}
		}
	}

	return content.String(), nil
}

func (cf *ContentFilter) checkContent(content string) *DetectionResult {
	if content == "" {
		return &DetectionResult{
			Detected:   false,
			Confidence: 0,
		}
	}

	var matches []string
	var maxConfidence float64

	// Check against keywords
	checkContent := content
	if !cf.config.CaseSensitive {
		checkContent = strings.ToLower(content)
	}

	for _, keyword := range cf.config.BlockedKeywords {
		checkKeyword := keyword
		if !cf.config.CaseSensitive {
			checkKeyword = strings.ToLower(keyword)
		}

		if strings.Contains(checkContent, checkKeyword) {
			matches = append(matches, keyword)
			maxConfidence = 0.9 // High confidence for exact keyword match
		}
	}

	// Check against regex patterns
	for i, pattern := range cf.patterns {
		if pattern.MatchString(content) {
			matches = append(matches, cf.config.BlockedPatterns[i])
			if maxConfidence < 0.8 {
				maxConfidence = 0.8 // Medium-high confidence for pattern match
			}
		}
	}

	detected := len(matches) > 0
	message := ""
	if detected {
		message = fmt.Sprintf("Detected inappropriate content: %s", strings.Join(matches, ", "))
	}

	return &DetectionResult{
		Detected:   detected,
		Matches:    matches,
		Confidence: maxConfidence,
		Action:     cf.config.Action,
		Message:    message,
	}
}

func (cf *ContentFilter) redactContent(body []byte, matches []string) []byte {
	content := string(body)
	
	// Simple redaction - replace matches with redaction text
	for _, match := range matches {
		if cf.config.CaseSensitive {
			content = strings.ReplaceAll(content, match, cf.config.RedactionText)
		} else {
			// Case-insensitive replacement
			re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(match))
			content = re.ReplaceAllString(content, cf.config.RedactionText)
		}
	}

	return []byte(content)
}
