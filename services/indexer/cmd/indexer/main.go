// Command indexer is the SkillPass certificate indexer service.
// It runs the chain worker and the gRPC CertificateQuery server concurrently,
// and shuts down gracefully on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/asynqjobs"
	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/cache"
	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/chain"
	grpcadapter "github.com/oksasatya/skillpass/services/indexer/internal/adapter/grpc"
	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/postgres"
	"github.com/oksasatya/skillpass/services/indexer/internal/config"
	"github.com/oksasatya/skillpass/services/indexer/internal/platform/broadcast"
	platformdb "github.com/oksasatya/skillpass/services/indexer/internal/platform/db"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
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

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("pgxpool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := platformdb.Migrate(ctx, pool); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}

	repo := postgres.NewCertificateRepo(pool)

	src, err := chain.NewEventSource(ctx, cfg.EthRPCURL, cfg.ContractAddress)
	if err != nil {
		log.Error("eventsource", "err", err)
		os.Exit(1)
	}
	defer src.Close()

	broadcaster := broadcast.NewBroadcaster()

	worker := usecase.NewWorker(src, repo, usecase.WorkerConfig{
		ChainID:      cfg.ChainID,
		StartBlock:   cfg.StartBlock,
		BatchSize:    cfg.BatchSize,
		PollInterval: cfg.PollInterval,
	}, log)
	worker.SetPublisher(broadcaster)

	trendService := usecase.NewTrendService(repo, cfg.ChainID)

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close() //nolint:errcheck // best-effort close on process exit

	trendService.SetCache(cache.NewRedisTrendCache(redisClient))

	asynqRedisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr}
	asynqClient := asynq.NewClient(asynqRedisOpt)
	defer asynqClient.Close() //nolint:errcheck // best-effort close on process exit

	worker.SetEnqueuer(asynqjobs.NewEnqueuer(asynqClient))

	s := buildGRPCServer(repo, src, broadcaster, trendService, log)

	asynqServer, asynqMux, scheduler := buildAsynqRuntime(asynqRedisOpt, trendService, log)

	svc := runtimeServices{
		grpcServer:  s,
		worker:      worker,
		asynqServer: asynqServer,
		asynqMux:    asynqMux,
		scheduler:   scheduler,
	}
	if err := runConcurrently(ctx, svc, cfg.GRPCAddr, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// buildGRPCServer wires the gRPC server with interceptors, health, and reflection.
func buildGRPCServer(repo usecase.CertificateRepo, src usecase.EventSource, sub usecase.EventSubscriber, trend *usecase.TrendService, log *slog.Logger) *grpc.Server {
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(log),
			loggingInterceptor(log),
		),
	)

	certv1.RegisterCertificateQueryServer(s, grpcadapter.NewServer(repo, src, sub, trend, log))

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(s)

	return s
}

// buildAsynqRuntime wires the asynq processing server (handles enqueued refresh tasks) and
// scheduler (15-minute cron backstop, in case an event-triggered enqueue is ever missed).
// Returns the mux alongside the server since Run(mux) needs the exact same instance the
// handler was registered on.
func buildAsynqRuntime(redisOpt asynq.RedisClientOpt, trend *usecase.TrendService, log *slog.Logger) (*asynq.Server, *asynq.ServeMux, *asynq.Scheduler) {
	server := asynq.NewServer(redisOpt, asynq.Config{Concurrency: 5})

	mux := asynq.NewServeMux()
	mux.Handle(usecase.TrendRefreshTaskType, asynqjobs.NewRefreshTrendCacheHandler(trend, log))

	scheduler := asynq.NewScheduler(redisOpt, nil)
	if _, err := scheduler.Register("*/15 * * * *", asynqjobs.NewRefreshTrendCacheTask()); err != nil {
		log.Error("register trend-refresh cron", "err", err)
	}

	return server, mux, scheduler
}

// runtimeServices bundles the long-running components runConcurrently supervises —
// introduced once adding the asynq server/scheduler would have pushed runConcurrently
// past the Sonar-preferred 5-param ceiling.
type runtimeServices struct {
	grpcServer  *grpc.Server
	worker      *usecase.Worker
	asynqServer *asynq.Server
	asynqMux    *asynq.ServeMux
	scheduler   *asynq.Scheduler
}

// runConcurrently starts the worker, gRPC server, asynq processing server, and asynq
// scheduler; monitors ctx for graceful shutdown of all four.
func runConcurrently(ctx context.Context, svc runtimeServices, addr string, log *slog.Logger) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return svc.worker.Run(gCtx)
	})

	g.Go(func() error {
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		log.Info("gRPC server listening", "addr", addr)
		return svc.grpcServer.Serve(lis)
	})

	g.Go(func() error {
		return svc.asynqServer.Run(svc.asynqMux)
	})

	g.Go(func() error {
		return svc.scheduler.Run()
	})

	g.Go(func() error {
		<-gCtx.Done()
		// asynqServer.Run/scheduler.Run each install their own internal signal.Notify
		// handler (asynq's own SIGTERM/SIGINT/SIGTSTP handling) alongside this ctx-driven
		// shutdown. Both paths call the same idempotent Shutdown/Stop methods, so calling
		// them again here is a safe no-op if asynq's own handler already fired first.
		svc.grpcServer.GracefulStop()
		svc.asynqServer.Shutdown()
		svc.scheduler.Shutdown()
		return nil
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	}
	return nil
}

// recoveryInterceptor catches panics and returns codes.Internal, preventing server crash.
func recoveryInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered", "method", info.FullMethod, "panic", r, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// loggingInterceptor logs method name, duration, and gRPC status code.
func loggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Info("grpc",
			"method", info.FullMethod,
			"duration_ms", time.Since(start).Milliseconds(),
			"code", status.Code(err).String(),
		)
		return resp, err
	}
}
