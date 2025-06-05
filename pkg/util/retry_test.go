package util

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	// Test successful operation
	t.Run("success", func(t *testing.T) {
		count := 0
		err := Retry(context.Background(), DefaultRetryConfig(), func() error {
			count++
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if count != 1 {
			t.Errorf("Expected operation to run once, ran %d times", count)
		}
	})

	// Test retry with eventual success
	t.Run("eventual success", func(t *testing.T) {
		count := 0
		config := DefaultRetryConfig()
		config.InitialBackoff = 1 * time.Millisecond              // Speed up test
		config.RetryableErrors = func(error) bool { return true } // All errors retryable for testing

		err := Retry(context.Background(), config, func() error {
			count++
			if count < 3 {
				return errors.New("temporary error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if count != 3 {
			t.Errorf("Expected operation to run 3 times, ran %d times", count)
		}
	})

	// Test maximum retries exceeded
	t.Run("max retries exceeded", func(t *testing.T) {
		count := 0
		config := DefaultRetryConfig()
		config.MaxAttempts = 3
		config.InitialBackoff = 1 * time.Millisecond              // Speed up test
		config.RetryableErrors = func(error) bool { return true } // All errors retryable for testing

		err := Retry(context.Background(), config, func() error {
			count++
			return errors.New("persistent error")
		})

		if err == nil {
			t.Error("Expected error, got nil")
		}

		if count != 3 {
			t.Errorf("Expected operation to run 3 times, ran %d times", count)
		}
	})

	// Test non-retryable error
	t.Run("non-retryable error", func(t *testing.T) {
		count := 0
		config := DefaultRetryConfig()
		config.RetryableErrors = func(err error) bool {
			return false // All errors are non-retryable
		}

		err := Retry(context.Background(), config, func() error {
			count++
			return errors.New("non-retryable error")
		})

		if err == nil {
			t.Error("Expected error, got nil")
		}

		if count != 1 {
			t.Errorf("Expected operation to run once, ran %d times", count)
		}
	})

	// Test context cancellation
	t.Run("context cancellation", func(t *testing.T) {
		count := 0
		config := DefaultRetryConfig()
		config.InitialBackoff = 100 * time.Millisecond

		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := Retry(ctx, config, func() error {
			count++
			return errors.New("error that would be retried")
		})

		if err == nil {
			t.Error("Expected error due to cancellation, got nil")
		}

		if count > 2 {
			t.Errorf("Expected at most 2 attempts before cancellation, got %d", count)
		}
	})
}

func TestRetryWithResult(t *testing.T) {
	// Test successful operation
	t.Run("success", func(t *testing.T) {
		count := 0
		result, err := RetryWithResult(context.Background(), DefaultRetryConfig(), func() (string, error) {
			count++
			return "success", nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != "success" {
			t.Errorf("Expected result 'success', got '%s'", result)
		}

		if count != 1 {
			t.Errorf("Expected operation to run once, ran %d times", count)
		}
	})

	// Test retry with eventual success
	t.Run("eventual success", func(t *testing.T) {
		count := 0
		config := DefaultRetryConfig()
		config.InitialBackoff = 1 * time.Millisecond              // Speed up test
		config.RetryableErrors = func(error) bool { return true } // All errors retryable for testing

		result, err := RetryWithResult(context.Background(), config, func() (int, error) {
			count++
			if count < 3 {
				return 0, errors.New("temporary error")
			}
			return 42, nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != 42 {
			t.Errorf("Expected result 42, got %d", result)
		}

		if count != 3 {
			t.Errorf("Expected operation to run 3 times, ran %d times", count)
		}
	})
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"connection refused", errors.New("dial tcp 127.0.0.1:8080: connection refused"), true},
		{"timeout", errors.New("context deadline exceeded: i/o timeout"), true},
		{"non-network error", errors.New("file not found"), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsNetworkError(test.err)
			if result != test.expected {
				t.Errorf("IsNetworkError(%v) = %v, expected %v", test.err, result, test.expected)
			}
		})
	}
}
