package util

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"time"
)

// RetryConfig holds the configuration for retry operations
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffFactor is the factor by which the backoff increases
	BackoffFactor float64
	// RetryableErrors is a function that determines if an error is retryable
	RetryableErrors func(error) bool
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:     250,
		InitialBackoff:  500 * time.Millisecond,
		MaxBackoff:      60 * time.Second,
		BackoffFactor:   1.5,
		RetryableErrors: IsNetworkError,
	}
}

// Retry executes the given function with exponential backoff retry
func Retry(ctx context.Context, config RetryConfig, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check if context is canceled before attempting
		if ctx.Err() != nil {
			return fmt.Errorf("context canceled during retry: %w", ctx.Err())
		}

		// Execute the operation
		err := operation()
		if err == nil {
			// Success, return nil
			return nil
		}

		// Check if error is retryable
		if !config.RetryableErrors(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Save the last error
		lastErr = err

		// If this is the last attempt, don't sleep
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate backoff duration with exponential increase and some jitter
		backoff := float64(config.InitialBackoff) * math.Pow(config.BackoffFactor, float64(attempt))

		// Add small random jitter (±10%)
		jitterFactor := 0.9 + 0.2*rand.Float64()
		backoff = backoff * jitterFactor

		// Cap at max backoff
		if backoff > float64(config.MaxBackoff) {
			backoff = float64(config.MaxBackoff)
		}

		backoffDuration := time.Duration(backoff)

		slog.Debug("Operation failed, retrying", "attempt", attempt+1, "max_attempts", config.MaxAttempts, "backoff", backoffDuration, "error", err)

		// Wait for backoff duration or until context is canceled
		timer := time.NewTimer(backoffDuration)
		select {
		case <-timer.C:
			// Timer expired, continue with next attempt
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("context canceled during retry backoff: %w", ctx.Err())
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// RetryWithResult executes the given function with exponential backoff retry and returns the result
func RetryWithResult[T any](ctx context.Context, config RetryConfig, operation func() (T, error)) (T, error) {
	var lastErr error
	var zeroVal T

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check if context is canceled before attempting
		if ctx.Err() != nil {
			return zeroVal, fmt.Errorf("context canceled during retry: %w", ctx.Err())
		}

		// Execute the operation
		result, err := operation()
		if err == nil {
			// Success, return result
			return result, nil
		}

		// Check if error is retryable
		if !config.RetryableErrors(err) {
			return zeroVal, fmt.Errorf("non-retryable error: %w", err)
		}

		// Save the last error
		lastErr = err

		// If this is the last attempt, don't sleep
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate backoff duration with exponential increase
		backoff := float64(config.InitialBackoff) * math.Pow(config.BackoffFactor, float64(attempt))

		// Add small random jitter (±10%)
		jitterFactor := 0.9 + 0.2*rand.Float64()
		backoff = backoff * jitterFactor

		// Cap at max backoff
		if backoff > float64(config.MaxBackoff) {
			backoff = float64(config.MaxBackoff)
		}

		backoffDuration := time.Duration(backoff)

		slog.Debug("Operation failed, retrying", "attempt", attempt+1, "max_attempts", config.MaxAttempts, "backoff", backoffDuration, "error", err)

		// Wait for backoff duration or until context is canceled
		timer := time.NewTimer(backoffDuration)
		select {
		case <-timer.C:
			// Timer expired, continue with next attempt
		case <-ctx.Done():
			timer.Stop()
			return zeroVal, fmt.Errorf("context canceled during retry backoff: %w", ctx.Err())
		}
	}

	return zeroVal, fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// Common retryable error predicates

// IsNetworkError returns true if the error appears to be a network error
func IsNetworkError(err error) bool {
	// Common network error strings - this can be extended
	networkErrors := []string{
		"connection refused",
		"connection reset",
		"connection closed",
		"no such host",
		"timeout",
		"too many open files",
		"network is unreachable",
		"i/o timeout",
		"temporary failure",
		"broken pipe",
	}

	errStr := err.Error()
	for _, netErr := range networkErrors {
		if strings.Contains(strings.ToLower(errStr), netErr) {
			return true
		}
	}

	return false
}

// IsTemporaryError returns true if the error appears to be temporary
func IsTemporaryError(err error) bool {
	var tempError interface {
		Temporary() bool
	}

	if errors.As(err, &tempError) {
		return tempError.Temporary()
	}

	return false
}

// IsRetryableError combines common retry conditions
func IsRetryableError(err error) bool {
	return IsNetworkError(err) || IsTemporaryError(err)
}
