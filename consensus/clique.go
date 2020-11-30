package consensus

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

type CliqueConsensus struct {
	cfg    *types.NodeConfig
	client *http.Client
}

// IstanbulSealActivity represents output of RPC istanbul_status
type CliqueStatus struct {
	InTurnPercent  int
	NumBlocks      int            `json:"numBlocks"`
	SealerActivity map[string]int `json:"sealerActivity"`
}

type CliqueStatusResp struct {
	Result CliqueStatus `json:"result"`
	Error  error        `json:"error"`
}

type CoinBaseResp struct {
	CoinBaseAccount string `json:"result"`
	Error           error  `json:"error"`
}

const (
	allowedSigningDiff = 2

	// Clique RPC APIs
	CliqueStatusReq = `{"jsonrpc":"2.0", "method":"clique_status", "params":[], "id":67}`
	CoinBaseReq     = `{"jsonrpc":"2.0", "method":"eth_coinbase", "id":67}`
)

func NewCliqueConsensus(qn *types.NodeConfig) Consensus {
	return &CliqueConsensus{cfg: qn, client: core.NewHttpClient()}
}

// returns true if the coinbase account of the node is one of the signer accounts
func (c *CliqueConsensus) getCoinBaseAccount() (string, error) {
	var result CoinBaseResp
	if err := core.CallRPC(c.cfg.BasicConfig.BcClntRpcUrl, []byte(CoinBaseReq), &result); err != nil {
		return "", err
	}
	return result.CoinBaseAccount, nil
}

func (c *CliqueConsensus) getConsensusStatus() (*CliqueStatus, error) {
	var respResult CliqueStatusResp
	if err := core.CallRPC(c.cfg.BasicConfig.BcClntRpcUrl, []byte(CliqueStatusReq), &respResult); err != nil {
		return nil, err
	}
	return &respResult.Result, respResult.Error
}

func (c *CliqueConsensus) ValidateShutdown() error {
	// get the signing status of the network
	status, err := c.getConsensusStatus()
	if err != nil {
		log.Error("failed to get the signing status for the network", "err", err)
		return err
	}

	coinbase, err := c.getCoinBaseAccount()
	if err != nil {
		log.Error("failed to read the coinbase account")
		return err
	}

	// check if the coinbase account is one of the signer accounts.
	// if not return nil
	if _, ok := status.SealerActivity[coinbase]; !ok {
		return nil
	}

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
		return errors.New(errMsg)
	}
	return nil
}
