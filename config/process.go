package config

import (
	"errors"
	"strings"
)

type Process struct {
	Name         string   `toml:"name" json:"name"`                   // name of process. should be bcclnt or privman
	ControlType  string   `toml:"controlType" json:"controlType"`     // control type supported. shell or docker
	ContainerId  string   `toml:"containerId" json:"containerId"`     // docker container id. required if controlType is docker
	StopCommand  []string `toml:"stopCommand" json:"stopCommand"`     // stop command. required if controlType is shell
	StartCommand []string `toml:"startCommand" json:"startCommand"`   // start command. required if controlType is shell
	UpcheckCfg   *Upcheck `toml:"upcheckConfig" json:"upcheckConfig"` // Upcheck config
}

func (c Process) IsShell() bool {
	return strings.ToLower(c.ControlType) == "shell"
}

func (c Process) IsDocker() bool {
	return strings.ToLower(c.ControlType) == "docker"
}

func (c Process) IsBcClient() bool {
	return strings.ToLower(c.Name) == "bcclnt"
}

func (c Process) IsPrivacyManager() bool {
	return strings.ToLower(c.Name) == "privman"
}

func (c Process) IsValid() error {
	if !c.IsDocker() && !c.IsShell() {
		return newFieldErr("controlType", errors.New("must be shell or docker"))
	}
	if !c.IsBcClient() && !c.IsPrivacyManager() {
		return newFieldErr("name", errors.New("must be bcclnt or privman"))
	}
	if c.IsDocker() && c.ContainerId == "" {
		return newFieldErr("containerId", errors.New("must be set as controlType is docker"))
	}
	if c.IsShell() && len(c.StartCommand) == 0 {
		return newFieldErr("startCommand", errors.New("must be set as controlType is shell"))
	}
	if c.IsShell() && len(c.StopCommand) == 0 {
		return newFieldErr("stopCommand", errors.New("must be set as controlType is shell"))
	}
	if c.UpcheckCfg == nil {
		return newFieldErr("upcheckConfig", isEmptyErr)
	}
	if err := c.UpcheckCfg.IsValid(); err != nil {
		return newFieldErr("upcheckConfig", err)
	}
	return nil
}
