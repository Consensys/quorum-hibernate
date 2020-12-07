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

// GetRandomRetryWaitTime returns a random wait time within a range of min to max
func GetRandomRetryWaitTime(min int, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}

// CallRPC makes a rpc call to rpcUrl. It makes http post req with rpcReq as body.
// The returned JSON result is decoded into resData.
// resData must be a pointer.
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("CallRPC - response status failed, not OK", "status", resp.Status)
		return fmt.Errorf("CallRPC - response status failed, not OK, status=%s", resp.Status)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	log.Debug("CallRPC - response Body:", string(body))

	if err := json.Unmarshal(body, resData); err != nil {
		log.Error("CallRPC - response json decode failed", "err", err)
		return err
	}

	log.Debug("CallRPC - response OK", "from", rpcUrl, "result", resData)
	return nil
}
