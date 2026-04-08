package internal

import (
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

// URLList is a flag.Value that collects multiple URL flags.
type URLList []*url.URL

func (u *URLList) String() string {
	return fmt.Sprint(*u)
}

// Set parses and appends a URL to the list.
func (u *URLList) Set(val string) error {
	parsed, err := url.Parse(val)
	if err != nil {
		return err
	}
	*u = append(*u, parsed)
	return nil
}

// URLParser returns a function that parses a string into a destination URL.
func URLParser(out *url.URL) func(string) error {
	return func(raw string) error {
		parsed, err := url.Parse(raw)
		if err != nil {
			return err
		}
		*out = *parsed
		return nil
	}
}

// UUIDParser returns a flag value parser that writes a UUID into out.
func UUIDParser(out *uuid.UUID) func(string) error {
	return func(raw string) error {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return err
		}
		*out = parsed
		return nil
	}
}
