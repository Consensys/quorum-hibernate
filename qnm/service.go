package qnm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

func NewNodeManager(cfg *types.NodeConfig) *NodeManager {
	return &NodeManager{cfg: cfg, client: core.NewHttpClient()}
}

func (nm *NodeManager) getNodeManagerConfigByTesseraKey(key string) *types.NodeManagerConfig {
	for _, n := range nm.getLatestNodeManagerConfig() {
		if n.TesseraKey == key {
			log.Info("tesseraKey matched", "node", n)
			return n
		}
	}
	return nil
}

func (nm *NodeManager) getLatestNodeManagerConfig() []*types.NodeManagerConfig {
	newCfg, err := types.ReadNodeManagerConfig(nm.cfg.NodeManagerConfigFile)
	if err != nil {
		log.Error("error updating node manager config. will use old config", "path", nm.cfg.NodeManagerConfigFile, "err", err)
		return nm.cfg.NodeManagers
	}
	log.Info("loaded new config", "cfg", newCfg)
	if len(newCfg) == 0 {
		log.Warn("node manager list is empty after reload")
	} else {
		log.Info("updated node manager config", "new cfg", newCfg)
	}
	nm.cfg.NodeManagers = newCfg
	return nm.cfg.NodeManagers
}

// TODO parallelize request
func (nm *NodeManager) ValidateForPrivateTx(tesseraKeys []string) (bool, error) {
	var blockNumberJsonStr = []byte(fmt.Sprintf(`{"jsonrpc":"2.0", "method":"node.PrepareForPrivateTx", "params":["%s"], "id":77}`, nm.cfg.Name))
	var statusArr []bool
	for _, tessKey := range tesseraKeys {
		nmCfg := nm.getNodeManagerConfigByTesseraKey(tessKey)
		if nmCfg != nil {
			req, err := http.NewRequest("POST", nmCfg.RpcUrl, bytes.NewBuffer(blockNumberJsonStr))
			if err != nil {
				return false, fmt.Errorf("node manager private tx prep reply - creating request failed err=%v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			log.Info("node manager private tx prep sending req", "to", nmCfg.RpcUrl)
			resp, err := nm.client.Do(req)
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
			log.Warn("tessera key not found, probably node not using qnm", "key", tessKey)
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

func (nm *NodeManager) ValidateOtherQnms() ([]NodeStatusInfo, error) {
	var nodeStatusReq = []byte(fmt.Sprintf(`{"jsonrpc":"2.0", "method":"node.NodeStatus", "params":["%s"], "id":77}`, nm.cfg.Name))
	var statusArr []NodeStatusInfo
	nodeManagerCount := 0
	for _, n := range nm.getLatestNodeManagerConfig() {

		//skip self
		if n.EnodeId == nm.cfg.EnodeId {
			continue
		}

		nodeManagerCount++

		req, err := http.NewRequest("POST", n.RpcUrl, bytes.NewBuffer(nodeStatusReq))
		if err != nil {
			return nil, fmt.Errorf("node manager - creating NodeStatus request failed for node manager=%s err=%v", n.Name, err)
		}
		req.Header.Set("Content-Type", "application/json")
		log.Info("node manager prep sending req", "to", n.RpcUrl)
		resp, err := nm.client.Do(req)
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
				log.Info("node manager NodeStatus - response OK", "from", n.RpcUrl, "result", respResult)
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
		if n.Status == types.ShutdownInitiated || n.Status == types.ShutdownInprogress {
			shutdownInProgress = true
			break
		}
	}
	if shutdownInProgress {
		return statusArr, errors.New("some qnm(s) have shutdown initiated/inprogress")
	}

	return statusArr, nil
}
