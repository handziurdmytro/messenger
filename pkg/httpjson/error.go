package httpjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ClientError struct {
	Status  int
	Message string
}

func (e *ClientError) Error() string {
	return e.Message
}

func Error(w http.ResponseWriter, status int, msg string) error {
	return Write(w, status, struct {
		Error string `json:"error"`
	}{msg})
}

func translateDecodeError(err error) error {
	var syntaxErr *json.SyntaxError
	var unmarshalTypeErr *json.UnmarshalTypeError
	var maxBytesErr *http.MaxBytesError

	switch {
	case errors.As(err, &syntaxErr):
		return &ClientError{
			http.StatusBadRequest,
			fmt.Sprintf("body contains malformed JSON (at character %d)", syntaxErr.Offset),
		}

	case errors.Is(err, io.ErrUnexpectedEOF):
		return &ClientError{http.StatusBadRequest, "body contains malformed JSON"}

	case errors.As(err, &unmarshalTypeErr):
		return &ClientError{
			http.StatusBadRequest,
			fmt.Sprintf("body contains incorrect JSON type for field %q", unmarshalTypeErr.Field),
		}

	case errors.Is(err, io.EOF):
		return &ClientError{http.StatusBadRequest, "body must not be empty"}

	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return &ClientError{
			http.StatusBadRequest,
			fmt.Sprintf("body contains unknown key %s", field),
		}

	case errors.As(err, &maxBytesErr):
		return &ClientError{
			http.StatusRequestEntityTooLarge,
			fmt.Sprintf("body must not be larger than %d bytes", maxBytesErr.Limit),
		}

	default:
		return err
	}
}
