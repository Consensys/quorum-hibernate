package process

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
	dtypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type DockerControl struct {
	cfg               *types.ProcessConfig
	gethRpcUrl        string
	tesseraUpcheckUrl string
	status            bool
	muxLock           sync.Mutex
}

func NewDockerProcess(p *types.ProcessConfig, grpc string, turl string, s bool) Process {
	sp := &DockerControl{p, grpc, turl, s, sync.Mutex{}}
	sp.IsUp()
	log.Debug("shell process created", "name", sp.cfg.Name)
	return sp
}

func (dp *DockerControl) setStatus(s bool) {
	dp.status = s
	log.Debug("setStatus process "+dp.cfg.Name, "status", dp.status)
}

func (dp *DockerControl) Status() bool {
	return dp.status
}

func (dp *DockerControl) IsUp() bool {
	s := false
	var err error
	switch strings.ToLower(dp.cfg.Name) {
	case "geth":
		s, err = IsGethUp(dp.gethRpcUrl)
		if err != nil {
			dp.setStatus(false)
			log.Error("geth is down", "err", err)
		} else {
			dp.setStatus(s)
		}
	case "tessera":
		s, err = IsTesseraUp(dp.tesseraUpcheckUrl)
		if err != nil {
			dp.setStatus(false)
			log.Error("tessera is down", "err", err)
		} else {
			dp.setStatus(s)
		}
	}
	log.Debug("IsUp", "name", dp.cfg.Name, "return", dp.status)
	return dp.status
}

func (dp *DockerControl) Stop() error {
	defer log.Info("defer stopped", "process", dp.cfg.Name, "status", dp.status)
	defer dp.muxLock.Unlock()
	dp.muxLock.Lock()
	if !dp.status {
		log.Info("process is already down", "name", dp.cfg.Name)
		return nil
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Error("new docker client failed", "err", err)
		return err
	}

	if err := cli.ContainerStop(context.Background(), dp.cfg.ContainerId, nil); err == nil {
		log.Info("docker container stopped", "name", dp.cfg.Name, "id", dp.cfg.ContainerId)
		if dp.WaitToBeDown() {
			dp.setStatus(false)
			log.Info("is down", "process", dp.cfg.Name, "status", dp.status)
		} else {
			dp.setStatus(true)
			log.Error("failed to stop " + dp.cfg.Name)
			return fmt.Errorf("%s failed to stop", dp.cfg.Name)
		}
	} else {
		log.Error("docker container stop failed", "name", dp.cfg.Name, "id", dp.cfg.ContainerId, "err", err)
		dp.setStatus(false)
		return err
	}

	return nil
}

func (dp *DockerControl) Start() error {
	defer log.Info("defer started", "process", dp.cfg.Name, "status", dp.status)
	defer dp.muxLock.Unlock()
	dp.muxLock.Lock()
	if dp.status {
		log.Info("process is already up", "name", dp.cfg.Name)
		return nil
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Error("new docker client failed", "err", err)
		return err
	}

	if err := cli.ContainerStart(context.Background(), dp.cfg.ContainerId, dtypes.ContainerStartOptions{}); err == nil {
		log.Info("docker container started", "name", dp.cfg.Name, "id", dp.cfg.ContainerId)
		//wait for process to come up
		if dp.WaitToComeUp() {
			dp.setStatus(true)
			log.Info("is up", "process", dp.cfg.Name, "status", dp.status)
		} else {
			dp.setStatus(false)
			log.Error("failed to start " + dp.cfg.Name)
			return fmt.Errorf("%s failed to start", dp.cfg.Name)
		}
	} else {
		log.Info("docker container start failed", "name", dp.cfg.Name, "id", dp.cfg.ContainerId, "err", err)
		return err
	}
	return nil
}

func (dp *DockerControl) WaitToComeUp() bool {
	retryCount := 30
	c := 1
	for c <= retryCount {
		if dp.IsUp() {
			return true
		}
		time.Sleep(time.Second)
		log.Info("wait for up "+dp.cfg.Name, "c", c)
		c++
	}
	return false
}

func (sp *DockerControl) WaitToBeDown() bool {
	retryCount := 30
	c := 1
	for c <= retryCount {
		if !sp.IsUp() {
			return true
		}
		time.Sleep(time.Second)
		log.Info("wait for down "+sp.cfg.Name, "c", c)
		c++
	}
	return false
}
