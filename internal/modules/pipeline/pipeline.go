package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bendiamant/leash-gateway/internal/modules/interface"
	"go.uber.org/zap"
)

// Pipeline manages the execution of modules in the correct order
type Pipeline struct {
	inspectors   []interfaces.Module
	policies     []interfaces.Module
	transformers []interfaces.Module
	sinks        []interfaces.Module
	logger       *zap.SugaredLogger
	mu           sync.RWMutex
}

// NewPipeline creates a new module pipeline
func NewPipeline(logger *zap.SugaredLogger) *Pipeline {
	return &Pipeline{
		inspectors:   make([]interfaces.Module, 0),
		policies:     make([]interfaces.Module, 0),
		transformers: make([]interfaces.Module, 0),
		sinks:        make([]interfaces.Module, 0),
		logger:       logger,
	}
}

// AddModule adds a module to the appropriate pipeline stage
func (p *Pipeline) AddModule(module interfaces.Module) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch module.Type() {
	case interfaces.ModuleTypeInspector:
		p.inspectors = append(p.inspectors, module)
	case interfaces.ModuleTypePolicy:
		p.policies = append(p.policies, module)
	case interfaces.ModuleTypeTransformer:
		p.transformers = append(p.transformers, module)
	case interfaces.ModuleTypeSink:
		p.sinks = append(p.sinks, module)
	default:
		return fmt.Errorf("unknown module type: %s", module.Type().String())
	}

	p.logger.Infof("Added module %s to %s pipeline", module.Name(), module.Type().String())
	return nil
}

// RemoveModule removes a module from the pipeline
func (p *Pipeline) RemoveModule(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove from all pipeline stages
	p.inspectors = p.removeModuleFromSlice(p.inspectors, name)
	p.policies = p.removeModuleFromSlice(p.policies, name)
	p.transformers = p.removeModuleFromSlice(p.transformers, name)
	p.sinks = p.removeModuleFromSlice(p.sinks, name)

	p.logger.Infof("Removed module %s from pipeline", name)
	return nil
}

// removeModuleFromSlice removes a module from a slice by name
func (p *Pipeline) removeModuleFromSlice(modules []interfaces.Module, name string) []interfaces.Module {
	for i, module := range modules {
		if module.Name() == name {
			return append(modules[:i], modules[i+1:]...)
		}
	}
	return modules
}

// ProcessRequest processes a request through the module pipeline
func (p *Pipeline) ProcessRequest(ctx context.Context, req *interfaces.ProcessRequestContext) (*interfaces.ProcessRequestResult, error) {
	start := time.Now()
	
	p.logger.Debugf("Processing request %s through pipeline", req.RequestID)

	// Phase 1: Run inspectors in parallel (fail-open)
	inspectionResults := p.runInspectorsParallel(ctx, req)
	
	// Merge inspection annotations
	if req.Annotations == nil {
		req.Annotations = make(map[string]interface{})
	}
	for _, result := range inspectionResults {
		for key, value := range result.Annotations {
			req.Annotations[key] = value
		}
	}

	// Phase 2: Run policies sequentially (fail-closed)
	p.mu.RLock()
	policies := make([]interfaces.Module, len(p.policies))
	copy(policies, p.policies)
	p.mu.RUnlock()

	for _, policy := range policies {
		if !p.shouldRunModule(policy, req) {
			continue
		}

		result, err := p.runModuleWithTimeout(ctx, policy, req)
		if err != nil {
			p.logger.Errorf("Policy %s failed: %v", policy.Name(), err)
			return &interfaces.ProcessRequestResult{
				Action:      interfaces.ActionBlock,
				BlockReason: fmt.Sprintf("Policy %s failed: %v", policy.Name(), err),
			}, nil
		}

		if result.Action == interfaces.ActionBlock {
			p.logger.Warnf("Request %s blocked by policy %s: %s", 
				req.RequestID, policy.Name(), result.BlockReason)
			return result, nil
		}

		// Merge annotations
		p.mergeAnnotations(req, result.Annotations)
	}

	// Phase 3: Run transformers sequentially
	p.mu.RLock()
	transformers := make([]interfaces.Module, len(p.transformers))
	copy(transformers, p.transformers)
	p.mu.RUnlock()

	for _, transformer := range transformers {
		if !p.shouldRunModule(transformer, req) {
			continue
		}

		result, err := p.runModuleWithTimeout(ctx, transformer, req)
		if err != nil {
			// Log error but continue (non-critical)
			p.logger.Warnf("Transformer %s failed: %v", transformer.Name(), err)
			continue
		}

		if result.Action == interfaces.ActionTransform && len(result.ModifiedBody) > 0 {
			req.Body = result.ModifiedBody
			p.logger.Debugf("Request %s transformed by %s", req.RequestID, transformer.Name())
		}

		// Merge annotations
		p.mergeAnnotations(req, result.Annotations)
	}

	// Phase 4: Run sinks (fire-and-forget)
	go p.runSinksAsync(context.Background(), req)

	processingTime := time.Since(start)
	p.logger.Debugf("Request %s processed through pipeline in %v", req.RequestID, processingTime)

	return &interfaces.ProcessRequestResult{
		Action:         interfaces.ActionContinue,
		ProcessingTime: processingTime,
		Annotations:    req.Annotations,
	}, nil
}

