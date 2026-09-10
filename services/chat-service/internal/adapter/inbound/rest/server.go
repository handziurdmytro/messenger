package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type Server struct {
	http *http.Server
}

func NewServer(port string, createChat *usecase.CreateChatUseCase, hub *chat.Hub, baseLogger *slog.Logger) *Server {
	handler := NewHandler(createChat, hub)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.CreateChat)

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
