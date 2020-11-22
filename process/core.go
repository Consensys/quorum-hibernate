package process

import (
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
	Status() bool
}

type BlockNumberResp struct {
	Result string `json:"result"`
	Error  error  `json:"error"`
}

const BlockNumberReq = `{"jsonrpc":"2.0", "method":"eth_blockNumber", "params":[], "id":67}`

var httpClnt = core.NewHttpClient()

// TODO when geth is started by QNM it starts and runs ok. but when QNM is shutdown, geth gets shutdown
func ExecuteShellCommand(desc string, cmdArr []string) error {
	log.Debug("ExecuteShellCommand", "desc", desc, "command", cmdArr)
	var cmd *exec.Cmd
	if len(cmdArr) == 1 {
		cmd = exec.Command(cmdArr[0])
	} else {
		cmd = exec.Command(cmdArr[0], cmdArr[1:]...)

	}
	err := cmd.Run()
	if err != nil {
		log.Error("ExecuteShellCommand - cmd failed", "desc", desc, "err", err)
		return err
	}
	return nil
}

// TODO - what is the right way to check if geth is up?
func IsGethUp(gethRpcUrl string) (bool, error) {
	var resp BlockNumberResp
	if err := core.CallRPC(gethRpcUrl, []byte(BlockNumberReq), &resp); err != nil {
		return false, core.ErrNodeDown
	}
	return true, nil
}

func IsTesseraUp(tesseraUpcheckUrl string) (bool, error) {

	req, err := http.NewRequest("GET", tesseraUpcheckUrl, nil)
	if err != nil {
		log.Error("IsTesseraUp - get req failed", "err", err)
		return false, err
	}

	resp, err := httpClnt.Do(req)
	if err != nil {
		log.Warn("IsTesseraUp - client do req failed", "err", err)
		return false, err
	}
	defer resp.Body.Close()

	log.Debug("IsTesseraUp check response Status", "status", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Debug("IsTesseraUp - up check response Body:", string(body))
	if resp.StatusCode == http.StatusOK && string(body) == "I'm up!" {
		log.Debug("IsTesseraUp - tessera is up, replied to upcheck call", "reply", string(body))
		return true, nil
	}
	return false, core.ErrNodeDown
}