// ProcessResponse processes a response through the module pipeline
func (p *Pipeline) ProcessResponse(ctx context.Context, resp *interfaces.ProcessResponseContext) (*interfaces.ProcessResponseResult, error) {
	start := time.Now()
	
	p.logger.Debugf("Processing response %s through pipeline", resp.RequestID)

	// Run response transformers
	p.mu.RLock()
	transformers := make([]interfaces.Module, len(p.transformers))
	copy(transformers, p.transformers)
	p.mu.RUnlock()

	for _, transformer := range transformers {
		if !p.shouldRunModuleForResponse(transformer, resp) {
			continue
		}

		result, err := p.runResponseModuleWithTimeout(ctx, transformer, resp)
		if err != nil {
			p.logger.Warnf("Response transformer %s failed: %v", transformer.Name(), err)
			continue
		}

		if result.Action == interfaces.ActionTransform && len(result.ModifiedBody) > 0 {
			resp.ResponseBody = result.ModifiedBody
			p.logger.Debugf("Response %s transformed by %s", resp.RequestID, transformer.Name())
		}

		// Merge annotations
		p.mergeAnnotations(resp.ProcessRequestContext, result.Annotations)
	}

	// Run response sinks
	go p.runResponseSinksAsync(context.Background(), resp)

	processingTime := time.Since(start)
	p.logger.Debugf("Response %s processed through pipeline in %v", resp.RequestID, processingTime)

	return &interfaces.ProcessResponseResult{
		Action:         interfaces.ActionContinue,
		ProcessingTime: processingTime,
		Annotations:    resp.Annotations,
	}, nil
}

