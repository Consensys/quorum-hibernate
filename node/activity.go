package node

import (
	"time"

	"github.com/ConsenSysQuorum/node-manager/log"
)

// InactivityMonitor tracks inactivity of the node
// once inactivity reaches the threshold it requests node controller to stop blockchain client/privacy manager
// it allows inactivity to be reset via NodeControl
type InactivityMonitor struct {
	nodeCtrl          *NodeControl
	inactiveTimeCount int
	stopCh            chan bool
}

func NewInactivityMonitor(qn *NodeControl) *InactivityMonitor {
	return &InactivityMonitor{qn, 0, make(chan bool)}
}

func (nm *InactivityMonitor) StartInactivityTimer() {
	go nm.trackInactivity()
}

// trackInactivity tracks node's inactivity time in seconds.
// when inactive time exceeds limit(as per config) it requests the node to be shutdown
func (nm *InactivityMonitor) trackInactivity() {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	log.Info("trackInactivity - node inactivity tracker started", "inactivityTime", nm.nodeCtrl.config.BasicConfig.InactivityTime)
	for {
		select {
		case <-timer.C:
			if nm.inactiveTimeCount == nm.nodeCtrl.config.BasicConfig.InactivityTime {
				nm.processInactivity()
			} else {
				log.Debug("trackInactivity - inactivity ticking", "inactive seconds", nm.inactiveTimeCount)
				nm.inactiveTimeCount++
			}
		case <-nm.nodeCtrl.inactivityResetCh:
			nm.ResetInactivity()
		case <-nm.stopCh:
			log.Info("trackInactivity - stopped inactivity monitor")
			return
		}
	}
}

// processInactivity requests the node to be stopped if the node  is not busy.
func (nm *InactivityMonitor) processInactivity() {
	log.Info("processInactivity - going to try stop node as it has been inactive", "inactivetime", nm.nodeCtrl.config.BasicConfig.InactivityTime)
	if err := nm.nodeCtrl.IsNodeBusy(); err != nil {
		log.Info("processInactivity - node is busy", "msg", err.Error())
		// reset inactivity as node is busy, to prevent shutdown right after node start up
		nm.ResetInactivity()
	} else {
		nm.nodeCtrl.RequestStopClient()
		log.Info("processInactivity - requested node shutdown, waiting for shutdown complete")
		status := nm.nodeCtrl.WaitStopClient()
		log.Info("processInactivity - resuming inactivity time tracker", "shutdown status", status)
		nm.ResetInactivity()
	}
}

func (nm *InactivityMonitor) ResetInactivity() {
	wasInactive := nm.inactiveTimeCount
	nm.inactiveTimeCount = 0
	log.Info("ResetInactivity - inactivity reset", "was inactive for (seconds)", wasInactive)
}

func (nm *InactivityMonitor) Stop() {
	nm.stopCh <- true
}

func (nm *InactivityMonitor) GetInactivityTimeCount() int {
	return nm.inactiveTimeCount
}
