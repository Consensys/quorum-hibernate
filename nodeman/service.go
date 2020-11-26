package nodeman

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

const (
	NodeStatusMethod   = `{"jsonrpc":"2.0", "method":"node.NodeStatus", "params":["%s"], "id":77}`
	PreparePvtTxMethod = `{"jsonrpc":"2.0", "method":"node.PrepareForPrivateTx", "params":["%s"], "id":77}`
)

func NewNodeManager(cfg *types.NodeConfig) *NodeManager {
	return &NodeManager{cfg: cfg, client: core.NewHttpClient()}
}

func (nm *NodeManager) getConfigByPrivManKey(key string) *types.NodeManagerConfig {
	for _, n := range nm.getLatestConfig() {
		if n.PrivManKey == key {
			log.Debug("getConfigByPrivManKey - privacy manager key matched", "node", n)
			return n
		}
	}
	return nil
}

func (nm *NodeManager) getLatestConfig() []*types.NodeManagerConfig {
	newCfg, err := types.ReadNodeManagerConfig(nm.cfg.BasicConfig.NodeManagerConfigFile)
	if err != nil {
		log.Error("getLatestConfig - error updating node manager config. will use old config", "path", nm.cfg.BasicConfig.NodeManagerConfigFile, "err", err)
		return nm.cfg.NodeManagers
	}
	log.Debug("getLatestConfig - loaded new config", "cfg", newCfg)
	if len(newCfg) == 0 {
		log.Warn("getLatestConfig - node manager list is empty after reload")
	}
	log.Debug("getLatestConfig - node manager config", "new cfg", newCfg)
	nm.cfg.NodeManagers = newCfg
	return nm.cfg.NodeManagers
}

// TODO if a qnm is down/not reachable should I mark it as down and proceed?
// ValidatePeerPrivateTxStatus validates participants readiness status to process private tx
func (nm *NodeManager) ValidatePeerPrivateTxStatus(participantKeys []string) (bool, error) {
	statusArr := nm.peerPrivateTxStatus(participantKeys)
	finalStatus := true
	for _, s := range statusArr {
		if !s {
			finalStatus = false
			break
		}
	}
	log.Info("ValidatePeerPrivateTxStatus completed", "final status", finalStatus, "statusArr", statusArr)
	return finalStatus, nil
}

func (nm *NodeManager) peersByParticipantKeyCount(participantKeys []string) int {
	c := 0
	for _, key := range participantKeys {
		if nm.getConfigByPrivManKey(key) != nil {
			c++
		}
	}
	return c
}

// peerPrivateTxStatus returns readiness status of peers to process private transaction
func (nm *NodeManager) peerPrivateTxStatus(participantKeys []string) []bool {
	var wg = sync.WaitGroup{}
	var resDoneCh = make(chan bool, 1)
	var resCh = make(chan NodeManagerPrivateTxPrepResult, 1)
	var expResCnt = nm.peersByParticipantKeyCount(participantKeys)
	var preparePvtTxReq = []byte(fmt.Sprintf(PreparePvtTxMethod, nm.cfg.BasicConfig.Name))
	var statusArr []bool

	if expResCnt == 0 {
		return nil
	}

	// go routine to receive responses from rpc call to peers for status
	go func() {
		resCnt := 0
		for {
			select {
			case r := <-resCh:
				resCnt++
				if r.Error == nil {
					statusArr = append(statusArr, r.Result.Status)
				}
				if resCnt == expResCnt {
					resDoneCh <- true
					log.Debug("peerPrivateTxStatus - all results received")
					return
				}
			}
		}
	}()

	for _, key := range participantKeys {
		nmCfg := nm.getConfigByPrivManKey(key)

		if nmCfg != nil {
			wg.Add(1)
			go func(nmc *types.NodeManagerConfig) {
				defer wg.Done()
				result := NodeManagerPrivateTxPrepResult{}
				if err := core.CallRPC(nmc.RpcUrl, preparePvtTxReq, &result); err != nil {
					log.Error("peerPrivateTxStatus rpc failed", "err", err)
					result.Error = err
				} else if result.Error != nil {
					log.Error("peerPrivateTxStatus rpc result failed", "err", result.Error)
				}
				resCh <- result
			}(nmCfg)
		} else {
			log.Warn("peerPrivateTxStatus - privacy manager key not found, probably node not managed by qnm", "key", key)
		}

	}
	wg.Wait()
	<-resDoneCh
	log.Info("peerPrivateTxStatus - completed", "status", statusArr)
	return statusArr
}

