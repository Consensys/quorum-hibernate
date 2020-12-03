package node

import (
	"errors"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/nodeman"
	"github.com/ConsenSysQuorum/node-manager/privatetx"

	cons "github.com/ConsenSysQuorum/node-manager/consensus"
	besu "github.com/ConsenSysQuorum/node-manager/consensus/besu"
	qnm "github.com/ConsenSysQuorum/node-manager/consensus/quorum"
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
	withPrivMan        bool                 // indicates if the node is running with a privacy manage
	consValid          bool                 // indicates if network level consensus is valid
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
	node := &NodeControl{
		cfg,
		nil,
		nodeman.NewNodeManager(cfg),
		nil,
		nil,
		nil,
		nil,
		cfg.BasicConfig.PrivManProcess != nil,
		false,
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
		node.bcclnt = proc.NewShellProcess(cfg.BasicConfig.BcClntProcess, cfg.BasicConfig.BcClntRpcUrl, cfg.BasicConfig.PrivManUpcheckUrl, true)
	} else if cfg.BasicConfig.BcClntProcess.IsDocker() {
		node.bcclnt = proc.NewDockerProcess(cfg.BasicConfig.BcClntProcess, cfg.BasicConfig.BcClntRpcUrl, cfg.BasicConfig.PrivManUpcheckUrl, true)
	}

	if node.WithPrivMan() {
		if cfg.BasicConfig.PrivManProcess.IsShell() {
			node.pmclnt = proc.NewShellProcess(cfg.BasicConfig.PrivManProcess, cfg.BasicConfig.BcClntRpcUrl, cfg.BasicConfig.PrivManUpcheckUrl, true)
		} else if cfg.BasicConfig.PrivManProcess.IsDocker() {
			node.pmclnt = proc.NewDockerProcess(cfg.BasicConfig.PrivManProcess, cfg.BasicConfig.BcClntRpcUrl, cfg.BasicConfig.PrivManUpcheckUrl, true)
		}
	}
	populateConsensusHandler(node)
	if node.config.BasicConfig.IsQuorumClient() {
		node.txh = privatetx.NewQuorumTxHandler(node.config)
	} // TODO add tx handler for Besu
	node.config.BasicConfig.InactivityTime += getRandomBufferTime(node.config.BasicConfig.InactivityTime)
	log.Debug("Node config - inactivity time after random buffer", "InactivityTime", node.config.BasicConfig.InactivityTime)
	return node
}

func (n *NodeControl) WithPrivMan() bool {
	return n.withPrivMan
}

func getRandomBufferTime(inactivityTime int) int {
	// introduce random delay of 2% of inactivity time that should
	// be added on top of inactivity time
	delay := (2 * inactivityTime) / 100
	if delay < 10 {
		delay = 10
	}
	return core.GetRandomRetryWaitTime(1, delay)
}

func populateConsensusHandler(n *NodeControl) {
	if n.config.BasicConfig.IsQuorumClient() {
		if n.config.BasicConfig.IsRaft() {
			n.consensus = qnm.NewRaftConsensus(n.config)
		} else if n.config.BasicConfig.IsIstanbul() {
			n.consensus = qnm.NewIstanbulConsensus(n.config)
		} else if n.config.BasicConfig.IsClique() {
			n.consensus = qnm.NewCliqueConsensus(n.config)
		}
	} else if n.config.BasicConfig.IsBesuClient() {
		if n.config.BasicConfig.IsClique() {
			n.consensus = besu.NewCliqueConsensus(n.config)
		}
	}
}

func (n *NodeControl) GetRPCConfig() *types.RPCServerConfig {
	return n.config.BasicConfig.Server
}

func (n *NodeControl) GetNodeConfig() *types.NodeConfig {
	return n.config
}

func (n *NodeControl) GetNodeStatus() types.NodeStatus {
	return n.nodeStatus
}

func (n *NodeControl) GetProxyConfig() []*types.ProxyConfig {
	return n.config.BasicConfig.Proxies
}

func (n *NodeControl) GetTxHandler() privatetx.TxHandler {
	return n.txh
}

