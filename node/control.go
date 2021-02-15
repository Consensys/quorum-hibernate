package node

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/ConsenSys/quorum-hibernate/config"
	cons "github.com/ConsenSys/quorum-hibernate/consensus"
	besu "github.com/ConsenSys/quorum-hibernate/consensus/besu"
	qnm "github.com/ConsenSys/quorum-hibernate/consensus/quorum"
	"github.com/ConsenSys/quorum-hibernate/core"
	"github.com/ConsenSys/quorum-hibernate/log"
	"github.com/ConsenSys/quorum-hibernate/p2p"
	"github.com/ConsenSys/quorum-hibernate/privatetx"
	proc "github.com/ConsenSys/quorum-hibernate/process"
)

const CONSENSUS_WAIT_TIME = 60

// NodeControl represents a node manager controller.
// It implements ControllerApiService
// It tracks blockchain client/privacyManager processes' inactivity and it allows inactivity to be reset when
// there is some activity.
// It accepts request to stop blockchain client/privacyManager when there is inactivity.
// It starts blockchain client/privacyManager processes when there is a activity.
// It takes care of managing combined status of blockchain client & privacyManager.
type NodeControl struct {
	config              *config.Node             // config of this node
	im                  *InactivityResyncMonitor // inactivity monitor
	nm                  *p2p.PeerManager         // node manager to communicate with other node manager
	bcclntProcess       proc.Process             // blockchain client process controller
	pmclntProcess       proc.Process             // privacy manager process controller
	bcclntHttpClient    *http.Client             // blockchain client http client
	pmclntHttpClient    *http.Client             // privacy manager http client
	consensus           cons.Consensus           // consensus validator
	txh                 privatetx.TxHandler      // Transaction handler
	withPrivMan         bool                     // indicates if the node is running with a privacy manage
	consValid           bool                     // indicates if network level consensus is valid
	clientStatus        core.ClientStatus        // combined status of blockchain client and privacy manager processes
	nodeStatus          core.NodeStatus          // status of node manager
	inactivityResetCh   chan bool                // channel to reset inactivity
	syncResetCh         chan bool                // channel to reset sync timer
	stopClntCh          chan bool                // channel to request stop node
	stopClntCompleteCh  chan bool                // channel to notify stop node action status
	startClntCh         chan bool                // channel to request start node
	startClntCompleteCh chan bool                // channel to notify start node action status
	stopCh              chan bool                // channel to stop start/stop node monitor
	clntStatMonStopCh   chan bool                // channel to stop node status monitor
	startStopMux        sync.Mutex               // lock for starting and stopping node
	clntStatusMux       sync.Mutex               // lock for setting the client status
	nodeStatusMux       sync.Mutex               // lock for setting the node status
}

func (n *NodeControl) ClientStatus() core.ClientStatus {
	n.clntStatusMux.Lock()
	defer n.clntStatusMux.Unlock()
	return n.clientStatus
}

func NewNodeControl(cfg *config.Node) *NodeControl {
	node := &NodeControl{
		config:              cfg,
		nm:                  p2p.NewPeerManager(cfg),
		withPrivMan:         cfg.BasicConfig.PrivacyManager != nil,
		nodeStatus:          core.OK,
		inactivityResetCh:   make(chan bool, 1),
		syncResetCh:         make(chan bool, 1),
		stopClntCh:          make(chan bool, 1),
		stopClntCompleteCh:  make(chan bool, 1),
		startClntCh:         make(chan bool, 1),
		startClntCompleteCh: make(chan bool, 1),
		stopCh:              make(chan bool, 1),
		clntStatMonStopCh:   make(chan bool, 1),
	}

	setHttpClients(cfg, node)

	if cfg.BasicConfig.BlockchainClient.BcClntProcess.IsShell() {
		node.bcclntProcess = proc.NewShellProcess(node.bcclntHttpClient, cfg.BasicConfig.BlockchainClient.BcClntProcess, true)
	} else if cfg.BasicConfig.BlockchainClient.BcClntProcess.IsDocker() {
		node.bcclntProcess = proc.NewDockerProcess(node.bcclntHttpClient, cfg.BasicConfig.BlockchainClient.BcClntProcess, true)
	}

	if node.WithPrivMan() {
		if cfg.BasicConfig.PrivacyManager.PrivManProcess.IsShell() {
			node.pmclntProcess = proc.NewShellProcess(node.pmclntHttpClient, cfg.BasicConfig.PrivacyManager.PrivManProcess, true)
		} else if cfg.BasicConfig.PrivacyManager.PrivManProcess.IsDocker() {
			node.pmclntProcess = proc.NewDockerProcess(node.pmclntHttpClient, cfg.BasicConfig.PrivacyManager.PrivManProcess, true)
		}
	}
	node.im = NewInactivityResyncMonitor(node)
	populateConsensusHandler(node)
	if node.config.BasicConfig.IsGoQuorumClient() {
		node.txh = privatetx.NewQuorumTxHandler(node.config)
	} // TODO add tx handler for Besu
	node.config.BasicConfig.InactivityTime += getRandomBufferTime(node.config.BasicConfig.InactivityTime)
	log.Debug("Node config - inactivity time after random buffer", "InactivityTime", node.config.BasicConfig.InactivityTime)
	return node
}

