package quorum

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/config"

	"github.com/ConsenSysQuorum/node-manager/consensus"
	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/log"
)

// RaftClusterEntry represents entries from the output of rpc method raft_cluster
type RaftClusterEntry struct {
	Hostname   string `json:"hostName"`
	NodeActive bool   `json:"nodeActive"`
	NodeId     string `json:"nodeId"`
	P2pPort    int    `json:"p2pPort"`
	RaftId     int    `json:"raftId"`
	Role       string `json:"role"`
}

type RaftClusterResp struct {
	Result []RaftClusterEntry `json:"result"`
	Error  *core.RpcError     `json:"error"`
}

type RaftRoleResp struct {
	Result string         `json:"result"`
	Error  *core.RpcError `json:"error"`
}

type RaftConsensus struct {
	cfg    *config.Node
	client *http.Client
}

const (
	// Raft roles
	MINTER  = "minter"
	LEARNER = "learner"

	// Raft RPC apis
	RaftRoleReq    = `{"jsonrpc":"2.0", "method":"raft_role", "params":[], "id":67}`
	RaftClusterReq = `{"jsonrpc":"2.0", "method":"raft_cluster", "params":[], "id":67}`
)

func NewRaftConsensus(qn *config.Node, c *http.Client) consensus.Consensus {
	return &RaftConsensus{cfg: qn, client: c}
}

func (r *RaftConsensus) getRole(rpcUrl string) (string, error) {
	var respResult RaftRoleResp
	if err := core.CallRPC(r.client, rpcUrl, []byte(RaftRoleReq), &respResult); err != nil {
		return "", err
	}
	if respResult.Error != nil {
		return "", respResult.Error
	}
	return respResult.Result, nil
}

func (r *RaftConsensus) getRaftClusterInfo(rpcUrl string) ([]RaftClusterEntry, error) {
	var respResult RaftClusterResp
	if err := core.CallRPC(r.client, rpcUrl, []byte(RaftClusterReq), &respResult); err != nil {
		return nil, err
	}
	if respResult.Error != nil {
		return nil, respResult.Error
	}
	return respResult.Result, nil
}

// ValidateShutdown implements Consensus.ValidateShutdown
func (r *RaftConsensus) ValidateShutdown() (bool, error) {
	var isConsensusNode bool

	role, err := r.getRole(r.cfg.BasicConfig.BlockchainClient.BcClntRpcUrl)
	if err != nil {
		log.Error("ValidateShutdown - raft role failed", "err", err)
		return isConsensusNode, fmt.Errorf("unable to check raft role: %v", err)
	}

	if role == LEARNER {
		log.Debug("ValidateShutdown - raft consensus check - role:learner, ok to shutdown")
		return isConsensusNode, nil
	}

	isConsensusNode = true

	if role == MINTER {
		return isConsensusNode, errors.New("minter node, cannot be shutdown")
	}

	cluster, err := r.getRaftClusterInfo(r.cfg.BasicConfig.BlockchainClient.BcClntRpcUrl)
	if err != nil {
		log.Error("ValidateShutdown - raft cluster failed", "err", err)
		return isConsensusNode, fmt.Errorf("unable to check raft cluster info: %v", err)
	}

	activeNodes := 0
	totalNodes := len(cluster)
	for _, n := range cluster {
		if n.NodeActive {
			activeNodes++
		}
	}
	minActiveNodes := (totalNodes / 2) + 1 //TODO(cjh) need floor or ceil?
	log.Info("ValidateShutdown - raft consensus check", "role", role, "minActiveNodes", minActiveNodes, "totalNodes", totalNodes, "ActiveNodes", activeNodes)

	if activeNodes <= minActiveNodes {
		return isConsensusNode, fmt.Errorf("raft quorum failed, activeNodes=%d minimumActiveNodesRequired=%d cannot be shutdown", activeNodes, minActiveNodes)
	}
	return isConsensusNode, nil
}
