package config

import (
	"errors"
	"strings"
)

type UpcheckConfig struct {
	UpcheckUrl string `toml:"upcheckUrl"` // http endpoint for up check
	ReturnType string `toml:"returnType"` // type of returned data. RPCResult or string
	Method     string `toml:"method"`     // GET or POST
	Body       string `toml:"body"`       // Body of up check request
	Expected   string `toml:"expected"`   // expected output string if return type is string
}

func (c UpcheckConfig) IsValid() error {
	if c.UpcheckUrl == "" {
		return errors.New("UpcheckConfig - upcheck url is empty")
	}
	c.ReturnType = strings.ToLower(c.ReturnType)
	if !c.IsRpcResult() && !c.IsStringResult() {
		return errors.New("UpcheckConfig - invalid returnType. it must be RPCResult or string")
	}

	c.Method = strings.ToUpper(c.Method)
	if c.Method != "GET" && c.Method != "POST" {
		return errors.New("UpcheckConfig - invalid Method. it must be POST or GET")
	}
	if c.IsStringResult() && c.Expected == "" {
		return errors.New("UpcheckConfig - expected value is empty. It must be provided for returnType string")
	}
	return nil
}
