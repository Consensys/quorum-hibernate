package process

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os/exec"
	"syscall"

	"github.com/ConsenSysQuorum/node-manager/core"
	"github.com/ConsenSysQuorum/node-manager/log"
)

// Process is an interface that represents blockchain client or privacy manager process
// It allows  a process to be stopped & started.
// It allows the process's status to be checked.
// This should be used by node controller to control blockchain client, privacyManager.
type Process interface {
	// Start starts the process. it returns error if it fails.
	Start() error
	// Stop stops the process. it returns error if it fails.
	Stop() error
	// IsUp performs Up check for the process by trying to execute the http get / rpc call
	// trying to connect to the process
	IsUp() bool
	// Status returns the cached status of last IsUp check
	Status() bool
}

type BlockNumberResp struct {
	Result string `json:"result"`
	Error  error  `json:"error"`
}

const BlockNumberReq = `{"jsonrpc":"2.0", "method":"eth_blockNumber", "params":[], "id":67}`

var httpClnt = core.NewHttpClient()

func ExecuteShellCommand(cmdArr []string) error {
	log.Debug("ExecuteShellCommand", "cmd", cmdArr)
	var cmd *exec.Cmd
	if len(cmdArr) == 1 {
		cmd = exec.Command(cmdArr[0])
	} else {
		cmd = exec.Command(cmdArr[0], cmdArr[1:]...)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // SIGINT interrupts the entire process group - to prevent SIGINT of node-manager killing this child process, give the child its own process group
	}

	errOut := &bytes.Buffer{}
	cmd.Stderr = errOut

	err := cmd.Run()
	if err != nil {
		log.Error("ExecuteShellCommand failed", "cmd", cmdArr, "out", string(errOut.Bytes()), "err", err)
		return err
	}
	log.Debug("ExecuteShellCommand success", "cmd", cmdArr)
	return nil
}

// TODO - what is the right way to check if blockchain client is up?
func IsBlockchainClientUp(rpcUrl string) (bool, error) {
	var resp BlockNumberResp
	if err := core.CallRPC(rpcUrl, []byte(BlockNumberReq), &resp); err != nil {
		log.Info("IsBlockchainClientUp - failed", "err", err)
		return false, core.ErrNodeDown
	}
	return true, nil
}

func IsPrivacyManagerUp(upcheckUrl string) (bool, error) {

	req, err := http.NewRequest("GET", upcheckUrl, nil)
	if err != nil {
		log.Error("IsPrivacyManagerUp - get req failed", "err", err)
		return false, err
	}

	resp, err := httpClnt.Do(req)
	if err != nil {
		log.Info("IsPrivacyManagerUp - client do req failed", "err", err)
		return false, err
	}
	defer resp.Body.Close()

	log.Debug("IsPrivacyManagerUp check response Status", "status", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Debug("IsPrivacyManagerUp - up check response Body:", string(body))
	if resp.StatusCode == http.StatusOK && string(body) == "I'm up!" {
		log.Debug("IsPrivacyManagerUp - privacy manager is up, replied to upcheck call", "reply", string(body))
		return true, nil
	}
	return false, core.ErrNodeDown
}
