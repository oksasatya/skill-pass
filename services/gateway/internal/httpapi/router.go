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

// NewRouter builds the gateway's HTTP mux, wrapped with request logging + panic recovery.
func NewRouter(d Deps) http.Handler {
	if d.Log == nil {
		d.Log = slog.Default()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", Healthz)
	mux.Handle("GET /readyz", Readyz(d.Health, d.RequestTimeout))
	mux.Handle("GET /certificates", ListCertificates(d))
	mux.Handle("GET /certificates/stream", StreamCertificateEvents(d))
	mux.Handle("GET /stats/trend", GetIssuanceTrend(d))
	mux.Handle("GET /certificates/{tokenId}", GetCertificate(d))
	mux.Handle("GET /certificates/{tokenId}/metadata", GetCertificateMetadata(d))
	return WithObservability(mux, d.Log)
}
