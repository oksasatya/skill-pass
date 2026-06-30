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

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/chain"
	grpcadapter "github.com/oksasatya/skillpass/services/indexer/internal/adapter/grpc"
	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/postgres"
	"github.com/oksasatya/skillpass/services/indexer/internal/config"
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

	worker := usecase.NewWorker(src, repo, usecase.WorkerConfig{
		ChainID:      cfg.ChainID,
		StartBlock:   cfg.StartBlock,
		BatchSize:    cfg.BatchSize,
		PollInterval: cfg.PollInterval,
	}, log)

	s := buildGRPCServer(repo, src, log)

	if err := runConcurrently(ctx, s, worker, cfg.GRPCAddr, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// buildGRPCServer wires the gRPC server with interceptors, health, and reflection.
func buildGRPCServer(repo usecase.CertificateRepo, src usecase.EventSource, log *slog.Logger) *grpc.Server {
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(log),
			loggingInterceptor(log),
		),
	)

	certv1.RegisterCertificateQueryServer(s, grpcadapter.NewServer(repo, src, log))

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(s)

	return s
}

// runConcurrently starts the worker and gRPC server, monitors ctx for shutdown.
func runConcurrently(ctx context.Context, s *grpc.Server, worker *usecase.Worker, addr string, log *slog.Logger) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return worker.Run(gCtx)
	})

	g.Go(func() error {
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		log.Info("gRPC server listening", "addr", addr)
		return s.Serve(lis)
	})

	g.Go(func() error {
		<-gCtx.Done()
		s.GracefulStop()
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
