package p2p

import (
	"github.com/ConsenSys/quorum-hibernate/config"
	"github.com/ConsenSys/quorum-hibernate/core"
)

type PeerPrivateTxPrepResult struct {
	Result PrivateTxPrepStatus `json:"result"`
	Error  error               `json:"error"`
}

type PrivateTxPrepStatus struct {
	Status bool `json:"status"`
}

type PeerManager struct {
	cfg          *config.Node
	configReader config.PeersReader
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
