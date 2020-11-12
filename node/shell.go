package node

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
)

type ShellProcessControl struct {
	cfg               *types.ProcessConfig
	gethRpcUrl        string
	tesseraUpcheckUrl string
	status            bool
	muxLock           sync.Mutex
}

func NewShellProcess(p *types.ProcessConfig, grpc string, turl string, s bool) Process {
	sp := &ShellProcessControl{p, grpc, turl, s, sync.Mutex{}}
	sp.IsUp()
	log.Debug("shell process created", "name", sp.cfg.Name)
	return sp
}

func (sp *ShellProcessControl) setStatus(s bool) {
	sp.status = s
	log.Debug("setStatus process "+sp.cfg.Name, "status", sp.status)
}

func (sp *ShellProcessControl) IsUp() bool {
	s := false
	var err error
	switch strings.ToLower(sp.cfg.Name) {
	case "geth":
		s, err = IsGethUp(sp.gethRpcUrl)
		if err != nil {
			sp.setStatus(false)
			log.Error("geth is down", "err", err)
		} else {
			sp.setStatus(s)
		}
	case "tessera":
		s, err = IsTesseraUp(sp.tesseraUpcheckUrl)
		if err != nil {
			sp.setStatus(false)
			log.Error("tessera is down", "err", err)
		} else {
			sp.setStatus(s)
		}
	}
	log.Debug("IsUp", "name", sp.cfg.Name, "return", sp.status)
	return sp.status
}

func (sp *ShellProcessControl) Stop() error {
	defer log.Info("defer stopped", "process", sp.cfg.Name, "status", sp.status)
	defer sp.muxLock.Unlock()
	sp.muxLock.Lock()
	if !sp.status {
		log.Info("process is already down", "name", sp.cfg.Name)
		return nil
	}
	if err := ExecuteShellCommand("stop "+sp.cfg.Name, sp.cfg.StopCommand); err == nil {
		time.Sleep(1 * time.Second)
		sp.setStatus(false)
		log.Info("stopped", "process", sp.cfg.Name, "status", sp.status)
	} else {
		log.Error("stop "+sp.cfg.Name+" failed", "err", err)
		return err
	}
	return nil
}

func (sp *ShellProcessControl) Start() error {
	defer log.Info("defer started", "process", sp.cfg.Name, "status", sp.status)
	defer sp.muxLock.Unlock()
	sp.muxLock.Lock()
	if sp.status {
		log.Info("process is already up", "name", sp.cfg.Name)
		return nil
	}
	if err := ExecuteShellCommand("start tessera node", sp.cfg.StartCommand); err == nil {
		//wait for process to come up
		if sp.WaitToComeUp() {
			sp.setStatus(true)
			log.Info("started", "process", sp.cfg.Name, "status", sp.status)
		} else {
			sp.setStatus(false)
			log.Error("failed to start " + sp.cfg.Name)
			return fmt.Errorf("%s failed to start", sp.cfg.Name)
		}

	} else {
		log.Error("failed to start " + sp.cfg.Name)
		return err
	}
	return nil
}

func (sp *ShellProcessControl) WaitToComeUp() bool {
	retryCount := 30
	c := 1
	for c <= retryCount {
		if sp.IsUp() {
			return true
		}
		time.Sleep(time.Second)
		log.Info("wait for up "+sp.cfg.Name, "c", c)
		c++
	}
	return false
}
