package internal

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
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

// HTTPIsTemporary reports whether err is retryable due to transport/network issues.
func HTTPIsTemporary(err error) bool {
	if err == nil {
		return false
	}

	// unwrap *url.Error (http.Client always wraps)
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() || urlErr.Temporary() || errors.Is(urlErr, io.EOF) {
			return true
		}
		err = urlErr.Err
	}

	// generic network timeout / temporary
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	// syscall-level classification
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		switch {
		case opErr.Temporary() || opErr.Timeout(),
			errors.Is(opErr.Err, syscall.ECONNREFUSED),
			errors.Is(opErr.Err, syscall.ECONNRESET),
			errors.Is(opErr.Err, syscall.ECONNABORTED),
			errors.Is(opErr.Err, syscall.ETIMEDOUT),
			errors.Is(opErr.Err, syscall.EHOSTDOWN),
			errors.Is(opErr.Err, syscall.EHOSTUNREACH),
			errors.Is(opErr.Err, syscall.ENETDOWN),
			errors.Is(opErr.Err, syscall.ENETUNREACH):
			return true
		}
	}

	return false
}
