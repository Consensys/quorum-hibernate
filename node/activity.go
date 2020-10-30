package node

import (
	"log"
	"time"
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
		log.Printf("node inactivity tracker started")
		for {
			select {
			case <-timer.C:
				nm.inactiveTimeCount++
				//log.Printf("node is inactive for %d seconds", nodeMonitor.inactiveTimeCount)
				if nm.inactiveTimeCount == quorumNode.inactiveTime {
					log.Printf("going to stop node as it has been inactive for %d seconds", quorumNode.inactiveTime)
					nm.qrmNode.stopNodeCh <- true
					log.Printf("waiting for shutdown complete")
					shutdownStatus := <-nm.qrmNode.shutdownCompleteCh
					if shutdownStatus {
						log.Printf("shutown completed resuming inactivity time tracker")
					} else {
						log.Printf("shutown not successful")
					}
					nm.inactiveTimeCount = 0
				}
			case <-nm.qrmNode.inactivityResetCh:
				wasInactive := nm.inactiveTimeCount
				nodeMonitor.inactiveTimeCount = 0
				log.Printf("iactivity reset, was inactive for %d seconds", wasInactive)
			}
		}
	}()
}
