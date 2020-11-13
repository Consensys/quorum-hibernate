package node

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os/exec"

	"github.com/ConsenSysQuorum/node-manager/core"

	"github.com/ConsenSysQuorum/node-manager/log"
)

type Process interface {
	Start() error
	Stop() error
	IsUp() bool
}

var httpClnt = core.NewHttpClient()

// TODO when geth is started by QNM it starts and runs ok. but when QNM is shutdown, geth gets shutdown
func ExecuteShellCommand(desc string, cmdArr []string) error {
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

// TODO - what is the right way to check if geth is up?
func IsGethUp(gethRpcUrl string) (bool, error) {
	var blockNumberJsonStr = []byte(`{"jsonrpc":"2.0", "method":"eth_blockNumber", "params":[], "id":67}`)
	req, err := http.NewRequest("POST", gethRpcUrl, bytes.NewBuffer(blockNumberJsonStr))
	if err != nil {
		log.Error("geth up check new req failed", "err", err)
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClnt.Do(req)
	if err != nil {
		log.Warn("geth up check client do req failed", "err", err)
		return false, err
	}
	defer resp.Body.Close()

	log.Debug("geth up check response Status", "status", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Debug("geth up check response Body:", string(body))
	if resp.StatusCode == http.StatusOK {
		log.Debug("geth is up, replied to eth_blockNumber call", "reply", string(body))
		return true, nil
	}
	return false, ErrNodeDown
}

func IsTesseraUp(tesseraUpcheckUrl string) (bool, error) {

	req, err := http.NewRequest("GET", tesseraUpcheckUrl, nil)
	if err != nil {
		log.Error("tessera up check new get req failed", "err", err)
		return false, err
	}

	resp, err := httpClnt.Do(req)
	if err != nil {
		log.Warn("geth up check client do req failed", "err", err)
		return false, err
	}
	defer resp.Body.Close()

	log.Debug("tessera up check response Status", "status", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Debug("tessera up check response Body:", string(body))
	if resp.StatusCode == http.StatusOK && string(body) == "I'm up!" {
		log.Debug("tessera is up, replied to upcheck call", "reply", string(body))
		return true, nil
	}
	return false, ErrNodeDown
}
