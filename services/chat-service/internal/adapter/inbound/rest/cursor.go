package rest

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
	"github.com/handziurdmytro/messenger/chat-service/internal/usecase"
)

const headerNextCursor = "X-Next-Cursor"

var (
	errInvalidLimit  = errors.New("limit must be a positive integer")
	errInvalidCursor = errors.New("invalid cursor")
)

func parseChatPage(r *http.Request) (int, *usecase.ChatCursor, error) {
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return 0, nil, errInvalidLimit
		}
		limit = parsed
	}

	raw := r.URL.Query().Get("before")
	if raw == "" {
		return limit, nil, nil
	}

	cursor, err := decodeCursor(raw)
	if err != nil {
		return 0, nil, err
	}
	return limit, &cursor, nil
}

func encodeCursor(c usecase.ChatCursor) string {
	raw := c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + string(c.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(raw string) (usecase.ChatCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return usecase.ChatCursor{}, errInvalidCursor
	}

	createdAtRaw, idRaw, ok := strings.Cut(string(data), "|")
	if !ok {
		return usecase.ChatCursor{}, errInvalidCursor
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return usecase.ChatCursor{}, errInvalidCursor
	}

	id, err := chat.ParseID(idRaw)
	if err != nil {
		return usecase.ChatCursor{}, errInvalidCursor
	}

	return usecase.ChatCursor{CreatedAt: createdAt, ID: id}, nil
}
