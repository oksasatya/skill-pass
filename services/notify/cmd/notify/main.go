// Command notify is the SkillPass webhook delivery service. It consumes webhook:deliver
// tasks from the shared Redis/asynq queue and POSTs signed payloads to one configured
// webhook URL. It never touches Postgres -- the indexer owns all durable webhook state
// (the outbox table + the sweep backstop).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"golang.org/x/sync/errgroup"

	"github.com/oksasatya/skillpass/services/notify/internal/adapter/webhook"
	"github.com/oksasatya/skillpass/services/notify/internal/config"
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

	redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr}
	server := asynq.NewServer(redisOpt, asynq.Config{Concurrency: 5})

	mux := asynq.NewServeMux()
	mux.Handle(webhook.DeliverTaskType, webhook.NewHandler(cfg.WebhookURL, cfg.WebhookSecret))

	healthSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: healthzMux()}

	if err := runConcurrently(ctx, server, mux, healthSrv, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// healthzMux serves a minimal liveness endpoint -- notify has no other public HTTP API.
func healthzMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

// runConcurrently starts the asynq consumer and the /healthz server; shuts both down
// gracefully on ctx cancellation.
func runConcurrently(ctx context.Context, server *asynq.Server, mux *asynq.ServeMux, healthSrv *http.Server, log *slog.Logger) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return server.Run(mux)
	})

	g.Go(func() error {
		log.Info("healthz server listening", "addr", healthSrv.Addr)
		if err := healthSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()
		server.Shutdown()
		return healthSrv.Close()
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}
