package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"sync"
	"time"

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
	nodeUp             bool
	inactivityResetCh  chan bool
	stopNodeCh         chan bool
	shutdownCompleteCh chan bool
	startNodeCh        chan bool
	startCompleteCh    chan bool
	startStopMux       sync.Mutex
}

var ErrNodeDown = errors.New("node is not up")

var quorumNode *QuorumNode

func NewQuorumNode(cfg *types.NodeConfig) *QuorumNode {
	quorumNode = &QuorumNode{cfg,
		true,
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		make(chan bool, 1),
		sync.Mutex{},
	}
	return quorumNode
}

func (qn *QuorumNode) Start() {
	qn.StartMonitor()
	ac := NewNodeInactivityMonitor(qn)
	ac.StartInactivityTimer()
}

func (qn *QuorumNode) StartMonitor() {
	qn.SetNodeInitialStatus()
	go func() {
		log.Info("node start/stop monitor started")
		for {
			select {
			case <-qn.stopNodeCh:
				log.Info("request received to stop node")
				qn.StopNode(false, true)
			case <-qn.startNodeCh:
				log.Info("request received to start node")
				qn.StartNode(false, true)
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

func (qn *QuorumNode) SetNodeInitialStatus() {
	log.Info("set node's initial status")
	if up, err := qn.PingNodeToCheckIfItIsUp(); err != nil || !up {
		qn.SetNodeDown()
		log.Info("node is down")
	} else {
		qn.SetNodeUp()
		log.Info("node is up")
	}
}

// TODO handle error if node failed to start
func (qn *QuorumNode) PrepareNode() bool {
	if !qn.IsNodeUp() {
		if up, err := qn.PingNodeToCheckIfItIsUp(); err != nil || !up {
			qn.RequestStartNode()
			log.Info("waiting for node start to complete...")
			status := qn.WaitStartNode()
			log.Info("node start completed", "status", status)
			return status
		}
		return true
	} else {
		log.Info("node is UP")
		return true
	}
}

func (qn *QuorumNode) StopNode(fake bool, completeCheck bool) error {
	defer qn.startStopMux.Unlock()
	defer qn.startStopMux.Lock()
	if fake {
		log.Info("node shutdown")
		if completeCheck {
			qn.shutdownCompleteCh <- true
		}
		qn.SetNodeDown()
		return nil
	}
	if err := qn.ExecuteShellCommand("stop node", qn.config.GethProcess.StopCommand); err == nil {
		qn.SetNodeDown()
		time.Sleep(time.Second)
		if completeCheck {
			qn.shutdownCompleteCh <- true
		}
		return nil
	} else {
		log.Error("stop node failed", "err", err)
		if completeCheck {
			qn.shutdownCompleteCh <- false
		}
		return err
	}
}

func (qn *QuorumNode) StartNode(fake bool, completeCheck bool) error {
	defer qn.startStopMux.Unlock()
	defer qn.startStopMux.Lock()
	if fake {
		log.Info("fake start node done")
		if completeCheck {
			qn.startCompleteCh <- true
		}
		qn.SetNodeUp()
		return nil
	}
	if err := qn.ExecuteShellCommand("start node", qn.config.GethProcess.StartCommand); err == nil {
		//wait for node to come up
		time.Sleep(2 * time.Second)
		log.Info("node started.")
		// TODO ping the node to confirm if its up
		qn.SetNodeUp()
		if completeCheck {
			qn.startCompleteCh <- true
		}
		return nil
	} else {
		log.Error("failed to start node")
		if completeCheck {
			qn.startCompleteCh <- false
		}
		return err
	}
}

func (qn *QuorumNode) ExecuteShellCommand(desc string, cmdArr []string) error {
	log.Info("executing command", "desc", desc, "command", cmdArr)
	var cmd *exec.Cmd
	if len(cmdArr) == 1 {
		cmd = exec.Command(cmdArr[0])
	} else {
		cmd = exec.Command(cmdArr[0], cmdArr[1:]...)

	}
	err := cmd.Run()
	if err != nil {
		log.Error("cmd failed", "desc", desc, "err", err)
		return err
	}
	return nil
}

func (qn *QuorumNode) SetNodeUp() {
	qn.nodeUp = true
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
					log.Info("node manager private tx prep - response OK", "result", respResult)
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

func (qn *QuorumNode) PingNodeToCheckIfItIsUp() (bool, error) {
	defer qn.startStopMux.Unlock()
	defer qn.startStopMux.Lock()
	var blockNumberJsonStr = []byte(`{"jsonrpc":"2.0", "method":"eth_blockNumber", "params":[], "id":67}`)
	req, err := http.NewRequest("POST", qn.config.GethRpcUrl, bytes.NewBuffer(blockNumberJsonStr))
	if err != nil {
		log.Error("node up check reading body failed", "err", err)
		qn.SetNodeDown()
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("node up check client do req failed", "err", err)
		qn.SetNodeDown()
		return false, err
	}
	defer resp.Body.Close()

	log.Debug("nodeUp check response Status", "status", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Debug("nodeUp check response Body:", string(body))
	if resp.StatusCode == http.StatusOK {
		log.Info("node is up, replied to eth_blockNumber call", "reply", string(body))
		qn.SetNodeUp()
		return true, nil
	}
	qn.SetNodeDown()
	return false, ErrNodeDown
}

func (qn *QuorumNode) SetNodeDown() {
	qn.nodeUp = false
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
	return qn.nodeUp
}
