// Command server is the online-shop REST API. It only handles public artifacts
// and ciphertexts; it never holds or uses a secret key and has no decryption
// endpoint. Configuration comes from environment variables (see .env.example).
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/filbertengyo/sisterpedia/internal/server/catalog"
	"github.com/filbertengyo/sisterpedia/internal/server/config"
	"github.com/filbertengyo/sisterpedia/internal/server/handlers"
	"github.com/filbertengyo/sisterpedia/internal/server/service"
	"github.com/filbertengyo/sisterpedia/internal/server/store"
	"github.com/filbertengyo/sisterpedia/pkg/ckks"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	all, err := ckks.LoadRegistry(cfg.ProfilesFile)
	if err != nil {
		return err
	}
	reg, err := all.Restrict(cfg.AllowedProfiles)
	if err != nil {
		return err
	}
	cat, err := catalog.Load(cfg.CatalogPath)
	if err != nil {
		return err
	}
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	svc := service.New(reg, st, cat, int(cfg.MaxBodyBytes))
	h := handlers.New(svc, handlers.Options{AllowedOrigins: cfg.AllowedOrigins, MaxBodyBytes: cfg.MaxBodyBytes})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second, // request bodies carry large keys/ciphertexts
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		log.Printf("server listening on %s (profiles %v, library %s, db %s)", cfg.Addr, reg.Names(), ckks.LibraryVersion, cfg.DBPath)
		errCh <- srv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Printf("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
