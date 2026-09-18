package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type Server struct {
	http *http.Server
}

func NewServer(
	port string,
	register *usecase.RegisterUseCase,
	login *usecase.LoginUseCase,
	refresh *usecase.RefreshUseCase,
	tokenSvc *auth.TokenService,
	baseLogger *slog.Logger,
) *Server {
	h := NewHandler(register, login, refresh, tokenSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /validate", h.Validate)
	mux.HandleFunc("POST /register", h.Register)
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("POST /refresh", h.Refresh)
	mux.HandleFunc("GET /.well-known/jwks.pem", h.JWKS)

	handler := logger.HTTPMiddleware(baseLogger)(mux)

	return &Server{
		http: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
