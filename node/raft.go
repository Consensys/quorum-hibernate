package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/core"

	"github.com/ConsenSysQuorum/node-manager/log"
)

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
	Error  error              `json:"error"`
}

type RaftConsensus struct {
	qn     *QuorumNodeControl
	client *http.Client
}

func NewRaftConsensus(qn *QuorumNodeControl) Consensus {
	return &RaftConsensus{qn: qn, client: core.NewHttpClient()}
}

func (r *RaftConsensus) GetRaftClusterInfo(qrmRpcUrl string) ([]RaftClusterEntry, error) {
	raftClusterJsonStr := []byte(`{"jsonrpc":"2.0", "method":"raft_cluster", "params":[], "id":67}`)
	req, err := http.NewRequest("POST", qrmRpcUrl, bytes.NewBuffer(raftClusterJsonStr))
	if err != nil {
		return nil, fmt.Errorf("raft cluster - creating request failed err=%v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("raft cluster do req failed err=%v", err)
	}
	var respResult RaftClusterResp
	if resp.StatusCode == http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Debug("raft cluster response Body:", string(body))
		jerr := json.Unmarshal(body, &respResult)
		if jerr == nil {
			log.Debug("raft cluster - response OK", "from", qrmRpcUrl, "result", respResult)
		} else {
			log.Error("response result json decode failed", "err", jerr)
			return nil, err
		}
	}
	return respResult.Result, respResult.Error
}

func (r *RaftConsensus) ValidateShutdown() error {
	cluster, err := r.GetRaftClusterInfo(r.qn.config.GethRpcUrl)
	if err != nil {
		log.Error("raft cluster failed", "err", err)
		return err
	}
	role := "verifier"
	activeNodes := 0
	totalNodes := len(cluster)
	for _, n := range cluster {
		if n.NodeActive {
			activeNodes++
		}
		if n.NodeId == r.qn.config.EnodeId {
			role = n.Role
		}
	}
	minActiveNodes := (totalNodes / 2) + 1
	log.Info("raft consensus check", "role", role, "minActiveNodes", minActiveNodes, "totalNodes", totalNodes, "ActiveNodes", activeNodes)
	if role == "minter" {
		return errors.New("minter node, cannot be shutdown")
	}
	if activeNodes <= minActiveNodes {
		return fmt.Errorf("raft quorum failed, activeNodes=%d minimmumActiveNodesRequired=%d cannot be shutdown", activeNodes, minActiveNodes)
	}
	return nil
}
