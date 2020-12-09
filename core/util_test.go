package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCallRPC(t *testing.T) {
	var (
		rpcMethod = "app.DoSomething"
		req       = fmt.Sprintf(`{"jsonrpc":2.0, "id":11, "method":"%v"}`, rpcMethod)
		respCode  = 200
		respBody  = `{"jsonrpc":"2.0","id":1,"result":{"someresponsedata": "val"}}`
	)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, req.Method, "POST")
		require.Equal(t, req.Header["Content-Type"], []string{"application/json"})

		type rpcRequest struct {
			Method string
		}

		rpcReq := rpcRequest{}

		err := json.NewDecoder(req.Body).Decode(&rpcReq)
		require.NoError(t, err)
		require.Equal(t, rpcMethod, rpcReq.Method)

		w.WriteHeader(respCode)
		_, err = w.Write([]byte(respBody))
		require.NoError(t, err)
	})

	mockServer := httptest.NewServer(serverMux)

	var got interface{}

	err := CallRPC(mockServer.URL, []byte(req), &got)
	require.NoError(t, err)

	rpcResp := got.(map[string]interface{})
	rpcResult := rpcResp["result"].(map[string]interface{})

	require.Contains(t, rpcResult, "someresponsedata")
}

func TestCallRPC_HTTPError(t *testing.T) {
	var (
		rpcMethod = "app.DoSomething"
		req       = fmt.Sprintf(`{"jsonrpc":2.0, "id":11, "method":"%v"}`, rpcMethod)
	)

	var tests = []struct {
		name     string
		respCode int
		respBody string
		wantErr  string
	}{
		{name: "clientError", respCode: 400, wantErr: "CallRPC - response status failed, not OK, status=400 Bad Request"},
		{name: "serverError", respCode: 500, wantErr: "CallRPC - response status failed, not OK, status=500 Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverMux := http.NewServeMux()
			serverMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

				require.Equal(t, req.Method, "POST")
				require.Equal(t, req.Header["Content-Type"], []string{"application/json"})

				type rpcRequest struct {
					Method string
				}

				rpcReq := rpcRequest{}

				err := json.NewDecoder(req.Body).Decode(&rpcReq)
				require.NoError(t, err)
				require.Equal(t, rpcMethod, rpcReq.Method)

				w.WriteHeader(tt.respCode)
				_, err = w.Write([]byte(tt.respBody))
				require.NoError(t, err)
			})

			mockServer := httptest.NewServer(serverMux)

			var resp interface{}

			err := CallRPC(mockServer.URL, []byte(req), &resp)
			require.EqualError(t, err, tt.wantErr)
			require.Empty(t, resp)
		})
	}
}

func TestCallRPC_RpcError(t *testing.T) {
	var (
		rpcMethod = "app.DoSomething"
		req       = fmt.Sprintf(`{"jsonrpc":2.0, "id":11, "method":"%v"}`, rpcMethod)
		respCode  = 200
		respBody  = `{"jsonrpc":"2.0","id":1,"error":{"code":100,"message":"someerrormessage", "data":{"field": "val"}}}`
	)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		require.Equal(t, req.Method, "POST")
		require.Equal(t, req.Header["Content-Type"], []string{"application/json"})

		type rpcRequest struct {
			Method string
		}

		rpcReq := rpcRequest{}

		err := json.NewDecoder(req.Body).Decode(&rpcReq)
		require.NoError(t, err)
		require.Equal(t, rpcMethod, rpcReq.Method)

		w.WriteHeader(respCode)
		_, err = w.Write([]byte(respBody))
		require.NoError(t, err)
	})

	mockServer := httptest.NewServer(serverMux)

	var got interface{}

	err := CallRPC(mockServer.URL, []byte(req), &got)
	require.NoError(t, err)

	rpcResp := got.(map[string]interface{})
	rpcErr := rpcResp["error"].(map[string]interface{})
	require.Equal(t, rpcErr["message"], "someerrormessage")

	rpcErrData := rpcErr["data"].(map[string]interface{})
	require.Contains(t, rpcErrData, "field")
}

