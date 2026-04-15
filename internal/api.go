// Package internal defines common types used across the codebase.
package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// PlayerID identifies a player.
type PlayerID = uuid.UUID

// ShortID returns the first 8 hex characters of a UUID for display purposes.
func ShortID(id uuid.UUID) string {
	return id.String()[:8]
}

// SessionID identifies a game session.
type SessionID = uuid.UUID

// WriteError writes an error response with the given status code.
func WriteError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(err.Error()))
}

// PathUUID parses a path variable as a UUID.
func PathUUID(r *http.Request, name string) (uuid.UUID, error) {
	raw := r.PathValue(name)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid param %q: %w", name, err)
	}
	return id, nil
}

// BindJSON decodes JSON from the request body into the given type.
func BindJSON[T any](r io.Reader) (T, error) {
	var data T
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&data)
	return data, err
}

// EncodeJSON encodes the given data as JSON bytes.
func EncodeJSON[T any](t T) ([]byte, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// RespondJSON encodes the given data as JSON and writes it to the response.
func RespondJSON(w http.ResponseWriter, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonData)
	return err
}

// CORS wraps a handler with permissive CORS headers for browser access.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
