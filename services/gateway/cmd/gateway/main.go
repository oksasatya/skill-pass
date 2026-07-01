// Command gateway is the SkillPass public BFF: REST + SSE over the indexer's gRPC
// CertificateQuery service. It never touches Postgres — the indexer owns the read model.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
	"github.com/oksasatya/skillpass/services/gateway/internal/config"
	"github.com/oksasatya/skillpass/services/gateway/internal/httpapi"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, err := grpc.NewClient(cfg.IndexerGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("indexer dial", "err", err)
		os.Exit(1)
	}
	defer conn.Close() //nolint:errcheck // best-effort close on process exit

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpapi.NewRouter(httpapi.Deps{
			Cert:           certv1.NewCertificateQueryClient(conn),
			Health:         grpc_health_v1.NewHealthClient(conn),
			Log:            log,
			RequestTimeout: cfg.RequestTimeout,
		}),
	}

	if err := runConcurrently(ctx, srv, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// runConcurrently starts the HTTP server and shuts it down gracefully on ctx cancellation.
func runConcurrently(ctx context.Context, srv *http.Server, log *slog.Logger) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
