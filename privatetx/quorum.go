package privatetx

import (
	"encoding/json"
	"fmt"
	"github.com/ConsenSysQuorum/node-manager/config"
	"strings"

	"github.com/ConsenSysQuorum/node-manager/log"
)

type QuorumTxHandler struct {
	cfg *config.NodeConfig
}

const (
	// send transaction
	ethSendTx    = "eth_sendTransaction"
	ethSendRawTx = "eth_sendRawPrivateTransaction"

	// sign transaction
	ethSignTx      = "eth_signTransaction"
	personalSignTx = "personal_signTransaction"

	// private state extension
	pvtStateExtAppr = "quorumExtension_approveExtension"
	pvtStateExtExt  = "quorumExtension_extendContract"

	// estimate gas
	estimateGas = "eth_estimateGas"
	privateFor  = "privateFor"
)

// map to validate requests that need privacy manager keys to be extracted.
var pvtReqParamMap = map[string]int{
	// method name : number items expected in params array
	ethSendTx:       1,
	ethSendRawTx:    2,
	ethSignTx:       1,
	personalSignTx:  2,
	pvtStateExtExt:  4,
	pvtStateExtAppr: 3,
	estimateGas:     1,
}

func NewQuorumTxHandler(cfg *config.NodeConfig) TxHandler {
	return &QuorumTxHandler{cfg: cfg}
}

// IsPrivateTx implements TxHandler.IsPrivateTx
func (q QuorumTxHandler) IsPrivateTx(msg []byte) ([]string, error) {
	if containsPrivateTxKeyWords(string(msg)) {
		return decodePvtTx(msg)
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
	keys := validatePrivateReq(txMap, method)
	return keys, nil
}

func validatePrivateReq(txMap map[string]interface{}, method string) []string {
	var expArgs int
	var ok bool
	if expArgs, ok = pvtReqParamMap[method]; !ok {
		log.Warn("validatePrivateReq - method is missing in private request param map", "method", method)
		return nil
	}
	if params, ok := txMap["params"].([]interface{}); ok {
		if len(params) == 0 {
			log.Warn("validatePrivateReq - params len is zero", "method", method)
			return nil
		}

		if len(params) != expArgs {
			log.Warn("validatePrivateReq - params does not have enough arguments", "expected", expArgs, "method", method, "params", params)
			return nil
		}

		for _, param := range params {
			if pvtMap, ok := param.(map[string]interface{}); ok {
				if keys, ok := pvtMap["privateFor"]; ok {
					return privKeys(keys)
				}
			}
		}
		log.Warn(fmt.Sprintf("privateFor missing in %s params %v", method, params), "txmap", txMap)
		return nil
	} else {
		log.Warn("params missing in " + method)
		return nil
	}
	return nil
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
