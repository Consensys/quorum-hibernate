package process

import (
	"context"
	"fmt"
	"github.com/ConsenSysQuorum/node-manager/config"
	"net/http"
	"sync"
	"time"

	"github.com/ConsenSysQuorum/node-manager/log"
	dtypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// DockerControl represents process control for a docker container
type DockerControl struct {
	cfg     *config.Process
	status  bool
	client  *http.Client
	muxLock sync.Mutex
}

func NewDockerProcess(c *http.Client, p *config.Process, s bool) Process {
	sp := &DockerControl{p, s, c, sync.Mutex{}}
	sp.UpdateStatus()
	log.Debug("docker process created", "name", sp.cfg.Name)
	return sp
}

func (dc *DockerControl) setStatus(s bool) {
	dc.status = s
	log.Debug("setStatus - process "+dc.cfg.Name, "status", dc.status)
}

// Status implements Process.Status
func (dc *DockerControl) Status() bool {
	return dc.status
}

// UpdateStatus implements Process.UpdateStatus
func (dc *DockerControl) UpdateStatus() bool {
	s := false
	var err error
	s, err = IsProcessUp(dc.client, dc.cfg.UpcheckCfg)
	if err != nil {
		dc.setStatus(false)
		log.Error("Update status - docker process is down", "err", err)
	} else {
		dc.setStatus(s)
	}
	log.Debug("UpdateStatus", "name", dc.cfg.Name, "return", dc.status)
	return dc.status
}

// Stop implements Process.Stop
func (dc *DockerControl) Stop() error {
	defer dc.muxLock.Unlock()
	dc.muxLock.Lock()
	if !dc.status {
		log.Info("Stop - process is already down", "name", dc.cfg.Name)
		return nil
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Error("Stop - new docker client failed", "err", err)
		return err
	}

	if err := cli.ContainerStop(context.Background(), dc.cfg.ContainerId, nil); err == nil {
		log.Info("Stop - docker container stopped", "name", dc.cfg.Name, "id", dc.cfg.ContainerId)
		if dc.WaitToBeDown() {
			dc.setStatus(false)
			log.Debug("Stop - is down", "process", dc.cfg.Name, "status", dc.status)
		} else {
			dc.setStatus(true)
			log.Error("failed to stop " + dc.cfg.Name)
			return fmt.Errorf("%s failed to stop", dc.cfg.Name)
		}
	} else {
		log.Error("Stop - docker container stop failed", "name", dc.cfg.Name, "id", dc.cfg.ContainerId, "err", err)
		dc.setStatus(false)
		return err
	}

	return nil
}

// Stop implements Process.Stop
func (dc *DockerControl) Start() error {
	defer dc.muxLock.Unlock()
	dc.muxLock.Lock()
	if dc.status {
		log.Info("Start - process is already up", "name", dc.cfg.Name)
		return nil
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Error("Start - new docker client failed", "err", err)
		return err
	}

	if err := cli.ContainerStart(context.Background(), dc.cfg.ContainerId, dtypes.ContainerStartOptions{}); err == nil {
		log.Info("Start - docker container started", "name", dc.cfg.Name, "id", dc.cfg.ContainerId)
		//wait for process to come up
		if dc.WaitToComeUp() {
			dc.setStatus(true)
			log.Debug("Start - is up", "process", dc.cfg.Name, "status", dc.status)
		} else {
			dc.setStatus(false)
			log.Error("Start - failed to start " + dc.cfg.Name)
			return fmt.Errorf("%s failed to start", dc.cfg.Name)
		}
	} else {
		log.Error("Start - docker container start failed", "name", dc.cfg.Name, "id", dc.cfg.ContainerId, "err", err)
		return err
	}
	return nil
}

// WaitToComeUp waits for the process status to be up by performing up check repeatedly
// for a certain duration
func (dc *DockerControl) WaitToComeUp() bool {
	retryCount := 30
	c := 1
	for c <= retryCount {
		if dc.UpdateStatus() {
			return true
		}
		time.Sleep(time.Second)
		log.Debug("WaitToComeUp - wait for up "+dc.cfg.Name, "c", c)
		c++
	}
	return false
}

// WaitToBeDown waits for the process status to be down by performing up check repeatedly
// for a certain duration
func (dc *DockerControl) WaitToBeDown() bool {
	retryCount := 30
	c := 1
	for c <= retryCount {
		if !dc.UpdateStatus() {
			return true
		}
		time.Sleep(time.Second)
		log.Debug("WaitToBeDown - wait for down "+dc.cfg.Name, "c", c)
		c++
	}
	return false
}
