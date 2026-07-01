package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// Deps holds the gateway HTTP layer's collaborators.
type Deps struct {
	Cert           certv1.CertificateQueryClient
	Health         grpc_health_v1.HealthClient
	Log            *slog.Logger
	RequestTimeout time.Duration
}

// NewRouter builds the gateway's HTTP mux. Routes are added incrementally as
// REST/SSE handlers land (certificates list/get, stream).
func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", Healthz)
	mux.Handle("GET /readyz", Readyz(d.Health, d.RequestTimeout))
	return mux
}
