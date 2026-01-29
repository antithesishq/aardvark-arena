// Package internal defines common types used across the codebase.
package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// PlayerID identifies a player.
type PlayerID = uuid.UUID

// SessionID identifies a game session.
type SessionID = uuid.UUID

// APIKey authenticates API requests.
type APIKey = uuid.UUID

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
		return uuid.UUID{}, fmt.Errorf("invalid param %q: %e", name, err)
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

// EncodeJSON encodes the given data as JSON and returns a reader.
func EncodeJSON[T any](t T) (*bytes.Reader, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
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
