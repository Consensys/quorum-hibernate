package core

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_RandomInt(t *testing.T) {
	c := 1
	for c <= 1000 {
		w := RandomInt(100, 1000)
		if w > 1000 || w < 100 {
			t.Error("wait time is out of range (100 - 1000)")
		}
		c++
	}
}

func Test_CallREST(t *testing.T) {
	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.RequestURI == "/upcheck" {
			// Send response to be tested
			rw.Write([]byte("I'am up!"))
		} else {
			http.Error(rw, "unsupported uri", http.StatusInternalServerError)
		}
	}))
	// Close the server when test finishes
	defer server.Close()
	res, err := CallREST(server.URL+"/upcheck", "GET", []byte(""))
	assert.NoError(t, err)
	expected := "I'am up!"
	assert.Equal(t, res, expected)

	_, err = CallREST(server.URL+"/invalid", "GET", []byte(""))
	assert.Error(t, err)

}

type DummyBlocknumResp struct {
	Result interface{} `json:"result"`
	Error  error       `json:"error"`
}

type DummyBlocknumInvalidResp struct {
	ResultDum interface{} `json:"resultDum"`
	Error     error       `json:"error"`
}

func Test_CallRPC(t *testing.T) {
	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.RequestURI == "/blocknumber" {
			// Send response to be tested
			rw.Write([]byte(`{"jsonrpc":"2.0","id":67,"result":"0x27a9"}`))
		} else {
			http.Error(rw, "unsupported uri", http.StatusInternalServerError)
		}
	}))
	// Close the server when test finishes
	defer server.Close()
	var bnResp DummyBlocknumResp
	err := CallRPC(server.URL+"/blocknumber", []byte("dummy req"), &bnResp)
	assert.NoError(t, err)
	assert.NotNil(t, bnResp)
	assert.NoError(t, bnResp.Error)
	assert.NotNil(t, bnResp.Result)
	blkNumExpected := "0x27a9"
	blkNumActual := bnResp.Result.(string)
	assert.Equal(t, blkNumActual, blkNumExpected)

	var badResp DummyBlocknumInvalidResp
	err = CallRPC(server.URL+"/blocknumber", []byte("dummy req"), &badResp)
	assert.NoError(t, err)
	assert.NotNil(t, badResp)
	assert.Nil(t, badResp.ResultDum)
}
