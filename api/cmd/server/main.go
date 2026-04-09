package main

import (
	"context"
	"errors"
	"log"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chrisneave/phem/api/internal/assumptions"
	"github.com/chrisneave/phem/api/internal/baseline"
	"github.com/chrisneave/phem/api/internal/config"
	"github.com/chrisneave/phem/api/internal/db"
	httpapi "github.com/chrisneave/phem/api/internal/http"
	"github.com/chrisneave/phem/api/internal/scenarios"
	"github.com/chrisneave/phem/api/internal/seed"
	"github.com/chrisneave/phem/api/internal/solar"
)

func main() {
	cfg := config.FromEnv()
	log.Printf("phem-api starting; db=%s:%d/%s addr=%s", cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.HTTPAddr)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Connect with retry — give Docker time to bring postgres up.
	pool := mustConnect(ctx, cfg.DSN())
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	asm := assumptions.New(pool)
	bsl := baseline.New(pool, asm)
	sol := solar.New(pool, cfg.SiteLat, cfg.SiteLon)
	eng := scenarios.New(pool, asm, bsl, sol)

	if cfg.SeedOnEmpty {
		if err := seed.SeedIfEmpty(ctx, pool, cfg.SiteLat, cfg.SiteLon); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}
	if err := eng.SeedDefaultScenarios(ctx); err != nil {
		log.Printf("warn: seed default scenarios: %v", err)
	}

	router := httpapi.NewRouter(httpapi.Deps{
		Pool:        pool,
		Baseline:    bsl,
		Scenarios:   eng,
		Assumptions: asm,
	})

	srv := &stdhttp.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	_ = srv.Shutdown(shutdownCtx)
	log.Printf("phem-api stopped")
}

func mustConnect(ctx context.Context, dsn string) *pgxpool.Pool {
	deadline := time.Now().Add(60 * time.Second)
	for {
		pool, err := db.Connect(ctx, dsn)
		if err == nil {
			return pool
		}
		if time.Now().After(deadline) {
			log.Fatalf("connect db: %v", err)
		}
		log.Printf("waiting for db: %v", err)
		select {
		case <-ctx.Done():
			log.Fatalf("aborted")
		case <-time.After(2 * time.Second):
		}
	}
}
