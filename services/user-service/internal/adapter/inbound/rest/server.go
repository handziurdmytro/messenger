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
	getProfile GetProfileUseCase,
	updateProfile UpdateProfileUseCase,
	searchUsers SearchUsersUseCase,
	addContact AddContactUseCase,
	getContacts GetContactsUseCase,
	baseLogger *slog.Logger,
) *Server {
	h := NewHandler(getProfile, updateProfile, searchUsers, addContact, getContacts)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /me", h.GetProfile)
	mux.HandleFunc("GET /profiles/{id}", h.GetProfile)
	mux.HandleFunc("PATCH /me", h.UpdateProfile)
	mux.HandleFunc("GET /search", h.SearchUsers)
	mux.HandleFunc("POST /me/contacts", h.AddContact)
	mux.HandleFunc("GET /me/contacts", h.GetContacts)

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
