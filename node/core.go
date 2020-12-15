package node

import (
	"github.com/ConsenSysQuorum/node-manager/core"
)

// TODO(cjh) for testing so methods can be mocked
type ControllerApiService interface {
	CheckClientUpStatus(connectToClient bool) bool
	IsClientUp() bool
	ResetInactiveSyncTime()
	IsNodeBusy() error
	PrepareClient() bool
	GetNodeStatus() core.NodeStatus
	GetInactivityTimeCount() int
}
