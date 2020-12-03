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

// ValidateShutdown implements Consensus.ValidateShutdown
// It validates if the node can be hibernated. returns error if it cannot be
// hibernated. The logic used for checking if the node can be hibernated or
// not is as below:
// 1. check if the node is a signer. if not return nil
// 2. if the node is a signer, get the total number of signers for the network,
//    get the signer metrics for the last 100 blocks
// 3. get the number of signer nodes down by checking if a signer node had
//    signed a block in the last cycle. If not consider the node to be down
// 4. Once the number of signer nodes that are down is calculated, check if the
//    current node can go down based on already down nodes and total number of
//    signer nodes.
// For clique the requirement is to have 51% of the nodes up and running
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
		log.Error("ValidateShutdown - failed to read the signers", "err", err)
		return isSigner, err
	}

	for _, signer := range signers {
		if signer == coinbase {
			isSigner = true
			break
		}
	}
	// not signer account, ok to stop. return nil
	if !isSigner {
		return isSigner, nil
	}

	curBlockNum, err := c.getCurrentBlockNumber()
	if err != nil {
		log.Error("ValidateShutdown - failed to read current block number", "err", err)
		return isSigner, err
	}

	// get the signing status of the network
	status, err := c.getConsensusStatus()
	if err != nil {
		log.Error("ValidateShutdown - failed to get the signing status for the network", "err", err)
		return isSigner, err
	}

	nodesDown := 0
	totalSigners := int64(len(signers))
	// calculate the no of nodes that are down
	for _, v := range *status {
		proposed, err := strconv.ParseInt(v.LastProposedBlockNumber[2:], 16, 64)
		if err != nil {
			log.Error("ValidateShutdown - error parsing LastProposedBlockNumber hex value to int value", "err", err)
			return isSigner, err
		}
		if curBlockNum-proposed > totalSigners {
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
