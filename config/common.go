package config

import (
	"net/url"
)

func isValidUrl(addr string) error {
	_, err := url.Parse(addr)
	return err
}