func (n *NodeControl) SetNodeStatus(ns types.NodeStatus) {
	defer n.statusMux.Unlock()
	n.statusMux.Lock()
	n.nodeStatus = ns
}

// IsNodeUp performs up check for blockchain client and privacy manager and returns the combined status
// if both blockchain client and privacy manager are up, the node status is up(true) else down(false)
func (n *NodeControl) IsNodeUp() bool {
	bcclntStatus, pmStatus := n.checkUpStatus()
	log.Debug("IsNodeUp", "blockchain client", bcclntStatus, "privacy manager", pmStatus)
	if bcclntStatus && pmStatus {
		n.SetNodeStatus(types.Up)
	} else {
		n.SetNodeStatus(types.Down)
	}
	return bcclntStatus && pmStatus
}

// checkUpStatus checks up status of blockchain client and privacy manager in parallel
func (n *NodeControl) checkUpStatus() (bool, bool) {
	var bcclntStatus bool
	var wg = sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		bcclntStatus = n.bcclnt.IsUp()
	}()

	pmStatus := true
	if n.WithPrivMan() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pmStatus = n.pmclnt.IsUp()
		}()
	}

	wg.Wait()
	return bcclntStatus, pmStatus
}

// IsNodeBusy returns error if the node is busy with shutdown/startup
func (n *NodeControl) IsNodeBusy() error {
	switch n.nodeStatus {
	case types.ShutdownInprogress, types.WaitingPeerConfirmation:
		return errors.New(core.NodeIsBeingShutdown)
	case types.StartupInprogress, types.StartupInitiated:
		return errors.New(core.NodeIsBeingStarted)
	case types.Up, types.Down:
		return nil
	}
	return nil
}

// Start starts blockchain client and privacy manager start/stop monitor and inactivity tracker
func (n *NodeControl) Start() {
	n.StartStopNodeMonitor()
	n.im = NewInactivityMonitor(n)
	n.im.StartInactivityTimer()
	n.startNodeStatusMonitor()
}

// Stop stops blockchain client and privacy manager start/stop monitor and inactivity tracker
func (n *NodeControl) Stop() {
	n.im.Stop()
	n.stopCh <- true
	n.nsStopCh <- true
}

// ResetInactiveTime resets inactivity time of the tracker
func (n *NodeControl) ResetInactiveTime() {
	n.inactivityResetCh <- true
}

func (n *NodeControl) startNodeStatusMonitor() {
	go func() {
		timer := time.NewTicker(time.Duration(n.config.BasicConfig.UpchkPollingInterval) * time.Second)
		defer timer.Stop()
		log.Info("NodeStatusMonitor - node status monitor started")
		for {
			select {
			case <-timer.C:
				status := n.IsNodeUp()
				log.Debug("startNodeStatusMonitor", "status", status)
			case <-n.nsStopCh:
				log.Info("startNodeStatusMonitor - node status monitor stopped")
				return
			}
		}
	}()
}

//StartStopNodeMonitor listens for requests to start/stop blockchain client and privacy manager
func (n *NodeControl) StartStopNodeMonitor() {
	go func() {
		log.Info("StartStopNodeMonitor - node start/stop monitor started")
		for {
			select {
			case <-n.stopNodeCh:
				log.Debug("StartStopNodeMonitor - request received to stop node")
				if !n.StopNode() {
					log.Error("StartStopNodeMonitor - stopping failed")
					n.shutdownCompleteCh <- false
				} else {
					n.shutdownCompleteCh <- true
				}
			case <-n.startNodeCh:
				log.Debug("StartStopNodeMonitor - request received to start node")
				if !n.StartNode() {
					log.Error("StartStopNodeMonitor - starting failed")
					n.startCompleteCh <- false
				} else {
					n.startCompleteCh <- true
				}
			case <-n.stopCh:
				log.Info("StartStopNodeMonitor - stopped node start/stop monitor service")
				return
			}
		}
	}()
}

func (n *NodeControl) RequestStartNode() {
	n.startNodeCh <- true
}

func (n *NodeControl) RequestStopNode() {
	n.stopNodeCh <- true
}

func (n *NodeControl) WaitStartNode() bool {
	status := <-n.startCompleteCh
	return status
}

