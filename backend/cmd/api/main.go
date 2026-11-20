package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"relayradar/internal/api"
	"relayradar/internal/app"
	"relayradar/internal/config"
	"relayradar/internal/store"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()

	ctx := context.Background()
	repo, closeFn, err := openRepo(ctx, cfg, log)
	if err != nil {
		log.Error("store unavailable", "err", err)
		os.Exit(1)
	}
	defer closeFn()

	svc := app.NewService(repo, cfg.FaviconProxy)
	appSrv := api.New(svc, cfg, log)
	appSrv.StartStatusRecorder(context.Background())
	appSrv.StartProviderStatusRecorder(context.Background())
	appSrv.StartReachabilityRecorder(context.Background())
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           appSrv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("relayradar api listening", "addr", cfg.Addr, "store", cfg.StoreMode)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("bye")
}

func openRepo(ctx context.Context, cfg config.Config, log *slog.Logger) (store.Repo, func(), error) {
	if strings.EqualFold(cfg.StoreMode, "memory") || cfg.DatabaseURL == "memory" {
		log.Warn("using in-memory demo store; data resets on restart")
		return store.NewMemory(), func() {}, nil
	}

	st, err := openWithRetry(ctx, cfg.DatabaseURL, 30)
	if err != nil {
		return nil, func() {}, err
	}
	if err := st.Migrate(ctx); err != nil {
		st.Close()
		return nil, func() {}, err
	}
	if err := st.SyncModels(ctx); err != nil {
		log.Info("model catalog sync skipped", "err", err)
	}
	empty, err := st.IsEmpty(ctx)
	if err != nil {
		st.Close()
		return nil, func() {}, err
	}
	if empty && cfg.SeedDemo {
		log.Info("seeding demo dataset")
		if err := st.Seed(ctx); err != nil {
			st.Close()
			return nil, func() {}, err
		}
	}
	if cfg.SeedDemo {
		// Append stations added to the seed since the last deploy without
		// touching existing rows or probe history.
		if err := st.SyncDemoStations(ctx); err != nil {
			log.Warn("demo station top-up skipped", "err", err)
		}
	}
	return st, st.Close, nil
}

func openWithRetry(ctx context.Context, dsn string, attempts int) (*store.Store, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		st, err := store.Open(ctx, dsn)
		if err == nil {
			return st, nil
		}
		lastErr = err
		time.Sleep(2 * time.Second)
	}
	return nil, lastErr
}
