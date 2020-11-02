package node

import (
	"time"

	"github.com/ConsenSysQuorum/node-manager/log"
)

type NodeMonitor struct {
	qrmNode           *QuorumNode
	inactiveTimeCount int
}

var nodeMonitor *NodeMonitor

func NewNodeInactivityMonitor(qn *QuorumNode) *NodeMonitor {
	nodeMonitor = &NodeMonitor{qn, 0}
	return nodeMonitor
}

func (nm *NodeMonitor) StartInactivityTimer() {
	go func() {
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		log.Info("node inactivity tracker started")
		for {
			select {
			case <-timer.C:
				nm.inactiveTimeCount++
				log.Debug("node is inactive", "seconds", nodeMonitor.inactiveTimeCount)
				if nm.inactiveTimeCount == quorumNode.config.GethInactivityTime {
					log.Info("going to stop node as it has been inactive", "inactivetime", quorumNode.config.GethInactivityTime)
					nm.qrmNode.RequestStopNode()
					log.Info("waiting for shutdown complete")
					nm.qrmNode.WaitStopNode()
					log.Info("shutown completed resuming inactivity time tracker")
					nm.inactiveTimeCount = 0
				}
			case <-nm.qrmNode.inactivityResetCh:
				wasInactive := nm.inactiveTimeCount
				nodeMonitor.inactiveTimeCount = 0
				log.Info("iactivity reset, was inactive", "seconds", wasInactive)
			}
		}
	}()
}
