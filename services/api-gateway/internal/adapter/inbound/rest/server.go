package rest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/auth"
	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/proxy"
)

type Server struct {
	http *http.Server
}

func NewServer(port string, router *proxy.Router, authenticator *auth.Authenticator) *Server {
	httpHandler := NewHandler(router, authenticator)
	wsHandler := NewWSProxyHandler(router, authenticator)

	authed := authMiddleware(authenticator)

	mux := http.NewServeMux()
	mux.Handle("/ws/", authed(wsHandler))
	mux.Handle("/", authed(httpHandler))

	return &Server{
		http: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 0,
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
