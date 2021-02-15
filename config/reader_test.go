package config

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestNewNodeHibernatorReader(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		wantImpl interface{}
	}{
		{
			name:     "toml",
			file:     "conf.toml",
			wantImpl: tomlNodeHibernatorReader{},
		},
		{
			name:     "json",
			file:     "conf.json",
			wantImpl: jsonNodeHibernatorReader{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewNodeHibernatorReader(tt.file)
			require.IsType(t, tt.wantImpl, r)
			require.NoError(t, err)
		})
	}
}

func TestNewNodeHibernatorReader_UnsupportedFileFormat(t *testing.T) {
	_, err := NewNodeHibernatorReader("conf.yaml")
	require.EqualError(t, err, "unsupported config file format")
}

func TestNodeHibernatorReader_Read(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "toml",
			config: `
name = "node1"
upcheckPollingInterval = 1
peersConfigFile = "./test/shell/nh1.toml"
inactivityTime = 60
disableStrictMode = true

proxies = [
    { name = "geth-rpc", type = "http", proxyAddress = "localhost:9091", upstreamAddress = "http://localhost:22000", proxyPaths = ["/"], readTimeout = 15, writeTimeout = 15 },
    { name = "geth-graphql", type = "http", proxyAddress = "localhost:9191", upstreamAddress = "http://localhost:8547/graphql", proxyPaths = ["/graphql"], readTimeout = 15, writeTimeout = 15 },
    { name = "geth-ws", type = "ws", proxyAddress = "localhost:9291", upstreamAddress = "ws://localhost:23000", proxyPaths = ["/"], readTimeout = 15, writeTimeout = 15 },
    { name = "tessera", type = "http", proxyAddress = "localhost:9391", upstreamAddress = "http://127.0.0.1:9001", proxyPaths = ["/version", "/upcheck", "/resend", "/push", "/partyinfo", "/partyinfo-mirror", "/partyinfo/validate"], readTimeout = 15, writeTimeout = 15 },
]

[server]
rpcAddress = "localhost:8081"
rpcCorsList = ["*"]
rpcvHosts = ["*"]

[blockchainClient]
type = "goquorum"
consensus = "raft"
rpcUrl = "http://localhost:22000"

[blockchainClient.process]
name = "bcclnt"
controlType = "shell"
stopCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopNode.sh", "22000"]
startCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startNode.sh", "1"]
upcheckConfig = { url = "http://localhost:22000", method = "POST", body = "{\"jsonrpc\":\"2.0\", \"method\":\"eth_blockNumber\", \"params\":[], \"id\":67}",returnType = "rpcresult"}

[privacyManager]
publicKey = "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8="

[privacyManager.process]
name = "privman"
controlType = "shell"
stopCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopTessera.sh", "2"]
startCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startTessera.sh", "2"]
upcheckConfig = { url = "http://localhost:9001/upcheck", method = "GET", body = "", returnType = "string", expected = "I'm up!"}
`,
		},
		{
			name: "json",
			config: `
{
	"name": "node1",
	"upcheckPollingInterval": 1,
	"peersConfigFile": "./test/shell/nh1.toml",
	"inactivityTime": 60,
	"disableStrictMode": true,
	"proxies": [
		{ "name": "geth-rpc", "type": "http", "proxyAddress": "localhost:9091", "upstreamAddress": "http://localhost:22000", "proxyPaths": ["/"], "readTimeout": 15, "writeTimeout": 15 },
		{ "name": "geth-graphql", "type": "http", "proxyAddress": "localhost:9191", "upstreamAddress": "http://localhost:8547/graphql", "proxyPaths": ["/graphql"], "readTimeout": 15, "writeTimeout": 15 },
		{ "name": "geth-ws", "type": "ws", "proxyAddress": "localhost:9291", "upstreamAddress": "ws://localhost:23000", "proxyPaths": ["/"], "readTimeout": 15, "writeTimeout": 15 },
		{ "name": "tessera", "type": "http", "proxyAddress": "localhost:9391", "upstreamAddress": "http://127.0.0.1:9001", "proxyPaths": ["/version", "/upcheck", "/resend", "/push", "/partyinfo", "/partyinfo-mirror", "/partyinfo/validate"], "readTimeout": 15, "writeTimeout": 15 }
	],
	"server": {
		"rpcAddress": "localhost:8081",
		"rpcCorsList": ["*"],
		"rpcvHosts": ["*"]
	},
	"blockchainClient": {
		"type": "goquorum",
		"consensus": "raft",
		"rpcUrl": "http://localhost:22000",
		"process": {
			"name": "bcclnt",
			"controlType": "shell",
			"stopCommand": ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopNode.sh", "22000"],
			"startCommand": ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startNode.sh", "1"],
			"upcheckConfig": { 
				"url": "http://localhost:22000", 
				"method": "POST", 
				"body": "{\"jsonrpc\":\"2.0\", \"method\":\"eth_blockNumber\", \"params\":[], \"id\":67}", 
				"returnType": "rpcresult"
			}	
		}
	},
	"privacyManager": {
		"publicKey": "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=",
		"process": {
			"name": "privman",
			"controlType": "shell",
			"stopCommand": ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopTessera.sh", "2"],
			"startCommand": ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startTessera.sh", "2"],
			"upcheckConfig": { 
				"url": "http://localhost:9001/upcheck", 
				"method": "GET", 
				"body": "",
				"returnType": "string", 
				"expected": "I'm up!"
			}
		}
	}
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ioutil.TempFile("", "qnhconfig")
			require.NoError(t, err)
			defer os.Remove(f.Name())

			_, err = f.Write([]byte(tt.config))
			require.NoError(t, err)

			var r NodeHibernatorReader
			if tt.name == "toml" {
				r = tomlNodeHibernatorReader{file: f.Name()}
			} else if tt.name == "json" {
				r = jsonNodeHibernatorReader{file: f.Name()}
			}
			got, err := r.Read()
			require.NoError(t, err)

			want := Basic{
				Name:                 "node1",
				UpchkPollingInterval: 1,
				PeersConfigFile:      "./test/shell/nh1.toml",
				InactivityTime:       60,
				DisableStrictMode:    true,
				Server: &RPCServer{
					RPCAddr:     "localhost:8081",
					RPCCorsList: []string{"*"},
					RPCVHosts:   []string{"*"},
				},
				BlockchainClient: &BlockchainClient{
					ClientType:   "goquorum",
					Consensus:    "raft",
					BcClntRpcUrl: "http://localhost:22000",
					//BcClntTLSConfig: nil,
					BcClntProcess: &Process{
						Name:         "bcclnt",
						ControlType:  "shell",
						ContainerId:  "",
						StopCommand:  []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopNode.sh", "22000"},
						StartCommand: []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startNode.sh", "1"},
						UpcheckCfg: &Upcheck{
							UpcheckUrl: "http://localhost:22000",
							Method:     "POST",
							Body:       "{\"jsonrpc\":\"2.0\", \"method\":\"eth_blockNumber\", \"params\":[], \"id\":67}",
							ReturnType: "rpcresult",
						},
					},
				},
				PrivacyManager: &PrivacyManager{
					PrivManKey: "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=",
					//PrivManTLSConfig: nil,
					PrivManProcess: &Process{
						Name:         "privman",
						ControlType:  "shell",
						ContainerId:  "",
						StopCommand:  []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopTessera.sh", "2"},
						StartCommand: []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startTessera.sh", "2"},
						UpcheckCfg: &Upcheck{
							UpcheckUrl: "http://localhost:9001/upcheck",
							Method:     "GET",
							Body:       "",
							ReturnType: "string",
							Expected:   "I'm up!",
						},
					},
				},
				Proxies: []*Proxy{
					{
						Name:         "geth-rpc",
						Type:         "http",
						ProxyAddr:    "localhost:9091",
						UpstreamAddr: "http://localhost:22000",
						ProxyPaths:   []string{"/"},
						ReadTimeout:  15,
						WriteTimeout: 15,
					},
					{
						Name:         "geth-graphql",
						Type:         "http",
						ProxyAddr:    "localhost:9191",
						UpstreamAddr: "http://localhost:8547/graphql",
						ProxyPaths:   []string{"/graphql"},
						ReadTimeout:  15,
						WriteTimeout: 15,
					},
					{
						Name:         "geth-ws",
						Type:         "ws",
						ProxyAddr:    "localhost:9291",
						UpstreamAddr: "ws://localhost:23000",
						ProxyPaths:   []string{"/"},
						ReadTimeout:  15,
						WriteTimeout: 15,
					},
					{
						Name:         "tessera",
						Type:         "http",
						ProxyAddr:    "localhost:9391",
						UpstreamAddr: "http://127.0.0.1:9001",
						ProxyPaths:   []string{"/version", "/upcheck", "/resend", "/push", "/partyinfo", "/partyinfo-mirror", "/partyinfo/validate"},
						ReadTimeout:  15,
						WriteTimeout: 15,
					},
				},
			}

			require.Equal(t, want, got)
		})
	}
}

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
