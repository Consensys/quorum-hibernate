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

// validates if the node can be hibernated. returns error if it cannot be
// hibernated. The logic used for checking if the node can be hibernated or
// not is as below:
// 1. check if the node is a signer. if not return nil
// 2. if the node is a signer, get the total number of signers for the network,
//    geth the signer metrics for the last 100 blocks
// 3. get the number of signer nodes down by checking if a signer node had
//    signed a block in the last cycle. If not consider the node to be down
// 4. Once the number of signer nodes that down is calculated, check if the
//    current node can go down based on already down nodes and total number
//    signer nodes.
// For clique the requirement is to have 51% of the nodes up and running
func (c *CliqueConsensus) ValidateShutdown() error {
	// get coinbase accout
	coinbase, err := c.getCoinBaseAccount()
	if err != nil {
		log.Error("failed to read the coinbase account", "err", err)
		return err
	}

	// get all signers
	signers, err := c.getSigners()
	if err != nil {
		log.Error("failed to read the coinbase account", "err", err)
		return err
	}

	isSigner := false
	for _, signer := range signers {
		if signer == coinbase {
			isSigner = true
		}
	}
	// not signer account return nil
	if !isSigner {
		return nil
	}

	curBlockNum, err := c.getCurrentBlockNumber()
	if err != nil {
		log.Error("failed to read current block number", "err", err)
		return err
	}

	// get the signing status of the network
	status, err := c.getConsensusStatus()
	if err != nil {
		log.Error("failed to get the signing status for the network", "err", err)
		return err
	}

	nodesDown := 0
	totalSigners := int64(len(signers))
	// check if the coinbase account is one of the signers
	for _, v := range *status {
		proposed, err := strconv.ParseInt(v.LastProposedBlockNumber[2:], 16, 64)
		if err != nil {
			log.Error("error is parsing value", "err", err)
			return err
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
		return errors.New(errMsg)
	}
	return nil
}
