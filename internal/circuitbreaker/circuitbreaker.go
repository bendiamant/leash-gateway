package circuitbreaker

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name             string
	maxFailures      int
	minRequests      int
	resetTimeout     time.Duration
	state            State
	failures         int
	requests         int
	lastFailureTime  time.Time
	lastSuccessTime  time.Time
	mu               sync.RWMutex
	onStateChange    func(name string, from State, to State)
}

// State represents circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config represents circuit breaker configuration
type Config struct {
	Name             string
	MaxFailures      int
	MinRequests      int
	ResetTimeout     time.Duration
	OnStateChange    func(name string, from State, to State)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		name:          config.Name,
		maxFailures:   config.MaxFailures,
		minRequests:   config.MinRequests,
		resetTimeout:  config.ResetTimeout,
		state:         StateClosed,
		onStateChange: config.OnStateChange,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	if !cb.allowRequest() {
		return fmt.Errorf("circuit breaker %s is open", cb.name)
	}

	err := fn()
	cb.recordResult(err)
	return err
}

// allowRequest determines if a request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.setState(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult records the result of a request
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.requests++

	if err != nil {
		cb.failures++
		cb.lastFailureTime = time.Now()

		// Check if we should open the circuit
		if cb.requests >= cb.minRequests {
			failureRate := float64(cb.failures) / float64(cb.requests)
			if failureRate >= float64(cb.maxFailures)/100.0 {
				cb.setState(StateOpen)
			}
		}
	} else {
		cb.lastSuccessTime = time.Now()

		// Check if we should close the circuit (from half-open)
		if cb.state == StateHalfOpen {
			cb.setState(StateClosed)
			cb.reset()
		}
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState State) {
	if cb.state != newState {
		oldState := cb.state
		cb.state = newState
		
		if cb.onStateChange != nil {
			go cb.onStateChange(cb.name, oldState, newState)
		}
	}
}

// reset resets the circuit breaker counters
func (cb *CircuitBreaker) reset() {
	cb.failures = 0
	cb.requests = 0
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var failureRate float64
	if cb.requests > 0 {
		failureRate = float64(cb.failures) / float64(cb.requests)
	}

	return Stats{
		Name:            cb.name,
		State:           cb.state,
		Failures:        cb.failures,
		Requests:        cb.requests,
		FailureRate:     failureRate,
		LastFailureTime: cb.lastFailureTime,
		LastSuccessTime: cb.lastSuccessTime,
	}
}

// Stats represents circuit breaker statistics
type Stats struct {
	Name            string    `json:"name"`
	State           State     `json:"state"`
	Failures        int       `json:"failures"`
	Requests        int       `json:"requests"`
	FailureRate     float64   `json:"failure_rate"`
	LastFailureTime time.Time `json:"last_failure_time"`
	LastSuccessTime time.Time `json:"last_success_time"`
}

// Manager manages multiple circuit breakers
type Manager struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

// NewManager creates a new circuit breaker manager
func NewManager() *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (m *Manager) GetOrCreate(name string, config Config) *CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if breaker, exists := m.breakers[name]; exists {
		return breaker
	}

	config.Name = name
	breaker := NewCircuitBreaker(config)
	m.breakers[name] = breaker
	return breaker
}

// Get gets a circuit breaker by name
func (m *Manager) Get(name string) (*CircuitBreaker, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	breaker, exists := m.breakers[name]
	if !exists {
		return nil, fmt.Errorf("circuit breaker %s not found", name)
	}

	return breaker, nil
}

// List returns all circuit breakers
func (m *Manager) List() []*CircuitBreaker {
	m.mu.RLock()
	defer m.mu.RUnlock()

	breakers := make([]*CircuitBreaker, 0, len(m.breakers))
	for _, breaker := range m.breakers {
		breakers = append(breakers, breaker)
	}

	return breakers
}

// GetStats returns statistics for all circuit breakers
func (m *Manager) GetStats() []Stats {
	breakers := m.List()
	stats := make([]Stats, len(breakers))
	
	for i, breaker := range breakers {
		stats[i] = breaker.GetStats()
	}

	return stats
}

// Remove removes a circuit breaker
func (m *Manager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.breakers[name]; !exists {
		return fmt.Errorf("circuit breaker %s not found", name)
	}

	delete(m.breakers, name)
	return nil
}
