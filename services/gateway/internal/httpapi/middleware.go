package httpapi

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// statusRecorder wraps http.ResponseWriter to capture the status code for logging —
// http.ResponseWriter itself exposes no way to read back what was written.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// WithObservability wraps h with panic recovery and structured request logging, mirroring
// the gRPC server's recoveryInterceptor/loggingInterceptor pair (cmd/indexer/main.go).
func WithObservability(h http.Handler, log *slog.Logger) http.Handler {
	return recoverMiddleware(logMiddleware(h, log), log)
}

func logMiddleware(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		log.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func recoverMiddleware(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic recovered", "method", r.Method, "path", r.URL.Path, "panic", rec, "stack", string(debug.Stack()))
				writeJSONError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
