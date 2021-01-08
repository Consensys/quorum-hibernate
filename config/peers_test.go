package config

import (
	"encoding/json"
	"fmt"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
	"testing"
)

func minimumValidPeer() Peer {
	return Peer{
		Name:       "mypeer",
		PrivManKey: "akey",
		RpcUrl:     "http://url",
		TLSConfig:  nil,
	}
}

func TestNodeManagerList_Unmarshal(t *testing.T) {
	tests := []struct {
		name, configTemplate string
	}{
		{
			name: "json",
			configTemplate: `
{
	"%v": [
		{
			"%v": "mypeer",		
			"%v": "akey",		
			"%v": "http://url",		
			"%v": {}		
		}
	]
}`,
		},
		{
			name: "toml",
			configTemplate: `
[[%v]]
%v = "mypeer"
%v = "akey"
%v = "http://url"
%v = {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := fmt.Sprintf(tt.configTemplate, peersField, nameField, privacyManagerKeyField, rpcUrlField, tlsConfigField)

			want := NodeManagerList{
				Peers: []*Peer{
					{
						Name:       "mypeer",
						PrivManKey: "akey",
						RpcUrl:     "http://url",
						TLSConfig:  &ClientTLS{},
					},
				},
			}

			var (
				got NodeManagerList
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

func TestPeerArr_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidPeer()

	a := PeerArr{&c}

	err := a.IsValid()

	require.NoError(t, err)
}

func TestPeerArr_IsValid(t *testing.T) {
	validPeer := minimumValidPeer()
	validPeer.Name = "mypeer1"

	invalidPeer := minimumValidPeer()
	invalidPeer.RpcUrl = ""
	invalidPeer.Name = "mypeer2"

	anotherValidPeer := minimumValidPeer()

	tests := []struct {
		name       string
		peers      PeerArr
		wantErrMsg string
	}{
		{
			name:       "invalid",
			peers:      PeerArr{&invalidPeer},
			wantErrMsg: fmt.Sprintf("%v[0].%v is empty", peersField, rpcUrlField),
		},
		{
			name:       "valid and invalid",
			peers:      PeerArr{&validPeer, &invalidPeer},
			wantErrMsg: fmt.Sprintf("%v[1].%v is empty", peersField, rpcUrlField),
		},
		{
			name:       "duplicate peers",
			peers:      PeerArr{&validPeer, &validPeer},
			wantErrMsg: fmt.Sprintf("%v[1].%v must be unique", peersField, nameField),
		},
		{
			name:       "no error",
			peers:      PeerArr{&validPeer, &anotherValidPeer},
			wantErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.peers.IsValid()

			if tt.wantErrMsg == "" {
				require.NoError(t, err)
			} else {
				require.IsType(t, &arrFieldErr{}, err)
				require.EqualError(t, err, tt.wantErrMsg)
			}

		})
	}

}

func TestPeer_IsValid_MinimumValid(t *testing.T) {
	c := minimumValidPeer()

	err := c.IsValid()

	require.NoError(t, err)
}

func TestPeer_IsValid_Name(t *testing.T) {
	c := minimumValidPeer()
	c.Name = ""

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, nameField+" is empty")
}

func TestPeer_IsValid_RpcUrl(t *testing.T) {
	tests := []struct {
		name, rpcUrl, wantErr string
	}{
		{
			name:    "not set",
			rpcUrl:  "",
			wantErr: rpcUrlField + " is empty",
		},
		{
			name:    "invalid url",
			rpcUrl:  "://no-scheme",
			wantErr: rpcUrlField + ` parse "://no-scheme": missing protocol scheme`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := minimumValidPeer()
			c.RpcUrl = tt.rpcUrl

			err := c.IsValid()

			require.IsType(t, &fieldErr{}, err)
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestPeer_IsValid_TLSConfig(t *testing.T) {
	c := minimumValidPeer()
	c.TLSConfig = &ClientTLS{}

	err := c.IsValid()

	require.IsType(t, &fieldErr{}, err)
	require.EqualError(t, err, fmt.Sprintf("%v.%v %v", tlsConfigField, caCertificateFileField, "is empty"))
}
