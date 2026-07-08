package rest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/usecase"
)

type Server struct {
	http *http.Server
}

func NewServer(
	port string,
	getProfile *usecase.GetProfileUseCase,
	updateProfile *usecase.UpdateProfileUseCase,
	searchUsers *usecase.SearchUsersUseCase,
) *Server {
	h := NewHandler(getProfile, updateProfile, searchUsers)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/me", h.GetProfile)
	mux.HandleFunc("PATCH /users/me", h.UpdateProfile)
	mux.HandleFunc("GET /users/search", h.SearchUsers)

	return &Server{
		http: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      mux,
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
