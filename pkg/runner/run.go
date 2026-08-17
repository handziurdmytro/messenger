package runner

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"
)

type Service interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

func Run(ctx context.Context, log *slog.Logger, shutdownTimeout time.Duration, services ...Service) error {
	g, gCtx := errgroup.WithContext(ctx)

	for _, s := range services {
		g.Go(func() error { return s.Start(gCtx) })
	}

	g.Go(func() error {
		<-gCtx.Done()
		log.Info("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		var errs []error
		for _, s := range services {
			if err := s.Shutdown(shutdownCtx); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	})

	return g.Wait()
}
