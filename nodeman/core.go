package nodeman

import (
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/core/types"
)

type NodeManagerPrivateTxPrepResult struct {
	Result PrivateTxPrepStatus `json:"result"`
	Error  error               `json:"error"`
}

type PrivateTxPrepStatus struct {
	Status bool `json:"status"`
}

type NodeManager struct {
	cfg    *types.NodeConfig
	client *http.Client
}

type NodeManagerNodeStatusResult struct {
	Result NodeStatusInfo `json:"result"`
	Error  error          `json:"error"`
}

type NodeStatusInfo struct {
	Status            types.NodeStatus
	InactiveTimeLimit int
	InactiveTime      int
	TimeToShutdown    int
}