func TestCallRPC_InvalidRespType(t *testing.T) {
	var (
		rpcMethod = "app.DoSomething"
		req       = fmt.Sprintf(`{"jsonrpc":2.0, "id":11, "method":"%v"}`, rpcMethod)
		respCode  = 200
		respBody  = `{"jsonrpc":"2.0","id":1,"result":{"someresponsedata": "val"}}`
	)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		require.Equal(t, req.Method, "POST")
		require.Equal(t, req.Header["Content-Type"], []string{"application/json"})

		type rpcRequest struct {
			Method string
		}

		rpcReq := rpcRequest{}

		err := json.NewDecoder(req.Body).Decode(&rpcReq)
		require.NoError(t, err)
		require.Equal(t, rpcMethod, rpcReq.Method)

		w.WriteHeader(respCode)
		_, err = w.Write([]byte(respBody))
		require.NoError(t, err)
	})

	mockServer := httptest.NewServer(serverMux)

	type invalid struct {
		NotKnown int
	}

	var got invalid

	err := CallRPC(mockServer.URL, []byte(req), &got)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestRpcError(t *testing.T) {
	var (
		rpcMethod = "app.DoSomething"
		req       = fmt.Sprintf(`{"jsonrpc":2.0, "id":11, "method":"%v"}`, rpcMethod)
		respCode  = 200
		respBody  = `{"jsonrpc":"2.0","id":1,"error":{"code":100,"message":"someerrormessage", "data":{"field": "val"}}}`
	)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		require.Equal(t, req.Method, "POST")
		require.Equal(t, req.Header["Content-Type"], []string{"application/json"})

		type rpcRequest struct {
			Method string
		}

		rpcReq := rpcRequest{}

		err := json.NewDecoder(req.Body).Decode(&rpcReq)
		require.NoError(t, err)
		require.Equal(t, rpcMethod, rpcReq.Method)

		w.WriteHeader(respCode)
		_, err = w.Write([]byte(respBody))
		require.NoError(t, err)
	})

	mockServer := httptest.NewServer(serverMux)

	type resp struct {
		Error RpcError `json:"error"`
	}

	var got resp

	err := CallRPC(mockServer.URL, []byte(req), &got)
	require.NoError(t, err)

	want := resp{
		Error: RpcError{
			Code:    100,
			Message: "someerrormessage",
			Data:    map[string]interface{}{"field": "val"},
		},
	}

	require.Equal(t, want, got)
}

func TestRpcError_Error(t *testing.T) {
	type nestedErrData struct {
		s string
	}

	type errData struct {
		i int
		s string
		d nestedErrData
	}

	err := &RpcError{
		Code:    100,
		Message: "someerrormessage",
		Data: errData{
			i: 55,
			s: "moreerrorstring",
			d: nestedErrData{
				s: "evenmorestring",
			},
		},
	}

	wantErrMsg := "code = 100, message = someerrormessage, data = {55 moreerrorstring {evenmorestring}}"

	require.EqualError(t, err, wantErrMsg)
}

func Test_CallREST_response_match_error(t *testing.T) {
	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		require.Equal(t, req.Method, "GET")
		if req.RequestURI == "/upcheck" {
			// Send response to be tested
			rw.Write([]byte("I'am up!"))
		} else {
			http.Error(rw, "unsupported uri", http.StatusInternalServerError)
		}
	}))
	// Close the server when test finishes
	defer server.Close()

	// test if response matches
	res, err := CallREST(server.URL+"/upcheck", "GET", []byte(""))
	assert.NoError(t, err)
	expected := "I'am up!"
	assert.Equal(t, res, expected)

	// test if error is handled
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

type UserData struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}
type DummyUserdResp struct {
	Result UserData `json:"result"`
	Error  error    `json:"error"`
}

func Test_CallRPC_validResult_invalidResult_json_error(t *testing.T) {
	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		require.Equal(t, req.Method, "POST")
		if req.RequestURI == "/blocknumber" {
			// Send response to be tested
			rw.Write([]byte(`{"jsonrpc":"2.0","id":67,"result":"0x27a9"}`))
		} else if req.RequestURI == "/userdata/valid" {
			// Send response to be tested
			rw.Write([]byte(`{"jsonrpc":"2.0","id":67,"result":{"name":"amalraj","age":41}}`))
		} else if req.RequestURI == "/userdata/jsonerror" {
			// Send response to be tested
			rw.Write([]byte(`{"jsonrpc":"2.0","id":67,"result":{"name":"amalraj","age":"abc"}}`))
		} else {
			http.Error(rw, "unsupported uri", http.StatusInternalServerError)
		}
	}))
	// Close the server when test finishes
	defer server.Close()

	// test if json decode works fine with interface{} type in result
	var bnResp DummyBlocknumResp
	err := CallRPC(server.URL+"/blocknumber", []byte("dummy req"), &bnResp)
	assert.NoError(t, err)
	assert.NotNil(t, bnResp)
	assert.NoError(t, bnResp.Error)
	assert.NotNil(t, bnResp.Result)
	blkNumExpected := "0x27a9"
	blkNumActual := bnResp.Result.(string)
	assert.Equal(t, blkNumActual, blkNumExpected)

	// test if json decode works fine with wrong json field name in result
	var badResp DummyBlocknumInvalidResp
	err = CallRPC(server.URL+"/blocknumber", []byte("dummy req"), &badResp)
	assert.NoError(t, err)
	assert.NotNil(t, badResp)
	assert.Nil(t, badResp.ResultDum)

	// test if json decode works fine with proper struct in result
	var validUserResp DummyUserdResp
	err = CallRPC(server.URL+"/userdata/valid", []byte("dummy req"), &validUserResp)
	assert.NoError(t, err)
	assert.NotNil(t, validUserResp)
	assert.NoError(t, validUserResp.Error)
	assert.NotNil(t, validUserResp.Result)
	assert.Equal(t, validUserResp.Result.Name, "amalraj")
	assert.Equal(t, validUserResp.Result.Age, 41)

	// test if json decode fails when result format is wrong
	var userResp DummyUserdResp
	err = CallRPC(server.URL+"/userdata/jsonerror", []byte("dummy req"), &userResp)
	assert.Error(t, err)
}
