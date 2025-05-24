package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// parseSizeString converts a size string (e.g., "1.5MB", "1024KB") to bytes.
func parseSizeString(sizeStr string) (int, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	sizeStr = strings.ToUpper(sizeStr)

	var multiplier int
	var suffix string

	if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		suffix = "GB"
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		suffix = "MB"
	} else if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		suffix = "KB"
	} else if strings.HasSuffix(sizeStr, "G") {
		multiplier = 1024 * 1024 * 1024
		suffix = "G"
	} else if strings.HasSuffix(sizeStr, "M") {
		multiplier = 1024 * 1024
		suffix = "M"
	} else if strings.HasSuffix(sizeStr, "K") {
		multiplier = 1024
		suffix = "K"
	} else {
		// Assume bytes if no suffix
		val, err := strconv.Atoi(sizeStr)
		if err != nil {
			return 0, fmt.Errorf("invalid size string format: %s. Expected number or number with K/KB/M/MB/G/GB suffix", sizeStr)
		}
		return val, nil
	}

	// Remove suffix
	numStr := strings.TrimSuffix(sizeStr, suffix)

	// Parse the number part
	valFloat, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in size string: %s", numStr)
	}

	if valFloat < 0 {
		return 0, fmt.Errorf("size cannot be negative: %s", numStr)
	}

	return int(valFloat * float64(multiplier)), nil
}

// logRequestMiddleware logs information about each HTTP request in a format similar to Nginx
func logRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture the status code
		rwWrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default to 200 OK
		}

		// Process the request with the wrapped response writer
		next.ServeHTTP(rwWrapper, r)

		// Calculate duration
		duration := time.Since(start)

		// Log the request details
		slog.Info("HTTP Request",
			"remote_addr", r.RemoteAddr,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rwWrapper.statusCode,
			"duration", duration,
			"user_agent", r.UserAgent(),
		)
	})
}

// responseWriterWrapper captures the status code of the response
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before passing it to the wrapped ResponseWriter
func (rww *responseWriterWrapper) WriteHeader(code int) {
	rww.statusCode = code
	rww.ResponseWriter.WriteHeader(code)
}
