package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/handziurdmytro/messenger/api-gateway/internal/domain/proxy"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: msg})
}

func errorAs(err error, target *proxy.DecisionError) bool {
	return errors.As(err, target)
}
