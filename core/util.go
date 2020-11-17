package core

import (
	"encoding/json"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

func IsPrivateTransaction(bodyStr string) bool {
	return strings.Contains(bodyStr, "eth_sendTransaction") && strings.Contains(bodyStr, "privateFor")
}

func GetPrivateTx(body []byte) (types.EthTransaction, error) {
	tx := types.EthTransaction{}
	err := json.Unmarshal(body, &tx)
	if err != nil {
		return types.EthTransaction{}, err
	} else {
		log.Debug("tx details", "Tx", tx)
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
