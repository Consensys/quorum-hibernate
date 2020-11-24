package process

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/core/types"
	"github.com/ConsenSysQuorum/node-manager/log"
	dtypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// DockerControl represents process control for a docker container
type DockerControl struct {
	cfg             *types.ProcessConfig
	bcClntRpcUrl    string
	privManUpchkUrl string
	status          bool
	muxLock         sync.Mutex
}

func NewDockerProcess(p *types.ProcessConfig, grpc string, turl string, s bool) Process {
	sp := &DockerControl{p, grpc, turl, s, sync.Mutex{}}
	sp.IsUp()
	log.Debug("shell process created", "name", sp.cfg.Name)
	return sp
}

func (dp *DockerControl) setStatus(s bool) {
	dp.status = s
	log.Debug("setStatus - process "+dp.cfg.Name, "status", dp.status)
}

// Status implements Process.Status
func (dp *DockerControl) Status() bool {
	return dp.status
}

// IsUp implements Process.IsUp
func (dp *DockerControl) IsUp() bool {
	s := false
	var err error
	if dp.cfg.IsBcClient() {
		s, err = IsBlockchainClientUp(dp.bcClntRpcUrl)
		if err != nil {
			dp.setStatus(false)
			log.Error("IsUp - blockchain client is down", "err", err)
		} else {
			dp.setStatus(s)
		}
	} else if dp.cfg.IsPrivacyManager() {
		s, err = IsPrivacyManagerUp(dp.privManUpchkUrl)
		if err != nil {
			dp.setStatus(false)
			log.Error("IsUp - privacy manager is down", "err", err)
		} else {
			dp.setStatus(s)
		}
	}
	log.Debug("IsUp", "name", dp.cfg.Name, "return", dp.status)
	return dp.status
}

// Stop implements Process.Stop
func (dp *DockerControl) Stop() error {
	defer dp.muxLock.Unlock()
	dp.muxLock.Lock()
	if !dp.status {
		log.Info("Stop - process is already down", "name", dp.cfg.Name)
		return nil
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Error("Stop - new docker client failed", "err", err)
		return err
	}

	if err := cli.ContainerStop(context.Background(), dp.cfg.ContainerId, nil); err == nil {
		log.Info("Stop - docker container stopped", "name", dp.cfg.Name, "id", dp.cfg.ContainerId)
		if dp.WaitToBeDown() {
			dp.setStatus(false)
			log.Debug("Stop - is down", "process", dp.cfg.Name, "status", dp.status)
		} else {
			dp.setStatus(true)
			log.Error("failed to stop " + dp.cfg.Name)
			return fmt.Errorf("%s failed to stop", dp.cfg.Name)
		}
	} else {
		log.Error("Stop - docker container stop failed", "name", dp.cfg.Name, "id", dp.cfg.ContainerId, "err", err)
		dp.setStatus(false)
		return err
	}

	return nil
}

// Stop implements Process.Stop
func (dp *DockerControl) Start() error {
	defer dp.muxLock.Unlock()
	dp.muxLock.Lock()
	if dp.status {
		log.Info("Start - process is already up", "name", dp.cfg.Name)
		return nil
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Error("Start - new docker client failed", "err", err)
		return err
	}

	if err := cli.ContainerStart(context.Background(), dp.cfg.ContainerId, dtypes.ContainerStartOptions{}); err == nil {
		log.Info("Start - docker container started", "name", dp.cfg.Name, "id", dp.cfg.ContainerId)
		//wait for process to come up
		if dp.WaitToComeUp() {
			dp.setStatus(true)
			log.Debug("Start - is up", "process", dp.cfg.Name, "status", dp.status)
		} else {
			dp.setStatus(false)
			log.Error("Start - failed to start " + dp.cfg.Name)
			return fmt.Errorf("%s failed to start", dp.cfg.Name)
		}
	} else {
		log.Error("Start - docker container start failed", "name", dp.cfg.Name, "id", dp.cfg.ContainerId, "err", err)
		return err
	}
	return nil
}

// WaitToComeUp waits for the process status to be up by performing up check repeatedly
// for a certain duration
func (dp *DockerControl) WaitToComeUp() bool {
	retryCount := 30
	c := 1
	for c <= retryCount {
		if dp.IsUp() {
			return true
		}
		time.Sleep(time.Second)
		log.Debug("WaitToComeUp - wait for up "+dp.cfg.Name, "c", c)
		c++
	}
	return false
}

// WaitToBeDown waits for the process status to be down by performing up check repeatedly
// for a certain duration
func (sp *DockerControl) WaitToBeDown() bool {
	retryCount := 30
	c := 1
	for c <= retryCount {
		if !sp.IsUp() {
			return true
		}
		time.Sleep(time.Second)
		log.Debug("WaitToBeDown - wait for down "+sp.cfg.Name, "c", c)
		c++
	}
	return false
}
