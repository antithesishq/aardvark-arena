package internal

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

func NewHttpClient() *http.Client {
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

func HttpIsTemporary(err error) bool {
	if urlerr, ok := err.(*url.Error); ok {
		return urlerr.Temporary() || urlerr.Timeout()
	}
	return false
}
