package p2p

import (
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/ConsenSysQuorum/node-manager/config"

	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/log"
)

const (
	NodeStatusMethod   = `{"jsonrpc":"2.0", "method":"node.NodeStatus", "params":["%s"], "id":77}`
	PreparePvtTxMethod = `{"jsonrpc":"2.0", "method":"node.PrepareForPrivateTx", "params":["%s"], "id":77}`
)

func NewPeerManager(cfg *config.Node) *PeerManager {
	configReader, _ := config.NewPeersReader(cfg.BasicConfig.PeersConfigFile)

	return &PeerManager{
		cfg:          cfg,
		configReader: configReader,
	}
}

func (pm *PeerManager) getConfigByPrivManKey(key string) *config.Peer {
	for _, n := range pm.readPeersConfig() {
		if n.PrivManKey == key {
			log.Debug("getConfigByPrivManKey - privacy manager key matched", "node", n)
			return n
		}
	}
	return nil
}

func (pm *PeerManager) readPeersConfig() []*config.Peer {
	newPeers, err := pm.configReader.Read()
	if err != nil {
		log.Error("readPeersConfig - error updating node manager config. will use old config", "path", pm.cfg.BasicConfig.PeersConfigFile, "err", err)
		return pm.cfg.Peers
	}
	if err = newPeers.IsValid(); err != nil {
		log.Error("readPeersConfig - error validation of node manager config failed.", "err", err)
		return pm.cfg.Peers
	}

	log.Debug("readPeersConfig - loaded new config", "cfg", newPeers)
	if len(newPeers) == 0 {
		log.Warn("readPeersConfig - node manager list is empty after reload")
	}
	log.Debug("readPeersConfig - node manager config", "new cfg", newPeers)
	pm.cfg.Peers = newPeers
	return pm.cfg.Peers
}

// TODO if a node manager is down/not reachable should we mark it as down and proceed?
// ValidatePeerPrivateTxStatus validates participants readiness status to process private tx
func (pm *PeerManager) ValidatePeerPrivateTxStatus(participantKeys []string) (bool, error) {
	statusArr := pm.peerPrivateTxStatus(participantKeys)
	finalStatus := true
	if len(statusArr) == 0 {
		finalStatus = false
	}
	for _, s := range statusArr {
		if !s {
			finalStatus = false
			break
		}
	}
	log.Debug("ValidatePeerPrivateTxStatus completed", "final status", finalStatus, "statusArr", statusArr)
	return finalStatus, nil
}

func (pm *PeerManager) peersByParticipantKeyCount(participantKeys []string) int {
	c := 0
	for _, key := range participantKeys {
		if pm.getConfigByPrivManKey(key) != nil {
			c++
		}
	}
	return c
}

// peerPrivateTxStatus returns readiness status of peers to process private transaction
func (pm *PeerManager) peerPrivateTxStatus(participantKeys []string) []bool {
	var wg = sync.WaitGroup{}
	var resDoneCh = make(chan bool, 1)
	var resCh = make(chan PeerPrivateTxPrepResult, 1)
	var expResCnt = pm.peersByParticipantKeyCount(participantKeys)
	var preparePvtTxReq = []byte(fmt.Sprintf(PreparePvtTxMethod, pm.cfg.BasicConfig.Name))
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
		nmCfg := pm.getConfigByPrivManKey(key)

		if nmCfg != nil {
			wg.Add(1)
			go func(nmc *config.Peer) {
				defer wg.Done()
				result := PeerPrivateTxPrepResult{}
				var client *http.Client
				if nmc.TLSConfig != nil {
					client = core.NewHttpClient(nmc.TLSConfig.TlsCfg)
				}
				if err := core.CallRPC(client, nmc.RpcUrl, preparePvtTxReq, &result); err != nil {
					log.Error("peerPrivateTxStatus rpc failed", "err", err)
					result.Error = err
				} else if result.Error != nil {
					log.Error("peerPrivateTxStatus rpc result failed", "err", result.Error)
				}
				resCh <- result
			}(nmCfg)
		} else {
			log.Warn("peerPrivateTxStatus - privacy manager key not found, probably node not managed by node manager", "key", key)
		}

	}
	wg.Wait()
	<-resDoneCh
	log.Debug("peerPrivateTxStatus - completed", "status", statusArr)
	return statusArr
}

// ValidatePeers checks the status of peer node managers.
// if one of them returns error during rpc call or not reachable then it returns error.
// if all of them responded and one of them in shutdown initiated or inprogress state it returns error
func (pm *PeerManager) ValidatePeers() ([]NodeStatusInfo, error) {
	nodeManagerCount, statusArr := pm.peerStatus()
	if len(statusArr) != nodeManagerCount {
		return statusArr, errors.New("some node managers did not respond")
	}
	shutdownInProgress := false
	for _, n := range statusArr {
		if n.Status == core.ShutdownInprogress || n.Status == core.ConsensusWait {
			shutdownInProgress = true
			break
		}
	}
	if shutdownInProgress {
		return statusArr, errors.New("ValidatePeers - some peer node managers have shutdown initiated/inprogress")
	}

	return statusArr, nil
}

func (pm *PeerManager) getPeersCount(nmCfgs []*config.Peer) int {
	nodeManagerCount := 0
	for _, n := range nmCfgs {
		//skip self
		if pm.isPeerSelf(n.Name) {
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
func (pm *PeerManager) peerStatus() (int, []NodeStatusInfo) {
	var nodeStatusReq = []byte(fmt.Sprintf(NodeStatusMethod, pm.cfg.BasicConfig.Name))
	var statusArr []NodeStatusInfo
	var wg = sync.WaitGroup{}
	var resDoneCh = make(chan bool, 1)
	var resCh = make(chan PeerNodeStatusResult, 1)
	peersConfig := pm.readPeersConfig()
	expResCnt := pm.getPeersCount(peersConfig)

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

	for _, n := range peersConfig {
		// skip self
		if pm.isPeerSelf(n.Name) {
			continue
		}
		wg.Add(1)
		go func(nmc *config.Peer) {
			defer wg.Done()
			var res = PeerNodeStatusResult{}
			var client *http.Client
			if nmc.TLSConfig != nil {
				client = core.NewHttpClient(nmc.TLSConfig.TlsCfg)
			}
			if err := core.CallRPC(client, nmc.RpcUrl, nodeStatusReq, &res); err != nil {
				log.Error("peerStatus - ClientStatus - failed", "err", err)
			}
			if res.Error != nil {
				log.Error("peerStatus - ClientStatus - response failed", "err", res.Error)
			}
			log.Debug("peerStatus", "res", res, "cfg", n)
			resCh <- res
		}(n)
	}
	wg.Wait()
	<-resDoneCh
	log.Info("peerStatus - completed", "status", fmt.Sprintf("%+v", statusArr))
	return expResCnt, statusArr
}

func (pm *PeerManager) isPeerSelf(peerName string) bool {
	return peerName != "" && peerName == pm.cfg.BasicConfig.Name
}
