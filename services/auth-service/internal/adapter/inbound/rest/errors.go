package rest

import (
	"net/http"

	"github.com/handziurdmytro/messenger/pkg/httpjson"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	_ = httpjson.Write(w, status, errorResponse{Error: msg})
}