// ValidatePeers checks the status of peer node managers.
// if one of them returns error during rpc call or not reachable then it returns error.
// if all of them responded and one of them in shutdown initiated or inprogress state it returns error
func (nm *NodeManager) ValidatePeers() ([]NodeStatusInfo, error) {
	nodeManagerCount, statusArr := nm.peerStatus()
	if len(statusArr) != nodeManagerCount {
		return statusArr, errors.New("some node managers did not respond")
	}

	shutdownInProgress := false
	for _, n := range statusArr {
		if n.Status == types.ShutdownInitiated || n.Status == types.ShutdownInprogress {
			shutdownInProgress = true
			break
		}
	}
	if shutdownInProgress {
		return statusArr, errors.New("ValidatePeers - some peer node managers have shutdown initiated/inprogress")
	}

	return statusArr, nil
}

func (nm *NodeManager) getConfigCount(nmCfgs []*types.NodeManagerConfig) int {
	nodeManagerCount := 0
	for _, n := range nmCfgs {
		//skip self
		if n.PrivManKey == nm.cfg.BasicConfig.PrivManKey {
			continue
		}
		nodeManagerCount++
	}
	return nodeManagerCount
}

// peerStatus makes rpc call to peers and gets their status.
// If returns expected result count and an array results(NodeStausInfo) received.
// If all of them responded and one of them in shutdown initiated or inprogress state it returns error.
// It creates as many go routines as the number of peer node managers. It should not be an issue
// as we would not have more than a few thousand peers.
// Golang easily supports creating thousands of goroutines
func (nm *NodeManager) peerStatus() (int, []NodeStatusInfo) {
	var nodeStatusReq = []byte(fmt.Sprintf(NodeStatusMethod, nm.cfg.BasicConfig.Name))
	var statusArr []NodeStatusInfo
	var wg = sync.WaitGroup{}
	var resDoneCh = make(chan bool, 1)
	var resCh = make(chan NodeManagerNodeStatusResult, 1)
	nmCfgs := nm.getLatestConfig()
	expResCnt := nm.getConfigCount(nmCfgs)

	if expResCnt == 0 {
		return 0, nil
	}

	// go routine to receive responses from rpc call to peers for status
	go func() {
		resCnt := 0
		for {
			select {
			case r := <-resCh:
				resCnt++
				if r.Error == nil {
					statusArr = append(statusArr, r.Result)
				}
				if resCnt == expResCnt {
					resDoneCh <- true
					log.Debug("peerStatus - all results received")
					return
				}
			}
		}
	}()

	for _, n := range nmCfgs {
		//skip self
		if n.PrivManKey == nm.cfg.BasicConfig.PrivManKey {
			continue
		}
		wg.Add(1)
		go func(nmc *types.NodeManagerConfig) {
			defer wg.Done()
			var res = NodeManagerNodeStatusResult{}
			if err := core.CallRPC(nmc.RpcUrl, nodeStatusReq, &res); err != nil {
				log.Error("peerStatus - NodeStatus - failed", "err", err)
			}
			if res.Error != nil {
				log.Error("peerStatus - NodeStatus - response failed", "err", res.Error)
			}
			log.Debug("peerStatus", "res", res, "cfg", n)
			resCh <- res
		}(n)
	}
	wg.Wait()
	<-resDoneCh
	log.Debug("peerStatus - completed", "status", statusArr)
	return expResCnt, statusArr
}
