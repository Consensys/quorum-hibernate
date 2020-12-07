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

const CONSENSUS_WAIT_TIME = 60

// NodeControl represents a node manager controller.
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
	consensus          cons.Consensus       // consensus validator
	txh                privatetx.TxHandler  // Transaction handler
	withPrivMan        bool                 // indicates if the node is running with a privacy manage
	consValid          bool                 // indicates if network level consensus is valid
	clientStatus       types.ClientStatus   // combined status of blockchain client and privacy manager processes
	nodeStatus         types.NodeStatus     // status of node manager
	inactivityResetCh  chan bool            // channel to reset inactivity
	syncResetCh        chan bool            // channel to reset sync timer
	stopClntCh         chan bool            // channel to request stop node
	stopClntCompleteCh chan bool            // channel to notify stop node action status
	stopCh             chan bool            // channel to stop start/stop node monitor
	clntStatMonStopCh  chan bool            // channel to stop node status monitor
	startStopMux       sync.Mutex           // lock for starting and stopping node
	clntStatusMux      sync.Mutex           // lock for setting the client status
	nodeStatusMux      sync.Mutex           // lock for setting the node status
}

func (n *NodeControl) ClientStatus() types.ClientStatus {
	n.clntStatusMux.Lock()
	defer n.clntStatusMux.Unlock()
	return n.clientStatus
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
		types.OK,
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		sync.Mutex{},
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

func (n *NodeControl) SetClntStatus(ns types.ClientStatus) {
	defer n.clntStatusMux.Unlock()
	n.clntStatusMux.Lock()
	n.clientStatus = ns
}

func (n *NodeControl) SetNodeStatus(ns types.NodeStatus) {
	defer n.nodeStatusMux.Unlock()
	n.nodeStatusMux.Lock()
	n.nodeStatus = ns
}

// IsClientUp performs up check for blockchain client and privacy manager and returns the combined status
// if both blockchain client and privacy manager are up, the node status is up(true) else down(false)
func (n *NodeControl) IsClientUp(connectToClient bool) bool {
	// it is possible that QNM status of the node is down and the node was brought up
	// in such cases, with forceMode true, a direct call to client is done to get the
	// real status
	if n.ClientStatus() != types.Down || connectToClient {
		bcclntStatus, pmStatus := n.checkUpStatus()
		log.Debug("IsClientUp", "blockchain client", bcclntStatus, "privacy manager", pmStatus)
		if bcclntStatus && pmStatus {
			n.SetClntStatus(types.Up)
		} else {
			n.SetClntStatus(types.Down)
		}
		return bcclntStatus && pmStatus
	}
	return false
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

// IsNodeBusy returns error if the node manager is busy with shutdown/startup
func (n *NodeControl) IsNodeBusy() error {
	switch n.nodeStatus {
	case types.ShutdownInprogress:
		return errors.New(core.NodeIsBeingShutdown)
	case types.StartupInprogress:
		return errors.New(core.NodeIsBeingStarted)
	case types.OK:
		return nil
	}
	return nil
}

// Start starts blockchain client and privacy manager start/stop monitor and inactivity tracker
func (n *NodeControl) Start() {
	n.StopNodeMonitor()
	n.im = NewInactivityMonitor(n)
	n.im.StartInactivitySyncTimer()
	n.startClientStatusMonitor()
}

// Stop stops blockchain client and privacy manager start/stop monitor and inactivity tracker
func (n *NodeControl) Stop() {
	n.im.Stop()
	n.stopCh <- true
	n.clntStatMonStopCh <- true
}

// ResetInactiveSyncTime resets inactivity time of the tracker
func (n *NodeControl) ResetInactiveSyncTime() {
	n.inactivityResetCh <- true
	n.syncResetCh <- true
}

func (n *NodeControl) startClientStatusMonitor() {
	go func() {
		var (
			isClientUp bool
			timer      = time.NewTicker(time.Duration(n.config.BasicConfig.UpchkPollingInterval) * time.Second)
		)
		defer timer.Stop()

		log.Info("clientStatusMonitor started")
		for {
			isClientUp = n.IsClientUp(false)
			log.Debug("clientStatusMonitor", "isClientUp", isClientUp)
			select {
			case <-timer.C:
				continue
			case <-n.clntStatMonStopCh:
				log.Info("clientStatusMonitor stopped")
				return
			}
		}
	}()
}

//StopNodeMonitor listens for requests to start/stop blockchain client and privacy manager
func (n *NodeControl) StopNodeMonitor() {
	go func() {
		log.Info("StopNodeMonitor - node start/stop monitor started")
		for {
			select {
			case <-n.stopClntCh:
				log.Debug("StopNodeMonitor - request received to stop node")
				if !n.StopClient() {
					log.Error("StopNodeMonitor - stopping failed")
					n.stopClntCompleteCh <- false
				} else {
					n.stopClntCompleteCh <- true
				}
			case <-n.stopCh:
				log.Info("StopNodeMonitor - stopped node start/stop monitor service")
				return
			}
		}
	}()
}

func (n *NodeControl) RequestStopClient() {
	n.stopClntCh <- true
}

func (n *NodeControl) WaitStopClient() bool {
	status := <-n.stopClntCompleteCh
	return status
}

// TODO handle error if node failed to start
func (n *NodeControl) PrepareClient() bool {
	log.Debug("PrepareClient - starting node")
	status := n.StartClient()
	log.Debug("PrepareClient - node start completed", "status", status)
	return status
}

func (n *NodeControl) StopClient() bool {
	defer n.startStopMux.Unlock()
	n.startStopMux.Lock()

	if n.ClientStatus() == types.Down {
		log.Debug("StopClient - node is already down")
		return true
	}
	if err := n.IsNodeBusy(); err != nil {
		log.Error("StopClient - cannot be shutdown", "err", err)
		return false
	}
	var peersStatus []nodeman.NodeStatusInfo
	var err error

	consensusNode, err := n.checkAndValidateConsensus()
	if err != nil {
		log.Info("StopClient - consensus check failed, node cannot be shutdown", "err", err)
		n.SetNodeStatus(types.OK)
		return false
	}
	log.Info("StopClient - consensus check passed, node can be shutdown")

	if consensusNode && n.config.BasicConfig.RunMode == types.STRICT_MODE {
		// consensus node running in strict mode. node cannot be brouwght down
		log.Info("StopClient - consensus node running in strict mode. cannot be shut down")
		return false
	}

	// consensus is ok. check with network to prevent multiple nodes
	// going down at the same time
	w := core.GetRandomRetryWaitTime(10, 5000)
	log.Info("StopClient - waiting for p2p validation try", "wait time in seconds", w)
	time.Sleep(time.Duration(w) * time.Millisecond)
	n.SetNodeStatus(types.ShutdownInprogress)

	if peersStatus, err = n.nm.ValidatePeers(); err != nil {
		n.SetNodeStatus(types.OK)
		log.Error("StopClient - node cannot be shutdown, p2p validation failed")
		return false
	}
	log.Info("StopClient - all checks passed for shutdown", "peerStatus", peersStatus)

	bcStatus, pmStatus := n.stopProcesses()
	if bcStatus && pmStatus {
		n.SetClntStatus(types.Down)

		// for IBFT and Clique, since we rely on block signed data
		// do not want to mark the QNM status as OK immediately
		// want to allow enough sleep period so that the consensus
		// engine can mint enough new blocks before another node hibernates
		n.SetNodeStatus(types.ConsensusWait)
		time.Sleep(CONSENSUS_WAIT_TIME * time.Second)
		n.SetNodeStatus(types.OK)

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

func (n *NodeControl) StartClient() bool {
	defer n.startStopMux.Unlock()
	n.startStopMux.Lock()
	// if the node status is down, enfornce client check to get the true client status
	// before initiating start up. This is to handle scenarios where the node was
	// brought up in the backend bypassing QNM
	if n.ClientStatus() == types.Up || (n.ClientStatus() == types.Down && n.IsClientUp(true)) {
		log.Debug("StartClient - node is already up")
		return true
	}
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
		n.SetClntStatus(types.Up)
		n.SetNodeStatus(types.OK)
	}
	// if start up of blockchain client or privacy manager fails Status will remain as StartupInprogress and node manager will not process any requests from clients
	// it will need some manual intervention to set it to the correct status
	return gs && ts
}

func (n *NodeControl) PrepareNodeManagerForPrivateTx(privateFor []string) (bool, error) {
	return n.nm.ValidatePeerPrivateTxStatus(privateFor)
}
