package node

import (
	"errors"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/privatetx"
	"github.com/ConsenSysQuorum/node-manager/qnm"

	cons "github.com/ConsenSysQuorum/node-manager/consensus"
	"github.com/ConsenSysQuorum/node-manager/core"
	proc "github.com/ConsenSysQuorum/node-manager/process"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

// NodeControl represents a node controller.
// It tracks blockchain client/privacyManager processes' inactivity and it allows inactivity to be reset when
// there is some activity.
// It accepts request to stop blockchain client/privacyManager when there is inactivity.
// It starts blockchain client/privacyManager processes when there is a activity.
// It takes care of managing combined status of blockchain client & privacyManager.
type NodeControl struct {
	config             *types.NodeConfig   // config of this node
	im                 *InactivityMonitor  // inactivity monitor
	nm                 *qnm.NodeManager    // node manager to communicate with other qnm
	bcclnt             proc.Process        // blockchain client process controller
	pmclnt             proc.Process        // privacy manager process controller
	consensus          cons.Consensus      // consenus validator
	txh                privatetx.TxHandler // Transaction handler
	nodeStatus         types.NodeStatus    // status of this node
	inactivityResetCh  chan bool           // channel to reset inactivity
	stopNodeCh         chan bool           // channel to request stop node
	shutdownCompleteCh chan bool           // channel to notify stop node action status
	startNodeCh        chan bool           // channel to request start node
	startCompleteCh    chan bool           // channel to notify start node action status
	stopCh             chan bool           // channel to stop start/stop node monitor
	startStopMux       sync.Mutex          // lock for starting and stopping node
	statusMux          sync.Mutex          // lock for setting the status
}

func NewQuorumNodeControl(cfg *types.NodeConfig) *NodeControl {
	quorumNode := &NodeControl{
		cfg,
		nil,
		qnm.NewNodeManager(cfg),
		nil,
		nil,
		nil,
		nil,
		types.Up,
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		sync.Mutex{},
		sync.Mutex{},
	}

	if cfg.BasicConfig.BcClntProcess.IsShell() {
		quorumNode.bcclnt = proc.NewShellProcess(cfg.BasicConfig.BcClntProcess, cfg.BasicConfig.BcClntRpcUrl, cfg.BasicConfig.PrivManUpcheckUrl, true)
	} else if cfg.BasicConfig.BcClntProcess.IsDocker() {
		quorumNode.bcclnt = proc.NewDockerProcess(cfg.BasicConfig.BcClntProcess, cfg.BasicConfig.BcClntRpcUrl, cfg.BasicConfig.PrivManUpcheckUrl, true)
	}

	if cfg.BasicConfig.PrivManProcess.IsShell() {
		quorumNode.pmclnt = proc.NewShellProcess(cfg.BasicConfig.PrivManProcess, cfg.BasicConfig.BcClntRpcUrl, cfg.BasicConfig.PrivManUpcheckUrl, true)
	} else if cfg.BasicConfig.PrivManProcess.IsDocker() {
		quorumNode.pmclnt = proc.NewDockerProcess(cfg.BasicConfig.PrivManProcess, cfg.BasicConfig.BcClntRpcUrl, cfg.BasicConfig.PrivManUpcheckUrl, true)
	}

	if quorumNode.bcclnt.Status() && quorumNode.pmclnt.Status() {
		quorumNode.SetNodeStatus(types.Up)
	} else {
		quorumNode.SetNodeStatus(types.Down)
	}

	if quorumNode.config.BasicConfig.IsRaft() {
		quorumNode.consensus = cons.NewRaftConsensus(quorumNode.config)
	} else if quorumNode.config.BasicConfig.IsIstanbul() {
		quorumNode.consensus = cons.NewIstanbulConsensus(quorumNode.config)
	}

	if quorumNode.config.BasicConfig.IsQuorumClient() {
		quorumNode.txh = privatetx.NewQuorumTxHandler(quorumNode.config)
	} // TODO add tx handler for Besu

	return quorumNode
}

func (qn *NodeControl) GetRPCConfig() *types.RPCServerConfig {
	return qn.config.BasicConfig.Server
}

func (qn *NodeControl) GetNodeConfig() *types.NodeConfig {
	return qn.config
}

func (qn *NodeControl) GetNodeStatus() types.NodeStatus {
	return qn.nodeStatus
}

func (qn *NodeControl) GetProxyConfig() []*types.ProxyConfig {
	return qn.config.BasicConfig.Proxies
}

func (qn *NodeControl) GetTxHandler() privatetx.TxHandler {
	return qn.txh
}

func (qn *NodeControl) SetNodeStatus(ns types.NodeStatus) {
	defer qn.statusMux.Unlock()
	qn.statusMux.Lock()
	qn.nodeStatus = ns
}

// IsNodeUp performs up check for blockchain client and privacy manager and returns the combined status
// if both blockchain client and privacy manager are up, the node status is up(true) else down(false)
func (qn *NodeControl) IsNodeUp() bool {
	gs := qn.bcclnt.IsUp()
	ts := qn.pmclnt.IsUp()
	log.Debug("IsNodeUp", "blockchain client", gs, "privacy manager", ts)
	if gs && ts {
		qn.SetNodeStatus(types.Up)
	} else {
		qn.SetNodeStatus(types.Down)
	}
	return gs && ts
}

// IsNodeBusy returns error if the node is busy with shutdown/startup
func (qn *NodeControl) IsNodeBusy() error {
	switch qn.nodeStatus {
	case types.ShutdownInprogress, types.ShutdownInitiated:
		return errors.New("node is being shutdown, try after sometime")
	case types.StartupInprogress, types.StartupInitiated:
		return errors.New("node is being started, try after sometime")
	case types.Up, types.Down:
		return nil
	}
	return nil
}

// Start starts blockchain client and privacy manager start/stop monitor and inactivity tracker
func (qn *NodeControl) Start() {
	qn.StartStopNodeMonitor()
	qn.im = NewInactivityMonitor(qn)
	qn.im.StartInactivityTimer()
}

// Stop stops blockchain client and privacy manager start/stop monitor and inactivity tracker
func (qn *NodeControl) Stop() {
	qn.im.Stop()
	qn.stopCh <- true
}

// ResetInactiveTime resets inactivity time of the tracker
func (nm *NodeControl) ResetInactiveTime() {
	nm.inactivityResetCh <- true
}

//StartStopNodeMonitor listens for requests to start/stop blockchain client and privacy manager
func (qn *NodeControl) StartStopNodeMonitor() {
	go func() {
		log.Info("StartStopNodeMonitor - node start/stop monitor started")
		for {
			select {
			case <-qn.stopNodeCh:
				log.Debug("StartStopNodeMonitor - request received to stop node")
				if !qn.StopNode() {
					log.Error("StartStopNodeMonitor - stopping failed")
					qn.shutdownCompleteCh <- false
				} else {
					qn.shutdownCompleteCh <- true
				}
			case <-qn.startNodeCh:
				log.Debug("StartStopNodeMonitor - request received to start node")
				if !qn.StartNode() {
					log.Error("StartStopNodeMonitor - starting failed")
					qn.startCompleteCh <- false
				} else {
					qn.startCompleteCh <- true
				}
			case <-qn.stopCh:
				log.Info("StartStopNodeMonitor - stopped node start/stop monitor service")
				return
			}
		}
	}()
}

func (qn *NodeControl) RequestStartNode() {
	qn.startNodeCh <- true
}

func (qn *NodeControl) RequestStopNode() {
	qn.stopNodeCh <- true
}

func (qn *NodeControl) WaitStartNode() bool {
	status := <-qn.startCompleteCh
	return status
}

func (qn *NodeControl) WaitStopNode() bool {
	status := <-qn.shutdownCompleteCh
	return status
}

// TODO handle error if node failed to start
func (qn *NodeControl) PrepareNode() bool {
	if !qn.IsNodeUp() {
		status := qn.StartNode()
		log.Debug("PrepareNode - node start completed", "status", status)
		return status
	} else {
		log.Info("node is UP")
		return true
	}
}

func (qn *NodeControl) StopNode() bool {
	defer qn.startStopMux.Unlock()
	qn.startStopMux.Lock()

	if qn.nodeStatus == types.Down {
		log.Info("StopNode - node is already down")
		return true
	}
	if err := qn.IsNodeBusy(); err != nil {
		log.Error("StopNode - cannot be shutdown", "err", err)
		return false
	}
	var qnms []qnm.NodeStatusInfo
	var err error

	retryCount := 1

	for retryCount <= core.Qnm2QnmValidationRetryLimit {
		w := core.GetRandomRetryWaitTime()
		log.Info("StopNode - waiting for qnm2qnm validation try", "wait time in seconds", w)
		time.Sleep(time.Duration(w) * time.Millisecond)
		qnms, err = qn.nm.ValidateOtherQnms()
		if err == nil {
			log.Info("StopNode - qnm2qnm validation passed")
			break
		}
		log.Error("StopNode - qnm2qnm validation failed", "retryLimit", core.Qnm2QnmValidationRetryLimit, "retryCount", retryCount, "err", err, "qnms", qnms)
		retryCount++
	}

	if retryCount > core.Qnm2QnmValidationRetryLimit {
		log.Error("StopNode - node cannot be shutdown, qnm2qnm validation failed after retrying")
		return false
	}

	qn.SetNodeStatus(types.ShutdownInitiated)

	if err := qn.consensus.ValidateShutdown(); err == nil {
		log.Info("StopNode - consensus check passed, node can be shutdown")
	} else {
		log.Info("StopNode - consensus check failed, node cannot be shutdown", "err", err)
		qn.SetNodeStatus(types.Up)
		return false
	}

	qn.SetNodeStatus(types.ShutdownInprogress)

	// TODO parallelize / loop
	gs := true
	ts := true
	if qn.bcclnt.Stop() != nil {
		gs = false
	}
	if qn.pmclnt.Stop() != nil {
		ts = false
	}
	if gs && ts {
		qn.SetNodeStatus(types.Down)
	}
	// if stopping of blockchain client or privacy manager fails Status will remain as ShutdownInprogress and qnm will not process any requests from clients
	// it will need some manual intervention to set it to the correct status
	return gs && ts
}

func (qn *NodeControl) StartNode() bool {
	defer qn.startStopMux.Unlock()
	qn.startStopMux.Lock()
	qn.SetNodeStatus(types.StartupInitiated)
	qn.SetNodeStatus(types.StartupInprogress)
	gs := true
	ts := true
	if qn.pmclnt.Start() != nil {
		gs = false
	}
	if qn.bcclnt.Start() != nil {
		ts = false
	}
	if gs && ts {
		qn.SetNodeStatus(types.Up)
	}
	// if start up of blockchain client or privacy manager fails Status will remain as StartupInprogress and qnm will not process any requests from clients
	// it will need some manual intervention to set it to the correct status
	return gs && ts
}

func (qn *NodeControl) PrepareNodeManagerForPrivateTx(privateFor []string) (bool, error) {
	return qn.nm.ValidateForPrivateTx(privateFor)
}