func setHttpClients(cfg *config.Node, node *NodeControl) {
	if cfg.BasicConfig.BlockchainClient.BcClntTLSConfig != nil {
		node.bcclntHttpClient = core.NewHttpClient(cfg.BasicConfig.BlockchainClient.BcClntTLSConfig.TlsCfg)
	} else {
		node.bcclntHttpClient = core.NewHttpClient(nil)
	}

	if node.WithPrivMan() {
		if cfg.BasicConfig.PrivacyManager.PrivManTLSConfig != nil {
			node.pmclntHttpClient = core.NewHttpClient(cfg.BasicConfig.PrivacyManager.PrivManTLSConfig.TlsCfg)
		} else {
			node.pmclntHttpClient = core.NewHttpClient(nil)
		}
	}
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
	return core.RandomInt(1, delay)
}

func populateConsensusHandler(n *NodeControl) {
	if n.config.BasicConfig.IsGoQuorumClient() {
		if n.config.BasicConfig.IsRaft() {
			n.consensus = qnm.NewRaftConsensus(n.config, n.bcclntHttpClient)
		} else if n.config.BasicConfig.IsIstanbul() {
			n.consensus = qnm.NewIstanbulConsensus(n.config, n.bcclntHttpClient)
		} else if n.config.BasicConfig.IsClique() {
			n.consensus = qnm.NewCliqueConsensus(n.config, n.bcclntHttpClient)
		}
	} else if n.config.BasicConfig.IsBesuClient() {
		if n.config.BasicConfig.IsClique() {
			n.consensus = besu.NewCliqueConsensus(n.config, n.bcclntHttpClient)
		}
	}
}

func (n *NodeControl) GetRPCConfig() *config.RPCServer {
	return n.config.BasicConfig.Server
}

func (n *NodeControl) GetNodeConfig() *config.Node {
	return n.config
}

func (n *NodeControl) GetNodeStatus() core.NodeStatus {
	n.nodeStatusMux.Lock()
	defer n.nodeStatusMux.Unlock()
	return n.nodeStatus
}

func (n *NodeControl) GetProxyConfig() []*config.Proxy {
	return n.config.BasicConfig.Proxies
}

func (n *NodeControl) GetTxHandler() privatetx.TxHandler {
	return n.txh
}

func (n *NodeControl) SetClntStatus(ns core.ClientStatus) {
	defer n.clntStatusMux.Unlock()
	n.clntStatusMux.Lock()
	log.Debug("SetClntStatus", "old", n.clientStatus, "new", ns)
	n.clientStatus = ns
}

func (n *NodeControl) SetNodeStatus(ns core.NodeStatus) {
	defer n.nodeStatusMux.Unlock()
	n.nodeStatusMux.Lock()
	log.Debug("SetNodeStatus", "old", n.nodeStatus, "new", ns)
	n.nodeStatus = ns
}

func (n *NodeControl) IsClientUp() bool {
	return n.ClientStatus() == core.Up
}

// CheckClientUpStatus performs up check for blockchain client and privacy manager and returns the combined status
// If both blockchain client and privacy manager are up, it returns true else returns false
// If connectToClient is true it connects to client to check the actual status otherwise it returns the status based on check done by status monitor
func (n *NodeControl) CheckClientUpStatus(connectToClient bool) bool {
	// it is possible that QNM status of the node is down and the node was brought up
	// in such cases, with forceMode true, a direct call to client is done to get the
	// real status
	if !n.IsClientUp() && !connectToClient {
		return false
	}

	bcclntStatus, pmStatus := n.fetchCurrentClientStatuses()
	log.Debug("CheckClientUpStatus", "blockchain client", bcclntStatus, "privacy manager", pmStatus)

	areClientsUp := bcclntStatus && pmStatus

	if areClientsUp {
		n.SetClntStatus(core.Up)
	} else {
		n.SetClntStatus(core.Down)
	}
	return areClientsUp
}

