package apiserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Server
type Server struct {
	http *http.Server
}

// New
func New(addr string) *Server {
	return &Server{
		http: &http.Server{
			Addr:         addr,
			Handler:      newRouter(nil),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}
}

// Start
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		slog.Info("api server started", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.shutdown()
	}
}

func (s *Server) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.http.Shutdown(ctx)
}
