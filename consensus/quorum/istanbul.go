package quorum

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/consensus"
	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

// IstanbulSealActivity represents output of RPC istanbul_status
type IstanbulSealActivity struct {
	NumBlocks      int            `json:"numBlocks"`
	SealerActivity map[string]int `json:"sealerActivity"`
}

type IstanbulSealActivityResp struct {
	Result IstanbulSealActivity `json:"result"`
	Error  *core.RpcError       `json:"error"`
}

type IstanbulIsValidatorResp struct {
	Result bool           `json:"result"`
	Error  *core.RpcError `json:"error"`
}

type IstanbulConsensus struct {
	cfg    *types.NodeConfig
	client *http.Client
}

const (
	validatorDownSealDiff = 2

	// Istanbul RPC APIs
	//TODO(cjh) deterministic rpc request ids - check other rpc requests too
	IstanbulStatusReq      = `{"jsonrpc":"2.0", "method":"istanbul_status", "params":[], "id":67}`
	IstanbulIsValidatorReq = `{"jsonrpc":"2.0", "method":"istanbul_isValidator", "params":[], "id":67}`
)

func NewIstanbulConsensus(qn *types.NodeConfig) consensus.Consensus {
	return &IstanbulConsensus{cfg: qn, client: core.NewHttpClient()}
}

func (i *IstanbulConsensus) getIstanbulSealerActivity() (*IstanbulSealActivity, error) {
	var respResult IstanbulSealActivityResp
	if err := core.CallRPC(i.cfg.BasicConfig.BcClntRpcUrl, []byte(IstanbulStatusReq), &respResult); err != nil {
		return nil, err
	}
	if respResult.Error != nil {
		return nil, respResult.Error
	}
	return &respResult.Result, nil
}

func (i *IstanbulConsensus) getIstanbulIsValidator() (bool, error) {
	var respResult IstanbulIsValidatorResp
	if err := core.CallRPC(i.cfg.BasicConfig.BcClntRpcUrl, []byte(IstanbulIsValidatorReq), &respResult); err != nil {
		return false, err
	}
	if respResult.Error != nil {
		return false, respResult.Error
	}
	return respResult.Result, nil
}

// TODO - if the number of validators are more than 64 this will not work as expected as signers return data for last 64 blocks only
// ValidateShutdown implements Consensus.ValidateShutdown
func (i *IstanbulConsensus) ValidateShutdown() (bool, error) {
	isValidator, err := i.getIstanbulIsValidator()
	if err != nil {
		log.Error("ValidateShutdown - istanbul isValidator check failed", "err", err)
		return isValidator, fmt.Errorf("unable to check if istanbul validator: %v", err)
	}

	if !isValidator {
		log.Info("ValidateShutdown - istanbul non-validator node, ok to shutdown")
		return isValidator, nil
	}

	activity, err := i.getIstanbulSealerActivity()
	if err != nil {
		log.Error("ValidateShutdown - istanbul status check failed", "err", err)
		return isValidator, fmt.Errorf("unable to check istanbul sealer status: %v", err)
	}

	totalValidators := len(activity.SealerActivity)

	if totalValidators == 0 {
		return isValidator, errors.New("istanbul consensus check failed - no signers")
	}

	maxSealBlocks := activity.NumBlocks / totalValidators

	if activity.NumBlocks == 0 {
		return isValidator, errors.New("istanbul consensus check failed - block minting not started at network")
	}

	var numNodesDown = 0
	for _, numBlocks := range activity.SealerActivity {
		sealDiff := maxSealBlocks - numBlocks
		if sealDiff >= validatorDownSealDiff {
			numNodesDown++
		}
	}

	numOfNodesThatCanBeDown := (totalValidators - 1) / 3

	log.Debug("ValidateShutdown - istanbul consensus check", "numOfNodesThatCanBeDown", numOfNodesThatCanBeDown, "numNodesDown", numNodesDown, "activityMap", activity)

	if numNodesDown >= numOfNodesThatCanBeDown {
		errMsg := fmt.Sprintf("istanbul consensus check - the number of nodes currently down has reached threshold, numOfNodesThatCanBeDown:%d numNodesDown:%d", numOfNodesThatCanBeDown, numNodesDown)
		log.Error(errMsg)
		return isValidator, errors.New(errMsg)
	}

	return isValidator, nil
}
