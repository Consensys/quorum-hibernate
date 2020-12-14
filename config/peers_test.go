package config

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestNewPeersReader(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		wantImpl interface{}
	}{
		{
			name:     "toml",
			file:     "conf.toml",
			wantImpl: tomlPeersReader{},
		},
		{
			name:     "json",
			file:     "conf.json",
			wantImpl: jsonPeersReader{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewPeersReader(tt.file)
			require.IsType(t, tt.wantImpl, r)
			require.NoError(t, err)
		})
	}
}

func TestNewPeersReader_UnsupportedFileFormat(t *testing.T) {
	_, err := NewPeersReader("conf.yaml")
	require.EqualError(t, err, "unsupported config file format")
}

func TestPeersReader_Read(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "toml",
			config: `
peers = [
	{ name = "node1", privacyManagerKey = "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=", rpcUrl = "http://localhost:8081" },
	{ name = "node2", privacyManagerKey = "QfeDAys9MPDs2XHExtc84jKGHxZg/aj52DTh0vtA3Xc=", rpcUrl = "http://localhost:8082" }
]`,
		},
		{
			name: "json",
			config: `
{
	"peers": [
		{ 
			"name": "node1", 
			"privacyManagerKey": "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=", 
			"rpcUrl": "http://localhost:8081" 
		},
		{ 
			"name": "node2", 
			"privacyManagerKey": "QfeDAys9MPDs2XHExtc84jKGHxZg/aj52DTh0vtA3Xc=", 
			"rpcUrl": "http://localhost:8082" 
		}
	]
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ioutil.TempFile("", "remotesconfig")
			require.NoError(t, err)
			defer os.Remove(f.Name())

			_, err = f.Write([]byte(tt.config))
			require.NoError(t, err)

			var r PeersReader
			if tt.name == "toml" {
				r = tomlPeersReader{file: f.Name()}
			} else if tt.name == "json" {
				r = jsonPeersReader{file: f.Name()}
			}

			got, err := r.Read()
			require.NoError(t, err)

			want := []*Peer{
				{
					Name:       "node1",
					PrivManKey: "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=",
					RpcUrl:     "http://localhost:8081",
				},
				{
					Name:       "node2",
					PrivManKey: "QfeDAys9MPDs2XHExtc84jKGHxZg/aj52DTh0vtA3Xc=",
					RpcUrl:     "http://localhost:8082",
				},
			}

			// dereference is required for require.Contains
			gotDeref := make([]Peer, len(got))
			for i := range got {
				gotDeref[i] = *got[i]
			}

			require.Len(t, got, 2)
			require.Contains(t, gotDeref, *want[0])
			require.Contains(t, gotDeref, *want[1])
		})
	}
}
