package rest

import (
	"errors"
	"net/http"
	"strconv"
)

const defaultSearchLimit = 20

var (
	errInvalidLimit  = errors.New("limit must be a positive integer")
	errInvalidOffset = errors.New("offset must be a non-negative integer")
)

func parsePage(r *http.Request, defaultLimit int) (limit, offset int, err error) {
	limit = defaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit <= 0 {
			return 0, 0, errInvalidLimit
		}
	}

	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return 0, 0, errInvalidOffset
		}
	}

	return limit, offset, nil
}
