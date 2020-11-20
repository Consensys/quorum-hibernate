package consensus

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

const validatorDownSealDiff = 3

func NewIstanbulConsensus(qn *types.NodeConfig) Consensus {
	return &IstanbulConsensus{cfg: qn, client: core.NewHttpClient()}
}

func (r *IstanbulConsensus) getIstanbulSealerActivity(qrmRpcUrl string) (*IstanbulSealActivity, error) {
	istanbulStatusReq := []byte(`{"jsonrpc":"2.0", "method":"istanbul_status", "params":[], "id":67}`)
	req, err := http.NewRequest("POST", qrmRpcUrl, bytes.NewBuffer(istanbulStatusReq))
	if err != nil {
		return nil, fmt.Errorf("istanbul status - creating request failed err=%v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("istanbul status do req failed err=%v", err)
	}
	var respResult IstanbulSealActivityResp
	if resp.StatusCode == http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Debug("istanbul status response Body:", string(body))
		jerr := json.Unmarshal(body, &respResult)
		if jerr == nil {
			log.Debug("istanbul status - response OK", "from", qrmRpcUrl, "result", respResult)
		} else {
			log.Error("istanbul status response result json decode failed", "err", jerr)
			return nil, err
		}
	}
	return &respResult.Result, respResult.Error
}

func (r *IstanbulConsensus) getIstanbulIsValidator(qrmRpcUrl string) (bool, error) {
	istanbulIsValidatorReq := []byte(`{"jsonrpc":"2.0", "method":"istanbul_isValidator", "params":[], "id":67}`)
	req, err := http.NewRequest("POST", qrmRpcUrl, bytes.NewBuffer(istanbulIsValidatorReq))
	if err != nil {
		return false, fmt.Errorf("istanbul isValidator - creating request failed err=%v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("istanbul isValidator do req failed err=%v", err)
	}
	var respResult IstanbulIsValidatorResp
	if resp.StatusCode == http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Debug("istanbul isValidator response Body:", string(body))
		jerr := json.Unmarshal(body, &respResult)
		if jerr == nil {
			log.Debug("istanbul isValidator - response OK", "from", qrmRpcUrl, "result", respResult)
		} else {
			log.Error("istanbul isValidator response result json decode failed", "err", jerr)
			return false, err
		}
	}
	return respResult.Result, respResult.Error
}

func (r *IstanbulConsensus) ValidateShutdown() error {
	isValidator, err := r.getIstanbulIsValidator(r.cfg.BasicConfig.GethRpcUrl)
	if err != nil {
		log.Error("istanbul isValidator check failed", "err", err)
		return err
	}

	if !isValidator {
		log.Info("istanbul non-validator node, ok to shutdown")
		return nil
	}

	activity, err := r.getIstanbulSealerActivity(r.cfg.BasicConfig.GethRpcUrl)
	if err != nil {
		log.Error("istanbul status check failed", "err", err)
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

	log.Info("istanbul consensus check", "totalValidators", totalValidators, "maxSealBlocks", maxSealBlocks, "activity", activity.SealerActivity)

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

	log.Info("istanbul consensus check", "numOfNodesThatCanBeDown", numOfNodesThatCanBeDown, "numNodesDown", numNodesDown, "percMap", percMap)

	if numNodesDown >= numOfNodesThatCanBeDown {
		errMsg := fmt.Sprintf("istanbul consensus check - the number of nodes currently down has reached threshold, numOfNodesThatCanBeDown:%d numNodesDown:%d", numOfNodesThatCanBeDown, numNodesDown)
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	return nil
}
