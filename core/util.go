package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/ConsenSysQuorum/node-manager/log"
)

var client *http.Client

func init() {
	client = NewHttpClient()
}

// NewHttpClient returns a new customized http client
func NewHttpClient() *http.Client {
	var netTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: HttpClientRequestDialerTimeout,
		}).DialContext,
		TLSHandshakeTimeout: TLSHandshakeTimeout,
	}
	var netClient = &http.Client{
		Timeout:   HttpClientRequestTimeout,
		Transport: netTransport,
	}
	return netClient
}

// RandomInt returns a random int within a range of min to max
func RandomInt(min int, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}

func CallRPC(rpcUrl string, rpcReq []byte, resData interface{}) error {
	_, err := httpRequest(rpcUrl, "POST", rpcReq, resData, false)
	return err
}

func CallREST(rpcUrl string, method string, rpcReq []byte) (string, error) {
	return httpRequest(rpcUrl, method, rpcReq, nil, true)
}

// httpRequest makes a http request to rpcUrl. It makes http req with rpcReq as body.
// The returned JSON result is decoded into resData.
// resData must be a pointer.
// If http request returns 200 OK, it returns response body decoded into resData
// It returns error if http request does not return 200 OK or json decoding of response fails
// if returnRaw is true it returns the response as string and does not set resData
func httpRequest(rpcUrl string, method string, rpcReq []byte, resData interface{}, returnRaw bool) (string, error) {
	log.Debug("CallRPC - making rpc call", "req", string(rpcReq))
	req, err := http.NewRequest(method, rpcUrl, bytes.NewBuffer(rpcReq))
	if err != nil {
		return "", fmt.Errorf("CallRPC - creating request failed err=%v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("CallRPC - do req failed err=%v", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Debug("CallRPC - response", "body", string(body))
		if returnRaw {
			return string(body), nil
		}
		err := json.Unmarshal(body, resData)
		if err == nil {
			log.Debug("CallRPC - response OK", "from", rpcUrl, "result", resData)
		} else {
			log.Error("CallRPC - response json decode failed", "err", err)
			return "", err
		}
	} else {
		log.Error("CallRPC - response status failed, not OK", "status", resp.Status)
		return "", fmt.Errorf("CallRPC - response status failed, not OK, status=%s", resp.Status)
	}
	return "", nil
}