// fetchCurrentClientStatuses gets the current statuses of the blockchain client and privacy manager in parallel
func (n *NodeControl) fetchCurrentClientStatuses() (bool, bool) {
	var bcclntStatus bool
	var wg = sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		bcclntStatus = n.bcclntProcess.UpdateStatus()
	}()

	pmStatus := true
	if n.WithPrivMan() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pmStatus = n.pmclntProcess.UpdateStatus()
		}()
	}

	wg.Wait()
	return bcclntStatus, pmStatus
}

// IsNodeBusy returns error if the node manager is busy with shutdown/startup
func (n *NodeControl) IsNodeBusy() error {
	switch n.GetNodeStatus() {
	case core.ShutdownInprogress:
		return errors.New(core.NodeIsBeingShutdown)
	case core.StartupInprogress:
		return errors.New(core.NodeIsBeingStarted)
	case core.OK:
		return nil
	}
	return nil
}

// Start starts blockchain client and privacy manager start/stop monitor and inactivity tracker
func (n *NodeControl) Start() {
	n.StartNodeMonitor()
	n.startClientStatusMonitor()
	n.im.Start()
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
	if n.config.BasicConfig.IsResyncTimerSet() {
		n.syncResetCh <- true
	}
}

func (n *NodeControl) startClientStatusMonitor() {
	go func() {
		var (
			isClientUp bool
			timer      = time.NewTicker(time.Duration(n.config.BasicConfig.UpchkPollingInterval) * time.Second)
			init       = false
		)
		defer timer.Stop()

		log.Info("clientStatusMonitor started")
		for {
			select {
			case <-timer.C:
				if !init {
					isClientUp = n.CheckClientUpStatus(true)
					init = true
				} else {
					isClientUp = n.CheckClientUpStatus(false)
				}
				log.Debug("clientStatusMonitor", "isClientUp", isClientUp)
				continue
			case <-n.clntStatMonStopCh:
				log.Info("clientStatusMonitor stopped")
				return
			}
		}
	}()
}

//StartNodeMonitor listens for requests to stop blockchain client and privacy manager and
// stops blockchain client and privacy manager when a request is received
func (n *NodeControl) StartNodeMonitor() {
	go func() {
		log.Info("StartNodeMonitor - node start/stop monitor started")
		for {
			select {
			case <-n.stopClntCh:
				log.Debug("StartNodeMonitor - request received to stop node")
				if !n.StopClient() {
					log.Error("StartNodeMonitor - stopping failed")
					n.stopClntCompleteCh <- false
				} else {
					log.Debug("StartNodeMonitor - stopping complete")
					n.stopClntCompleteCh <- true
				}
			case <-n.startClntCh:
				log.Debug("StartNodeMonitor - request received to start node")
				if !n.StartClient() {
					log.Error("StartNodeMonitor - starting failed")
					n.startClntCompleteCh <- false
				} else {
					log.Debug("StartNodeMonitor - starting complete")
					n.startClntCompleteCh <- true
				}
			case <-n.stopCh:
				log.Info("StartNodeMonitor - stopped node start/stop monitor service")
				return
			}
		}
	}()
}

func (n *NodeControl) RequestStartClient() {
	n.startClntCh <- true
}

func (n *NodeControl) RequestStopClient() {
	n.stopClntCh <- true
}

func (n *NodeControl) WaitStartClient() bool {
	status := <-n.startClntCompleteCh
	return status
}

func (n *NodeControl) WaitStopClient() bool {
	status := <-n.stopClntCompleteCh
	return status
}

func (n *NodeControl) PrepareClient() bool {
	log.Debug("PrepareClient - starting node")
	status := n.StartClient()
	log.Debug("PrepareClient - node start completed", "status", status)
	return status
}

