package consensus

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/core/types"

	"github.com/ConsenSysQuorum/node-manager/core"

	"github.com/ConsenSysQuorum/node-manager/log"
)

type IstanbulSealActivity struct {
	NumBlocks      int            `json:"numBlocks"`
	SealerActivity map[string]int `json:"sealerActivity"`
}

type IstanbulSealActivityResp struct {
	Result IstanbulSealActivity `json:"result"`
	Error  error                `json:"error"`
}

type IstanbulIsValidatorResp struct {
	Result bool  `json:"result"`
	Error  error `json:"error"`
}

type IstanbulConsensus struct {
	cfg    *types.NodeConfig
	client *http.Client
}

const (
	validatorDownSealDiff = 3

	IstanbulStatusReq      = `{"jsonrpc":"2.0", "method":"istanbul_status", "params":[], "id":67}`
	IstanbulIsValidatorReq = `{"jsonrpc":"2.0", "method":"istanbul_isValidator", "params":[], "id":67}`
)

func NewIstanbulConsensus(qn *types.NodeConfig) Consensus {
	return &IstanbulConsensus{cfg: qn, client: core.NewHttpClient()}
}

func (r *IstanbulConsensus) getIstanbulSealerActivity(qrmRpcUrl string) (*IstanbulSealActivity, error) {
	var respResult IstanbulSealActivityResp
	if err := core.MakeRpcCall(qrmRpcUrl, []byte(IstanbulStatusReq), &respResult); err != nil {
		return nil, err
	}
	return &respResult.Result, respResult.Error
}

func (r *IstanbulConsensus) getIstanbulIsValidator(qrmRpcUrl string) (bool, error) {
	var respResult IstanbulIsValidatorResp
	if err := core.MakeRpcCall(qrmRpcUrl, []byte(IstanbulIsValidatorReq), &respResult); err != nil {
		return false, err
	}
	return respResult.Result, respResult.Error
}

// TODO - if the number of validators are more than 64 this will not work as expected as signers return data for last 64 blocks only
func (r *IstanbulConsensus) ValidateShutdown() error {
	isValidator, err := r.getIstanbulIsValidator(r.cfg.BasicConfig.GethRpcUrl)
	if err != nil {
		log.Error("ValidateShutdown - istanbul isValidator check failed", "err", err)
		return err
	}

	if !isValidator {
		log.Info("ValidateShutdown - istanbul non-validator node, ok to shutdown")
		return nil
	}

	activity, err := r.getIstanbulSealerActivity(r.cfg.BasicConfig.GethRpcUrl)
	if err != nil {
		log.Error("ValidateShutdown - istanbul status check failed", "err", err)
		return err
	}

	totalValidators := len(activity.SealerActivity)
	maxSealBlocks := activity.NumBlocks / totalValidators
	zeroBlockSealCnt := 0
	for _, numBlocks := range activity.SealerActivity {
		if numBlocks == 0 {
			zeroBlockSealCnt++
		}
	}

	log.Info("ValidateShutdown - istanbul consensus check", "totalValidators", totalValidators, "maxSealBlocks", maxSealBlocks, "activity", activity.SealerActivity)

	if zeroBlockSealCnt == totalValidators {
		return errors.New("istanbul consensus check - looks like all validators are down")
	}

	var percMap = make(map[string]int)
	var numNodesDown = 0
	for id, numBlocks := range activity.SealerActivity {
		sealDiff := maxSealBlocks - numBlocks
		if sealDiff >= validatorDownSealDiff {
			numNodesDown++
		}
		percMap[id] = sealDiff
	}

	numOfNodesThatCanBeDown := (totalValidators - 1) / 3

	log.Info("ValidateShutdown - istanbul consensus check", "numOfNodesThatCanBeDown", numOfNodesThatCanBeDown, "numNodesDown", numNodesDown, "percMap", percMap)

	if numNodesDown >= numOfNodesThatCanBeDown {
		errMsg := fmt.Sprintf("istanbul consensus check - the number of nodes currently down has reached threshold, numOfNodesThatCanBeDown:%d numNodesDown:%d", numOfNodesThatCanBeDown, numNodesDown)
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	return nil
}
