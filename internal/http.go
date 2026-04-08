package internal

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// NewHTTPClient returns an http.Client with sensible default timeouts.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 60 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
}

// HTTPIsTemporary reports whether err is a temporary or timeout URL error.
func HTTPIsTemporary(err error) bool {
	if urlerr, ok := err.(*url.Error); ok {
		return urlerr.Temporary() || urlerr.Timeout() || errors.Is(urlerr, io.EOF)
	}
	return false
}
