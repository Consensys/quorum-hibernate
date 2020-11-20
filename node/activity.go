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

func NewNodeInactivityMonitor(qn *QuorumNodeControl) *InactivityMonitor {
	return &InactivityMonitor{qn, 0, make(chan bool)}
}

func (nm *InactivityMonitor) StartInactivityTimer() {
	go func() {
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		log.Info("node inactivity tracker started", "inactivityTime", nm.qrmNode.config.BasicConfig.InactivityTime)
		for {
			select {
			case <-timer.C:
				if nm.inactiveTimeCount == nm.qrmNode.config.BasicConfig.InactivityTime {
					log.Info("going to try stop node as it has been inactive", "inactivetime", nm.qrmNode.config.BasicConfig.InactivityTime)
					if err := nm.qrmNode.IsNodeBusy(); err != nil {
						log.Info("node is busy", "msg", err.Error())
						// reset inactivity as node is busy, to prevent shutdown right after node start up
						nm.ResetInactivity()
					} else {
						nm.qrmNode.RequestStopNode()
						log.Info("requested node shutdown, waiting for shutdown complete")
						nm.qrmNode.WaitStopNode()
						log.Info("shutown completed resuming inactivity time tracker")
						nm.ResetInactivity()
					}
				} else {
					log.Debug("inactivity ticking", "inactive seconds", nm.inactiveTimeCount)
					nm.inactiveTimeCount++
				}
			case <-nm.qrmNode.inactivityResetCh:
				nm.ResetInactivity()
			case <-nm.stopCh:
				log.Info("stopped inactivity monitor")
				return
			}
		}
	}()
}

func (nm *InactivityMonitor) ResetInactivity() {
	wasInactive := nm.inactiveTimeCount
	nm.inactiveTimeCount = 0
	log.Info("inactivity reset", "was inactive for (seconds)", wasInactive)
}

func (nm *InactivityMonitor) Stop() {
	nm.stopCh <- true
}

func (nm *InactivityMonitor) GetInactivityTimeCount() int {
	return nm.inactiveTimeCount
}
