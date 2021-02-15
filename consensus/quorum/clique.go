package quorum

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ConsenSys/quorum-hibernate/config"

	"github.com/ConsenSys/quorum-hibernate/consensus"
	"github.com/ConsenSys/quorum-hibernate/core"
	"github.com/ConsenSys/quorum-hibernate/log"
)

type CliqueConsensus struct {
	cfg    *config.Node
	client *http.Client
}

// CliqueStatus represents output of RPC clique_status
type CliqueStatus struct {
	InTurnPercent  float32        `json:"inTurnPercent"`
	NumBlocks      int            `json:"numBlocks"`
	SealerActivity map[string]int `json:"sealerActivity"`
}

type CliqueStatusResp struct {
	Result CliqueStatus   `json:"result"`
	Error  *core.RpcError `json:"error"`
}

type CoinBaseResp struct {
	CoinBaseAccount string         `json:"result"`
	Error           *core.RpcError `json:"error"`
}

const (
	allowedSigningDiff = 2

	// Clique RPC APIs
	CliqueStatusReq = `{"jsonrpc":"2.0", "method":"clique_status", "params":[], "id":67}`
	CoinBaseReq     = `{"jsonrpc":"2.0", "method":"eth_coinbase", "id":67}`
)

func NewCliqueConsensus(qn *config.Node, c *http.Client) consensus.Consensus {
	return &CliqueConsensus{cfg: qn, client: c}
}

// returns true if the coinbase account of the node is one of the signer accounts
func (c *CliqueConsensus) getCoinBaseAccount() (string, error) {
	var result CoinBaseResp
	if err := core.CallRPC(c.client, c.cfg.BasicConfig.BlockchainClient.BcClntRpcUrl, []byte(CoinBaseReq), &result); err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", result.Error
	}
	return result.CoinBaseAccount, nil
}

func (c *CliqueConsensus) getConsensusStatus() (*CliqueStatus, error) {
	var respResult CliqueStatusResp
	if err := core.CallRPC(c.client, c.cfg.BasicConfig.BlockchainClient.BcClntRpcUrl, []byte(CliqueStatusReq), &respResult); err != nil {
		return nil, err
	}
	if respResult.Error != nil {
		return nil, respResult.Error
	}
	return &respResult.Result, nil
}

func (c *CliqueConsensus) ValidateShutdown() (bool, error) {
	isSigner := false
	// get the signing status of the network
	status, err := c.getConsensusStatus()
	if err != nil {
		log.Error("failed to get the signing status for the network", "err", err)
		return isSigner, err
	}

	coinbase, err := c.getCoinBaseAccount()
	if err != nil {
		log.Error("failed to read the coinbase account")
		return isSigner, err
	}

	// check if the coinbase account is one of the signer accounts.
	// if not return nil
	if _, ok := status.SealerActivity[coinbase]; !ok {
		return isSigner, nil
	}

	isSigner = true

	// the node account is a signer account and hence need to check if it can go down
	totalSealers := len(status.SealerActivity)
	maxSealingPerNode := status.NumBlocks / totalSealers
	maxDownNodesAllowed := (totalSealers - 1) / 2
	potentialDownNodes := 0

	for _, v := range status.SealerActivity {
		if maxSealingPerNode-v >= allowedSigningDiff {
			potentialDownNodes++
		}
	}
	if potentialDownNodes >= maxDownNodesAllowed {
		errMsg := fmt.Sprintf("clique consensus check - the number of nodes currently down has reached threshold, numOfNodesThatCanBeDown:%d numNodesDown:%d", maxDownNodesAllowed, potentialDownNodes)
		// current node cannot go down. return error
		log.Error(errMsg)
		return isSigner, errors.New(errMsg)
	}
	return isSigner, nil
}
