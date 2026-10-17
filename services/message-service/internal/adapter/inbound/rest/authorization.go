package rest

import (
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

func authorizeChat(baseURL string, next http.HandlerFunc) http.HandlerFunc {
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if _, err := uuid.Parse(userID); err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		chatID := r.PathValue("chat_id")
		if _, err := uuid.Parse(chatID); err != nil {
			writeError(w, http.StatusBadRequest, "invalid chat ID")
			return
		}
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, baseURL+"/chats/"+url.PathEscape(chatID), nil)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "authorization unavailable")
			return
		}
		req.Header.Set("X-User-ID", userID)
		response, err := client.Do(req)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "authorization unavailable")
			return
		}
		defer response.Body.Close()
		switch response.StatusCode {
		case http.StatusOK:
			next(w, r)
		case http.StatusForbidden, http.StatusNotFound:
			writeError(w, http.StatusForbidden, "access denied")
		default:
			writeError(w, http.StatusServiceUnavailable, "authorization unavailable")
		}
	}
}
