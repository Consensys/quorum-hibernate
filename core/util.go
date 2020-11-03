package core

import (
	"encoding/json"
	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
	"strings"
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
