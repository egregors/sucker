package internal

import (
	"errors"
	"fmt"
	"net/url"
)

// ParseArgs validate cli args
func ParseArgs(args []string) ([]string, error) {
	if len(args) < 1 {
		return nil, errors.New("not enough args")
	}

	for _, u := range args {
		_, err := url.ParseRequestURI(u)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("bad URL: %v", err))
		}
	}

	return args, nil
}
