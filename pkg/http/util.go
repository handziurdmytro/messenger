package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if reflect.ValueOf(dst).Kind() != reflect.Ptr {
		return errors.New("dst must be a non-nil pointer")
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&dst); err != nil {
		return fmt.Errorf("failed to decode: %w", err)
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("body must contain a single JSON value")
	}

	return nil
}

func WriteJSON(w http.ResponseWriter, status int, data any) error {
	js, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(js)
	return nil
}

func WriteError(w http.ResponseWriter, status int, err error) error {
	return WriteJSON(w, status, struct {
		Error string `json:"error"`
	}{err.Error()})
}
