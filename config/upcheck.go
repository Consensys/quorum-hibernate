package config

import (
	"errors"
	"strings"
)

type Upcheck struct {
	UpcheckUrl string `toml:"url" json:"url"`               // http endpoint for up check
	ReturnType string `toml:"returnType" json:"returnType"` // type of returned data. RPCResult or string
	Method     string `toml:"method" json:"method"`         // GET or POST
	Body       string `toml:"body" json:"body"`             // Body of up check request
	Expected   string `toml:"expected" json:"expected"`     // expected output string if return type is string
}

func (c Upcheck) IsRpcResult() bool {
	return c.ReturnType == "rpcresult"
}

func (c Upcheck) IsStringResult() bool {
	return c.ReturnType == "string"
}

func (c Upcheck) IsValid() error {
	if c.UpcheckUrl == "" {
		return newFieldErr("url", isEmptyErr)
	}
	c.ReturnType = strings.ToLower(c.ReturnType)
	if !c.IsRpcResult() && !c.IsStringResult() {
		return newFieldErr("returnType", errors.New("must be rpcresult or string"))
	}

	c.Method = strings.ToUpper(c.Method)
	if c.Method != "GET" && c.Method != "POST" {
		return newFieldErr("method", errors.New("must be POST or GET"))
	}
	if c.IsStringResult() && c.Expected == "" {
		return newFieldErr("expected", errors.New("must be set as returnType is string"))
	}
	return nil
}
