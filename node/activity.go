package node

import (
	"time"

	"github.com/ConsenSysQuorum/node-manager/log"
)

type InactivityMonitor struct {
	qrmNode           *QuorumNodeControl
	inactiveTimeCount int
	stopCh            chan bool
}

var nodeMonitor *InactivityMonitor

func NewNodeInactivityMonitor(qn *QuorumNodeControl) *InactivityMonitor {
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
				if nm.inactiveTimeCount == quorumNode.config.GethInactivityTime {
					log.Info("going to try stop node as it has been inactive", "inactivetime", quorumNode.config.GethInactivityTime)
					if err := nm.qrmNode.IsNodeBusy(); err != nil {
						log.Info("node is busy", "msg", err.Error())
						// reset inactivity as node is busy, to prevent shutdown right after node start up
						nm.inactiveTimeCount = 0
					} else {
						nm.qrmNode.RequestStopNode()
						log.Info("requested node shutdown, waiting for shutdown complete")
						nm.qrmNode.WaitStopNode()
						log.Info("shutown completed resuming inactivity time tracker")
						nm.inactiveTimeCount = 0
					}
				} else {
					log.Debug("inactivity ticking", "inactive seconds", nodeMonitor.inactiveTimeCount)
					nm.inactiveTimeCount++
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
