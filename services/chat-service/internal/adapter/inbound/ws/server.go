package ws

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/port/outbound"
)

type Server struct {
	http *http.Server
}

func NewServer(port string, hub *chat.Hub, pub outbound.MessagePublisher) *Server {
	handler := NewHandler(hub, pub)

	mux := http.NewServeMux()
	mux.Handle("/ws", handler)

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
