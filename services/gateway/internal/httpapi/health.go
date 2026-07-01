// Package httpapi is the gateway's inbound adapter: translates REST/SSE requests into
// gRPC calls against the indexer's CertificateQuery service. It holds zero business logic —
// the indexer (and ultimately the chain) is the sole source of truth.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"
)

// Healthz is the liveness probe — 200 iff the process is up, no dependency checks.
func Healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Readyz is the readiness probe — 200 iff the indexer's gRPC health service reports SERVING,
// 503 otherwise (unreachable or degraded). Load balancers should target this, not /healthz.
func Readyz(health grpc_health_v1.HealthClient, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		resp, err := health.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		w.Header().Set("Content-Type", "application/json")
		if err != nil || resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "indexer unreachable"})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
