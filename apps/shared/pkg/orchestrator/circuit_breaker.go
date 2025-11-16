package orchestrator

import (
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for Docker API calls
type CircuitBreaker struct {
	state                CircuitState
	failureCount         int
	halfOpenSuccessCount int
	lastFailureTime      time.Time

	failureThreshold int
	cooldownPeriod   time.Duration
	halfOpenMaxCalls int

	mu sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, cooldownPeriod time.Duration, halfOpenMaxCalls int) *CircuitBreaker {
	return &CircuitBreaker{
		state:            CircuitClosed,
		failureThreshold: failureThreshold,
		cooldownPeriod:   cooldownPeriod,
		halfOpenMaxCalls: halfOpenMaxCalls,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()

	// Check if circuit should transition from open to half-open
	if cb.state == CircuitOpen {
		if time.Since(cb.lastFailureTime) >= cb.cooldownPeriod {
			cb.state = CircuitHalfOpen
			cb.halfOpenSuccessCount = 0
			cb.failureCount = 0
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}

	cb.mu.Unlock()

	// Execute the function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.state == CircuitHalfOpen || cb.failureCount >= cb.failureThreshold {
			cb.state = CircuitOpen
			cb.halfOpenSuccessCount = 0
		}
		return err
	}

	// Success
	if cb.state == CircuitHalfOpen {
		cb.halfOpenSuccessCount++
		if cb.halfOpenSuccessCount >= cb.halfOpenMaxCalls {
			cb.state = CircuitClosed
			cb.failureCount = 0
			cb.halfOpenSuccessCount = 0
		}
	} else {
		cb.failureCount = 0 // Reset on success
	}

	return nil
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailureCount returns the current failure count
func (cb *CircuitBreaker) GetFailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount
}

var ErrCircuitOpen = &CircuitOpenError{}

type CircuitOpenError struct{}

func (e *CircuitOpenError) Error() string {
	return "circuit breaker is open"
}
