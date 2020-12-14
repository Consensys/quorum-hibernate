package process

import (
	"bytes"
	"github.com/ConsenSysQuorum/node-manager/config"
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
	// UpdateStatus performs a status check of the process, and caches the result before returning
	UpdateStatus() bool
	// Status returns the cached status
	Status() bool
}

type UpcheckResponse struct {
	Result interface{}    `json:"result"`
	Error  *core.RpcError `json:"error"`
}

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

func IsProcessUp(client *http.Client, cfg config.Upcheck) (bool, error) {
	if cfg.IsRpcResult() {
		var resp UpcheckResponse
		if err := core.CallRPC(client, cfg.UpcheckUrl, []byte(cfg.Body), &resp); err != nil || resp.Error != nil {
			log.Info("IsProcessUp - failed", "err", err, "resp.err", resp.Error, "resp", resp)
			return false, core.ErrNodeDown
		}
	} else if cfg.IsStringResult() {
		if resp, err := core.CallREST(client, cfg.UpcheckUrl, cfg.Method, []byte(cfg.Body)); err != nil {
			log.Info("IsProcessUp - failed", "err", err, "resp", resp)
			return false, core.ErrNodeDown
		} else {
			if resp == cfg.Expected {
				log.Debug("IsProcessUp - privacy manager is up, replied to upcheck call", "reply", resp)
				return true, nil
			}
		}
		return false, core.ErrNodeDown
	} else {
		log.Error("IsProcessUp - unsupported return type")
	}
	return true, nil
}
