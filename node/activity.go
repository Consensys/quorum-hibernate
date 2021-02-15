package node

import (
	"time"

	"github.com/ConsenSys/quorum-hibernate/log"
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

func (nh *InactivityResyncMonitor) Start() {
	go nh.trackInactivity()
	go nh.trackResyncTimer()
}

// trackInactivity tracks node's inactivity time in seconds.
// when inactive time exceeds limit(as per config) it requests the node to be shutdown
func (nh *InactivityResyncMonitor) trackInactivity() {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	log.Info("trackInactivity - node inactivity tracker started", "inactivityTime", nh.nodeCtrl.config.BasicConfig.InactivityTime)
	for {
		select {
		case <-timer.C:
			if nh.inactiveTimeCount == nh.nodeCtrl.config.BasicConfig.InactivityTime {
				nh.processInactivity()
			} else {
				log.Trace("trackInactivity - inactivity ticking", "inactive seconds", nh.inactiveTimeCount)
				nh.inactiveTimeCount++
			}
		case <-nh.nodeCtrl.inactivityResetCh:
			nh.ResetInactivity()
		case <-nh.stopCh:
			log.Info("trackInactivity - stopped inactivity monitor")
			return
		}
	}
}

// trackResyncTimer brings up the node after certain period of hibernation to
// resync with the network
func (nh *InactivityResyncMonitor) trackResyncTimer() {
	if !nh.nodeCtrl.config.BasicConfig.IsResyncTimerSet() {
		// resyncing feature not enabled. return
		return
	}
	resyncTime := time.Duration(nh.nodeCtrl.config.BasicConfig.ResyncTime) * time.Second
	timer := time.NewTimer(resyncTime)
	defer timer.Stop()

	log.Info("trackResyncTimer - node resync tracker started", "resyncTime", nh.nodeCtrl.config.BasicConfig.ResyncTime)

	for {
		select {
		case <-timer.C:
			nh.processResyncRequest()

		case <-nh.nodeCtrl.syncResetCh:
			timer.Reset(resyncTime)

		case <-nh.stopCh:
			log.Info("trackResyncTimer - stopped inactivity monitor")
			return
		}
	}

}

func (nh *InactivityResyncMonitor) processResyncRequest() {
	if err := nh.nodeCtrl.IsNodeBusy(); err == nil {
		nh.ResetInactivity()
		// restart node for sync. node shut down will happen based on inactivity
		nh.nodeCtrl.RequestStartClient()
		log.Info("trackResyncTimer - requested node start, waiting for start complete")
		status := nh.nodeCtrl.WaitStartClient()
		log.Info("trackResyncTimer - resuming resync timer", "start status", status)
	} else {
		log.Warn("trackResyncTimer - failed to start node", "err", err)
	}
}

// processInactivity requests the node to be stopped if the node  is not busy.
func (nh *InactivityResyncMonitor) processInactivity() {

	log.Info("processInactivity - going to try stop node as it has been inactive", "inactivetime", nh.nodeCtrl.config.BasicConfig.InactivityTime)
	if err := nh.nodeCtrl.IsNodeBusy(); err != nil {
		log.Info("processInactivity - node is busy", "msg", err.Error())
		// reset inactivity as node is busy, to prevent shutdown right after node start up
		nh.ResetInactivity()
	} else {
		// at the end of inactivity period force status check with
		// client. This is to handle scenarios where in the node was
		// brought up in the backend bypassing node hibernator
		if nh.nodeCtrl.CheckClientUpStatus(true) {
			nh.nodeCtrl.RequestStopClient()
			log.Info("processInactivity - requested node shutdown, waiting for shutdown complete")
			status := nh.nodeCtrl.WaitStopClient()
			log.Info("processInactivity - resuming inactivity time tracker", "shutdown status", status)
		} else {
			log.Info("processInactivity - node is already down")
		}
		nh.ResetInactivity()
	}
}

func (nh *InactivityResyncMonitor) ResetInactivity() {
	wasInactive := nh.inactiveTimeCount
	nh.inactiveTimeCount = 0
	log.Info("ResetInactivity - inactivity reset", "was inactive for (seconds)", wasInactive)
}

func (nh *InactivityResyncMonitor) Stop() {
	close(nh.stopCh)
}

func (nh *InactivityResyncMonitor) GetInactivityTimeCount() int {
	return nh.inactiveTimeCount
}