func (n *NodeControl) WaitStopNode() bool {
	status := <-n.shutdownCompleteCh
	return status
}

// TODO handle error if node failed to start
func (n *NodeControl) PrepareNode() bool {
	if n.nodeStatus == types.Up {
		log.Info("PrepareNode - node is up")
		return true
	} else {
		status := n.StartNode()
		log.Debug("PrepareNode - node start completed", "status", status)
		return status
	}
}

func (n *NodeControl) StopNode() bool {
	defer n.startStopMux.Unlock()
	n.startStopMux.Lock()

	if n.nodeStatus == types.Down {
		log.Info("StopNode - node is already down")
		return true
	}
	if err := n.IsNodeBusy(); err != nil {
		log.Error("StopNode - cannot be shutdown", "err", err)
		return false
	}
	var peersStatus []nodeman.NodeStatusInfo
	var err error

	consensusNode, err := n.checkAndValidateConsensus()
	if err != nil {
		log.Info("StopNode - consensus check failed, node cannot be shutdown", "err", err)
		return false
	}
	log.Debug("StopNode - consensus check passed, node can be shutdown")

	if consensusNode {
		// consensus is ok. check with network to prevent multiple nodes
		// going down at the same time
		n.SetNodeStatus(types.WaitingPeerConfirmation)
		//for retryCount <= core.Peer2PeerValidationRetryLimit {
		w := core.GetRandomRetryWaitTime(10, 1000)
		log.Info("StopNode - waiting for p2p validation try", "wait time in seconds", w)
		time.Sleep(time.Duration(w) * time.Millisecond)

		if peersStatus, err = n.nm.ValidatePeers(); err != nil {
			n.SetNodeStatus(types.Up)
			log.Error("StopNode - node cannot be shutdown, p2p validation failed after retrying")
			return false
		}
	}
	log.Debug("StopNode - all checks passed for shutdown", "peerStatus", peersStatus)

	n.SetNodeStatus(types.ShutdownInprogress)

	bcStatus, pmStatus := n.stopProcesses()
	if bcStatus && pmStatus {
		n.SetNodeStatus(types.Down)
	}
	// if stopping of blockchain client or privacy manager fails Status will remain as ShutdownInprogress and node manager will not process any requests from clients
	// it will need some manual intervention to set it to the correct status
	return bcStatus && pmStatus
}

func (n *NodeControl) checkAndValidateConsensus() (bool, error) {
	// validate if the consensus passed in config is correct.
	// for besu bypass this check as it does not provide any rpc api to confirm consensus
	if !n.consValid && n.config.BasicConfig.IsQuorumClient() {
		if err := n.config.IsConsensusValid(); err != nil {
			return false, err
		}
		n.consValid = true
	}
	// perform consensus level validations for node hibernation
	return n.consensus.ValidateShutdown()
}

// stopProcesses stops blockchain client and privacy manager processes in parallel
func (n *NodeControl) stopProcesses() (bool, bool) {
	gs := true
	ts := true
	var wg = sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if n.bcclnt.Stop() != nil {
			gs = false
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if n.pmclnt.Stop() != nil {
			ts = false
		}
	}()
	wg.Wait()
	return gs, ts
}

func (n *NodeControl) StartNode() bool {
	defer n.startStopMux.Unlock()
	n.startStopMux.Lock()
	if n.nodeStatus == types.Up {
		log.Debug("StartNode - node is already up")
		return true
	}
	n.SetNodeStatus(types.StartupInitiated)
	n.SetNodeStatus(types.StartupInprogress)
	gs := true
	ts := true
	if n.withPrivMan && n.pmclnt.Start() != nil {
		gs = false
	}
	if n.bcclnt.Start() != nil {
		ts = false
	}
	if gs && ts {
		n.SetNodeStatus(types.Up)
	}
	// if start up of blockchain client or privacy manager fails Status will remain as StartupInprogress and node manager will not process any requests from clients
	// it will need some manual intervention to set it to the correct status
	return gs && ts
}

func (n *NodeControl) PrepareNodeManagerForPrivateTx(privateFor []string) (bool, error) {
	return n.nm.ValidatePeerPrivateTxStatus(privateFor)
}
