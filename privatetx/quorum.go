package privatetx

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

type QuorumTxHandler struct {
	cfg *types.NodeConfig
}

const (
	ethSendTx  = "eth_sendTransaction"
	privateFor = "privateFor"
)

func NewQuorumTxHandler(cfg *types.NodeConfig) TxHandler {
	return &QuorumTxHandler{cfg: cfg}
}

// IsPrivateTx implements TxHandler.IsPrivateTx
func (q QuorumTxHandler) IsPrivateTx(msg []byte) ([]string, error) {
	if containsPrivateTxKeyWords(string(msg)) {
		if tx, err := decodePvtTx(msg); err != nil {
			log.Error("IsPrivateTx - failed to unmarshal private tx from request", "err", err)
			return nil, fmt.Errorf("IsPrivateTx - failed to unmarshal private tx from request err=%v", err)
		} else {
			if tx.Method == ethSendTx {
				log.Info("IsPrivateTx - private transaction request")
				return tx.Params[0].PrivateFor, nil
			}
			return nil, nil
		}
	}
	return nil, nil
}

// TODO needs to be expanded to cover private tx for all apis like contract extension
func containsPrivateTxKeyWords(bodyStr string) bool {
	return strings.Contains(bodyStr, ethSendTx) || strings.Contains(bodyStr, privateFor)
}

func decodePvtTx(body []byte) (types.EthTransaction, error) {
	tx := types.EthTransaction{}
	err := json.Unmarshal(body, &tx)
	if err != nil {
		return types.EthTransaction{}, err
	} else {
		log.Debug("decodePvtTx - tx details", "Tx", tx)
	}
	return tx, nil
}
