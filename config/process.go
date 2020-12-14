package config

import (
	"errors"
	"strings"
)

type Process struct {
	Name         string   `toml:"name" json:"name"`                 // name of process. should be bcclnt or privman
	ControlType  string   `toml:"controlType" json:"controlType"`   // control type supported. shell or docker
	ContainerId  string   `toml:"containerId" json:"containerId"`   // docker container id. required if controlType is docker
	StopCommand  []string `toml:"stopCommand" json:"stopCommand"`   // stop command. required if controlType is shell
	StartCommand []string `toml:"startCommand" json:"startCommand"` // start command. required if controlType is shell
	UpcheckCfg   Upcheck  `toml:"upcheckCfg" json:"upcheckCfg"`     // Upcheck config
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
		return errors.New("invalid controlType. supports only shell or docker")
	}
	if !c.IsBcClient() && !c.IsPrivacyManager() {
		return errors.New("invalid name. supports only bcclnt or privman")
	}
	if c.IsDocker() && c.ContainerId == "" {
		return errors.New("containerId is empty for docker controlType")
	}
	if c.IsShell() && (len(c.StartCommand) == 0 || len(c.StopCommand) == 0) {
		return errors.New("startCommand or stopCommand is empty for shell controlType")
	}
	if err := c.UpcheckCfg.IsValid(); err != nil {
		return err
	}
	return nil
}
