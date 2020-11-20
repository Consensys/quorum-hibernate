package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

// TODO needs to be expanded to cover private tx for all apis like contract extension
func IsPrivateTransaction(bodyStr string) bool {
	return strings.Contains(bodyStr, "eth_sendTransaction") && strings.Contains(bodyStr, "privateFor")
}

func GetPrivateTx(body []byte) (types.EthTransaction, error) {
	tx := types.EthTransaction{}
	err := json.Unmarshal(body, &tx)
	if err != nil {
		return types.EthTransaction{}, err
	} else {
		log.Debug("GetPrivateTx - tx details", "Tx", tx)
	}
	return tx, nil
}

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

func GetRandomRetryWaitTime() int {
	rand.Seed(time.Now().UnixNano())
	min := 100
	max := 1000
	return rand.Intn(max-min+1) + min
}

func MakeRpcCall(qrmRpcUrl string, rpcReq []byte, resData interface{}) error {
	client := NewHttpClient()
	log.Debug("MakeRpcCall - making rpc call", "req", string(rpcReq))
	req, err := http.NewRequest("POST", qrmRpcUrl, bytes.NewBuffer(rpcReq))
	if err != nil {
		return fmt.Errorf("MakeRpcCall - creating request failed err=%v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("MakeRpcCall - do req failed err=%v", err)
	}
	if resp.StatusCode == http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Debug("MakeRpcCall - response Body:", string(body))
		err := json.Unmarshal(body, resData)
		if err == nil {
			log.Debug("MakeRpcCall - response OK", "from", qrmRpcUrl, "result", resData)
		} else {
			log.Error("MakeRpcCall - response json decode failed", "err", err)
			return err
		}
	} else {
		log.Error("MakeRpcCall - response status failed, not OK", "status", resp.Status)
		return fmt.Errorf("MakeRpcCall - response status failed, not OK, status=%s", resp.Status)
	}
	return nil
}