func (n *NodeControl) StopClient() bool {
	defer n.startStopMux.Unlock()
	n.startStopMux.Lock()

	if !n.IsClientUp() {
		log.Debug("StopClient - node is already down")
		return true
	}
	if err := n.IsNodeBusy(); err != nil {
		log.Error("StopClient - cannot be shutdown", "err", err)
		return false
	}
	var peersStatus []p2p.NodeStatusInfo
	var err error

	consensusNode, err := n.checkAndValidateConsensus()
	if err != nil {
		log.Info("StopClient - consensus check failed, node cannot be shutdown", "err", err)
		n.SetNodeStatus(core.OK)
		return false
	}
	log.Info("StopClient - consensus check passed, node can be shutdown")

	if consensusNode && !n.config.BasicConfig.DisableStrictMode {
		// consensus node running in strict mode. node cannot be brought down
		log.Info("StopClient - node manager running in strict mode. consensus node cannot be shut down")
		return false
	}

	// consensus is ok. check with network to prevent multiple nodes
	// going down at the same time
	w := core.RandomInt(10, 5000)
	log.Info("StopClient - waiting for p2p validation try", "wait time in milliseconds", w)
	time.Sleep(time.Duration(w) * time.Millisecond)
	n.SetNodeStatus(core.ShutdownInprogress)

	if peersStatus, err = n.nm.ValidatePeers(); err != nil {
		n.SetNodeStatus(core.OK)
		log.Error("StopClient - node cannot be shutdown, p2p validation failed", "err", err)
		return false
	}
	log.Info("StopClient - all checks passed for shutdown", "peerStatus", peersStatus)

	bcStatus, pmStatus := n.stopProcesses()
	if bcStatus && pmStatus {
		log.Debug("StopClient - bcclnt and privman processes stopped")
		n.SetClntStatus(core.Down)

		// for IBFT and Clique, since we rely on block signed data
		// do not want to mark the QNM status as OK immediately
		// want to allow enough sleep period so that the consensus
		// engine can mint enough new blocks before another node hibernates
		n.SetNodeStatus(core.ConsensusWait)
		time.Sleep(CONSENSUS_WAIT_TIME * time.Second)
		n.SetNodeStatus(core.OK)
		log.Debug("StopClient", "nodeStatus", n.GetNodeStatus())
	} else {
		log.Error("StopClient - bcclnt and privman processes not stopped")
	}
	// if stopping of blockchain client or privacy manager fails Status will remain as ShutdownInprogress and node manager will not process any requests from clients
	// it will need some manual intervention to set it to the correct status
	return bcStatus && pmStatus
}

func (n *NodeControl) checkAndValidateConsensus() (bool, error) {
	// validate if the consensus passed in config is correct.
	// for besu bypass this check as it does not provide any rpc api to confirm consensus
	if !n.consValid && n.config.BasicConfig.IsGoQuorumClient() {
		if err := n.config.IsConsensusValid(n.pmclntHttpClient); err != nil {
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
		if n.bcclntProcess.Stop() != nil {
			gs = false
		}
	}()
	if n.pmclntProcess != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if n.pmclntProcess.Stop() != nil {
				ts = false
			}
		}()
	}
	wg.Wait()
	return gs, ts
}

func (n *NodeControl) StartClient() bool {
	defer n.startStopMux.Unlock()
	n.startStopMux.Lock()
	// if the node status is down, enfornce client check to get the true client status
	// before initiating start up. This is to handle scenarios where the node was
	// brought up in the backend bypassing QNM
	if n.IsClientUp() || (!n.IsClientUp() && n.CheckClientUpStatus(true)) {
		log.Debug("StartClient - node is already up")
		return true
	}
	n.SetNodeStatus(core.StartupInprogress)
	gs := true
	ts := true
	if n.withPrivMan && n.pmclntProcess.Start() != nil {
		gs = false
	}
	if n.bcclntProcess.Start() != nil {
		ts = false
	}
	if gs && ts {
		n.SetClntStatus(core.Up)
		n.SetNodeStatus(core.OK)
	}
	// if start up of blockchain client or privacy manager fails Status will remain as StartupInprogress and node manager will not process any requests from clients
	// it will need some manual intervention to set it to the correct status
	return gs && ts
}

func (n *NodeControl) PrepareNodeManagerForPrivateTx(privateFor []string) (bool, error) {
	return n.nm.ValidatePeerPrivateTxStatus(privateFor)
}

func (n *NodeControl) GetInactivityTimeCount() int {
	return n.im.GetInactivityTimeCount()
}
