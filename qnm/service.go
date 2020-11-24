package qnm

import (
	"errors"
	"fmt"
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

func (nm *NodeManager) getNodeManagerConfigByPrivManKey(key string) *types.NodeManagerConfig {
	for _, n := range nm.getLatestNodeManagerConfig() {
		if n.PrivManKey == key {
			log.Debug("getNodeManagerConfigByPrivManKey - privacy manager key matched", "node", n)
			return n
		}
	}
	return nil
}

func (nm *NodeManager) getLatestNodeManagerConfig() []*types.NodeManagerConfig {
	newCfg, err := types.ReadNodeManagerConfig(nm.cfg.BasicConfig.NodeManagerConfigFile)
	if err != nil {
		log.Error("getLatestNodeManagerConfig - error updating node manager config. will use old config", "path", nm.cfg.BasicConfig.NodeManagerConfigFile, "err", err)
		return nm.cfg.NodeManagers
	}
	log.Debug("getLatestNodeManagerConfig - loaded new config", "cfg", newCfg)
	if len(newCfg) == 0 {
		log.Warn("getLatestNodeManagerConfig - node manager list is empty after reload")
	}
	log.Debug("getLatestNodeManagerConfig - node manager config", "new cfg", newCfg)
	nm.cfg.NodeManagers = newCfg
	return nm.cfg.NodeManagers
}

// TODO if a qnm is down/not reachable should I mark it as down and proceed?
// TODO parallelize request
func (nm *NodeManager) ValidateForPrivateTx(prvManKeys []string) (bool, error) {
	var preparePvtTxReq = []byte(fmt.Sprintf(PreparePvtTxMethod, nm.cfg.BasicConfig.Name))
	var statusArr []bool
	for _, key := range prvManKeys {
		nmCfg := nm.getNodeManagerConfigByPrivManKey(key)
		if nmCfg != nil {
			respResult := NodeManagerPrivateTxPrepResult{}
			if err := core.CallRPC(nmCfg.RpcUrl, preparePvtTxReq, &respResult); err != nil {
				log.Error("ValidateForPrivateTx failed", "err", err)
				statusArr = append(statusArr, false)
			} else if respResult.Error != nil {
				log.Error("ValidateForPrivateTx result failed", "err", respResult.Error)
				statusArr = append(statusArr, false)
			} else {
				statusArr = append(statusArr, respResult.Result.Status)
			}
		} else {
			log.Warn("ValidateForPrivateTx - privacy manager key not found, probably node not using qnm", "key", key)
		}
	}

	finalStatus := true
	for _, s := range statusArr {
		if !s {
			finalStatus = false
			break
		}
	}
	log.Info("ValidateForPrivateTx completed", "final status", finalStatus, "statusArr", statusArr)
	return finalStatus, nil
}

// TODO if a qnm is down/not reachable should I mark it as down and proceed?
// TODO parallelize req
func (nm *NodeManager) ValidateOtherQnms() ([]NodeStatusInfo, error) {
	var nodeStatusReq = []byte(fmt.Sprintf(NodeStatusMethod, nm.cfg.BasicConfig.Name))
	var statusArr []NodeStatusInfo
	nodeManagerCount := 0
	for _, n := range nm.getLatestNodeManagerConfig() {
		//skip self
		if n.PrivManKey == nm.cfg.BasicConfig.PrivManKey {
			continue
		}
		nodeManagerCount++
		var respResult = NodeManagerNodeStatusResult{}
		if err := core.CallRPC(n.RpcUrl, nodeStatusReq, &respResult); err != nil {
			log.Error("ValidateOtherQnms NodeStatus - failed", "err", err)
			return nil, err
		}
		if respResult.Error != nil {
			log.Error("ValidateOtherQnms NodeStatus - error in response", "err", respResult.Error)
			return nil, respResult.Error
		}
		statusArr = append(statusArr, respResult.Result)
	}

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
		return statusArr, errors.New("ValidateOtherQnms - some qnm(s) have shutdown initiated/inprogress")
	}

	return statusArr, nil
}
