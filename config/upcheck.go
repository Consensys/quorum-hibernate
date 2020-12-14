package config

import (
	"errors"
	"strings"
)

type Upcheck struct {
	UpcheckUrl string `toml:"upcheckUrl"` // http endpoint for up check
	ReturnType string `toml:"returnType"` // type of returned data. RPCResult or string
	Method     string `toml:"method"`     // GET or POST
	Body       string `toml:"body"`       // Body of up check request
	Expected   string `toml:"expected"`   // expected output string if return type is string
}

func (c Upcheck) IsRpcResult() bool {
	return c.ReturnType == "rpcresult"
}

func (c Upcheck) IsStringResult() bool {
	return c.ReturnType == "string"
}

func (c Upcheck) IsValid() error {
	if c.UpcheckUrl == "" {
		return errors.New("Upcheck - upcheck url is empty")
	}
	c.ReturnType = strings.ToLower(c.ReturnType)
	if !c.IsRpcResult() && !c.IsStringResult() {
		return errors.New("Upcheck - invalid returnType. it must be RPCResult or string")
	}

	c.Method = strings.ToUpper(c.Method)
	if c.Method != "GET" && c.Method != "POST" {
		return errors.New("Upcheck - invalid Method. it must be POST or GET")
	}
	if c.IsStringResult() && c.Expected == "" {
		return errors.New("Upcheck - expected value is empty. It must be provided for returnType string")
	}
	return nil
}
