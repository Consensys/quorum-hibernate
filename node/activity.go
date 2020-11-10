package node

import (
	"time"

	"github.com/ConsenSysQuorum/node-manager/log"
)

type InactivityMonitor struct {
	qrmNode           *QuorumNode
	inactiveTimeCount int
	stopCh            chan bool
}

var nodeMonitor *InactivityMonitor

func NewNodeInactivityMonitor(qn *QuorumNode) *InactivityMonitor {
	nodeMonitor = &InactivityMonitor{qn, 0, make(chan bool)}
	return nodeMonitor
}

func (nm *InactivityMonitor) StartInactivityTimer() {
	go func() {
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		log.Info("node inactivity tracker started", "inactivityTime", nm.qrmNode.config.GethInactivityTime)
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
				log.Info("inactivity reset, was inactive", "seconds", wasInactive)
			case <-nm.stopCh:
				log.Info("stopped inactivity monitor")
				return
			}
		}
	}()
}

func (nm *InactivityMonitor) Stop() {
	nm.stopCh <- true
}
