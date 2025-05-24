package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// LogRequestMiddleware logs information about each HTTP request in a format similar to Nginx
func LogRequestMiddleware(next http.Handler) http.Handler {
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
