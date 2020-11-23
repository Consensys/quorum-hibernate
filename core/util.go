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

// GetRandomRetryWaitTime returns a random wait time within a range of 100 to 1000
func GetRandomRetryWaitTime() int {
	rand.Seed(time.Now().UnixNano())
	min := 100
	max := 1000
	return rand.Intn(max-min+1) + min
}

// CallRPC makes a rpc call to rpcUrl. It makes http post req with rpcTeq as body.
// The returned JSON result is decoded into resData.
func CallRPC(rpcUrl string, rpcReq []byte, resData interface{}) error {
	client := NewHttpClient()
	log.Debug("CallRPC - making rpc call", "req", string(rpcReq))
	req, err := http.NewRequest("POST", rpcUrl, bytes.NewBuffer(rpcReq))
	if err != nil {
		return fmt.Errorf("CallRPC - creating request failed err=%v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("CallRPC - do req failed err=%v", err)
	}
	if resp.StatusCode == http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Debug("CallRPC - response Body:", string(body))
		err := json.Unmarshal(body, resData)
		if err == nil {
			log.Debug("CallRPC - response OK", "from", rpcUrl, "result", resData)
		} else {
			log.Error("CallRPC - response json decode failed", "err", err)
			return err
		}
	} else {
		log.Error("CallRPC - response status failed, not OK", "status", resp.Status)
		return fmt.Errorf("CallRPC - response status failed, not OK, status=%s", resp.Status)
	}
	return nil
}
