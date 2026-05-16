package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/webui"
)

const (
	defaultAddr     = ":8080"
	shutdownTimeout = 5 * time.Second
)

// Config contains web server settings.
type Config struct {
	Addr             string
	SnapshotProvider webui.SnapshotProvider
	Branding         string
	Title            string
}

// Run starts the web UI HTTP server and blocks until the server exits or ctx is canceled.
func Run(ctx context.Context, config Config) error {
	server := &http.Server{
		Addr:              config.listenAddr(),
		Handler:           webui.NewHandler(config.handlerConfig()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		errc <- server.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("run web server: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown web server: %w", err)
		}
		if err := <-errc; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("run web server: %w", err)
		}
		return nil
	}
}

func (config Config) listenAddr() string {
	if config.Addr == "" {
		return defaultAddr
	}
	return config.Addr
}

func (config Config) handlerConfig() webui.Config {
	return webui.Config{
		SnapshotProvider: config.SnapshotProvider,
		Branding:         config.Branding,
		Title:            config.Title,
	}
}
