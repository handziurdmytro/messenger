package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/pkg/logger"
)

type Server struct {
	http *http.Server
}

func NewServer(
	port string,
	services Services,
	baseLogger *slog.Logger,
) *Server {
	h := NewHandler(services)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /me", h.GetProfile)
	mux.HandleFunc("GET /profiles", h.GetProfiles)
	mux.HandleFunc("GET /profiles/{id}", h.GetProfile)
	mux.HandleFunc("PATCH /me", h.UpdateProfile)
	mux.HandleFunc("GET /search", h.SearchUsers)
	mux.HandleFunc("POST /me/contacts", h.AddContact)
	mux.HandleFunc("GET /me/contacts", h.GetContacts)
	mux.HandleFunc("DELETE /me/contacts/{id}", h.RemoveContact)
	mux.HandleFunc("POST /me/blocks", h.BlockUser)
	mux.HandleFunc("GET /me/blocks", h.ListBlocked)
	mux.HandleFunc("DELETE /me/blocks/{id}", h.UnblockUser)

	handlerWithLogger := logger.HTTPMiddleware(baseLogger)(mux)

	return &Server{
		http: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      handlerWithLogger,
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
