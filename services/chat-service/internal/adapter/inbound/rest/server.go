package rest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type Server struct {
	http *http.Server
}

func NewServer(
	port string,
	createChat *usecase.CreateChatUseCase,
	getChat *usecase.GetChatUseCase,
	wsHandler http.Handler,
	baseLogger *slog.Logger,
) *Server {
	handler := NewHandler(createChat, getChat)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /chats", handler.CreateChat)
	mux.HandleFunc("GET /chats/{id}", handler.GetChat)

	if wsHandler != nil {
		mux.Handle("GET /ws", wsHandler)
	}

	return &Server{
		http: &http.Server{
			Addr:              fmt.Sprintf(":%s", port),
			Handler:           logger.HTTPMiddleware(baseLogger)(mux),
			ReadHeaderTimeout: 5 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20,
		},
	}
}

func (s *Server) Start() error {
	err := s.http.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
