package node

import (
	"errors"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/nodeman"
	"github.com/ConsenSysQuorum/node-manager/privatetx"

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
	config             *types.NodeConfig    // config of this node
	im                 *InactivityMonitor   // inactivity monitor
	nm                 *nodeman.NodeManager // node manager to communicate with other node manager
	bcclnt             proc.Process         // blockchain client process controller
	pmclnt             proc.Process         // privacy manager process controller
	consensus          cons.Consensus       // consenus validator
	txh                privatetx.TxHandler  // Transaction handler
	nodeStatus         types.NodeStatus     // status of this node
	inactivityResetCh  chan bool            // channel to reset inactivity
	stopNodeCh         chan bool            // channel to request stop node
	shutdownCompleteCh chan bool            // channel to notify stop node action status
	startNodeCh        chan bool            // channel to request start node
	startCompleteCh    chan bool            // channel to notify start node action status
	stopCh             chan bool            // channel to stop start/stop node monitor
	nsStopCh           chan bool            // channel to stop node status monitor
	startStopMux       sync.Mutex           // lock for starting and stopping node
	statusMux          sync.Mutex           // lock for setting the status
}

func NewNodeControl(cfg *types.NodeConfig) *NodeControl {
	quorumNode := &NodeControl{
		cfg,
		nil,
		nodeman.NewNodeManager(cfg),
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

func (nc *NodeControl) GetRPCConfig() *types.RPCServerConfig {
	return nc.config.BasicConfig.Server
}

func (nc *NodeControl) GetNodeConfig() *types.NodeConfig {
	return nc.config
}

func (nc *NodeControl) GetNodeStatus() types.NodeStatus {
	return nc.nodeStatus
}

func (nc *NodeControl) GetProxyConfig() []*types.ProxyConfig {
	return nc.config.BasicConfig.Proxies
}

func (nc *NodeControl) GetTxHandler() privatetx.TxHandler {
	return nc.txh
}

func (nc *NodeControl) SetNodeStatus(ns types.NodeStatus) {
	defer nc.statusMux.Unlock()
	nc.statusMux.Lock()
	nc.nodeStatus = ns
}

// IsNodeUp performs up check for blockchain client and privacy manager and returns the combined status
// if both blockchain client and privacy manager are up, the node status is up(true) else down(false)
func (nc *NodeControl) IsNodeUp() bool {
	bcclntStatus, pmStatus := nc.checkUpStatus()
	log.Debug("IsNodeUp", "blockchain client", bcclntStatus, "privacy manager", pmStatus)
	if bcclntStatus && pmStatus {
		nc.SetNodeStatus(types.Up)
	} else {
		nc.SetNodeStatus(types.Down)
	}
	return bcclntStatus && pmStatus
}

// checkUpStatus checks up status of blockchain client and privacy manager in parallel
func (nc *NodeControl) checkUpStatus() (bool, bool) {
	var bcclntStatus bool
	var pmStatus bool
	var wg = sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		bcclntStatus = nc.bcclnt.IsUp()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		pmStatus = nc.pmclnt.IsUp()
	}()
	wg.Wait()
	return bcclntStatus, pmStatus
}

// IsNodeBusy returns error if the node is busy with shutdown/startup
func (nc *NodeControl) IsNodeBusy() error {
	switch nc.nodeStatus {
	case types.ShutdownInprogress, types.ShutdownInitiated:
		return errors.New(core.NodeIsBeingShutdown)
	case types.StartupInprogress, types.StartupInitiated:
		return errors.New(core.NodeIsBeingStarted)
	case types.Up, types.Down:
		return nil
	}
	return nil
}

// Start starts blockchain client and privacy manager start/stop monitor and inactivity tracker
func (nc *NodeControl) Start() {
	nc.StartStopNodeMonitor()
	nc.im = NewInactivityMonitor(nc)
	nc.im.StartInactivityTimer()
	nc.startNodeStatusMonitor()
}

// Stop stops blockchain client and privacy manager start/stop monitor and inactivity tracker
func (nc *NodeControl) Stop() {
	nc.im.Stop()
	nc.stopCh <- true
	nc.nsStopCh <- true
}

// ResetInactiveTime resets inactivity time of the tracker
func (nc *NodeControl) ResetInactiveTime() {
	nc.inactivityResetCh <- true
}

func (nc *NodeControl) startNodeStatusMonitor() {
	go func() {
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		log.Info("NodeStatusMonitor - node status monitor started")
		for {
			select {
			case <-timer.C:
				status := nc.IsNodeUp()
				log.Debug("startNodeStatusMonitor", "status", status)
			case <-nc.nsStopCh:
				log.Info("startNodeStatusMonitor - node status monitor stopped")
				return
			}
		}
	}()
}

//StartStopNodeMonitor listens for requests to start/stop blockchain client and privacy manager
func (nc *NodeControl) StartStopNodeMonitor() {
	go func() {
		log.Info("StartStopNodeMonitor - node start/stop monitor started")
		for {
			select {
			case <-nc.stopNodeCh:
				log.Debug("StartStopNodeMonitor - request received to stop node")
				if !nc.StopNode() {
					log.Error("StartStopNodeMonitor - stopping failed")
					nc.shutdownCompleteCh <- false
				} else {
					nc.shutdownCompleteCh <- true
				}
			case <-nc.startNodeCh:
				log.Debug("StartStopNodeMonitor - request received to start node")
				if !nc.StartNode() {
					log.Error("StartStopNodeMonitor - starting failed")
					nc.startCompleteCh <- false
				} else {
					nc.startCompleteCh <- true
				}
			case <-nc.stopCh:
				log.Info("StartStopNodeMonitor - stopped node start/stop monitor service")
				return
			}
		}
	}()
}

func (nc *NodeControl) RequestStartNode() {
	nc.startNodeCh <- true
}

func (nc *NodeControl) RequestStopNode() {
	nc.stopNodeCh <- true
}

func (nc *NodeControl) WaitStartNode() bool {
	status := <-nc.startCompleteCh
	return status
}

func (nc *NodeControl) WaitStopNode() bool {
	status := <-nc.shutdownCompleteCh
	return status
}

// TODO handle error if node failed to start
func (nc *NodeControl) PrepareNode() bool {
	if nc.nodeStatus == types.Up {
		log.Info("PrepareNode - node is up")
		return true
	} else {
		status := nc.StartNode()
		log.Debug("PrepareNode - node start completed", "status", status)
		return status
	}
}

func (nc *NodeControl) StopNode() bool {
	defer nc.startStopMux.Unlock()
	nc.startStopMux.Lock()

	if nc.nodeStatus == types.Down {
		log.Info("StopNode - node is already down")
		return true
	}
	if err := nc.IsNodeBusy(); err != nil {
		log.Error("StopNode - cannot be shutdown", "err", err)
		return false
	}
	var peersStatus []nodeman.NodeStatusInfo
	var err error

	// 1st check if hibernating node will break the consensus model
	if err := nc.consensus.ValidateShutdown(); err == nil {
		log.Info("StopNode - consensus check passed, node can be shutdown")
	} else {
		log.Info("StopNode - consensus check failed, node cannot be shutdown", "err", err)
		return false
	}

	// consensus is ok. check with network to prevent multiple nodes
	// going down at the same time
	retryCount := 1
	for retryCount <= core.Peer2PeerValidationRetryLimit {
		w := core.GetRandomRetryWaitTime()
		log.Info("StopNode - waiting for p2p validation try", "wait time in seconds", w)
		time.Sleep(time.Duration(w) * time.Millisecond)
		peersStatus, err = nc.nm.ValidatePeers()
		if err == nil {
			log.Info("StopNode - p2p validation passed")
			break
		}
		log.Error("StopNode - p2p validation failed", "retryLimit", core.Peer2PeerValidationRetryLimit, "retryCount", retryCount, "err", err, "peersStatus", peersStatus)
		retryCount++
	}

	if retryCount > core.Peer2PeerValidationRetryLimit {
		log.Error("StopNode - node cannot be shutdown, p2p validation failed after retrying")
		return false
	}

	nc.SetNodeStatus(types.ShutdownInitiated)

	nc.SetNodeStatus(types.ShutdownInprogress)

	bcStatus, pmStatus := nc.stopProcesses()
	if bcStatus && pmStatus {
		nc.SetNodeStatus(types.Down)
	}
	// if stopping of blockchain client or privacy manager fails Status will remain as ShutdownInprogress and node manager will not process any requests from clients
	// it will need some manual intervention to set it to the correct status
	return bcStatus && pmStatus
}

// stopProcesses stops blockchain client and privacy manager processes in parallel
func (nc *NodeControl) stopProcesses() (bool, bool) {
	gs := true
	ts := true
	var wg = sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if nc.bcclnt.Stop() != nil {
			gs = false
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if nc.pmclnt.Stop() != nil {
			ts = false
		}
	}()
	wg.Wait()
	return gs, ts
}

func (nc *NodeControl) StartNode() bool {
	defer nc.startStopMux.Unlock()
	nc.startStopMux.Lock()
	if nc.nodeStatus == types.Up {
		log.Debug("StartNode - node is already up")
		return true
	}
	nc.SetNodeStatus(types.StartupInitiated)
	nc.SetNodeStatus(types.StartupInprogress)
	gs := true
	ts := true
	if nc.pmclnt.Start() != nil {
		gs = false
	}
	if nc.bcclnt.Start() != nil {
		ts = false
	}
	if gs && ts {
		nc.SetNodeStatus(types.Up)
	}
	// if start up of blockchain client or privacy manager fails Status will remain as StartupInprogress and node manager will not process any requests from clients
	// it will need some manual intervention to set it to the correct status
	return gs && ts
}

func (nc *NodeControl) PrepareNodeManagerForPrivateTx(privateFor []string) (bool, error) {
	return nc.nm.ValidatePeerPrivateTxStatus(privateFor)
}
