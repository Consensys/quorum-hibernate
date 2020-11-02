package node

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

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
	go func() {
		log.Info("node start/stop monitor started")
		for {
			select {
			case <-qn.stopNodeCh:
				log.Info("request recieved to stop node as it was inactive")
				qn.StopNode(false)
			case <-qn.startNodeCh:
				log.Info("request recieved to start node as it was down")
				qn.StartNode(false)
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

func (qn *QuorumNode) WaitStartNode() {
	<-qn.startCompleteCh
}

func (qn *QuorumNode) WaitStopNode() {
	<-qn.shutdownCompleteCh
}

func (qn *QuorumNode) StopNode(fake bool) error {
	defer qn.startStopMux.Unlock()
	defer qn.startStopMux.Lock()
	if fake {
		log.Info("node shutdown")
		qn.shutdownCompleteCh <- true
		return nil
	}
	if err := qn.ExecuteShellCommand("stop node", qn.config.GethProcess.StopCommand); err == nil {
		qn.SetNodeDown()
		time.Sleep(time.Second)
		qn.shutdownCompleteCh <- true
	} else {
		log.Error("stop node failed", "err", err)
	}
	return nil
}

func (qn *QuorumNode) StartNode(fake bool) error {
	defer qn.startStopMux.Unlock()
	defer qn.startStopMux.Lock()
	if fake {
		log.Info("start node done")
		qn.startCompleteCh <- true
	}
	if err := qn.ExecuteShellCommand("start node", qn.config.GethProcess.StartCommand); err == nil {
		time.Sleep(time.Second)
		qn.SetNodeUp()
		qn.startCompleteCh <- true
	} else {
		log.Error("failed to start node")
		return err
	}
	return nil
}

func (qn *QuorumNode) ExecuteShellCommand(desc string, cmdArr []string) error {
	defer log.Info("finished executing command", "desc", desc, "cmd", cmdArr)
	log.Info("executing command", "desc", desc, "command", cmdArr)
	var cmd *exec.Cmd
	if len(cmdArr) == 1 {
		cmd = exec.Command(cmdArr[0])
	} else {
		cmd = exec.Command(cmdArr[0], cmdArr[1:]...)

	}
	err := cmd.Run()
	if err != nil {
		log.Error("cmd failed", "err", err)
		return err
	} else {
		log.Info("cmd executed successfully")
	}
	return nil
}

func (qn *QuorumNode) SetNodeUp() {
	qn.nodeUp = true
}

func (qn *QuorumNode) IsNodeUp() (bool, error) {
	defer qn.startStopMux.Unlock()
	defer qn.startStopMux.Lock()
	var blockNumberJsonStr = []byte(`{"jsonrpc":"2.0", "method":"eth_blockNumber", "params":[], "id":67}`)
	req, err := http.NewRequest("POST", qn.config.GethRpcUrl, bytes.NewBuffer(blockNumberJsonStr))
	if err != nil {
		log.Error("ERROR: reading body failed", "err", err)
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("client do req", "err", err)
		return false, err
	}
	defer resp.Body.Close()

	log.Info("nodeUp check response Status", "status", resp.Status)
	log.Info("nodeUp check response Headers", "header", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Info("nodeUp check response Body:", string(body))
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, ErrNodeDown
}

func (qn *QuorumNode) SetNodeDown() {
	qn.nodeUp = false
}

func (nm *QuorumNode) ResetInactiveTime() {
	nm.inactivityResetCh <- true
}
