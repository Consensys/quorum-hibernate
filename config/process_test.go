package config

import (
	"encoding/json"
	"fmt"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
	"testing"
)

func minimumValidProcess() Process {
	upcheckConfig := minimumValidUpcheck()

	return Process{
		Name:         "bcclnt",
		ControlType:  "shell",
		ContainerId:  "",
		StopCommand:  []string{"stop.sh"},
		StartCommand: []string{"start.sh"},
		UpcheckCfg:   &upcheckConfig,
	}
}

func TestProcess_Unmarshal(t *testing.T) {
	tests := []struct {
		name, configTemplate string
	}{
		{
			name: "json",
			configTemplate: `
{
	"%v": "bcclnt",
	"%v": "shell",
	"%v": "mycontainer",
	"%v": ["stop.sh"],
	"%v": ["start.sh"],
	"%v": {}
}`,
		},
		{
			name: "toml",
			configTemplate: `
%v = "bcclnt"
%v = "shell"
%v = "mycontainer"
%v = ["stop.sh"]
%v = ["start.sh"]
%v = {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := fmt.Sprintf(
				tt.configTemplate,
				nameField,
				controlTypeField,
				containerIdField,
				stopCommandField,
				startCommandField,
				upcheckConfigField,
			)

			want := Process{
				Name:         "bcclnt",
				ControlType:  "shell",
				ContainerId:  "mycontainer",
				StopCommand:  []string{"stop.sh"},
				StartCommand: []string{"start.sh"},
				UpcheckCfg:   &Upcheck{},
			}

			var (
				got Process
				err error
			)

			if tt.name == "json" {
				err = json.Unmarshal([]byte(conf), &got)
			} else if tt.name == "toml" {
				err = toml.Unmarshal([]byte(conf), &got)
			}

			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestProcess_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidProcess()

	err := c.IsValid()

	require.NoError(t, err)
}

func TestProcess_IsValid_Name(t *testing.T) {
	tests := []struct {
		name, nameField, wantErrMsg string
	}{
		{
			name:       "not set",
			nameField:  "",
			wantErrMsg: nameField + " must be bcclnt or privman",
		},
		{
			name:       "invalid",
			nameField:  "invalid",
			wantErrMsg: nameField + " must be bcclnt or privman",
		},
		{
			name:       "bcclnt",
			nameField:  "bcclnt",
			wantErrMsg: "",
		},
		{
			name:       "privman",
			nameField:  "privman",
			wantErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidProcess()
			c.Name = tt.nameField

			err := c.IsValid()

			if tt.wantErrMsg == "" {
				require.NoError(t, err)
			} else {
				require.IsType(t, &fieldErr{}, err)
				require.EqualError(t, err, tt.wantErrMsg)
			}
		})
	}
}

func TestProcess_IsValid_ControlType(t *testing.T) {
	tests := []struct {
		name, controlType, wantErrMsg string
	}{
		{
			name:        "not set",
			controlType: "",
			wantErrMsg:  controlTypeField + " must be shell or docker",
		},
		{
			name:        "invalid",
			controlType: "invalid",
			wantErrMsg:  controlTypeField + " must be shell or docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidProcess()
			c.ControlType = tt.controlType

			err := c.IsValid()

			if tt.wantErrMsg == "" {
				require.NoError(t, err)
			} else {
				require.IsType(t, &fieldErr{}, err)
				require.EqualError(t, err, tt.wantErrMsg)
			}
		})
	}
}

func TestProcess_IsValid_ContainerId(t *testing.T) {
	c := minimumValidProcess()
	c.ControlType = "docker"
	c.ContainerId = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, fmt.Sprintf("%v must be set as %v is docker", containerIdField, controlTypeField))
}

func TestProcess_IsValid_StartCommand(t *testing.T) {
	c := minimumValidProcess()
	c.ControlType = "shell"
	c.StartCommand = nil

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, fmt.Sprintf("%v must be set as %v is shell", startCommandField, controlTypeField))
}

func TestProcess_IsValid_StopCommand(t *testing.T) {
	c := minimumValidProcess()
	c.ControlType = "shell"
	c.StopCommand = nil

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, fmt.Sprintf("%v must be set as %v is shell", stopCommandField, controlTypeField))
}

func TestProcess_IsValid_UpcheckConfig(t *testing.T) {
	invalidUpcheck := minimumValidUpcheck()
	invalidUpcheck.UpcheckUrl = ""

	tests := []struct {
		name          string
		upcheckConfig *Upcheck
		wantErrMsg    string
	}{
		{
			name:          "not set",
			upcheckConfig: nil,
			wantErrMsg:    upcheckConfigField + " is empty",
		},
		{
			name:          "invalid",
			upcheckConfig: &invalidUpcheck,
			wantErrMsg:    fmt.Sprintf("%v.%v is empty", upcheckConfigField, urlField),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidProcess()
			c.UpcheckCfg = tt.upcheckConfig

			err := c.IsValid()

			if tt.wantErrMsg == "" {
				require.NoError(t, err)
			} else {
				require.IsType(t, &fieldErr{}, err)
				require.EqualError(t, err, tt.wantErrMsg)
			}
		})
	}
}

func TestProcess_IsShell(t *testing.T) {
	tests := []struct {
		name, controlType string
		want              bool
	}{
		{
			name:        "not shell",
			controlType: "docker",
			want:        false,
		},
		{
			name:        "is shell",
			controlType: "shell",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Process{
				ControlType: tt.controlType,
			}
			require.Equal(t, tt.want, c.IsShell())
		})
	}
}

func TestProcess_IsDocker(t *testing.T) {
	tests := []struct {
		name, controlType string
		want              bool
	}{
		{
			name:        "not docker",
			controlType: "shell",
			want:        false,
		},
		{
			name:        "is docker",
			controlType: "docker",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Process{
				ControlType: tt.controlType,
			}
			require.Equal(t, tt.want, c.IsDocker())
		})
	}
}

func TestProcess_IsBcClient(t *testing.T) {
	tests := []struct {
		name, processName string
		want              bool
	}{
		{
			name:        "not bcclnt",
			processName: "privman",
			want:        false,
		},
		{
			name:        "is bcclnt",
			processName: "bcclnt",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Process{
				Name: tt.processName,
			}
			require.Equal(t, tt.want, c.IsBcClient())
		})
	}
}

func TestProcess_IsPrivacyManager(t *testing.T) {
	tests := []struct {
		name, processName string
		want              bool
	}{
		{
			name:        "not privman",
			processName: "bcclnt",
			want:        false,
		},
		{
			name:        "is privman",
			processName: "privman",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Process{
				Name: tt.processName,
			}
			require.Equal(t, tt.want, c.IsPrivacyManager())
		})
	}
}
