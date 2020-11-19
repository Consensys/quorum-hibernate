package node

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/qnm"

	cons "github.com/ConsenSysQuorum/node-manager/consensus"
	"github.com/ConsenSysQuorum/node-manager/core"
	proc "github.com/ConsenSysQuorum/node-manager/process"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

type QuorumNodeControl struct {
	config             *types.NodeConfig
	im                 *InactivityMonitor
	nm                 *qnm.NodeManager
	gethp              proc.Process
	tesserap           proc.Process
	consensus          cons.Consensus
	nodeStatus         types.NodeStatus
	client             *http.Client
	inactivityResetCh  chan bool
	stopNodeCh         chan bool
	shutdownCompleteCh chan bool
	startNodeCh        chan bool
	startCompleteCh    chan bool
	stopCh             chan bool
	startStopMux       sync.Mutex
	statusMux          sync.Mutex
}

func NewQuorumNodeControl(cfg *types.NodeConfig) *QuorumNodeControl {
	quorumNode := &QuorumNodeControl{
		cfg,
		nil,
		qnm.NewNodeManager(cfg),
		nil,
		nil,
		nil,
		types.Up,
		core.NewHttpClient(),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		sync.Mutex{},
		sync.Mutex{},
	}

	if cfg.GethProcess.IsShell() {
		quorumNode.gethp = proc.NewShellProcess(cfg.GethProcess, cfg.GethRpcUrl, cfg.TesseraUpcheckUrl, true)
	} else if cfg.GethProcess.IsDocker() {
		quorumNode.gethp = proc.NewDockerProcess(cfg.GethProcess, cfg.GethRpcUrl, cfg.TesseraUpcheckUrl, true)
	}

	if cfg.TesseraProcess.IsShell() {
		quorumNode.tesserap = proc.NewShellProcess(cfg.TesseraProcess, cfg.GethRpcUrl, cfg.TesseraUpcheckUrl, true)
	} else if cfg.TesseraProcess.IsDocker() {
		quorumNode.tesserap = proc.NewDockerProcess(cfg.TesseraProcess, cfg.GethRpcUrl, cfg.TesseraUpcheckUrl, true)
	}

	if quorumNode.gethp.Status() && quorumNode.tesserap.Status() {
		quorumNode.SetNodeStatus(types.Up)
	} else {
		quorumNode.SetNodeStatus(types.Down)
	}

	if quorumNode.config.IsRaft() {
		quorumNode.consensus = cons.NewRaftConsensus(quorumNode.config)
	} else if quorumNode.config.IsIstanbul() {
		quorumNode.consensus = cons.NewIstanbulConsensus(quorumNode.config)
	}
	return quorumNode
}

func (qn *QuorumNodeControl) GetRPCConfig() *types.RPCServerConfig {
	return qn.config.Server
}

func (qn *QuorumNodeControl) GetNodeConfig() *types.NodeConfig {
	return qn.config
}

func (qn *QuorumNodeControl) GetNodeStatus() types.NodeStatus {
	return qn.nodeStatus
}

func (qn *QuorumNodeControl) GetProxyConfig() []*types.ProxyConfig {
	return qn.config.Proxies
}

func (qn *QuorumNodeControl) SetNodeStatus(ns types.NodeStatus) {
	defer qn.statusMux.Unlock()
	qn.statusMux.Lock()
	qn.nodeStatus = ns
}

func (qn *QuorumNodeControl) IsNodeUp() bool {
	gs := qn.gethp.IsUp()
	ts := qn.tesserap.IsUp()
	log.Info("IsNodeUp", "geth", gs, "tessera", ts)
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
	qn.im = NewNodeInactivityMonitor(qn)
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
		log.Info("node start/stop monitor started")
		for {
			select {
			case <-qn.stopNodeCh:
				log.Info("request received to stop node")
				if !qn.StopNode() {
					log.Error("stopping failed")
					qn.shutdownCompleteCh <- false
				} else {
					qn.shutdownCompleteCh <- true
				}
			case <-qn.startNodeCh:
				log.Info("request received to start node")
				if !qn.StartNode() {
					log.Error("starting failed")
					qn.startCompleteCh <- false
				} else {
					qn.startCompleteCh <- true
				}
			case <-qn.stopCh:
				log.Info("stopped node start/stop monitor service")
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
		log.Info("node start completed", "status", status)
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
		log.Info("node is already down")
		return true
	}
	var qnms []qnm.NodeStatusInfo
	var err error

	retryCount := 1

	for retryCount <= core.Qnm2QnmValidationRetryLimit {
		qnms, err = qn.nm.ValidateOtherQnms()
		if err == nil {
			log.Info("qnm2qnm validation passed")
			break
		}
		log.Error("qnm2qnm validation failed", "retryLimit", core.Qnm2QnmValidationRetryLimit, "retryCount", retryCount, "err", err, "qnms", qnms)
		retryCount++
		w := core.GetRandomRetryWaitTime()
		log.Info("waiting for next qnm2qnm validation try", "wait time in seconds", w)
		time.Sleep(time.Duration(w) * time.Millisecond)
	}

	if retryCount > core.Qnm2QnmValidationRetryLimit {
		log.Error("node cannot be shutdown, qnm2qnm validation failed after retrying")
		return false
	}

	qn.SetNodeStatus(types.ShutdownInitiated)

	if err := qn.consensus.ValidateShutdown(); err == nil {
		log.Info("consensus check passed, node can be shutdown")
	} else {
		log.Info("consensus check failed, node cannot be shutdown", "err", err)
		qn.SetNodeStatus(types.Up)
		return false
	}

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
	return gs && ts
}

func (qn *QuorumNodeControl) PrepareNodeManagerForPrivateTx(privateFor []string) (bool, error) {
	return qn.nm.ValidateForPrivateTx(privateFor)
}
