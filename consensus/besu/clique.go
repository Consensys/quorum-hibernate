package quorum

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ConsenSysQuorum/node-manager/consensus"
	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

type CliqueConsensus struct {
	cfg    *types.NodeConfig
	client *http.Client
}

//
type BlockNumberResp struct {
	Result string `json:"result"`
	Error  error  `json:"error"`
}

type CliqueSignersResp struct {
	Result []string `json:"result"`
	Error  error    `json:"error"`
}

// clique seal status represents output of RPC clique_getSignerMetrics
type CliqueStatus struct {
	Address                 string `json:"address"`
	ProposedBlockCount      string `json:"proposedBlockCount"`
	LastProposedBlockNumber string `json:"lastProposedBlockNumber"`
}

type CliqueStatusResp struct {
	Result []CliqueStatus `json:"result"`
	Error  error          `json:"error"`
}

type CoinBaseResp struct {
	CoinBaseAccount string `json:"result"`
	Error           error  `json:"error"`
}

const (
	allowedSigningDiff = 2

	// Clique RPC APIs
	CliqueStatusReq = `{"jsonrpc":"2.0", "method":"clique_getSignerMetrics", "params":[], "id":67}`
	CoinBaseReq     = `{"jsonrpc":"2.0", "method":"eth_coinbase", "id":67}`
	BlockNumberReq  = `{"jsonrpc":"2.0", "method":"eth_blockNumber", "params":[], "id":67}`
	CLiqueSigners   = `{"jsonrpc":"2.0", "method":"clique_getSigners", "params":[], "id":67}`
)

func NewCliqueConsensus(qn *types.NodeConfig) consensus.Consensus {
	return &CliqueConsensus{cfg: qn, client: core.NewHttpClient()}
}

func (c *CliqueConsensus) getCurrentBlockNumber() (int64, error) {
	var result BlockNumberResp
	if err := core.CallRPC(c.cfg.BasicConfig.BcClntRpcUrl, []byte(BlockNumberReq), &result); err != nil {
		return 0, err
	}
	blockNumber, err := strconv.ParseInt(result.Result[2:], 16, 64)
	if err != nil {
		return 0, err
	}
	return blockNumber, nil
}

func (c *CliqueConsensus) getSigners() ([]string, error) {
	var result CliqueSignersResp
	if err := core.CallRPC(c.cfg.BasicConfig.BcClntRpcUrl, []byte(CLiqueSigners), &result); err != nil {
		return nil, err
	}

	return result.Result, nil
}

// returns true if the coinbase account of the node is one of the signer accounts
func (c *CliqueConsensus) getCoinBaseAccount() (string, error) {
	var result CoinBaseResp
	if err := core.CallRPC(c.cfg.BasicConfig.BcClntRpcUrl, []byte(CoinBaseReq), &result); err != nil {
		return "", err
	}
	return result.CoinBaseAccount, nil
}

func (c *CliqueConsensus) getConsensusStatus() (*[]CliqueStatus, error) {
	var respResult CliqueStatusResp
	if err := core.CallRPC(c.cfg.BasicConfig.BcClntRpcUrl, []byte(CliqueStatusReq), &respResult); err != nil {
		return nil, err
	}
	return &respResult.Result, respResult.Error
}

func (c *CliqueConsensus) ValidateShutdown() (bool, error) {
	isSigner := false

	// get coinbase accout
	coinbase, err := c.getCoinBaseAccount()
	if err != nil {
		log.Error("failed to read the coinbase account", "err", err)
		return isSigner, err
	}

	// get all signers
	signers, err := c.getSigners()
	if err != nil {
		log.Error("failed to read the coinbase account", "err", err)
		return isSigner, err
	}

	for _, signer := range signers {
		if signer == coinbase {
			isSigner = true
		}
	}
	// not signer account return nil
	if !isSigner {
		return isSigner, nil
	}

	curBlockNum, err := c.getCurrentBlockNumber()
	if err != nil {
		log.Error("failed to read current block number", "err", err)
		return isSigner, err
	}

	// get the signing status of the network
	status, err := c.getConsensusStatus()
	if err != nil {
		log.Error("failed to get the signing status for the network", "err", err)
		return isSigner, err
	}

	totalSigners := int64(len(signers))
	minProposedBlock := curBlockNum - totalSigners

	nodesDown := 0

	// check if the coinbase account is one of the signers
	for _, v := range *status {
		proposed, err := strconv.ParseInt(v.LastProposedBlockNumber[2:], 16, 64)
		if err != nil {
			log.Error("error is parsing value", "err", err)
			return isSigner, err
		}
		if proposed < minProposedBlock {
			nodesDown++
		}
	}

	allowedDownNodes := (totalSigners - 1) / 2

	if nodesDown >= int(allowedDownNodes) {
		errMsg := fmt.Sprintf("clique consensus check - the number of nodes currently down has reached threshold, numOfNodesThatCanBeDown:%d numNodesDown:%d", allowedDownNodes, nodesDown)
		// current node cannot go down. return error
		log.Error(errMsg)
		return isSigner, errors.New(errMsg)
	}

	return isSigner, nil
}
