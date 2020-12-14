package p2p

import (
	"github.com/ConsenSysQuorum/node-manager/core/types"
)

type PeerPrivateTxPrepResult struct {
	Result PrivateTxPrepStatus `json:"result"`
	Error  error               `json:"error"`
}

type PrivateTxPrepStatus struct {
	Status bool `json:"status"`
}

type PeerManager struct {
	cfg *types.NodeConfig
}

type PeerNodeStatusResult struct {
	Result NodeStatusInfo `json:"result"`
	Error  error          `json:"error"`
}

type NodeStatusInfo struct {
	Status            types.NodeStatus
	InactiveTimeLimit int
	InactiveTime      int
	TimeToShutdown    int
}
