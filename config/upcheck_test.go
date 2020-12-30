package config

import (
	"encoding/json"
	"fmt"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
	"testing"
)

func minimumValidUpcheck() Upcheck {
	return Upcheck{
		UpcheckUrl: "http://url",
		ReturnType: "string",
		Method:     "GET",
		Body:       "",
		Expected:   "status = up",
	}
}

func TestUpcheck_Unmarshal(t *testing.T) {
	tests := []struct {
		name, configTemplate string
	}{
		{
			name: "json",
			configTemplate: `
{
	"%v": "http://url",
	"%v": "string",
	"%v": "GET",
	"%v": "some-body",
	"%v": "status = up"
}`,
		},
		{
			name: "toml",
			configTemplate: `
%v = "http://url"
%v = "string"
%v = "GET"
%v = "some-body"
%v = "status = up"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := fmt.Sprintf(tt.configTemplate, urlField, returnTypeField, methodField, bodyField, expectedField)

			want := Upcheck{
				UpcheckUrl: "http://url",
				ReturnType: "string",
				Method:     "GET",
				Body:       "some-body",
				Expected:   "status = up",
			}

			var (
				got Upcheck
				err error
			)

			if tt.name == "json" {
				err = json.Unmarshal([]byte(conf), &got)
			} else if tt.name == "toml" {
				err = toml.Unmarshal([]byte(conf), &got)
			}

			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestUpcheck_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidUpcheck()

	err := c.IsValid()

	require.NoError(t, err)
}

func TestUpcheck_IsValid_Url(t *testing.T) {
	c := minimumValidUpcheck()

	c.UpcheckUrl = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, urlField+" is empty")
}

func TestUpcheck_IsValid_ReturnType(t *testing.T) {
	tests := []struct {
		name, returnType, wantErrMsg string
	}{
		{
			name:       "not set",
			returnType: "",
			wantErrMsg: returnTypeField + " must be rpcresult or string",
		},
		{
			name:       "not valid",
			returnType: "notvalid",
			wantErrMsg: returnTypeField + " must be rpcresult or string",
		},
		{
			name:       "rpcresult",
			returnType: "rpcresult",
			wantErrMsg: "",
		},
		{
			name:       "rpcresult char case",
			returnType: "rPcReSuLt",
			wantErrMsg: "",
		},
		{
			name:       "string",
			returnType: "string",
			wantErrMsg: "",
		},
		{
			name:       "string char case",
			returnType: "sTrInG",
			wantErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidUpcheck()
			c.ReturnType = tt.returnType

			err := c.IsValid()

			if tt.wantErrMsg == "" {
				require.NoError(t, err)
			} else {
				require.IsType(t, &fieldErr{}, err)
				require.EqualError(t, err, tt.wantErrMsg)
			}
		})
	}
}

func TestUpcheck_IsValid_Method(t *testing.T) {
	tests := []struct {
		name, method, wantErrMsg string
	}{
		{
			name:       "not set",
			method:     "",
			wantErrMsg: methodField + " must be POST or GET",
		},
		{
			name:       "not valid",
			method:     "notvalid",
			wantErrMsg: methodField + " must be POST or GET",
		},
		{
			name:       "GET",
			method:     "GET",
			wantErrMsg: "",
		},
		{
			name:       "GET char case",
			method:     "gEt",
			wantErrMsg: "",
		},
		{
			name:       "POST",
			method:     "POST",
			wantErrMsg: "",
		},
		{
			name:       "POST char case",
			method:     "pOsT",
			wantErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidUpcheck()
			c.Method = tt.method

			err := c.IsValid()

			if tt.wantErrMsg == "" {
				require.NoError(t, err)
			} else {
				require.IsType(t, &fieldErr{}, err)
				require.EqualError(t, err, tt.wantErrMsg)
			}
		})
	}
}

func TestUpcheck_IsValid_Expected(t *testing.T) {
	tests := []struct {
		name, returnType, expected, wantErrMsg string
	}{
		{
			name:       "not set and string result",
			returnType: "string",
			expected:   "",
			wantErrMsg: expectedField + " must be set as returnType is string",
		},
		{
			name:       "set and string result",
			returnType: "string",
			expected:   "status = up",
			wantErrMsg: "",
		},
		{
			name:       "not set and rpcresult result",
			returnType: "rpcresult",
			expected:   "",
			wantErrMsg: "",
		},
		{
			name:       "set and rpcresult result",
			returnType: "rpcresult",
			expected:   "status = up",
			wantErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidUpcheck()
			c.ReturnType = tt.returnType
			c.Expected = tt.expected

			err := c.IsValid()

			if tt.wantErrMsg == "" {
				require.NoError(t, err)
			} else {
				require.IsType(t, &fieldErr{}, err)
				require.EqualError(t, err, tt.wantErrMsg)
			}
		})
	}
}
