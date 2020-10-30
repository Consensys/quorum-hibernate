package node

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type QuorumNode struct {
	nodeName           string
	rpcUrl             string
	WsUrl              string
	GraphqlUrl         string
	rpcProxyPort       int
	wsProxyPort        int
	graphqlProxyPort   int
	processId          string
	port               string
	inactiveTime       int // in seconds
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

func NewQuorumNode() *QuorumNode {
	quorumNode = &QuorumNode{"node1",
		"http://localhost:22000",
		":23000",
		"http://localhost:8547/graphql",
		9090,
		9091,
		9092,
		"10002",
		"22000",
		1000000,
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
		log.Printf("node start/stop monitor started")
		for {
			select {
			case <-qn.stopNodeCh:
				log.Printf("request recieved to stop node as it was inactive")
				qn.StopNode(true)
			case <-qn.startNodeCh:
				log.Printf("request recieved to start node as it was inactive")
				qn.StartNode(true)
			}
		}
	}()
}

func (qn *QuorumNode) GetProxyInfo(name string) (string, int) {
	switch name {
	case "RPC":
		return qn.rpcUrl, qn.rpcProxyPort
	case "GRAPHQL":
		return qn.GraphqlUrl, qn.graphqlProxyPort
	case "WS":
		return qn.WsUrl, qn.wsProxyPort
	}
	return "", 0
}

func (qn *QuorumNode) StopNode(fake bool) error {
	defer qn.startStopMux.Unlock()
	defer qn.startStopMux.Lock()
	if fake {
		log.Printf("node shutdown")
		qn.shutdownCompleteCh <- true
		return nil
	}
	nodeStopCmd := "kill"
	arg0 := quorumNode.processId
	log.Printf("stopping node %s procid:%s\n", quorumNode.rpcUrl, arg0)
	cmd := exec.Command(nodeStopCmd, arg0)
	stdout, err := cmd.Output()
	if err != nil {
		log.Printf("node failed to stop err=%v\n", err.Error())
		qn.shutdownCompleteCh <- false
		return err
	}
	log.Printf(string(stdout))
	qn.SetNodeDown()
	time.Sleep(time.Second)
	qn.shutdownCompleteCh <- true
	return nil
}

func (qn *QuorumNode) GetProcessIdOfNode() error {
	defer qn.startStopMux.Unlock()
	defer qn.startStopMux.Lock()
	nodeStopCmd := "ps"
	arg0 := "-eaf"
	log.Printf("grepping for node %s procid:%s %s\n", qn.rpcUrl, nodeStopCmd, arg0)
	cmd := exec.Command(nodeStopCmd, arg0)
	stdout, err := cmd.Output()
	if err != nil {
		log.Printf("node failed to run ps -eaf err=%v\n", err.Error())
		return err
	}
	processIdIndexPos := 4
	for _, l := range strings.Split(string(stdout), "\n") {
		if strings.Contains(l, "geth") && strings.Contains(l, qn.port) {
			prArr := strings.Split(l, " ")
			log.Printf("prArr = %v", prArr)
			qn.processId = prArr[processIdIndexPos]
			log.Printf("node process id found. processId=%s", qn.processId)
			return nil
		}
	}
	return errors.New("process id not found")
}

func (qn *QuorumNode) RequestStartNode() {
	qn.startNodeCh <- true
}

func (qn *QuorumNode) WaitStartNode() {
	<-qn.startCompleteCh
}

func (qn *QuorumNode) RequestStopNode() {
	qn.stopNodeCh <- true
}

func (qn *QuorumNode) StartNode(fake bool) error {
	defer qn.startStopMux.Unlock()
	defer qn.startStopMux.Lock()
	if fake {
		log.Printf("start node done")
		qn.startCompleteCh <- true
	}
	nodeStartCmd := "/Users/maniam/consensys/quorum-examples/examples/7nodes/raft-s-n.sh"
	arg0 := "5"
	arg1 := "5"
	defer log.Printf("finished starting node")
	log.Printf("starting node %s %s %s %s\n", qn.rpcUrl, nodeStartCmd, arg0, arg1)
	cmd := exec.Command(nodeStartCmd, arg0, arg1)
	err := cmd.Run()
	if err != nil {
		log.Printf("node failed to start err=%v\n", err)
		qn.startCompleteCh <- false
		return err
	} else {
		log.Printf("node started successfully")
	}
	time.Sleep(time.Second)
	qn.SetNodeUp()
	qn.startCompleteCh <- true
	return nil
}

func (qn *QuorumNode) SetNodeUp() {
	qn.nodeUp = true
}

func (qn *QuorumNode) IsNodeUp() (bool, error) {
	defer qn.startStopMux.Unlock()
	defer qn.startStopMux.Lock()
	var blockNumberJsonStr = []byte(`{"jsonrpc":"2.0", "method":"eth_blockNumber", "params":[], "id":67}`)
	req, err := http.NewRequest("POST", qn.rpcUrl, bytes.NewBuffer(blockNumberJsonStr))
	if err != nil {
		log.Printf("ERROR: reading body failed err:%v", err)

		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: client do req err:%v", err)
		return false, err
	}
	defer resp.Body.Close()

	log.Println("nodeUp check response Status:", resp.Status)
	log.Println("nodeUp check response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Println("nodeUp check response Body:", string(body))
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
