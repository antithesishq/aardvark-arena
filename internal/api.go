// Package internal defines common types used across the codebase.
package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// PlayerID identifies a player.
type PlayerID = uuid.UUID

// SessionID identifies a game session.
type SessionID = uuid.UUID

// Token authenticates API requests.
type Token uuid.UUID

// IsNil returns true if the token is nil or the zero UUID.
func (t *Token) IsNil() bool {
	return t == nil || uuid.UUID(*t) == uuid.Nil
}

// String returns the string representation of the token.
func (t *Token) String() string {
	return uuid.UUID(*t).String()
}

// Set parses a UUID string and sets the token value.
func (t *Token) Set(val string) error {
	parsed, err := uuid.Parse(val)
	if err != nil {
		return err
	}
	*t = Token(parsed)
	return nil
}

// TokenAuth wraps a handler with token authentication.
// If token is all zeros, authentication is skipped.
func TokenAuth(token Token, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token.IsNil() {
			next(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			WriteError(w, http.StatusUnauthorized, errors.New("missing authorization header"))
			return
		}
		provided, found := strings.CutPrefix(authHeader, "Bearer ")
		if !found {
			WriteError(w, http.StatusUnauthorized, errors.New("invalid authorization header"))
			return
		}
		if provided != token.String() {
			WriteError(w, http.StatusUnauthorized, errors.New("invalid token"))
			return
		}
		next(w, r)
	}
}

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
