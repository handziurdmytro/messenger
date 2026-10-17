package rest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/handziurdmytro/messenger/message-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/logger"
)

type Server struct {
	http *http.Server
}

func NewServer(
	port string,
	history *usecase.GetHistory,
	get *usecase.GetMessage,
	upload *usecase.UploadFile,
	download *usecase.DownloadFile,
	baseLogger *slog.Logger,
) *Server {
	handler := NewHandler(history, get, upload, download)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /chats/{chat_id}/messages", handler.GetHistory)
	mux.HandleFunc("GET /chats/{chat_id}/messages/{message_id}", handler.GetMessage)
	mux.HandleFunc("POST /files", handler.UploadFile)
	mux.HandleFunc("GET /files/{id}", handler.DownloadFile)

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
