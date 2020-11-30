package privatetx

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

type QuorumTxHandler struct {
	cfg *types.NodeConfig
}

const (
	ethSendTx    = "eth_sendTransaction"
	ethSendRawTx = "eth_sendRawPrivateTransaction"

	ethSignTx      = "eth_signTransaction"
	personalSignTx = "personal_signTransaction"

	privateFor = "privateFor"
)

func NewQuorumTxHandler(cfg *types.NodeConfig) TxHandler {
	return &QuorumTxHandler{cfg: cfg}
}

// IsPrivateTx implements TxHandler.IsPrivateTx
func (q QuorumTxHandler) IsPrivateTx(msg []byte) ([]string, error) {
	if containsPrivateTxKeyWords(string(msg)) {
		if keys, err := decodePvtTx(msg); err != nil {
			log.Error("IsPrivateTx - failed to unmarshal private tx from request", "err", err)
			return nil, fmt.Errorf("IsPrivateTx - failed to unmarshal private tx from request err=%v", err)
		} else {
			log.Info("AJ KEYS", "K", keys)
			return keys, nil
		}
	}
	return nil, nil
}

// TODO improvement -  convert it to a regular expression
func containsPrivateTxKeyWords(bodyStr string) bool {
	return strings.Contains(bodyStr, privateFor)
}

func decodePvtTx(body []byte) ([]string, error) {
	var txMap map[string]interface{}
	err := json.Unmarshal(body, &txMap)
	if err != nil {
		return nil, err
	} else {
		log.Debug("decodePvtTx - txMap details", "Tx", txMap)
	}
	var method string
	var ok bool
	if _, ok = txMap["method"]; !ok {
		return nil, nil
	}
	method = txMap["method"].(string)
	if method == ethSendTx || method == ethSignTx || method == personalSignTx {
		if params, ok := txMap["params"].([]interface{}); ok {
			if len(params) == 0 {
				log.Warn("decodePvtTx - params len is zero", "method", method)
				return nil, nil
			}
			if dataMap, ok := params[0].(map[string]interface{}); ok {
				if keys, ok := dataMap["privateFor"]; ok {
					return privKeys(keys), nil
				} else {
					return nil, errors.New("privateFor missing in " + method)
				}
			} else {
				return nil, errors.New("tx data map missing in params " + method)
			}
		} else {
			return nil, errors.New("params missing in " + method)
		}
	} else if method == ethSendRawTx {
		if params, ok := txMap["params"].([]interface{}); ok {
			if len(params) == 0 {
				log.Warn("decodePvtTx - params len is zero", "method", method)
				return nil, nil
			}

			if len(params) == 1 {
				log.Warn("decodePvtTx - params is having only one parameter", "method", method, "params", params)
				return nil, nil
			}

			for _, param := range params {
				if pvtMap, ok := param.(map[string]interface{}); ok {
					if keys, ok := pvtMap["privateFor"]; ok {
						return privKeys(keys), nil
					}
				}
			}
			return nil, fmt.Errorf("privateFor missing in %s params %v", method, params)
		} else {
			return nil, errors.New("params missing in " + ethSendRawTx)
		}
	} else {
		log.Warn("decodePvtTx - unhandled private transaction", "method", method)
	}
	return nil, nil
}

func privKeys(keys interface{}) []string {
	arrK := keys.([]interface{})
	var privKeys []string
	for _, v := range arrK {
		s := v.(string)
		privKeys = append(privKeys, s)
	}
	return privKeys
}
