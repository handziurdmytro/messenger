package inbound

import "net/http"

type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
