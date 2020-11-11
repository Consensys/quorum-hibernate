package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

type NodeManagerPrivateTxPrepResult struct {
	Result PrivateTxPrepStatus `json:"result"`
	Error  error               `json:"error"`
}

type PrivateTxPrepStatus struct {
	Status bool `json:"status"`
}

type QuorumNode struct {
	config             *types.NodeConfig
	im                 *InactivityMonitor
	gethp              Process
	tesserap           Process
	inactivityResetCh  chan bool
	stopNodeCh         chan bool
	shutdownCompleteCh chan bool
	startNodeCh        chan bool
	startCompleteCh    chan bool
	stopCh             chan bool
	startStopMux       sync.Mutex
}

var ErrNodeDown = errors.New("node is not up")

var quorumNode *QuorumNode

func NewQuorumNode(cfg *types.NodeConfig) *QuorumNode {
	quorumNode = &QuorumNode{cfg,
		nil,
		nil,
		nil,
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
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
	return quorumNode
}

func (qn *QuorumNode) Start() {
	qn.StartMonitor()
	qn.im = NewNodeInactivityMonitor(qn)
	qn.im.StartInactivityTimer()
}

func (qn *QuorumNode) Stop() {
	qn.im.Stop()
	qn.stopCh <- true
}

func (qn *QuorumNode) StartMonitor() {
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

func (qn *QuorumNode) GetProxyConfig() []*types.ProxyConfig {
	return qn.config.Proxies
}

func (qn *QuorumNode) RequestStartNode() {
	qn.startNodeCh <- true
}

func (qn *QuorumNode) RequestStopNode() {
	qn.stopNodeCh <- true
}

func (qn *QuorumNode) WaitStartNode() bool {
	status := <-qn.startCompleteCh
	return status
}

func (qn *QuorumNode) WaitStopNode() bool {
	status := <-qn.shutdownCompleteCh
	return status
}

// TODO handle error if node failed to start
func (qn *QuorumNode) PrepareNode() bool {
	if !qn.IsNodeUp() {
		qn.RequestStartNode()
		log.Info("waiting for node start to complete...")
		status := qn.WaitStartNode()
		log.Info("node start completed", "status", status)
		return status
	} else {
		log.Info("node is UP")
		return true
	}
}

// TODO request node managers in parallel
func (qn *QuorumNode) RequestNodeManagerForPrivateTxPrep(tesseraKeys []string) (bool, error) {
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
			client := &http.Client{}
			resp, err := client.Do(req)
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

func (nm *QuorumNode) ResetInactiveTime() {
	nm.inactivityResetCh <- true
}

func (qn *QuorumNode) GetRPCConfig() *types.RPCServerConfig {
	return qn.config.Server
}

func (qn *QuorumNode) GetNodeManagerConfig(key string) *types.NodeManagerConfig {
	for _, n := range qn.config.NodeManagers {
		if n.TesseraKey == key {
			log.Info("tesseraKey matched", "node", n)
			return n
		}
	}
	return nil
}

func (qn *QuorumNode) IsNodeUp() bool {
	gs := qn.gethp.IsUp()
	ts := qn.tesserap.IsUp()
	log.Info("IsNodeUp", "geth", gs, "tessera", ts)
	return gs && ts
}

func (qn *QuorumNode) StopNode() bool {
	defer qn.startStopMux.Unlock()
	qn.startStopMux.Lock()
	gs := true
	ts := true
	if qn.gethp.Stop() != nil {
		gs = false
	}
	if qn.tesserap.Stop() != nil {
		ts = false
	}
	return gs && ts
}

func (qn *QuorumNode) StartNode() bool {
	defer qn.startStopMux.Unlock()
	qn.startStopMux.Lock()
	gs := true
	ts := true
	if qn.tesserap.Start() != nil {
		gs = false
	}
	if qn.gethp.Start() != nil {
		ts = false
	}
	return gs && ts
}
