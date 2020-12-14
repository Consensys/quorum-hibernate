package config

import (
	"fmt"
	"net/url"
)

// namedValidationError provides additional context to an error, useful for providing context when there is a validation error with an element in an array
type namedValidationError struct {
	name, errMsg string
}

func (e namedValidationError) Error() string {
	return fmt.Sprintf("name = %v: %v", e.name, e.errMsg)
}

func isValidUrl(addr string) error {
	_, err := url.Parse(addr)
	return err
}
