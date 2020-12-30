package config

import (
	"encoding/json"
	"fmt"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
	"testing"
)

func minimumValidBlockchainClient() BlockchainClient {
	bcClntProcess := minimumValidProcess()

	return BlockchainClient{
		ClientType:      "goquorum",
		Consensus:       "istanbul",
		BcClntRpcUrl:    "http://url",
		BcClntTLSConfig: nil,
		BcClntProcess:   &bcClntProcess,
	}
}

func minimumValidPrivacyManager() PrivacyManager {
	privManProcess := minimumValidProcess()

	return PrivacyManager{
		PrivManKey:       "mykey",
		PrivManTLSConfig: nil,
		PrivManProcess:   &privManProcess,
	}
}

func TestBlockchainClient_Unmarshal(t *testing.T) {
	tests := []struct {
		name, configTemplate string
	}{
		{
			name: "json",
			configTemplate: `
{
	"%v": "goquorum",
	"%v": "istanbul",
	"%v": "http://url",
	"%v": {},
	"%v": {}
}`,
		},
		{
			name: "toml",
			configTemplate: `
%v = "goquorum"
%v = "istanbul"
%v = "http://url"
%v = {}
%v = {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := fmt.Sprintf(
				tt.configTemplate,
				typeField,
				consensusField,
				rpcUrlField,
				tlsConfigField,
				processField,
			)

			want := BlockchainClient{
				ClientType:      "goquorum",
				Consensus:       "istanbul",
				BcClntRpcUrl:    "http://url",
				BcClntTLSConfig: &ClientTLS{},
				BcClntProcess:   &Process{},
			}

			var (
				got BlockchainClient
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

func TestPrivacyManager_Unmarshal(t *testing.T) {
	tests := []struct {
		name, configTemplate string
	}{
		{
			name: "json",
			configTemplate: `
{
	"%v": "mykey",
	"%v": {},
	"%v": {}
}`,
		},
		{
			name: "toml",
			configTemplate: `
%v = "mykey"
%v = {}
%v = {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := fmt.Sprintf(
				tt.configTemplate,
				publicKeyField,
				tlsConfigField,
				processField,
			)

			want := PrivacyManager{
				PrivManKey:       "mykey",
				PrivManTLSConfig: &ClientTLS{},
				PrivManProcess:   &Process{},
			}

			var (
				got PrivacyManager
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

func TestBlockchainClient_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidBlockchainClient()

	err := c.IsValid()

	require.NoError(t, err)
}

func TestBlockchainClient_IsValid_Type(t *testing.T) {
	tests := []struct {
		name, clientType, wantErrMsg string
	}{
		{
			name:       "not set",
			clientType: "",
			wantErrMsg: typeField + " is empty",
		},
		{
			name:       "invalid",
			clientType: "notvalid",
			wantErrMsg: typeField + " must be goquorum or besu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidBlockchainClient()
			c.ClientType = tt.clientType

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

func TestBlockchainClient_IsValid_Consensus(t *testing.T) {
	tests := []struct {
		name, clientType, consensus, wantErrMsg string
	}{
		{
			name:       "not set and type goquorum",
			clientType: "goquorum",
			consensus:  "",
			wantErrMsg: consensusField + " is empty",
		},
		{
			name:       "invalid and type goquorum",
			clientType: "goquorum",
			consensus:  "notvalid",
			wantErrMsg: consensusField + " must be raft, istanbul, or clique",
		},
		{
			name:       "raft and type goquorum",
			clientType: "goquorum",
			consensus:  "raft",
			wantErrMsg: "",
		},
		{
			name:       "istanbul and type goquorum",
			clientType: "goquorum",
			consensus:  "istanbul",
			wantErrMsg: "",
		},
		{
			name:       "clique and type goquorum",
			clientType: "goquorum",
			consensus:  "clique",
			wantErrMsg: "",
		},
		{
			name:       "not set and type besu",
			clientType: "besu",
			consensus:  "",
			wantErrMsg: consensusField + " is empty",
		},
		{
			name:       "invalid and type besu",
			clientType: "besu",
			consensus:  "notvalid",
			wantErrMsg: consensusField + " must be clique",
		},
		{
			name:       "raft and type besu",
			clientType: "besu",
			consensus:  "raft",
			wantErrMsg: consensusField + " must be clique",
		},
		{
			name:       "istanbul and type besu",
			clientType: "besu",
			consensus:  "istanbul",
			wantErrMsg: consensusField + " must be clique",
		},
		{
			name:       "clique and type besu",
			clientType: "besu",
			consensus:  "clique",
			wantErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidBlockchainClient()
			c.ClientType = tt.clientType
			c.Consensus = tt.consensus

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

func TestBlockchainClient_IsValid_RpcUrl(t *testing.T) {
	c := minimumValidBlockchainClient()
	c.BcClntRpcUrl = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, rpcUrlField+" is empty")
}

func TestBlockchainClient_IsValid_Process(t *testing.T) {
	invalidProcess := minimumValidProcess()
	invalidProcess.Name = ""

	tests := []struct {
		name       string
		process    *Process
		wantErrMsg string
	}{
		{
			name:       "not set",
			process:    nil,
			wantErrMsg: processField + " is empty",
		},
		{
			name:       "invalid",
			process:    &invalidProcess,
			wantErrMsg: fmt.Sprintf("%v.%v must be bcclnt or privman", processField, nameField),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidBlockchainClient()
			c.BcClntProcess = tt.process

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

func TestBlockchainClient_IsValid_TLSConfig(t *testing.T) {
	c := minimumValidBlockchainClient()
	c.BcClntTLSConfig = &ClientTLS{}

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, fmt.Sprintf("%v.%v %v", tlsConfigField, caCertificateFileField, "is empty"))
}

func TestPrivacyManager_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidPrivacyManager()

	err := c.IsValid()

	require.NoError(t, err)
}

func TestPrivacyManager_IsValid_PublicKey(t *testing.T) {
	c := minimumValidPrivacyManager()
	c.PrivManKey = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, publicKeyField+" is empty")
}

func TestPrivacyManager_IsValid_Process(t *testing.T) {
	invalidProcess := minimumValidProcess()
	invalidProcess.Name = ""

	tests := []struct {
		name       string
		process    *Process
		wantErrMsg string
	}{
		{
			name:       "not set",
			process:    nil,
			wantErrMsg: processField + " is empty",
		},
		{
			name:       "invalid",
			process:    &invalidProcess,
			wantErrMsg: fmt.Sprintf("%v.%v must be bcclnt or privman", processField, nameField),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidPrivacyManager()
			c.PrivManProcess = tt.process

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

func TestPrivacyManager_IsValid_TLSConfig(t *testing.T) {
	c := minimumValidPrivacyManager()
	c.PrivManTLSConfig = &ClientTLS{}

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, fmt.Sprintf("%v.%v %v", tlsConfigField, caCertificateFileField, "is empty"))
}
