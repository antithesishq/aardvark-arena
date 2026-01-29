package internal

import (
	"fmt"
	"net/url"
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
// Can we used with `flag.Func`:
// ```go
// var matchmakerURL *url.URL
// flag.Func("matchmaker-url", "matchmaker base URL", internal.URLParser(matchmakerURL))
// ```
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
