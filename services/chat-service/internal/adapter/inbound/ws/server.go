package ws

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type Server struct {
	http *http.Server
}

func NewServer(
	port string,
	hub *chat.Hub,
	pub outbound.MessagePublisher,
	chats outbound.ChatRepository,
	baseLogger *slog.Logger,
) *Server {
	handler := NewHandler(hub, pub, chats)

	mux := http.NewServeMux()
	mux.Handle("/ws", handler)

	handlerWithLogger := logger.HTTPMiddleware(baseLogger)(mux)

	return &Server{
		http: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      handlerWithLogger,
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
