package p2p

import (
	"github.com/ConsenSysQuorum/node-manager/config"
	"github.com/ConsenSysQuorum/node-manager/core"
)

type PeerPrivateTxPrepResult struct {
	Result PrivateTxPrepStatus `json:"result"`
	Error  error               `json:"error"`
}

type PrivateTxPrepStatus struct {
	Status bool `json:"status"`
}

type PeerManager struct {
	cfg *config.Node
}

type PeerNodeStatusResult struct {
	Result NodeStatusInfo `json:"result"`
	Error  error          `json:"error"`
}

type NodeStatusInfo struct {
	Status            core.NodeStatus
	InactiveTimeLimit int
	InactiveTime      int
	TimeToShutdown    int
}
