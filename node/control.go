package node

import (
	"errors"
	"github.com/ConsenSysQuorum/node-manager/privatetx"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/qnm"

	cons "github.com/ConsenSysQuorum/node-manager/consensus"
	"github.com/ConsenSysQuorum/node-manager/core"
	proc "github.com/ConsenSysQuorum/node-manager/process"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

// QuorumNodeControl represents a quorum node controller.
// it takes care of managing combined status of geth and tessera.
// it takes care of starting & stopping of geth and tessera.
type QuorumNodeControl struct {
	config             *types.NodeConfig   // config of this node
	im                 *InactivityMonitor  // inactivity monitor
	nm                 *qnm.NodeManager    // node manager to communicate with other qnm
	gethp              proc.Process        // geth process controller
	tesserap           proc.Process        // tessera process controller
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

func NewQuorumNodeControl(cfg *types.NodeConfig) *QuorumNodeControl {
	quorumNode := &QuorumNodeControl{
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

	if cfg.BasicConfig.GethProcess.IsShell() {
		quorumNode.gethp = proc.NewShellProcess(cfg.BasicConfig.GethProcess, cfg.BasicConfig.GethRpcUrl, cfg.BasicConfig.TesseraUpcheckUrl, true)
	} else if cfg.BasicConfig.GethProcess.IsDocker() {
		quorumNode.gethp = proc.NewDockerProcess(cfg.BasicConfig.GethProcess, cfg.BasicConfig.GethRpcUrl, cfg.BasicConfig.TesseraUpcheckUrl, true)
	}

	if cfg.BasicConfig.TesseraProcess.IsShell() {
		quorumNode.tesserap = proc.NewShellProcess(cfg.BasicConfig.TesseraProcess, cfg.BasicConfig.GethRpcUrl, cfg.BasicConfig.TesseraUpcheckUrl, true)
	} else if cfg.BasicConfig.TesseraProcess.IsDocker() {
		quorumNode.tesserap = proc.NewDockerProcess(cfg.BasicConfig.TesseraProcess, cfg.BasicConfig.GethRpcUrl, cfg.BasicConfig.TesseraUpcheckUrl, true)
	}

	if quorumNode.gethp.Status() && quorumNode.tesserap.Status() {
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

func (qn *QuorumNodeControl) GetRPCConfig() *types.RPCServerConfig {
	return qn.config.BasicConfig.Server
}

func (qn *QuorumNodeControl) GetNodeConfig() *types.NodeConfig {
	return qn.config
}

func (qn *QuorumNodeControl) GetNodeStatus() types.NodeStatus {
	return qn.nodeStatus
}

func (qn *QuorumNodeControl) GetProxyConfig() []*types.ProxyConfig {
	return qn.config.BasicConfig.Proxies
}

func (qn *QuorumNodeControl) GetTxHandler() privatetx.TxHandler {
	return qn.txh
}

func (qn *QuorumNodeControl) SetNodeStatus(ns types.NodeStatus) {
	defer qn.statusMux.Unlock()
	qn.statusMux.Lock()
	qn.nodeStatus = ns
}

func (qn *QuorumNodeControl) IsNodeUp() bool {
	gs := qn.gethp.IsUp()
	ts := qn.tesserap.IsUp()
	log.Debug("IsNodeUp", "geth", gs, "tessera", ts)
	if gs && ts {
		qn.SetNodeStatus(types.Up)
	} else {
		qn.SetNodeStatus(types.Down)
	}
	return gs && ts
}

func (qn *QuorumNodeControl) IsNodeBusy() error {
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

func (qn *QuorumNodeControl) Start() {
	qn.StartStopNodeMonitor()
	qn.im = NewInactivityMonitor(qn)
	qn.im.StartInactivityTimer()
}

func (qn *QuorumNodeControl) Stop() {
	qn.im.Stop()
	qn.stopCh <- true
}

func (nm *QuorumNodeControl) ResetInactiveTime() {
	nm.inactivityResetCh <- true
}

func (qn *QuorumNodeControl) StartStopNodeMonitor() {
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

func (qn *QuorumNodeControl) RequestStartNode() {
	qn.startNodeCh <- true
}

func (qn *QuorumNodeControl) RequestStopNode() {
	qn.stopNodeCh <- true
}

func (qn *QuorumNodeControl) WaitStartNode() bool {
	status := <-qn.startCompleteCh
	return status
}

func (qn *QuorumNodeControl) WaitStopNode() bool {
	status := <-qn.shutdownCompleteCh
	return status
}

// TODO handle error if node failed to start
func (qn *QuorumNodeControl) PrepareNode() bool {
	if !qn.IsNodeUp() {
		status := qn.StartNode()
		log.Debug("PrepareNode - node start completed", "status", status)
		return status
	} else {
		log.Info("node is UP")
		return true
	}
}

func (qn *QuorumNodeControl) StopNode() bool {
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

	// 1st check if hibernating node will break the consensus model
	if err := qn.consensus.ValidateShutdown(); err == nil {
		log.Info("StopNode - consensus check passed, node can be shutdown")
	} else {
		log.Info("StopNode - consensus check failed, node cannot be shutdown", "err", err)
		qn.SetNodeStatus(types.Up)
		return false
	}

	// consensus is ok. check with network to prevent multiple nodes
	// going down at the same time
	retryCount := 1
	for retryCount <= core.Qnm2QnmValidationRetryLimit {
		qnms, err = qn.nm.ValidateOtherQnms()
		if err == nil {
			log.Info("StopNode - qnm2qnm validation passed")
			break
		}
		log.Error("StopNode - qnm2qnm validation failed", "retryLimit", core.Qnm2QnmValidationRetryLimit, "retryCount", retryCount, "err", err, "qnms", qnms)
		retryCount++
		w := core.GetRandomRetryWaitTime()
		log.Info("StopNode - waiting for next qnm2qnm validation try", "wait time in seconds", w)
		time.Sleep(time.Duration(w) * time.Millisecond)
	}

	if retryCount > core.Qnm2QnmValidationRetryLimit {
		log.Error("StopNode - node cannot be shutdown, qnm2qnm validation failed after retrying")
		return false
	}

	qn.SetNodeStatus(types.ShutdownInitiated)

	qn.SetNodeStatus(types.ShutdownInprogress)

	// TODO parallelize / loop
	gs := true
	ts := true
	if qn.gethp.Stop() != nil {
		gs = false
	}
	if qn.tesserap.Stop() != nil {
		ts = false
	}
	if gs && ts {
		qn.SetNodeStatus(types.Down)
	}
	// if stopping of geth or tessera fails Status will remain as ShutdownInprogress and qnm will not process any requests from clients
	// it will need some manual intervention to set it to the correct status
	return gs && ts
}

func (qn *QuorumNodeControl) StartNode() bool {
	defer qn.startStopMux.Unlock()
	qn.startStopMux.Lock()
	qn.SetNodeStatus(types.StartupInitiated)
	qn.SetNodeStatus(types.StartupInprogress)
	gs := true
	ts := true
	if qn.tesserap.Start() != nil {
		gs = false
	}
	if qn.gethp.Start() != nil {
		ts = false
	}
	if gs && ts {
		qn.SetNodeStatus(types.Up)
	}
	// if start up of geth or tessera fails Status will remain as StartupInprogress and qnm will not process any requests from clients
	// it will need some manual intervention to set it to the correct status
	return gs && ts
}

func (qn *QuorumNodeControl) PrepareNodeManagerForPrivateTx(privateFor []string) (bool, error) {
	return qn.nm.ValidateForPrivateTx(privateFor)
}