// runInspectorsParallel runs inspectors in parallel for better performance
func (p *Pipeline) runInspectorsParallel(ctx context.Context, req *interfaces.ProcessRequestContext) []*interfaces.ProcessRequestResult {
	p.mu.RLock()
	inspectors := make([]interfaces.Module, len(p.inspectors))
	copy(inspectors, p.inspectors)
	p.mu.RUnlock()

	results := make([]*interfaces.ProcessRequestResult, 0, len(inspectors))
	resultsChan := make(chan *interfaces.ProcessRequestResult, len(inspectors))
	
	var wg sync.WaitGroup

	for _, inspector := range inspectors {
		if !p.shouldRunModule(inspector, req) {
			continue
		}

		wg.Add(1)
		go func(module interfaces.Module) {
			defer wg.Done()
			
			result, err := p.runModuleWithTimeout(ctx, module, req)
			if err != nil {
				p.logger.Warnf("Inspector %s failed: %v", module.Name(), err)
				return
			}
			
			resultsChan <- result
		}(inspector)
	}

	// Wait for all inspectors to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

// runSinksAsync runs sinks asynchronously
func (p *Pipeline) runSinksAsync(ctx context.Context, req *interfaces.ProcessRequestContext) {
	p.mu.RLock()
	sinks := make([]interfaces.Module, len(p.sinks))
	copy(sinks, p.sinks)
	p.mu.RUnlock()

	for _, sink := range sinks {
		if !p.shouldRunModule(sink, req) {
			continue
		}

		go func(module interfaces.Module) {
			_, err := p.runModuleWithTimeout(ctx, module, req)
			if err != nil {
				p.logger.Warnf("Sink %s failed: %v", module.Name(), err)
			}
		}(sink)
	}
}

// runResponseSinksAsync runs response sinks asynchronously
func (p *Pipeline) runResponseSinksAsync(ctx context.Context, resp *interfaces.ProcessResponseContext) {
	p.mu.RLock()
	sinks := make([]interfaces.Module, len(p.sinks))
	copy(sinks, p.sinks)
	p.mu.RUnlock()

	for _, sink := range sinks {
		if !p.shouldRunModuleForResponse(sink, resp) {
			continue
		}

		go func(module interfaces.Module) {
			_, err := p.runResponseModuleWithTimeout(ctx, module, resp)
			if err != nil {
				p.logger.Warnf("Response sink %s failed: %v", module.Name(), err)
			}
		}(sink)
	}
}

// runModuleWithTimeout runs a module with timeout protection
func (p *Pipeline) runModuleWithTimeout(ctx context.Context, module interfaces.Module, req *interfaces.ProcessRequestContext) (*interfaces.ProcessRequestResult, error) {
	// Create timeout context
	timeout := 2 * time.Second // Default timeout
	if req.ModuleConfig != nil && req.ModuleConfig.Timeouts != nil && req.ModuleConfig.Timeouts.Processing > 0 {
		timeout = req.ModuleConfig.Timeouts.Processing
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Run module in goroutine
	resultChan := make(chan *interfaces.ProcessRequestResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		result, err := module.ProcessRequest(timeoutCtx, req)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("module %s timed out after %v", module.Name(), timeout)
	}
}

// runResponseModuleWithTimeout runs a response module with timeout protection
func (p *Pipeline) runResponseModuleWithTimeout(ctx context.Context, module interfaces.Module, resp *interfaces.ProcessResponseContext) (*interfaces.ProcessResponseResult, error) {
	timeout := 2 * time.Second // Default timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultChan := make(chan *interfaces.ProcessResponseResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		result, err := module.ProcessResponse(timeoutCtx, resp)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("module %s timed out after %v", module.Name(), timeout)
	}
}

// shouldRunModule checks if a module should run based on conditions
func (p *Pipeline) shouldRunModule(module interfaces.Module, req *interfaces.ProcessRequestContext) bool {
	config := module.GetConfig()
	if config == nil || !config.Enabled {
		return false
	}

	// Check conditions
	for _, condition := range config.Conditions {
		if !p.evaluateCondition(condition, req) {
			return false
		}
	}

	return true
}

// shouldRunModuleForResponse checks if a module should run for response processing
func (p *Pipeline) shouldRunModuleForResponse(module interfaces.Module, resp *interfaces.ProcessResponseContext) bool {
	return p.shouldRunModule(module, resp.ProcessRequestContext)
}

// evaluateCondition evaluates a single condition
func (p *Pipeline) evaluateCondition(condition interfaces.Condition, req *interfaces.ProcessRequestContext) bool {
	var fieldValue interface{}

	// Extract field value based on field name
	switch condition.Field {
	case "tenant":
		fieldValue = req.TenantID
	case "provider":
		fieldValue = req.Provider
	case "model":
		fieldValue = req.Model
	case "method":
		fieldValue = req.Method
	case "path":
		fieldValue = req.Path
	default:
		// Check in annotations
		if req.Annotations != nil {
			fieldValue = req.Annotations[condition.Field]
		}
	}

	// Evaluate condition based on operator
	switch condition.Operator {
	case "eq":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", condition.Value)
	case "ne":
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", condition.Value)
	case "in":
		// Value should be a slice
		if valueSlice, ok := condition.Value.([]interface{}); ok {
			for _, v := range valueSlice {
				if fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", v) {
					return true
				}
			}
		}
		return false
	case "not_in":
		// Value should be a slice
		if valueSlice, ok := condition.Value.([]interface{}); ok {
			for _, v := range valueSlice {
				if fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", v) {
					return false
				}
			}
		}
		return true
	default:
		p.logger.Warnf("Unknown condition operator: %s", condition.Operator)
		return true // Default to allow
	}
}

// mergeAnnotations merges annotations from module results
func (p *Pipeline) mergeAnnotations(req *interfaces.ProcessRequestContext, annotations map[string]interface{}) {
	if req.Annotations == nil {
		req.Annotations = make(map[string]interface{})
	}

	for key, value := range annotations {
		req.Annotations[key] = value
	}
}

// GetPipelineStatus returns the current pipeline configuration
func (p *Pipeline) GetPipelineStatus() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"inspectors":   len(p.inspectors),
		"policies":     len(p.policies),
		"transformers": len(p.transformers),
		"sinks":        len(p.sinks),
		"total_modules": len(p.inspectors) + len(p.policies) + len(p.transformers) + len(p.sinks),
	}
}

// ValidatePipeline validates the current pipeline configuration
func (p *Pipeline) ValidatePipeline() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check for at least one module
	totalModules := len(p.inspectors) + len(p.policies) + len(p.transformers) + len(p.sinks)
	if totalModules == 0 {
		return fmt.Errorf("pipeline has no modules configured")
	}

	// Validate each module
	allModules := append(append(append(p.inspectors, p.policies...), p.transformers...), p.sinks...)
	for _, module := range allModules {
		config := module.GetConfig()
		if err := module.ValidateConfig(config); err != nil {
			return fmt.Errorf("module %s config validation failed: %w", module.Name(), err)
		}
	}

	return nil
}
