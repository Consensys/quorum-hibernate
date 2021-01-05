package config

import (
	"encoding/json"
	"fmt"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
	"testing"
)

func minimumValidBasic() Basic {
	blockchainClient := minimumValidBlockchainClient()
	privacyManager := minimumValidPrivacyManager()
	server := minimumValidRPCServer()
	proxy := minimumValidProxy()

	return Basic{
		Name:                 "myname",
		DisableStrictMode:    false,
		UpchkPollingInterval: 1,
		PeersConfigFile:      "/path/to/conf.json",
		InactivityTime:       60,
		ResyncTime:           0,
		BlockchainClient:     &blockchainClient,
		PrivacyManager:       &privacyManager,
		Server:               &server,
		Proxies:              []*Proxy{&proxy},
	}
}

func TestBasic_Unmarshal(t *testing.T) {
	tests := []struct {
		name, configTemplate string
	}{
		{
			name: "json",
			configTemplate: `
{
	"%v": "myname",
	"%v": true,
	"%v": 1,
	"%v": "/path/to/conf.json",
	"%v": 60,
	"%v": 120,
	"%v": {},
	"%v": {},
	"%v": {},
	"%v": [{}]
}`,
		},
		{
			name: "toml",
			configTemplate: `
%v = "myname"
%v = true
%v = 1
%v = "/path/to/conf.json"
%v = 60
%v = 120
%v = {}
%v = {}
%v = {}
%v = [{}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := fmt.Sprintf(
				tt.configTemplate,
				nameField,
				disableStrictModeField,
				upcheckPollingIntervalField,
				peersConfigFileField,
				inactivityTimeField,
				resyncTimeField,
				blockchainClientField,
				privacyManagerField,
				serverField,
				proxiesField,
			)

			want := Basic{
				Name:                 "myname",
				DisableStrictMode:    true,
				UpchkPollingInterval: 1,
				PeersConfigFile:      "/path/to/conf.json",
				InactivityTime:       60,
				ResyncTime:           120,
				BlockchainClient:     &BlockchainClient{},
				PrivacyManager:       &PrivacyManager{},
				Server:               &RPCServer{},
				Proxies:              []*Proxy{{}},
			}

			var (
				got Basic
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

func TestBasic_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidBasic()

	err := c.IsValid()

	require.NoError(t, err)
}

func TestBasic_IsValid_Name(t *testing.T) {
	c := minimumValidBasic()
	c.Name = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, nameField+" is empty")
}

func TestBasic_IsValid_PeersConfigFile(t *testing.T) {
	c := minimumValidBasic()
	c.PeersConfigFile = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, peersConfigFileField+" is empty")
}

func TestBasic_IsValid_UpcheckPollingInterval(t *testing.T) {
	tests := []struct {
		name                   string
		upcheckPollingInterval int
		wantErrMsg             string
	}{
		{
			name:                   "zero",
			upcheckPollingInterval: 0,
			wantErrMsg:             upcheckPollingIntervalField + " must be > 0",
		},
		{
			name:                   "negative",
			upcheckPollingInterval: -10,
			wantErrMsg:             upcheckPollingIntervalField + " must be > 0",
		},
		{
			name:                   "positive",
			upcheckPollingInterval: 10,
			wantErrMsg:             "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidBasic()
			c.UpchkPollingInterval = tt.upcheckPollingInterval

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

func TestBasic_IsValid_InactivityTime(t *testing.T) {
	tests := []struct {
		name           string
		inactivityTime int
		wantErrMsg     string
	}{
		{
			name:           "less than 60",
			inactivityTime: 59,
			wantErrMsg:     inactivityTimeField + " must be >= 60",
		},
		{
			name:           "60",
			inactivityTime: 60,
			wantErrMsg:     "",
		},
		{
			name:           "greater than 60",
			inactivityTime: 61,
			wantErrMsg:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidBasic()
			c.InactivityTime = tt.inactivityTime

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

func TestBasic_IsValid_ResyncTime(t *testing.T) {
	tests := []struct {
		name           string
		resyncTime     int
		inactivityTime int
		wantErrMsg     string
	}{
		{
			name:           "not set",
			resyncTime:     0,
			inactivityTime: 60,
			wantErrMsg:     "",
		},
		{
			name:           "less than inactivityTime",
			resyncTime:     59,
			inactivityTime: 60,
			wantErrMsg:     fmt.Sprintf("%v must be >= %v", resyncTimeField, inactivityTimeField),
		},
		{
			name:           "equal to inactivityTime",
			resyncTime:     60,
			inactivityTime: 60,
			wantErrMsg:     "",
		},
		{
			name:           "greater than inactivityTime",
			resyncTime:     61,
			inactivityTime: 60,
			wantErrMsg:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidBasic()
			c.ResyncTime = tt.resyncTime
			c.InactivityTime = tt.inactivityTime

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

func TestBasic_IsValid_Server(t *testing.T) {
	invalidServer := minimumValidRPCServer()
	invalidServer.RPCAddr = ""

	tests := []struct {
		name       string
		server     *RPCServer
		wantErrMsg string
	}{
		{
			name:       "not set",
			server:     nil,
			wantErrMsg: serverField + " is empty",
		},
		{
			name:       "invalid",
			server:     &invalidServer,
			wantErrMsg: fmt.Sprintf("%v.%v is empty", serverField, rpcAddressField),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidBasic()
			c.Server = tt.server

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

func TestBasic_IsValid_BlockchainClient(t *testing.T) {
	invalidBlockchainClient := minimumValidBlockchainClient()
	invalidBlockchainClient.BcClntRpcUrl = ""

	tests := []struct {
		name             string
		blockchainClient *BlockchainClient
		wantErrMsg       string
	}{
		{
			name:             "not set",
			blockchainClient: nil,
			wantErrMsg:       blockchainClientField + " is empty",
		},
		{
			name:             "invalid",
			blockchainClient: &invalidBlockchainClient,
			wantErrMsg:       fmt.Sprintf("%v.%v is empty", blockchainClientField, rpcUrlField),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidBasic()
			c.BlockchainClient = tt.blockchainClient

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

func TestBasic_IsValid_PrivacyManager(t *testing.T) {
	invalidPrivacyManager := minimumValidPrivacyManager()
	invalidPrivacyManager.PrivManKey = ""

	tests := []struct {
		name           string
		privacyManager *PrivacyManager
		wantErrMsg     string
	}{
		{
			name:           "not set",
			privacyManager: nil,
			wantErrMsg:     "",
		},
		{
			name:           "invalid",
			privacyManager: &invalidPrivacyManager,
			wantErrMsg:     fmt.Sprintf("%v.%v is empty", privacyManagerField, publicKeyField),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidBasic()
			c.PrivacyManager = tt.privacyManager

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

func TestBasic_IsValid_Proxies_NotSet(t *testing.T) {
	c := minimumValidBasic()
	c.Proxies = nil

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, proxiesField+" is empty")
}

func TestBasic_IsValid_Proxies_Invalid(t *testing.T) {
	validProxy := minimumValidProxy()

	invalidProxy := minimumValidProxy()
	invalidProxy.Name = ""

	tests := []struct {
		name       string
		proxies    []*Proxy
		wantErrMsg string
	}{
		{
			name:       "invalid",
			proxies:    []*Proxy{&invalidProxy},
			wantErrMsg: fmt.Sprintf("%v[0].%v is empty", proxiesField, nameField),
		},
		{
			name:       "mix of valid and invalid",
			proxies:    []*Proxy{&validProxy, &invalidProxy},
			wantErrMsg: fmt.Sprintf("%v[1].%v is empty", proxiesField, nameField),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidBasic()
			c.Proxies = tt.proxies

			err := c.IsValid()

			if tt.wantErrMsg == "" {
				require.NoError(t, err)
			} else {
				require.IsType(t, &arrFieldErr{}, err)
				require.EqualError(t, err, tt.wantErrMsg)
			}
		})
	}
}

func TestBasic_IsResyncTimerSet(t *testing.T) {
	tests := []struct {
		name       string
		resyncTime int
		want       bool
	}{
		{
			name:       "not set",
			resyncTime: 0,
			want:       false,
		},
		{
			name:       "set",
			resyncTime: 10,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Basic{
				ResyncTime: tt.resyncTime,
			}
			require.Equal(t, tt.want, c.IsResyncTimerSet())
		})
	}
}

func TestBasic_IsRaft(t *testing.T) {
	tests := []struct {
		name, consensus string
		want            bool
	}{
		{
			name:      "not raft",
			consensus: "istanbul",
			want:      false,
		},
		{
			name:      "raft",
			consensus: "raft",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Basic{
				BlockchainClient: &BlockchainClient{
					Consensus: tt.consensus,
				},
			}
			require.Equal(t, tt.want, c.IsRaft())
		})
	}
}

func TestBasic_IsIstanbul(t *testing.T) {
	tests := []struct {
		name, consensus string
		want            bool
	}{
		{
			name:      "not istanbul",
			consensus: "raft",
			want:      false,
		},
		{
			name:      "istanbul",
			consensus: "istanbul",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Basic{
				BlockchainClient: &BlockchainClient{
					Consensus: tt.consensus,
				},
			}
			require.Equal(t, tt.want, c.IsIstanbul())
		})
	}
}

func TestBasic_IsClique(t *testing.T) {
	tests := []struct {
		name, consensus string
		want            bool
	}{
		{
			name:      "not clique",
			consensus: "istanbul",
			want:      false,
		},
		{
			name:      "clique",
			consensus: "clique",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Basic{
				BlockchainClient: &BlockchainClient{
					Consensus: tt.consensus,
				},
			}
			require.Equal(t, tt.want, c.IsClique())
		})
	}
}

func TestBasic_IsGoQuorumClient(t *testing.T) {
	tests := []struct {
		name, client string
		want         bool
	}{
		{
			name:   "not goquorum",
			client: "besu",
			want:   false,
		},
		{
			name:   "is goquorum",
			client: "goquorum",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Basic{
				BlockchainClient: &BlockchainClient{
					ClientType: tt.client,
				},
			}
			require.Equal(t, tt.want, c.IsGoQuorumClient())
		})
	}
}

func TestBasic_IsBesuClient(t *testing.T) {
	tests := []struct {
		name, client string
		want         bool
	}{
		{
			name:   "not besu",
			client: "goquorum",
			want:   false,
		},
		{
			name:   "is besu",
			client: "besu",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Basic{
				BlockchainClient: &BlockchainClient{
					ClientType: tt.client,
				},
			}
			require.Equal(t, tt.want, c.IsBesuClient())
		})
	}
}
