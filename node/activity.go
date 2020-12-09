package node

import (
	"time"

	"github.com/ConsenSysQuorum/node-manager/log"
)

// InactivityResyncMonitor tracks inactivity of the node and starts the blcokchain client and privacy manager based on resync timer
// once inactivity reaches the threshold it requests node controller to stop blockchain client/privacy manager
// it allows inactivity to be reset via NodeControl
// once resync timer is up it starts the blcokchain client and privacy manager
type InactivityResyncMonitor struct {
	nodeCtrl          *NodeControl
	inactiveTimeCount int
	stopCh            chan bool
}

func NewInactivityResyncMonitor(qn *NodeControl) *InactivityResyncMonitor {
	return &InactivityResyncMonitor{qn, 0, make(chan bool)}
}

func (nm *InactivityResyncMonitor) Start() {
	go nm.trackInactivity()
	go nm.trackResyncTimer()
}

// trackInactivity tracks node's inactivity time in seconds.
// when inactive time exceeds limit(as per config) it requests the node to be shutdown
func (nm *InactivityResyncMonitor) trackInactivity() {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	log.Info("trackInactivity - node inactivity tracker started", "inactivityTime", nm.nodeCtrl.config.BasicConfig.InactivityTime)
	for {
		select {
		case <-timer.C:
			if nm.inactiveTimeCount == nm.nodeCtrl.config.BasicConfig.InactivityTime {
				nm.processInactivity()
			} else {
				log.Trace("trackInactivity - inactivity ticking", "inactive seconds", nm.inactiveTimeCount)
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

// trackResyncTimer brings up the node after certain period of hibernation to
// resync with the network
func (nm *InactivityResyncMonitor) trackResyncTimer() {
	if !nm.nodeCtrl.config.BasicConfig.IsResyncTimerSet() {
		// resyncing feature not enabled. return
		return
	}
	resyncTime := time.Duration(nm.nodeCtrl.config.BasicConfig.ResyncTime) * time.Second
	timer := time.NewTimer(resyncTime)
	defer timer.Stop()

	log.Info("trackResyncTimer - node resync tracker started", "resyncTime", nm.nodeCtrl.config.BasicConfig.ResyncTime)

	for {
		select {
		case <-timer.C:
			nm.processResyncRequest()

		case <-nm.nodeCtrl.syncResetCh:
			timer.Reset(resyncTime)

		case <-nm.stopCh:
			log.Info("trackResyncTimer - stopped inactivity monitor")
			return
		}
	}

}

func (nm *InactivityResyncMonitor) processResyncRequest() {
	if err := nm.nodeCtrl.IsNodeBusy(); err == nil {
		nm.ResetInactivity()
		// restart node for sync. node shut down will happen based on inactivity
		nm.nodeCtrl.RequestStartClient()
		log.Info("trackResyncTimer - requested node start, waiting for start complete")
		status := nm.nodeCtrl.WaitStartClient()
		log.Info("trackResyncTimer - resuming resync timer", "start status", status)
	} else {
		log.Warn("trackResyncTimer - failed to start node", "err", err)
	}
}

// processInactivity requests the node to be stopped if the node  is not busy.
func (nm *InactivityResyncMonitor) processInactivity() {
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

func (nm *InactivityResyncMonitor) ResetInactivity() {
	wasInactive := nm.inactiveTimeCount
	nm.inactiveTimeCount = 0
	log.Info("ResetInactivity - inactivity reset", "was inactive for (seconds)", wasInactive)
}

func (nm *InactivityResyncMonitor) Stop() {
	close(nm.stopCh)
}

func (nm *InactivityResyncMonitor) GetInactivityTimeCount() int {
	return nm.inactiveTimeCount
}
