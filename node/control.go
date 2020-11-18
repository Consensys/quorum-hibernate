package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/core"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

type NodeManagerPrivateTxPrepResult struct {
	Result PrivateTxPrepStatus `json:"result"`
	Error  error               `json:"error"`
}

type NodeManagerNodeStatusResult struct {
	Result NodeStatusInfo `json:"result"`
	Error  error          `json:"error"`
}

type PrivateTxPrepStatus struct {
	Status bool `json:"status"`
}

type NodeStatus uint8

const (
	ShutdownInitiated NodeStatus = iota
	ShutdownInprogress
	StartupInitiated
	StartupInprogress
	Up
	Down
)

type NodeStatusInfo struct {
	Status            NodeStatus
	InactiveTimeLimit int
	InactiveTime      int
	TimeToShutdown    int
}

type QuorumNodeControl struct {
	config             *types.NodeConfig
	im                 *InactivityMonitor
	gethp              Process
	tesserap           Process
	consensus          Consensus
	nodeStatus         NodeStatus
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

var ErrNodeDown = errors.New("node is not up")

var quorumNode *QuorumNodeControl

func NewQuorumNodeControl(cfg *types.NodeConfig) *QuorumNodeControl {
	quorumNode = &QuorumNodeControl{cfg,
		nil,
		nil,
		nil,
		nil,
		Up,
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
		quorumNode.gethp = NewShellProcess(cfg.GethProcess, cfg.GethRpcUrl, cfg.TesseraUpcheckUrl, true)
	} else if cfg.GethProcess.IsDocker() {
		quorumNode.gethp = NewDockerProcess(cfg.GethProcess, cfg.GethRpcUrl, cfg.TesseraUpcheckUrl, true)
	}

	if cfg.TesseraProcess.IsShell() {
		quorumNode.tesserap = NewShellProcess(cfg.TesseraProcess, cfg.GethRpcUrl, cfg.TesseraUpcheckUrl, true)
	} else if cfg.TesseraProcess.IsDocker() {
		quorumNode.tesserap = NewDockerProcess(cfg.TesseraProcess, cfg.GethRpcUrl, cfg.TesseraUpcheckUrl, true)
	}

	if quorumNode.gethp.Status() && quorumNode.tesserap.Status() {
		quorumNode.SetNodeStatus(Up)
	} else {
		quorumNode.SetNodeStatus(Down)
	}

	if quorumNode.IsRaft() {
		quorumNode.consensus = NewRaftConsensus(quorumNode)
	} else if quorumNode.IsIstanbul() {
		quorumNode.consensus = NewIstanbulConsensus(quorumNode)
	}
	return quorumNode
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

func (qn *QuorumNodeControl) GetProxyConfig() []*types.ProxyConfig {
	return qn.config.Proxies
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

func (qn *QuorumNodeControl) SetNodeStatus(ns NodeStatus) {
	defer qn.statusMux.Unlock()
	qn.statusMux.Lock()
	qn.nodeStatus = ns
}

func (qn *QuorumNodeControl) IsNodeBusy() error {
	switch qn.nodeStatus {
	case ShutdownInprogress, ShutdownInitiated:
		return errors.New("node is being shutdown, try after sometime")
	case StartupInprogress, StartupInitiated:
		return errors.New("node is being started, try after sometime")
	case Up, Down:
		return nil
	}
	return nil
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

// TODO request node managers in parallel
func (qn *QuorumNodeControl) RequestNodeManagerForPrivateTxPrep(tesseraKeys []string) (bool, error) {
	var blockNumberJsonStr = []byte(fmt.Sprintf(`{"jsonrpc":"2.0", "method":"node.PrepareForPrivateTx", "params":["%s"], "id":77}`, qn.config.Name))
	var statusArr []bool
	for _, tessKey := range tesseraKeys {
		nmCfg := qn.GetNodeManagerConfig(tessKey)
		if nmCfg != nil {
			req, err := http.NewRequest("POST", nmCfg.RpcUrl, bytes.NewBuffer(blockNumberJsonStr))
			if err != nil {
				return false, fmt.Errorf("node manager private tx prep reply - creating request failed err=%v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			log.Info("node manager private tx prep sending req", "to", nmCfg.RpcUrl)
			resp, err := qn.client.Do(req)
			if err != nil {
				return false, fmt.Errorf("node manager private tx prep do req failed err=%v", err)
			}

			log.Debug("node manager private tx prep response Status", "status", resp.Status)
			if resp.StatusCode == http.StatusOK {
				body, _ := ioutil.ReadAll(resp.Body)
				log.Debug("node manager private tx prep response Body:", string(body))
				respResult := NodeManagerPrivateTxPrepResult{}
				jerr := json.Unmarshal(body, &respResult)
				if jerr == nil {
					log.Info("node manager private tx prep - response OK", "from", nmCfg.RpcUrl, "result", respResult)
					statusArr = append(statusArr, respResult.Result.Status)
				} else {
					log.Info("response result json decode failed", "err", jerr)
					statusArr = append(statusArr, false)
				}
			} else {
				statusArr = append(statusArr, false)
			}
			resp.Body.Close()
		} else {
			return false, fmt.Errorf("tesseraKey's node manager config missing, key=%s", tessKey)
		}

	}

	finalStatus := true
	for _, s := range statusArr {
		if !s {
			finalStatus = false
			break
		}
	}
	log.Info("node manager private tx prep completed", "final status", finalStatus, "statusArr", statusArr)
	return finalStatus, nil
}

func (nm *QuorumNodeControl) ResetInactiveTime() {
	nm.inactivityResetCh <- true
}

func (qn *QuorumNodeControl) GetRPCConfig() *types.RPCServerConfig {
	return qn.config.Server
}

func (qn *QuorumNodeControl) GetNodeManagerConfig(key string) *types.NodeManagerConfig {
	for _, n := range qn.config.NodeManagers {
		if n.TesseraKey == key {
			log.Info("tesseraKey matched", "node", n)
			return n
		}
	}
	return nil
}

func (qn *QuorumNodeControl) IsNodeUp() bool {
	gs := qn.gethp.IsUp()
	ts := qn.tesserap.IsUp()
	log.Info("IsNodeUp", "geth", gs, "tessera", ts)
	if gs && ts {
		qn.SetNodeStatus(Up)
	} else {
		qn.SetNodeStatus(Down)
	}
	return gs && ts
}

func (qn *QuorumNodeControl) IsRaft() bool {
	return strings.ToLower(qn.config.Consensus) == "raft"
}
func (qn *QuorumNodeControl) StopNode() bool {
	defer qn.startStopMux.Unlock()
	qn.startStopMux.Lock()

	if qn.nodeStatus == Down {
		log.Info("node is already down")
		return true
	}
	var qnms []NodeStatusInfo
	var err error

	retryCount := 1

	for retryCount <= core.Qnm2QnmValidationRetryLimit {
		qnms, err = qn.validateOtherQnms()
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

	qn.SetNodeStatus(ShutdownInitiated)

	if err := qn.consensus.ValidateShutdown(); err == nil {
		log.Info("raft consensus check passed, node can be shutdown")
	} else {
		log.Info("raft consensus check failed, node cannot be shutdown", "err", err)
		return false
	}

	qn.SetNodeStatus(ShutdownInprogress)

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
		qn.SetNodeStatus(Down)
	}
	return gs && ts
}

func (qn *QuorumNodeControl) StartNode() bool {
	defer qn.startStopMux.Unlock()
	qn.startStopMux.Lock()
	qn.SetNodeStatus(StartupInitiated)
	qn.SetNodeStatus(StartupInprogress)
	gs := true
	ts := true
	if qn.tesserap.Start() != nil {
		gs = false
	}
	if qn.gethp.Start() != nil {
		ts = false
	}
	if gs && ts {
		qn.SetNodeStatus(Up)
	}
	return gs && ts
}

func (qn *QuorumNodeControl) GetNodeStatus() NodeStatus {
	return qn.nodeStatus
}

func (qn *QuorumNodeControl) validateOtherQnms() ([]NodeStatusInfo, error) {
	var nodeStatusReq = []byte(fmt.Sprintf(`{"jsonrpc":"2.0", "method":"node.NodeStatus", "params":["%s"], "id":77}`, qn.config.Name))
	var statusArr []NodeStatusInfo
	nodeManagerCount := 0
	for _, nm := range qn.config.NodeManagers {

		//skip self
		if nm.EnodeId == qn.config.EnodeId {
			continue
		}

		nodeManagerCount++

		req, err := http.NewRequest("POST", nm.RpcUrl, bytes.NewBuffer(nodeStatusReq))
		if err != nil {
			return nil, fmt.Errorf("node manager - creating NodeStatus request failed for node manager=%s err=%v", nm.Name, err)
		}
		req.Header.Set("Content-Type", "application/json")
		log.Info("node manager prep sending req", "to", nm.RpcUrl)
		resp, err := qn.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("node manager NodeStatus do req failed err=%v", err)
		}

		log.Info("node manager NodeStatus response Status", "status", resp.Status)
		if resp.StatusCode == http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			log.Debug("node manager NodeStatus response Body:", string(body))
			respResult := NodeManagerNodeStatusResult{}
			jerr := json.Unmarshal(body, &respResult)
			if jerr == nil {
				log.Info("node manager NodeStatus - response OK", "from", nm.RpcUrl, "result", respResult)
				if respResult.Error != nil {
					log.Error("node manager NodeStatus - error in response", "err", respResult.Error)
					return nil, respResult.Error
				}
				statusArr = append(statusArr, respResult.Result)
			} else {
				log.Error("node manager NodeStatus response result json decode failed", "err", jerr)
			}
		} else {
			log.Error("node manager NodeStatus response failed", "status", resp.Status)
		}
		resp.Body.Close()

	}

	if len(statusArr) != nodeManagerCount {
		return statusArr, errors.New("some node managers did not respond")
	}

	shutdownInProgress := false
	for _, n := range statusArr {
		if n.Status == ShutdownInitiated || n.Status == ShutdownInprogress {
			shutdownInProgress = true
			break
		}
	}
	if shutdownInProgress {
		return statusArr, errors.New("some qnm(s) have shutdown initiated/inprogress")
	}

	return statusArr, nil
}

func (qn *QuorumNodeControl) IsIstanbul() bool {
	return strings.ToLower(qn.config.Consensus) == "istanbul"
}
